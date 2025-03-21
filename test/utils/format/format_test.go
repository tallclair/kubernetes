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

package format_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/utils/format"
)

func TestFormatObject(t *testing.T) {
	for name, test := range map[string]struct {
		value       interface{}
		expected    string
		indentation uint
	}{
		"int":            {value: 1, expected: `<int>: 1`},
		"string":         {value: "hello world", expected: `<string>: "hello world"`},
		"struct":         {value: myStruct{a: 1, b: 2}, expected: `<format_test.myStruct>: {a: 1, b: 2}`},
		"gomegastringer": {value: typeWithGomegaStringer(2), expected: `<format_test.typeWithGomegaStringer>: my stringer 2`},
		"pod": {value: v1.Pod{}, expected: `<v1.Pod>: 
    metadata:
      creationTimestamp: null
    spec:
      containers: null
    status: {}`},
		"pod-indented": {value: v1.Pod{}, indentation: 1, expected: `    <v1.Pod>: 
        metadata:
          creationTimestamp: null
        spec:
          containers: null
        status: {}`},
		"pod-ptr": {value: &v1.Pod{}, expected: `<*v1.Pod | <hex>>: 
    metadata:
      creationTimestamp: null
    spec:
      containers: null
    status: {}`},
		"pod-hash": {value: map[string]v1.Pod{}, expected: `<map[string]v1.Pod | len:0>: 
    {}`},
		"podlist": {value: v1.PodList{}, expected: `<v1.PodList>: 
    items: null
    metadata: {}`},
	} {
		t.Run(name, func(t *testing.T) {
			actual := format.Object(test.value, test.indentation)
			actual = regexp.MustCompile(`\| 0x[a-z0-9]+`).ReplaceAllString(actual, `| <hex>`)
			assert.Equal(t, test.expected, actual)

			actualKObj := format.KObject(test.value, format.Indentation(test.indentation))
			actualKObj = regexp.MustCompile(`\| 0x[a-z0-9]+`).ReplaceAllString(actual, `| <hex>`)
			assert.Equal(t, test.expected, actualKObj)
		})
	}
}

func TestFormatKObject(t *testing.T) {
	for _, test := range []struct {
		name     string
		value    any
		expected string
		opts     []format.Option
	}{{
		name:  "empty-pod",
		value: &v1.Pod{},
		opts:  []format.Option{format.Indentation(0)}, // Suppress default options.
		expected: `<*v1.Pod | <hex>>: 
    metadata:
      creationTimestamp: null
    spec:
      containers: null
    status: {}`,
	}, {
		name:  "pod-indented",
		value: v1.Pod{},
		opts:  []format.Option{format.Indentation(1)},
		expected: `    <v1.Pod>: 
        metadata:
          creationTimestamp: null
        spec:
          containers: null
        status: {}`,
	}, {
		name:  "pod-ptr",
		value: &v1.Pod{},
		opts:  []format.Option{format.Indentation(0)}, // Suppress default options.
		expected: `<*v1.Pod | <hex>>: 
    metadata:
      creationTimestamp: null
    spec:
      containers: null
    status: {}`,
	}, {
		name:  "podlist",
		value: v1.PodList{},
		opts:  []format.Option{format.Indentation(0)}, // Suppress default options.
		expected: `<v1.PodList>: 
    items: null
    metadata: {}`,
	}, {
		name:  "pod-with-managed-fields",
		value: podWithManagedFields(),
		opts:  []format.Option{format.Indentation(0)}, // Suppress default options.
		expected: `<*v1.Pod | <hex>>: 
    metadata:
      creationTimestamp: null
      managedFields:
      - apiVersion: v1
        fieldsType: FieldsV1
        fieldsV1:
          f:spec:
            f:hostNetwork: {}
        manager: manager-1
        operation: Apply
      - apiVersion: v1
        fieldsType: FieldsV1
        fieldsV1:
          f:spec:
            f:restartPolicy: {}
        manager: manager-2
        operation: Apply
      name: managed-pod
    spec:
      containers: null
    status: {}`,
	}, {
		name:  "pod-with-managed-fields-suppressed",
		value: podWithManagedFields(),
		opts:  []format.Option{format.SuppressManagedFields()},
		expected: `<*v1.Pod | <hex>>: 
    metadata:
      creationTimestamp: null
      name: managed-pod
    spec:
      containers: null
    status: {}`,
	}, {
		name:  "podlist-default-opts",
		value: &v1.PodList{Items: []v1.Pod{*podWithManagedFields()}},
		expected: `    <*v1.PodList | <hex>>: 
        items:
        - metadata:
            creationTimestamp: null
            name: managed-pod
          spec:
            containers: null
          status: {}
        metadata: {}`,
	}, {
		name:  "pod-default-opts",
		value: podWithManagedFields(),
		expected: `    <*v1.Pod | <hex>>: 
        metadata:
          creationTimestamp: null
          name: managed-pod
        spec:
          containers: null
        status: {}`,
	}} {
		t.Run(test.name, func(t *testing.T) {
			actual := format.KObject(test.value, test.opts...)
			actual = regexp.MustCompile(`\| 0x[a-z0-9]+`).ReplaceAllString(actual, `| <hex>`)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestFormatKObjects(t *testing.T) {
	values := []v1.Pod{*podWithManagedFields(), v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other-pod"}}}
	expected := `    <[]v1.Pod | len:2, cap:2>: 
        - metadata:
            creationTimestamp: null
            name: managed-pod
          spec:
            containers: null
          status: {}
        - metadata:
            creationTimestamp: null
            name: other-pod
          spec:
            containers: null
          status: {}`

	actual := format.KObjects(values)
	assert.Equal(t, expected, actual)

	ptrValues := []*v1.Pod{podWithManagedFields(), &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other-pod"}}}
	expected = `    <[]*v1.Pod | len:2, cap:2>: 
        - metadata:
            creationTimestamp: null
            name: managed-pod
          spec:
            containers: null
          status: {}
        - metadata:
            creationTimestamp: null
            name: other-pod
          spec:
            containers: null
          status: {}`

	actual = format.KObjects(ptrValues)
	assert.Equal(t, expected, actual)
}

func podWithManagedFields() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "managed-pod",
			ManagedFields: []metav1.ManagedFieldsEntry{{
				Manager:    "manager-1",
				Operation:  metav1.ManagedFieldsOperationApply,
				APIVersion: "v1",
				FieldsType: "FieldsV1",
				FieldsV1:   &metav1.FieldsV1{Raw: []byte(`{"f:spec":{"f:hostNetwork":{}}}`)},
			}, {
				Manager:    "manager-2",
				Operation:  metav1.ManagedFieldsOperationApply,
				APIVersion: "v1",
				FieldsType: "FieldsV1",
				FieldsV1:   &metav1.FieldsV1{Raw: []byte(`{"f:spec":{"f:restartPolicy":{}}}`)},
			}},
		},
	}
}

type typeWithGomegaStringer int

func (v typeWithGomegaStringer) GomegaString() string {
	return fmt.Sprintf("my stringer %d", v)
}

type myStruct struct {
	a, b int
}
