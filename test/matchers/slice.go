/*
Copyright 2016 The Kubernetes Authors.

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

package matchers

import (
	"fmt"
	"reflect"

	"k8s.io/kubernetes/pkg/util/errors"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type SliceMatcher struct {
	// Matchers for each element.
	Elements Elements
	// Whether missing or extra elements are considered an error.
	Strict bool
	// Function for identifying a slice element.
	Identifier Identifier
}

// Element ID to matcher.
type Elements map[string]types.GomegaMatcher

// Function for identifying elements of a slice.
type Identifier func(element interface{}) string

func (m *SliceMatcher) Match(actual interface{}) (success bool, err error) {
	if reflect.TypeOf(actual).Kind() != reflect.Slice {
		return false, fmt.Errorf("%v is type %T, expected slice", actual, actual)
	}

	errs := m.matchElements(actual)
	if len(errs) > 0 {
		return false, errors.NewAggregate(errs)
	}
	return true, nil
}

func (m *SliceMatcher) matchElements(actual interface{}) (errs []error) {
	// Provide more useful error messages in the case of a panic.
	defer func() {
		if err := recover(); err != nil {
			errs = append(errs, fmt.Errorf("panic checking %v: %v", actual, err))
		}
	}()

	val := reflect.ValueOf(actual)
	elements := map[string]bool{}
	for i := 0; i < val.Len(); i++ {
		element := val.Index(i).Interface()
		id := m.Identifier(element)
		if elements[id] {
			errs = append(errs, fmt.Errorf("found duplicate element ID %s", id))
			continue
		}
		elements[id] = true

		matcher, expected := m.Elements[id]
		if !expected {
			if m.Strict {
				errs = append(errs, fmt.Errorf("unexpected element %s", id))
			}
			continue
		}

		match, err := matcher.Match(element)
		if match {
			continue
		}

		errs = append(errs, Nest(fmt.Sprintf("[%s]", id), err))
	}

	if m.Strict {
		for id := range m.Elements {
			if !elements[id] {
				errs = append(errs, fmt.Errorf("missing expected element %s", id))
			}
		}
	}

	return errs
}

func (m *SliceMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to match slice matcher")
}

func (m *SliceMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to match slice matcher")
}
