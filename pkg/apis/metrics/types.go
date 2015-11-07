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

/*
This file (together with pkg/apis/extensions/v1beta1/types.go) contain the experimental
types in kubernetes. These API objects are experimental, meaning that the
APIs may be broken at any time by the kubernetes team.

DISCLAIMER: The implementation of the experimental API group itself is
a temporary one meant as a stopgap solution until kubernetes has proper
support for multiple API groups. The transition may require changes
beyond registration differences. In other words, experimental API group
support is experimental.
*/

package metrics

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// FIXME - remove this
// Replace: \([A-Z]\)\([A-Za-z]+\) \(.*\) `json:"[A-Za-z_]+\(,.*\)?"`
// With:    \1\2 \3 `json:"\,(downcase \1)\2\4"`

// RawNode holds node-level unprocessed sample metrics.
type RawNode struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata, applying to the lists of stats.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:",inline"`
	// Reference to the measured Node.
	NodeRef api.ObjectReference `json:"nodeRef,omitempty"`
	// Overall machine metrics.
	Machine RawContainer `json:"machine,omitempty"`
	// Metrics of system components.
	SystemContainers []RawContainer `json:"systemContainers,omitempty"`
}

// RawPod holds pod-level unprocessed sample metrics.
type RawPod struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata, applying to the lists of stats.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:",inline"`
	// Reference to the measured Pod.
	PodRef api.ObjectReference `json:"podRef,omitempty"`
	// Metrics of containers in the measured pod.
	Containers []RawContainer `json:"containers,omitempty"`
}

// RawContainer holds container-level unprocessed sample metrics.
type RawContainer struct {
	// Reference to the measured container.
	Name string `json:"name,omitempty"`
	// Describes the container resources.
	Spec ContainerSpec `json:"spec,omitempty"`
	// Historical metric samples gathered from the container.
	Stats []ContainerStats `json:"stats,omitempty"`
}

type ContainerSpec struct {
	// Time at which the container was created.
	CreationTime time.Time `json:"creationTime,omitempty"`

	// Other names by which the container is known within a certain namespace.
	// This is unique within that namespace.
	Aliases []string `json:"aliases,omitempty"`

	// Namespace under which the aliases of a container are unique (not a Kubernetes namespace).
	// An example of a namespace is "docker" for Docker containers.
	Namespace string `json:"namespace,omitempty"`

	// FIXME - what's this? docker labels?
	// Metadata labels associated with this container.
	Labels map[string]string `json:"labels,omitempty"`

	// FIXME - verify pointer usage
	Cpu           *CpuSpec     `json:"cpu,omitempty"`
	Memory        *MemorySpec  `json:"memory,omitempty"`
	CustomMetrics []MetricSpec `json:"customMetrics,omitempty"`

	// Following resources have no associated spec, but are being isolated.
	HasNetwork    bool `json:"hasNetwork,omitempty"`
	HasFilesystem bool `json:"hasFilesystem,omitempty"`
	HasDiskIo     bool `json:"hasDiskIo,omitempty"`

	// Image name used for this container.
	Image string `json:"image,omitempty"`
}

type CpuSpec struct {
	// FIXME - unit (milli-cpus?) - yes
	// Requested cpu shares. Default is 1024.
	Limit uint64 `json:"limit,omitempty"`
	// Requested cpu hard limit. Default is unlimited (0).
	// Units: milli-cpus.
	MaxLimitMilliCpus uint64 `json:"maxLimitMilliCpus,omitempty"`
	// Cpu affinity mask.
	Mask string `json:"mask,omitempty"`
}

type MemorySpec struct {
	// The amount of memory requested. Default is unlimited (-1).
	LimitBytes uint64 `json:"limitBytes,omitempty"`

	// The amount of guaranteed memory.  Default is 0.
	ReservationBytes uint64 `json:"reservationBytes,omitempty"`

	// The amount of swap space requested. Default is unlimited (-1).
	SwapLimitBytes uint64 `json:"swapLimitBytes,omitempty"`
}

type ContainerStats struct {
	// The time of this stat point.
	Timestamp time.Time `json:"timestamp,omitempty"`
	// CPU statistics, in nanoseconds (aggregated)
	Cpu *CpuStats `json:"cpu,omitempty"`
	// CPU statistics, in nanocores per second (instantaneous)
	CpuInst *CpuInstStats `json:"cpuInst,omitempty"`
	// Disk IO statistics
	DiskIo *DiskIoStats `json:"diskIo,omitempty"`
	// Memory statistics
	Memory *MemoryStats `json:"memory,omitempty"`
	// Network statistics
	Network *NetworkStats `json:"network,omitempty"`
	// Filesystem statistics
	Filesystem []FsStats `json:"filesystem,omitempty"`
	// Task load statistics
	Load *LoadStats `json:"load,omitempty"`
	// Custom Metrics
	CustomMetrics []CustomMetric `json:"customMetrics,omitempty"`
}

// Percentile statistics for a resource. Unit depends on resource.
type Percentiles struct {
	// Average over the collected sample.
	Mean uint64 `json:"mean,omitempty"`
	// Max seen over the collected sample.
	Max uint64 `json:"max,omitempty"`
	// 50th percentile over the collected sample.
	Fifty uint64 `json:"fifty,omitempty"`
	// 90th percentile over the collected sample.
	Ninety uint64 `json:"ninety,omitempty"`
	// 95th percentile over the collected sample.
	NinetyFive uint64 `json:"ninetyFive,omitempty"`
}

type TcpStats struct { // FIXME - cummulative counts
	Established uint64 `json:"established,omitempty"`
	SynSent     uint64 `json:"synSent,omitempty"`
	SynRecv     uint64 `json:"synRecv,omitempty"`
	FinWait1    uint64 `json:"finWait1,omitempty"`
	FinWait2    uint64 `json:"finWait2,omitempty"`
	TimeWait    uint64 `json:"timeWait,omitempty"`
	Close       uint64 `json:"close,omitempty"`
	CloseWait   uint64 `json:"closeWait,omitempty"`
	LastAck     uint64 `json:"lastAck,omitempty"`
	Listen      uint64 `json:"listen,omitempty"`
	Closing     uint64 `json:"closing,omitempty"`
}

type NetworkStats struct {
	// Network stats by interface.
	Interfaces []InterfaceStats `json:"interfaces,omitempty"`
	// TCP connection stats.
	Tcp TcpStats `json:"tcp,omitempty"`
	// TCP6 connection stats.
	Tcp6 TcpStats `json:"tcp6,omitempty"`
}

type InterfaceStats struct {
	// The name of the interface.
	Name string `json:"name,omitempty"`
	// Cumulative count of bytes received.
	RxBytes uint64 `json:"rxBytes,omitempty"` // FIXME - does Rx,Tx fall under extremely common abbreviations?
	// Cumulative count of packets received.  // FIXME - since container start
	RxPackets uint64 `json:"rxPackets,omitempty"`
	// Cumulative count of receive errors encountered.
	RxErrors uint64 `json:"rxErrors,omitempty"`
	// Cumulative count of packets dropped while receiving.
	RxDropped uint64 `json:"rxDropped,omitempty"`
	// Cumulative count of bytes transmitted.
	TxBytes uint64 `json:"txBytes,omitempty"`
	// Cumulative count of packets transmitted.
	TxPackets uint64 `json:"txPackets,omitempty"`
	// Cumulative count of transmit errors encountered.
	TxErrors uint64 `json:"txErrors,omitempty"`
	// Cumulative count of packets dropped while transmitting.
	TxDropped uint64 `json:"txDropped,omitempty"`
}

// Instantaneous CPU stats
type CpuInstStats struct {
	Usage CpuInstUsage `json:"usage,omitempty"`
}

// CPU usage time statistics.
type CpuInstUsage struct {
	// Total CPU usage.
	// Units: nanocores per second
	Total uint64 `json:"total,omitempty"`

	// Per CPU/core usage of the container.
	// Unit: nanocores per second
	PerCpu []uint64 `json:"perCpu,omitempty"`

	// Time spent in user space.
	// Unit: nanocores per second
	User uint64 `json:"user,omitempty"`

	// Time spent in kernel space.
	// Unit: nanocores per second
	System uint64 `json:"system,omitempty"`
}

// CPU usage time statistics.
type CpuUsage struct {
	// Total CPU usage.
	TotalNanoSeconds int64 `json:"total,omitempty"`

	// Per CPU/core usage of the container.
	PerCpu []uint64 `json:"perCpu,omitempty"` // FIXME - nanoseconds? & cumulative

	// Time spent in user space.
	UserNanoseconds uint64 `json:"user,omitempty"`

	// Time spent in kernel space.
	SystemNanoseconds uint64 `json:"system,omitempty"`
}

// All CPU usage metrics are cumulative from the creation of the container
type CpuStats struct {
	Usage CpuUsage `json:"usage,omitempty"`
	// Smoothed average of number of runnable threads x 1000.
	// We multiply by thousand to avoid using floats, but preserving precision.
	// Load is smoothed over the last 10 seconds. Instantaneous value can be read
	// from LoadStats.NrRunning.  // FIXME: NrRunning
	LoadAverage int32 `json:"loadAverage,omitempty"` // FIXME - units?
}

// FIXME - docs
type PerDiskStats struct {
	// Device identifiers
	Major uint64            `json:"major,omitempty"`
	Minor uint64            `json:"minor,omitempty"`
	Stats map[string]uint64 `json:"stats,omitempty"`
}

// FIXME - units
type DiskIoStats struct {
	IoServiceBytes []PerDiskStats `json:"ioServiceBytes,omitempty"`
	IoServiced     []PerDiskStats `json:"ioServiced,omitempty"` // cumulative number of IOs
	IoQueued       []PerDiskStats `json:"ioQueued,omitempty"`
	Sectors        []PerDiskStats `json:"sectors,omitempty"`
	IoServiceTime  []PerDiskStats `json:"ioServiceTime,omitempty"`
	IoWaitTime     []PerDiskStats `json:"ioWaitTime,omitempty"`
	IoMerged       []PerDiskStats `json:"ioMerged,omitempty"`
	IoTime         []PerDiskStats `json:"ioTime,omitempty"`
}

type MemoryStats struct {
	// Current memory usage, this includes all memory regardless of when it was
	// accessed.
	// Units: Bytes.
	Usage uint64 `json:"usage,omitempty"` // FIXME - TotalUsage

	// The amount of working set memory, this includes recently accessed memory,
	// dirty memory, and kernel memory. Working set is <= "usage".
	WorkingSetBytes uint64 `json:"workingSetBytes,omitempty"` // FIXME - usage

	FailCount uint64 `json:"failCount,omitempty"`

	ContainerData    MemoryStatsMemoryData `json:"containerData,omitempty"`
	HierarchicalData MemoryStatsMemoryData `json:"hierarchicalData,omitempty"`
}

type MemoryStatsMemoryData struct {
	Pgfault    uint64 `json:"pgfault,omitempty"`    // FIXME - cumulative counts
	Pgmajfault uint64 `json:"pgmajfault,omitempty"` // FIXME - rename? What's this?
}

type FsStats struct {
	// The block device name associated with the filesystem.
	DeviceName string `json:"deviceName,omitempty"`

	// Number of bytes that can be consumed by the container on this filesystem.
	LimitBytes uint64 `json:"limitBytes,omitempty"`

	// Number of bytes that is consumed by the container on this filesystem.
	UsageBytes uint64 `json:"usageBytes,omitempty"`

	// Number of bytes available for non-root user.
	AvailableBytes uint64 `json:"availableBytes,omitempty"`

	// Number of reads completed
	// This is the total number of reads completed successfully.
	ReadsCompleted uint64 `json:"readsCompleted,omitempty"`

	// Number of reads merged
	// Reads and writes which are adjacent to each other may be merged for
	// efficiency.  Thus two 4K reads may become one 8K read before it is
	// ultimately handed to the disk, and so it will be counted (and queued)
	// as only one I/O.  This field lets you know how often this was done.
	ReadsMerged uint64 `json:"readsMerged,omitempty"`

	// Number of sectors read
	// This is the total number of sectors read successfully.
	SectorsRead uint64 `json:"sectorsRead,omitempty"`

	// Number of milliseconds spent reading
	// This is the total number of milliseconds spent by all reads (as
	// measured from __make_request() to end_that_request_last()).
	ReadTime uint64 `json:"readMilliSeconds,omitempty"` // FIXME - or Millis? (and below)

	// Number of writes completed
	// This is the total number of writes completed successfully.
	WritesCompleted uint64 `json:"writesCompleted,omitempty"`

	// Number of writes merged
	// See the description of reads merged.
	WritesMerged uint64 `json:"writesMerged,omitempty"`

	// Number of sectors written
	// This is the total number of sectors written successfully.
	SectorsWritten uint64 `json:"sectorsWritten,omitempty"`

	// Number of milliseconds spent writing
	// This is the total number of milliseconds spent by all writes (as
	// measured from __make_request() to end_that_request_last()).
	WriteMilliSeconds uint64 `json:"writeMilliSeconds,omitempty"`

	// Number of I/Os currently in progress
	// The only field that should go to zero. Incremented as requests are
	// given to appropriate struct request_queue and decremented as they finish.
	IoInProgress uint64 `json:"ioInProgress,omitempty"`

	// Number of milliseconds spent doing I/Os
	// This field increases so long as field 9 is nonzero.
	IoMilliSeconds uint64 `json:"ioMilliSeconds,omitempty"`

	// weighted number of milliseconds spent doing I/Os
	// This field is incremented at each I/O start, I/O completion, I/O
	// merge, or read of these stats by the number of I/Os in progress
	// (field 9) times the number of milliseconds spent doing I/O since the
	// last update of this field.  This can provide an easy measure of both
	// I/O completion time and the backlog that may be accumulating.
	WeightedIoMilliSeconds uint64 `json:"weightedIoMilliSeconds,omitempty"`
}

// This mirrors kernel internal structure.
type LoadStats struct {
	// Number of sleeping tasks.
	SleepingTaskCount uint64 `json:"sleepingTaskCount,omitempty"`

	// Number of running tasks.
	RunningTaskCount uint64 `json:"runningTaskCount,omitempty"`

	// Number of tasks in stopped state
	StoppedTaskCount uint64 `json:"stoppedTaskCount,omitempty"`

	// Number of tasks in uninterruptible state
	UninterruptibleTaskCount uint64 `json:"uninterruptibleTaskCount,omitempty"`

	// Number of tasks waiting on IO
	IoWaitTaskCount uint64 `json:"ioWaitTaskCount,omitempty"`
}

// Type of metric being exported.
type MetricType string

const (
	// Instantaneous value. May increase or decrease.
	MetricGauge MetricType = "gauge"

	// A counter-like value that is only expected to increase.
	MetricCumulative MetricType = "cumulative"

	// Rate over a time period.
	MetricDelta MetricType = "delta"
)

// DataType for metric being exported.
type DataType string

const (
	IntType   DataType = "int"
	FloatType DataType = "float"
)

// Spec for custom metric.
type MetricSpec struct {
	// The name of the metric.
	Name string `json:"name,omitempty"`

	// Type of the metric.
	Type MetricType `json:"type,omitempty"`

	// Data Type for the stats.
	Format DataType `json:"format,omitempty"`

	// Display Unit for the stats.
	Unit string `json:"unit,omitempty"`
}

// An exported metric.
type MetricVal struct {
	// Label associated with a metric
	Label string `json:"label,omitempty"`

	// Time at which the metric was queried
	Timestamp time.Time `json:"timestamp,omitempty"`

	// The value of the metric at this point.
	IntValue   int64   `json:"intValue,omitempty"`
	FloatValue float64 `json:"floatValue,omitempty"`
}

type CustomMetric struct {
	Name   string      `json:"name,omitempty"`
	Values []MetricVal `json:"values,omitempty"`
}
