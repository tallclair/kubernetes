/*
Copyright 2015 The Kubernetes Authors.

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

package container

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func TestPodSyncResult(t *testing.T) {
	okResults := []*SyncResult{
		NewSyncResult(StartContainer, "container_0"),
		NewSyncResult(SetupNetwork, "pod"),
	}
	errResults := []*SyncResult{
		NewSyncResult(KillContainer, "container_1"),
		NewSyncResult(TeardownNetwork, "pod"),
	}
	errResults[0].Fail(errors.New("error_0"), "message_0")
	errResults[1].Fail(errors.New("error_1"), "message_1")

	// If the PodSyncResult doesn't contain error result, it should not be error
	result := PodSyncResult{}
	result.AddSyncResult(okResults...)
	if result.Error() != nil {
		t.Errorf("PodSyncResult should not be error: %v", result)
	}

	// If the PodSyncResult contains error result, it should be error
	result = PodSyncResult{}
	result.AddSyncResult(okResults...)
	result.AddSyncResult(errResults...)
	if result.Error() == nil {
		t.Errorf("PodSyncResult should be error: %v", result)
	}

	// If the PodSyncResult is failed, it should be error
	result = PodSyncResult{}
	result.AddSyncResult(okResults...)
	result.Fail(errors.New("error"))
	if result.Error() == nil {
		t.Errorf("PodSyncResult should be error: %v", result)
	}

	// If the PodSyncResult is added an error PodSyncResult, it should be error
	errResult := PodSyncResult{}
	errResult.AddSyncResult(errResults...)
	result = PodSyncResult{}
	result.AddSyncResult(okResults...)
	result.AddPodSyncResult(errResult)
	if result.Error() == nil {
		t.Errorf("PodSyncResult should be error: %v", result)
	}
}

func TestMinBackoff(t *testing.T) {
	start := time.Now()
	backoffErr := func(d time.Duration) *BackoffError {
		return NewBackoffError("backoff test", start.Add(d))
	}
	tests := []struct {
		name          string
		err           error
		expectBackoff bool
		expectMin     time.Duration
	}{{
		name: "no backoff",
		err:  errors.New("unrelated error"),
	}, {
		name:          "simple backoff",
		err:           backoffErr(time.Minute),
		expectBackoff: true,
		expectMin:     time.Minute,
	}, {
		name:          "wrapped backoff",
		err:           fmt.Errorf("wrapped: %w", backoffErr(time.Minute)),
		expectBackoff: true,
		expectMin:     time.Minute,
	}, {
		name:          "aggregated backoff",
		err:           utilerrors.NewAggregate([]error{errors.New("unrelated"), backoffErr(time.Hour), backoffErr(time.Minute), backoffErr(time.Hour)}),
		expectBackoff: true,
		expectMin:     time.Minute,
	}, {
		name: "aggregated recursive",
		err: utilerrors.NewAggregate([]error{
			errors.New("unrelated"),
			utilerrors.NewAggregate([]error{
				backoffErr(time.Hour),
				fmt.Errorf("wrapped: %w", backoffErr(time.Minute)),
				backoffErr(time.Hour),
			}),
		}),
		expectBackoff: true,
		expectMin:     time.Minute,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backoffTime, ok := MinBackoff(test.err)
			require.Equal(t, test.expectBackoff, ok)
			if ok {
				actual := backoffTime.Sub(start)
				assert.Equal(t, test.expectMin, actual)
			}
		})
	}
}
