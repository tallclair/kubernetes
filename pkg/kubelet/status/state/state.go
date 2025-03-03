/*
Copyright 2021 The Kubernetes Authors.

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

package state

import (
	v1 "k8s.io/api/core/v1"
)

// PodResizeStatus type is used in tracking the last resize decision for pod
type PodResizeStatus map[string]v1.PodResizeStatus

// Reader interface used to read current pod resource allocation state
type Reader interface {
	GetContainerResourceAllocation(podUID string, containerName string) (v1.ResourceRequirements, bool)
	GetPodResourceAllocation() PodResourceAllocationInfo
	GetPodResizeStatus(podUID string) v1.PodResizeStatus
}

type writer interface {
	SetContainerResourceAllocation(podUID string, containerName string, alloc v1.ResourceRequirements) error
	SetPodResourceAllocation(PodResourceAllocationInfo) error
	SetPodResizeStatus(podUID string, resizeStatus v1.PodResizeStatus)
	Delete(podUID string, containerName string) error
	ClearState() error
}

// State interface provides methods for tracking and setting pod resource allocation
type State interface {
	Reader
	writer
}
