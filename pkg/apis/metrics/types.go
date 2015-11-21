/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package metrics

import (
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// RawNode holds node-level unprocessed sample metrics.
type RawNodeMetrics struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata, since this is a synthetic resource.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:"metadata"`
	// Reference to the measured Node.
	NodeName string `json:"nodeName"`
	// Overall node metrics.
	Total []AggregateSample `json:"total" patchStrategy:"merge" patchMergeKey:"sampleTime"`
	// Metrics of system daemons tracked as raw containers, which may include:
	//   "/kubelet", "/docker-daemon", "kube-proxy" - Tracks respective component metrics
	//   "/system" - Tracks metrics of non-kubernetes and non-kernel processes (grouped together)
	SystemContainers []RawContainer `json:"systemContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

// RawPod holds pod-level unprocessed sample metrics.
type RawPodMetrics struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata, since this is a synthetic resource.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:"metadata"`
	// Reference to the measured Pod.
	PodRef NonLocalObjectReference `json:"podRef"`
	// Metrics of containers in the measured pod.
	Containers []RawContainer `json:"containers,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// Historical metric samples of pod-level resources.
	Samples []PodSample `json:"samples,omitempty" patchStrategy:"merge" patchMergeKey:"sampleTime"`
}

// RawContainer holds container-level unprocessed sample metrics.
type RawContainerMetrics struct {
	// Reference to the measured container.
	Name string `json:"name"`
	// Metadata labels associated with this container (not Kubernetes labels).
	// For example, docker labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Historical metric samples gathered from the container.
	Samples []ContainerSample `json:"samples,omitempty" patchStrategy:"merge" patchMergeKey:"sampleTime"`
}

// NonLocalObjectReference contains enough information to locate the referenced object.
type NonLocalObjectReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// AggregateSample contains a metric sample point of data aggregated across containers.
type AggregateSample struct {
	// The time this data point was collected at.
	SampleTime unversioned.Time `json:"sampleTime"`
	// Metrics pertaining to CPU resources.
	CPU *CPUMetrics `json:"cpu,omitempty"`
	// Metrics pertaining to disk IO resources.
	// Organized by device name.
	DiskIO []DiskIOMetrics `json:"diskIO,omitempty" patchStrategy:"merge" patchMergeKey:"device"`
	// Metrics pertaining to memory (RAM) resources.
	Memory *MemoryMetrics `json:"memory,omitempty"`
	// Metrics pertaining to filesystem usage.
	// Organized by device name.
	Filesystem []FilesystemMetrics `json:"filesystem,omitempty" patchStrategy:"merge" patchMergeKey:"device"`
	// Metrics pertaining to network resources.
	Network *NetworkMetrics `json:"network,omitempty"`
}

// PodSample contains a metric sample point of pod-level resources.
type PodSample struct {
	// The time this data point was collected at.
	SampleTime unversioned.Time `json:"sampleTime"`
	// Metrics pertaining to network resources.
	Network *NetworkMetrics `json:"network,omitempty"`
}

// ContainerSample contains a metric sample point of container-level resources.
type ContainerSample struct {
	// The time this data point was collected at.
	SampleTime unversioned.Time `json:"sampleTime"`
	// Metrics pertaining to CPU resources.
	CPU *CPUMetrics `json:"cpu,omitempty"`
	// Metrics pertaining to disk IO resources.
	// Organized by device name.
	DiskIO []DiskIOMetrics `json:"diskIO,omitempty" patchStrategy:"merge" patchMergeKey:"device"`
	// Metrics pertaining to memory (RAM) resources.
	Memory *MemoryMetrics `json:"memory,omitempty"`
	// Metrics pertaining to filesystem usage.
	// Organized by device name.
	Filesystem []FilesystemMetrics `json:"filesystem,omitempty" patchStrategy:"merge" patchMergeKey:"device"`
}

// NetworkMetrics contains data about network resources.
type NetworkMetrics struct {
	// Network per-interface metrics.
	Interfaces []InterfaceMetrics `json:"interfaces,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// Number of TCP connections in various states (Established, Listen...)
	TCP TCPMetrics `json:"tcp,omitempty"`
	// Number of TCP6 connections in various states (Established, Listen...)
	TCP6 TCPMetrics `json:"tcp6,omitempty"`
}

// InterfaceMetrics contains per-interface data.
type InterfaceMetrics struct {
	// The name of the interface.
	Name string `json:"name"`
	// Cumulative count of bytes received.
	RxBytes int64 `json:"rxBytes"`
	// Cumulative count of packets received.
	RxPackets int64 `json:"rxPackets"`
	// Cumulative count of receive errors encountered.
	RxErrors int64 `json:"rxErrors"`
	// Cumulative count of packets dropped while receiving.
	RxDropped int64 `json:"rxDropped"`
	// Cumulative count of bytes transmitted.
	TxBytes int64 `json:"txBytes"`
	// Cumulative count of packets transmitted.
	TxPackets int64 `json:"txPackets"`
	// Cumulative count of transmit errors encountered.
	TxErrors int64 `json:"txErrors"`
	// Cumulative count of packets dropped while transmitting.
	TxDropped int64 `json:"txDropped"`
}

// TCPMetrics contains data about TCP connection states.
type TCPMetrics struct {
	// Number of connections in the Established state.
	Established int64 `json:"established"`
	// Number of connections in the SynSent state.
	SynSent int64 `json:"synSent"`
	// Number of connections in the SynRecv state.
	SynRecv int64 `json:"synRecv"`
	// Number of connections in the FinWait1 state.
	FinWait1 int64 `json:"finWait1"`
	// Number of connections in the FinWait2 state.
	FinWait2 int64 `json:"finWait2"`
	// Number of connections in the TimeWait state.
	TimeWait int64 `json:"timeWait"`
	// Number of connections in the Close state.
	Close int64 `json:"close"`
	// Number of connections in the CloseWait state.
	CloseWait int64 `json:"closeWait"`
	// Number of connections in the LastAck state.
	LastAck int64 `json:"lastAck"`
	// Number of connections in the Listen state.
	Listen int64 `json:"listen"`
	// Number of connections in the Closing state.
	Closing int64 `json:"closing"`
}

// CPUMetrics contains data about CPU usage.
type CPUMetrics struct {
	Cumulative    CPUCumulativeMetrics    `json:"cumulative,omitempty"`
	Instantaneous CPUInstantaneousMetrics `json:"instantaneous,omitempty"`
	// CPU load that the container is experiencing, represented as a smoothed
	// average of number of runnable threads x 1000.  We multiply by thousand to
	// avoid using floats, but preserving precision.  Load is smoothed over the
	// last 10 seconds.
	LoadAverage *int64 `json:"loadAverage,omitempty"`
}

// CPUCumulativeMetrics contains cumulative data about CPU usage.
type CPUCumulativeMetrics struct {
	// Total CPU usage (sum of all cores).
	TotalCoreSeconds resource.Quantity `json:"total"`
	// Per core usage of the container, indexed by CPU index (0 to CPU MAX).
	PerCPUCoreSeconds []resource.Quantity `json:"perCPU,omitempty"`
	// Usage spent in user space.
	UserCoreSeconds resource.Quantity `json:"user"`
	// Usage spent in kernel space.
	SystemCoreSeconds resource.Quantity `json:"system"`
}

// CPUInstantaneousMetrics contains data about CPU usage averaged over the sampling window.
// The "core" unit can be thought of as CPU core-seconds per second.
type CPUInstantaneousMetrics struct {
	// Total CPU usage (sum of all cores).
	TotalCores resource.Quantity `json:"total"`
	// Per core usage of the container, indexed by CPU index (0 to CPU MAX).
	PerCPUCores []resource.Quantity `json:"perCPU,omitempty"`
	// Usage spent in user space.
	UserCores resource.Quantity `json:"user"`
	// Usage spent in kernel space.
	SystemCores resource.Quantity `json:"system"`
}

// Disk IO stats, as reported by the cgroup block io controller.
// See https://www.kernel.org/doc/Documentation/cgroups/blkio-controller.txt
type DiskIOMetrics struct {
	// The block device name.
	Device string `json:"device"`
	// Disk time allocated for this device.
	TimeSeconds resource.Quantity `json:"time,omitempty"`
	// Cumulative number of sectors transferred to/from disk.
	Sectors resource.Quantity `json:"sectors,omitempty"`
	// Cumulative number of bytes transferred to/from the disk.
	IOServiceBytes IOOperationMetrics `json:"ioServiceBytes,omitempty"`
	// Cumulative number of IOs issued to the disk.
	IOServiced IOOperationMetrics `json:"ioServiced,omitempty"`
	// Cumulative amount of time between request dispatch and request completion for the IOs done.
	IOServiceTimeSeconds IOOperationMetrics `json:"ioServiceTime,omitempty"`
	// Cumulative amount of time the IOs spent waiting in the scheduler queues for service.
	IOWaitTimeSeconds IOOperationMetrics `json:"ioWaitTime,omitempty"`
	// Cumulative number of bios/requests merged into requests belonging to this cgroup.
	IOMerged IOOperationMetrics `json:"ioMerged,omitempty"`
	// Total number of requests queued up at any given instant.
	IOQueued IOOperationMetrics `json:"ioQueued,omitempty"`
}

// IOOperationMetrics contains disk IO data, broken down by IO operation type.
// What each value represents, as well as the unit varies according to the parent field.
type IOOperationMetrics struct {
	// Data aggregated across all operation types.
	Total resource.Quantity `json:"total,omitempty"`
	// Data for read operations.
	Read resource.Quantity `json:"read,omitempty"`
	// Data for write operations.
	Write resource.Quantity `json:"write,omitempty"`
	// Data for synchronous operations.
	Sync resource.Quantity `json:"sync,omitempty"`
	// Data for asynchronous operations.
	Async resource.Quantity `json:"async,omitempty"`
}

// MemoryMetrics contains data about memory usage.
type MemoryMetrics struct {
	// Total memory in use. This includes all memory regardless of when it was accessed.
	TotalBytes resource.Quantity `json:"total,omitempty"`
	// The amount of working set memory. This includes recently accessed memory,
	// dirty memory, and kernel memory. UsageBytes is <= TotalBytes.
	UsageBytes resource.Quantity `json:"usage,omitempty"`
	// Cumulative number of times that a usage counter hit its limit
	FailCount *int64 `json:"failCount,omitempty"`

	ContainerData    PageFaultMetrics `json:"containerData,omitempty"`
	HierarchicalData PageFaultMetrics `json:"hierarchicalData,omitempty"` // FIXME - can we eliminate this?
}

type PageFaultMetrics struct {
	// Cumulative number of minor page faults.
	MinorCount *int64 `json:"minorCount,omitempty"`
	// Cumulative number of major page faults.
	MajorCount *int64 `json:"majorCount,omitempty"`
}

type FilesystemMetrics struct {
	// The block device name associated with the filesystem.
	Device string `json:"device"`

	// Number of bytes that can be consumed by the container on this filesystem.
	Limit resource.Quantity `json:"limit,omitempty"`

	// Number of bytes that is consumed by the container on this filesystem.
	Usage resource.Quantity `json:"usage,omitempty"`
}

type MetricsOptions struct {
	Start         unversioned.Time `json:"start,omitempty"`
	End           unversioned.Time `json:"end,omitempty"`
	Step          int              `json:"step,omitempty"`
	Count         int              `json:"count,omitempty"`
	Pretty        bool             `json:"pretty,omitempty"`
	LabelSelector labels.Selector  `json:"labelSelector,omitempty"`
	FieldSelector fields.Selector  `json:"fieldSelector,omitempty"`
}
