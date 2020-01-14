/*
Copyright 2019 The Kubernetes Authors.

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

package node

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/node/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclasstest "k8s.io/kubernetes/pkg/kubelet/runtimeclass/testing"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/e2e/scheduling"
	imageutils "k8s.io/kubernetes/test/utils/image"
	utilpointer "k8s.io/utils/pointer"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("[sig-node] RuntimeClass", func() {
	f := framework.NewDefaultFramework("runtimeclass")

	ginkgo.It("should reject a Pod requesting a RuntimeClass with conflicting node selector", func() {
		scheduling := &v1beta1.Scheduling{
			NodeSelector: map[string]string{
				"foo": "conflict",
			},
		}

		runtimeClass := newRuntimeClass(f.Namespace.Name, "conflict-runtimeclass")
		runtimeClass.Scheduling = scheduling
		rc, err := f.ClientSet.NodeV1beta1().RuntimeClasses().Create(runtimeClass)
		framework.ExpectNoError(err, "failed to create RuntimeClass resource")

		pod := newRuntimeClassPod(rc.GetName())
		pod.Spec.NodeSelector = map[string]string{
			"foo": "bar",
		}
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		framework.ExpectError(err, "should be forbidden")
		gomega.Expect(apierrs.IsForbidden(err)).To(gomega.BeTrue(), "should be forbidden error")
	})

	ginkgo.It("should run a Pod requesting a RuntimeClass with NodeSelector & Tolerations [NodeFeature:RuntimeHandler] [Disruptive]", func() {
		testRuntimeClassScheduling(f, true)
	})

	ginkgo.It("should run a Pod requesting a RuntimeClass with NodeSelector [NodeFeature:RuntimeHandler]", func() {
		testRuntimeClassScheduling(f, false)
	})
})

func testRuntimeClassScheduling(f *framework.Framework, testTaints bool) {
	nodeName := scheduling.GetNodeThatCanRunPod(f)
	scheduling := &v1beta1.Scheduling{}

	scheduling.NodeSelector = map[string]string{
		"test-runtimeclass-ns":         f.Namespace.Name,
		"test-runtimeclass-scheduling": fmt.Sprintf("testTaints=%t", testTaints),
	}

	ginkgo.By("Trying to apply a label on the found node.")
	for key, value := range scheduling.NodeSelector {
		framework.AddOrUpdateLabelOnNode(f.ClientSet, nodeName, key, value)
		framework.ExpectNodeHasLabel(f.ClientSet, nodeName, key, value)
		defer framework.RemoveLabelOffNode(f.ClientSet, nodeName, key)
	}

	if testTaints {
		taint := v1.Taint{
			Key:    "test-runtimeclass-ns",
			Value:  f.Namespace.Name,
			Effect: v1.TaintEffectNoSchedule,
		}
		scheduling.Tolerations = []v1.Toleration{{
			Key:      taint.Key,
			Operator: v1.TolerationOpEqual,
			Value:    taint.Value,
			Effect:   taint.Effect,
		}}

		ginkgo.By("Trying to apply taint on the found node.")
		framework.AddOrUpdateTaintOnNode(f.ClientSet, nodeName, taint)
		framework.ExpectNodeHasTaint(f.ClientSet, nodeName, &taint)
		defer framework.RemoveTaintOffNode(f.ClientSet, nodeName, taint)
	}

	ginkgo.By("Trying to create runtimeclass and pod")
	runtimeClass := newRuntimeClass(f.Namespace.Name, "non-conflict-runtimeclass")
	runtimeClass.Scheduling = scheduling
	rc, err := f.ClientSet.NodeV1beta1().RuntimeClasses().Create(runtimeClass)
	framework.ExpectNoError(err, "failed to create RuntimeClass resource")

	pod := newRuntimeClassPod(rc.GetName())
	pod.Spec.NodeSelector = map[string]string{
		"foo": "bar",
	}
	pod = f.PodClient().Create(pod)
	framework.ExpectNoError(e2epod.WaitForPodSuccessInNamespace(f.ClientSet, pod.Name, f.Namespace.Name))

	// check that pod got scheduled on specified node.
	scheduledPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(pod.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)
	framework.ExpectEqual(nodeName, scheduledPod.Spec.NodeName)

	expectedLabels := map[string]string{
		"foo": "bar",
	}
	for k, v := range scheduling.NodeSelector {
		expectedLabels[k] = v
	}
	framework.ExpectEqual(expectedLabels, pod.Spec.NodeSelector)

	var expectedTolerations []v1.Toleration
	if testTaints {
		expectedTolerations = append(expectedTolerations, scheduling.Tolerations...)
	}
	framework.ExpectEqual(expectedTolerations, pod.Spec.Tolerations)
}

// newRuntimeClass returns a test runtime class.
func newRuntimeClass(namespace, name string) *v1beta1.RuntimeClass {
	uniqueName := fmt.Sprintf("%s-%s", namespace, name)
	return runtimeclasstest.NewRuntimeClass(uniqueName, framework.PreconfiguredRuntimeClassHandler())
}

// newRuntimeClassPod returns a test pod with the given runtimeClassName.
func newRuntimeClassPod(runtimeClassName string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("test-runtimeclass-%s-", runtimeClassName),
		},
		Spec: v1.PodSpec{
			RuntimeClassName: &runtimeClassName,
			Containers: []v1.Container{{
				Name:    "test",
				Image:   imageutils.GetE2EImage(imageutils.BusyBox),
				Command: []string{"true"},
			}},
			RestartPolicy:                v1.RestartPolicyNever,
			AutomountServiceAccountToken: utilpointer.BoolPtr(false),
		},
	}
}
