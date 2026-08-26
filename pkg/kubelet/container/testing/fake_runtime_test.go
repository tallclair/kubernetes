/*
Copyright The Kubernetes Authors.

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

package testing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/flowcontrol"
)

func TestFakeRuntime_SyncPod_WithoutRuntimeHelper(t *testing.T) {
	fakeRuntime := &FakeRuntime{T: t}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("test-pod-1"),
			Name: "test-pod",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "c1"},
			},
		},
	}

	result := fakeRuntime.SyncPod(context.Background(), pod, nil, nil, flowcontrol.NewBackOff(0, 0), false)
	require.NoError(t, result.Error())
	assert.True(t, fakeRuntime.AssertCalls([]string{"SyncPod"}))
	assert.True(t, fakeRuntime.AssertStartedPods([]string{"test-pod-1"}))
	assert.True(t, fakeRuntime.AssertStartedContainers([]string{"c1"}))
}

func TestFakeRuntime_SyncPod_WithRuntimeHelper_Success(t *testing.T) {
	fakeHelper := &FakeRuntimeHelper{}
	fakeRuntime := &FakeRuntime{
		RuntimeHelper: fakeHelper,
		T:             t,
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("test-pod-2"),
			Name: "test-pod",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "c1"},
			},
		},
	}

	result := fakeRuntime.SyncPod(context.Background(), pod, nil, nil, flowcontrol.NewBackOff(0, 0), false)
	require.NoError(t, result.Error())
	assert.True(t, fakeRuntime.AssertCalls([]string{"SyncPod"}))
	assert.True(t, fakeRuntime.AssertStartedPods([]string{"test-pod-2"}))
}

func TestFakeRuntime_SyncPod_WithRuntimeHelper_WaitForAttachAndMountError(t *testing.T) {
	expectedErr := errors.New("mount failed")
	fakeHelper := &FakeRuntimeHelper{
		WaitForAttachAndMountErr: expectedErr,
	}
	fakeRuntime := &FakeRuntime{
		RuntimeHelper: fakeHelper,
		T:             t,
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("test-pod-3"),
			Name: "test-pod",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "c1"},
			},
		},
	}

	result := fakeRuntime.SyncPod(context.Background(), pod, nil, nil, flowcontrol.NewBackOff(0, 0), false)
	require.Error(t, result.Error())
	assert.ErrorIs(t, result.SyncError, expectedErr)
}
