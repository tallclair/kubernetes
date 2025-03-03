/*
Copyright 2024 The Kubernetes Authors.

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
	"maps"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PodAllocationInfo is the container for the allocation information for all pods scheduled to the node.
// Note: these types may only change in a backwards-compatible way.
type PodResourceAllocationInfo struct {
	// Pods are the allocated pods.
	AllocationEntries map[types.UID]PodAllocation `json:"pods,omitempty"`
}

// PodAllocation holds the allocation information for a single pod.
type PodAllocation struct {
	// Containers are the allocated containers for this pod.
	Containers map[string]ContainerAllocation `json:"containers,omitempty"`
}

type ContainerAllocation struct {
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

func (pr *PodResourceAllocationInfo) DeepCopy() *PodResourceAllocationInfo {
	prCopy := &PodResourceAllocationInfo{
		AllocationEntries: maps.Clone(pr.AllocationEntries), // shallow copy: deep copy entries below
	}
	for uid, pod := range prCopy.AllocationEntries {
		prCopy.AllocationEntries[uid] = *pod.DeepCopy()
	}
	return prCopy
}

func (pa *PodAllocation) DeepCopy() *PodAllocation {
	paCopy := &PodAllocation{
		Containers: maps.Clone(pa.Containers), // shallow copy: deep copy entries below
	}
	for name, c := range paCopy.Containers {
		paCopy.Containers[name] = *c.DeepCopy()
	}
	return paCopy
}

func (ca *ContainerAllocation) DeepCopy() *ContainerAllocation {
	return &ContainerAllocation{
		Resources: *ca.Resources.DeepCopy(),
	}
}
