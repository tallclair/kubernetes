/*
Copyright 2025 The Kubernetes Authors.

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
	"iter"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type IndexedPod struct {
	*v1.Pod
	Status *PodStatus

	// IndexedContainers maintains the set of indexed containers, ordered as follows:
	// init containers, main containers, ephemeral containers
	IndexedContainers []IndexedContainer
}

type IndexedContainer struct {
	*v1.Container

	// The type of this container.
	Type ContainerType

	// Index is the index this container is found at in the pod spec,
	// within the slice for the corresponding ContainerType
	Index int

	Status *Status
}

type ContainerType int

const (
	MainContainer ContainerType = 1 << iota
	InitContainer
	EphemeralContainer
)

func IndexPod(pod *v1.Pod, status *PodStatus) *IndexedPod {
	indexedPod := &IndexedPod{
		Pod:    pod,
		Status: status,
	}

	for i := range pod.Spec.InitContainers {
		indexedPod.IndexedContainers = append(indexedPod.IndexedContainers,
			indexContainer(i, &pod.Spec.InitContainers[i], InitContainer, status))
	}
	for i := range pod.Spec.Containers {
		indexedPod.IndexedContainers = append(indexedPod.IndexedContainers,
			indexContainer(i, &pod.Spec.Containers[i], MainContainer, status))
	}
	for i := range pod.Spec.EphemeralContainers {
		indexedPod.IndexedContainers = append(indexedPod.IndexedContainers,
			indexContainer(i, (*v1.Container)(&pod.Spec.EphemeralContainers[i].EphemeralContainerCommon), EphemeralContainer, status))
	}

	return indexedPod
}

func indexContainer(index int, c *v1.Container, cType ContainerType, status *PodStatus) IndexedContainer {
	return IndexedContainer{
		Container: c,
		Type:      cType,
		Index:     index,
		Status:    status.FindContainerStatusByName(c.Name),
	}
}

func (pod *IndexedPod) Ref() klog.ObjectRef {
	return klog.KRef(pod.Namespace, pod.Name)
}

func (pod *IndexedPod) Containers(typeMask ContainerType) iter.Seq2[*IndexedContainer, ContainerType] {
	return func(yield func(*IndexedContainer, ContainerType) bool) {
		for _, c := range pod.IndexedContainers {
			if typeMask&c.Type != 0 {
				if !yield(&c, c.Type) {
					return
				}
			}
		}
	}
}
