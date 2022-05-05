/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wait

import (
	"fmt"
	"reflect"
	"time"

	"github.com/onsi/gomega/format"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
)

const (
	defaultPollPeriod = 2 * time.Second
	defaultTimeout    = 5 * time.Minute
)

type Opts struct {
	PollPeriod time.Duration
	Timeout    time.Duration

	// RetryNotFound indicates whether NotFound errors should be treated as retryable (e.g. waiting for a resource to appear).
	RetryNotFound bool
	// DisableRetries causes all errors returned by the objectFetcher function to immediately exit the polling loop.
	DisableRetries bool
}

func (o *Opts) complete() {
	if o.PollPeriod == 0 {
		o.PollPeriod = defaultPollPeriod
	}
	if o.Timeout == 0 {
		o.Timeout = defaultTimeout
	}
}

func ForObjectCondition[O any](
	objectIdentifier string, objectFetcher func() (O, error),
	conditionDesc string, condition func(O) (bool, error),
	opts Opts) error {
	opts.complete()
	e2elog.Logf("Waiting up to %v for %s to be %s", opts.Timeout, objectIdentifier, conditionDesc)
	var (
		lastFetchError error
		lastObj        O
		fetched        = false // Whether the object was successfully fetched.
		start          = time.Now()
		end            = start.Add(opts.Timeout)
	)
	err := wait.PollImmediate(opts.PollPeriod, opts.Timeout, func() (bool, error) {
		obj, err := objectFetcher()
		lastFetchError = err
		if err != nil {
			if retry, delay := shouldRetry(err, opts); retry {
				e2elog.Logf("Retryable error while fetching %s", objectIdentifier)
				if time.Now().Add(delay).After(end) {
					// Retry is after timeout, just exit early.
					return false, wait.ErrWaitTimeout
				}
				if delay > 0 {
					time.Sleep(delay)
				}
				return false, nil
			}
			e2elog.Logf("Encountered non-retryable error while fetching %s: %v", objectIdentifier, err)
			return false, err
		}
		fetched = true
		lastObj = obj // Don't overwrite if an error occurs after successfully retrieving.

		return condition(obj)
	})

	if IsTimeout(err) {
		if fetched {
			e2elog.Logf("Timed out while waiting for %s to be %s. It was never successfully fetched.", objectIdentifier, conditionDesc)
		} else {
			e2elog.Logf("Timed out while waiting for %s to be %s. Last observed as: %s",
				objectIdentifier, conditionDesc, dumpObject(lastObj))
		}
		if lastFetchError != nil {
			// If the last API call was an error, return that instead of a timeout.
			return lastFetchError
		}
		return TimeoutError("timed out while waiting for %s to be %s ", objectIdentifier, conditionDesc)
	} else if err != nil {
		return fmt.Errorf("error while waiting for %s to be %s: %w", objectIdentifier, conditionDesc, err)
	} else {
		return nil
	}
}

type ListOpts struct {
	Opts

	// MinObjects is the minimum number of items which must be fetched to pass.
	MinObjects int
	// MaxObjects is the maximum number of items which must be fetched to pass.
	// If MaxObjects is nil, then there is no maximum.
	MaxObjects *int
	// MinMatching is the minimum number of items which must match the condition to pass.
	// If MinMatching is 0, then all items must match.
	MinMatching int
}

func ForObjectsCondition[O runtime.Object](
	listIdentifier string, objectsFetcher func() ([]O, error),
	conditionDesc string, condition func(O) (bool, error),
	opts ListOpts) error {
	bulkCondition := func(objs []O) (bool, error) {
		if len(objs) < opts.MinObjects || len(objs) < opts.MinMatching {
			return false, nil
		}
		if opts.MaxObjects != nil && len(objs) > *opts.MaxObjects {
			return false, nil
		}
		matching := 0
		for _, obj := range objs {
			if matched, err := condition(obj); err != nil {
				return false, err
			} else if matched {
				matching++
			}
		}
		done := matching == len(objs) ||
			(opts.MinMatching > 0 && matching >= opts.MinMatching)
		return done, nil
	}
	return ForObjectCondition(
		listIdentifier, objectsFetcher,
		conditionDesc, bulkCondition,
		opts.Opts)
}

type timeoutError struct {
	msg string
}

func (e *timeoutError) Error() string {
	return e.msg
}

func TimeoutError(format string, args ...interface{}) *timeoutError {
	return &timeoutError{
		msg: fmt.Sprintf(format, args...),
	}
}

func IsTimeout(err error) bool {
	if err == wait.ErrWaitTimeout {
		return true
	}
	if _, ok := err.(*timeoutError); ok {
		return true
	}
	return false
}

// Decide whether to retry an API request. Optionally include a delay to retry after.
func shouldRetry(err error, opts Opts) (retry bool, retryAfter time.Duration) {
	if opts.DisableRetries {
		return false, 0
	}

	// if the error sends the Retry-After header, we respect it as an explicit confirmation we should retry.
	if delay, shouldRetry := apierrors.SuggestsClientDelay(err); shouldRetry {
		return shouldRetry, time.Duration(delay) * time.Second
	}

	// these errors indicate a transient error that should be retried.
	if apierrors.IsTimeout(err) || apierrors.IsTooManyRequests(err) ||
		(apierrors.IsNotFound(err) && opts.RetryNotFound) {
		return true, 0
	}
	return false, 0
}

func dumpObject(obj any) string {
	if t, err := meta.TypeAccessor(obj); err == nil {
		if _, err := meta.ListAccessor(obj); err == nil {
			// If obj is a list type, just output the number of items rather than dumping the full list.
			v := reflect.ValueOf(obj)
			for ; v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer; v = v.Elem() {
			}
			if items := v.FieldByName("Items"); items.IsValid() {
				return fmt.Sprintf("%s %s with %d items", t.GetAPIVersion(), t.GetKind(), items.Len())
			}
		}
	}
	return format.Object(obj, 1)
}
