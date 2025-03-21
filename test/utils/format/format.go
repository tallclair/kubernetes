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

// Package format is an extension of Gomega's format package which
// improves printing of objects that can be serialized well as YAML,
// like the structs in the Kubernetes API.
//
// Just importing it is enough to activate this special YAML support
// in Gomega.
package format

import (
	"reflect"
	"strings"

	"github.com/onsi/gomega/format"

	"k8s.io/apimachinery/pkg/api/meta"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func init() {
	format.RegisterCustomFormatter(handleYAML)
}

// Object makes Gomega's [format.Object] available without having to import that
// package.
func Object(object interface{}, indentation uint) string {
	return format.Object(object, indentation)
}

// KObject pretty-prints a Kubernetes object according to the provided options.
// If no options are provided, the default set of options are used.
func KObject[T any, PT interface{ *T }](object T, opts ...Option) string {
	o := makeOptions(opts...)

	if _, ok := any(object).(runtime.Object); ok { // If T implements runtime.Object
		object = processOptions(any(object).(runtime.Object), o).(T)
	} else if _, ok := any(object).(runtime.Object); ok { // If *T implements runtime.Object
		object = *(processOptions(any(&object).(runtime.Object), o).(PT))
	}
	return Object(object, o.indentation)
}

// KObjects works just like KObject, but for slices of Kubernetes objects.
func KObjects[T any, PT interface{ *T }](objects []T, opts ...Option) string {
	o := makeOptions(opts...)

	var t T
	if _, ok := any(t).(runtime.Object); ok { // If T implements runtime.Object
		for i, obj := range objects {
			objects[i] = processOptions(any(obj).(runtime.Object), o).(T)
		}
	} else if _, ok := any(&t).(runtime.Object); ok { // If *T implements runtime.Object
		for i := range objects {
			objects[i] = *(processOptions(any(&objects[i]).(runtime.Object), o).(PT))
		}
	}
	return Object(objects, o.indentation)
}

func makeOptions(opts ...Option) *options {
	if len(opts) == 0 {
		opts = defaultOpts
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func processOptions[T runtime.Object](object T, o *options) T {
	for _, filter := range o.filters {
		object = filter(runtime.Object(object)).(T)
	}
	return object
}

type Option func(*options)
type options struct {
	indentation uint
	filters     []func(runtime.Object) runtime.Object
}

var defaultOpts = []Option{
	Indentation(1),
	SuppressManagedFields(),
}

func Indentation(indentation uint) Option {
	return func(o *options) {
		o.indentation = indentation
	}
}

func SuppressManagedFields() Option {
	return func(o *options) {
		o.filters = append(o.filters, suppressManagedFields)
	}
}

func suppressManagedFields(obj runtime.Object) runtime.Object {
	if meta.IsListType(obj) {
		obj = obj.DeepCopyObject()
		_ = meta.EachListItem(obj, func(item runtime.Object) error {
			omitManagedFields(item)
			return nil
		})
	} else if _, err := meta.Accessor(obj); err == nil {
		obj = omitManagedFields(obj.DeepCopyObject())
	}
	return obj
}

func omitManagedFields(o runtime.Object) runtime.Object {
	a, err := meta.Accessor(o)
	if err != nil {
		// The object is not a `metav1.Object`, ignore it.
		return o
	}
	a.SetManagedFields(nil)
	return o
}

// handleYAML formats all values as YAML where the result
// is likely to look better as YAML:
//   - pointer to struct or struct where all fields
//     have `json` tags
//   - slices containing such a value
//   - maps where the key or value are such a value
func handleYAML(object interface{}) (string, bool) {
	value := reflect.ValueOf(object)
	if !useYAML(value.Type()) {
		return "", false
	}
	y, err := yaml.Marshal(object)
	if err != nil {
		return "", false
	}
	return "\n" + strings.TrimSpace(string(y)), true
}

func useYAML(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return useYAML(t.Elem())
	case reflect.Map:
		return useYAML(t.Key()) || useYAML(t.Elem())
	case reflect.Struct:
		// All fields must have a `json` tag.
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if _, ok := field.Tag.Lookup("json"); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}
