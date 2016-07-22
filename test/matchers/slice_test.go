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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func TestSliceMatcher(t *testing.T) {
	allElements := []string{"a", "b"}
	missingElements := []string{"a"}
	extraElements := []string{"a", "b", "c"}
	empty := []string{}

	strict := StrictSlice(id, Elements{
		"b": gomega.Equal("b"),
		"a": gomega.Equal("a"),
	})
	strictFail := StrictSlice(id, Elements{
		"a": gomega.Equal("a"),
		"b": gomega.Equal("fail"),
	})
	strictEmpty := StrictSlice(id, Elements{})
	ignoreExtras := LooseSlice(id, IgnoreExtras, Elements{
		"b": gomega.Equal("b"),
		"a": gomega.Equal("a"),
	})
	ignoreMissing := LooseSlice(id, IgnoreMissing, Elements{
		"a": gomega.Equal("a"),
		"b": gomega.Equal("b"),
	})
	looseFail := LooseSlice(id, IgnoreExtras|IgnoreMissing, Elements{
		"a": gomega.Equal("a"),
		"b": gomega.Equal("fail"),
	})

	tests := []struct {
		actual      interface{}
		matcher     types.GomegaMatcher
		expectMatch bool
		msg         string
	}{
		{allElements, strict, true, "StrictSlice should match all elements"},
		{missingElements, strict, false, "StrictSlice should fail with missing elements"},
		{extraElements, strict, false, "StrictSlice should fail with extra elements"},
		{allElements, strictFail, false, "StrictSlice should fail with fail"},
		{empty, strictEmpty, true, "StrictSlice should handle empty slices"},
		{allElements, ignoreExtras, true, "LooseSlice 'ignoreExtras' should match all elements"},
		{missingElements, ignoreExtras, false, "LooseSlice 'ignoreExtras' should fail with missing elements"},
		{extraElements, ignoreExtras, true, "LooseSlice 'ignoreExtras' should ignore extra elements"},
		{allElements, ignoreMissing, true, "LooseSlice 'ignoreMissing' should match all elements"},
		{missingElements, ignoreMissing, true, "LooseSlice 'ignoreMissing' should ignore missing elements"},
		{extraElements, ignoreMissing, false, "LooseSlice 'ignoreMissing' should fail with extra elements"},
		{allElements, looseFail, false, "LooseSlice should fail with fail"},
	}

	for i, test := range tests {
		match, err := test.matcher.Match(test.actual)
		assert.NoError(t, err, "[%d] %s", i, test.msg)
		assert.Equal(t, test.expectMatch, match,
			"[%d] %s: %s", i, test.msg, test.matcher.FailureMessage(test.actual))
	}
}

func id(element interface{}) string {
	return element.(string)
}
