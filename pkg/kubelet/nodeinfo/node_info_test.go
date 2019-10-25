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

package nodeinfo

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
)

const (
	testNodeName = "test-node-123"
	testNodeUID  = "test-uid-12345678"
)

func TestGetNode(t *testing.T) {
	initial := fakeInitialNode{}
	getter := &fakeNodeGetter{}

	p := NewProvider(testNodeName, getter, initial.fn)

	initNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
	}
	fetchNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			UID:  testNodeUID,
		},
	}

	testCases := []struct {
		desc                    string
		init, fetched, expected *v1.Node
	}{{
		desc: "init, no cache, fail",
	}, {
		desc:     "init, no cache, success",
		init:     initNode,
		expected: initNode,
	}, {
		desc:     "init, cached, success",
		expected: initNode,
	}, {
		desc:     "fetch, cached, success",
		fetched:  fetchNode,
		expected: fetchNode,
	}, {
		desc:     "fetch, cached, fail",
		expected: initNode,
	}}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			initial.t = t
			initial.node = test.init
			getter.t = t
			getter.node = test.fetched

			actual, err := p.GetNode()
			if test.expected == nil {
				assert.Nil(t, actual)
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestGetObjectRef(t *testing.T) {
	getter := &fakeNodeGetter{}

	p := NewProvider(testNodeName, getter, nil)

	fetchNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			UID:  testNodeUID,
		},
	}

	testCases := []struct {
		desc          string
		fetched       *v1.Node
		expectSuccess bool
	}{{
		desc: "no cache, fail",
	}, {
		desc:          "no cache, success",
		fetched:       fetchNode,
		expectSuccess: true,
	}, {
		desc:          "cached, fail",
		expectSuccess: true,
	}, {
		desc:          "cached, success",
		fetched:       fetchNode,
		expectSuccess: true,
	}}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			getter.t = t
			getter.node = test.fetched

			expectedRef := &v1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       testNodeName,
				UID:        testNodeUID,
			}

			actual, err := p.GetObjectRef()
			if test.expectSuccess {
				assert.NoError(t, err)
				assert.Equal(t, expectedRef, actual)
			} else {
				assert.Error(t, err)
				assert.Nil(t, actual)
			}
		})
	}
}

func TestGetEventRef(t *testing.T) {
	getter := &fakeNodeGetter{}

	p := NewProvider(testNodeName, getter, nil)

	fetchNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			UID:  testNodeUID,
		},
	}

	testCases := []struct {
		desc        string
		fetched     *v1.Node
		expectedUID types.UID
	}{{
		desc:        "no cache, fail",
		expectedUID: types.UID(testNodeName),
	}, {
		desc:        "no cache, success",
		fetched:     fetchNode,
		expectedUID: testNodeUID,
	}, {
		desc:        "cached, fail",
		expectedUID: testNodeUID,
	}, {
		desc:        "cached, success",
		expectedUID: testNodeUID,
	}}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			getter.t = t
			getter.node = test.fetched

			expectedRef := &v1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       testNodeName,
				UID:        test.expectedUID,
			}

			actual := p.GetEventRef()
			assert.Equal(t, expectedRef, actual)
		})
	}
}

type fakeNodeGetter struct {
	t *testing.T

	node *v1.Node
}

func (f *fakeNodeGetter) Get(name string) (*v1.Node, error) {
	if name != testNodeName {
		f.t.Fatalf("Unexpected node requested: %s", name)
	}

	if f.node != nil {
		return f.node, nil
	}
	return nil, errors.New("fakeNodeGetter: no node")
}

type fakeInitialNode struct {
	t *testing.T

	node      *v1.Node
	succeeded bool
}

func (f *fakeInitialNode) fn() (*v1.Node, error) {
	if f.succeeded {
		f.t.Errorf("Expected only 1 successful call to initialNode")
	}

	if f.node != nil {
		f.succeeded = true
		return f.node, nil
	}
	return nil, errors.New("fakeInitialNode: no node")
}
