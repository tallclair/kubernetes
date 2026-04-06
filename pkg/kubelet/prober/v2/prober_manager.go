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

package v2

import (
	"context"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/kubelet/status"
	"k8s.io/utils/clock"
)

// Manager manages pod probing. It creates a probe "worker" for every container that specifies a
// probe (AddPod). The worker periodically probes its assigned container and caches the results. The
// manager use the cached probe results to set the appropriate Ready state in the PodStatus when
// requested (UpdatePodStatus). Updating probe parameters is not currently supported.
type Manager interface {
	// AddPod creates new probe workers for every container probe. This should be called for every
	// pod created.
	// AddPod(ctx context.Context, pod *v1.Pod)

	StartContainerProbes(container *v1.Container, containerID string) error
	StopContainerProbes(container *v1.Container) error

	// StopLivenessAndStartup handles stopping liveness and startup probes during termination.
	// StopLivenessAndStartup(pod *v1.Pod)

	// RemovePod handles cleaning up the removed pod state, including terminating probe workers and
	// deleting cached results.
	RemovePod(pod *v1.Pod)

	// CleanupPods handles cleaning up pods which should no longer be running.
	// It takes a map of "desired pods" which should not be cleaned up.
	CleanupPods(desiredPods map[types.UID]sets.Empty)
}

type manager struct {
	// Map of active workers for probes
	workers map[probeKey]*worker
	// Lock for accessing & mutating workers
	workerLock sync.RWMutex

	// The statusManager cache provides pod IP and container IDs for probing.
	statusManager status.Manager

	// readinessManager manages the results of readiness probes
	readinessManager results.Manager

	// livenessManager manages the results of liveness probes
	livenessManager results.Manager

	// startupManager manages the results of startup probes
	startupManager results.Manager

	// prober executes the probe actions.
	prober *prober.Prober

	start time.Time
}

// NewManager creates a Manager for pod probing.
func NewManager(
	statusManager status.Manager,
	livenessManager results.Manager,
	readinessManager results.Manager,
	startupManager results.Manager,
	runner kubecontainer.CommandRunner,
	recorder record.EventRecorderLogger) Manager {

	prober := newProber(runner, recorder)
	return &manager{
		statusManager:    statusManager,
		prober:           prober,
		readinessManager: readinessManager,
		livenessManager:  livenessManager,
		startupManager:   startupManager,
		workers:          make(map[probeKey]*worker),
		start:            clock.RealClock{}.Now(),
	}
}

// Key uniquely identifying container probes
type probeKey struct {
	podUID        types.UID
	containerName string
	probeType     prober.ProbeType
}

const (
	probeResultSuccessful string = "successful"
	probeResultFailed     string = "failed"
	probeResultUnknown    string = "unknown"
)

func (m *manager) StartContainerProbes(ctx context.Context, pod *v1.Pod, c *v1.Container, containerID string) error {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()

	logger := klog.FromContext(ctx)
	key := probeKey{podUID: pod.UID, containerName: c.Name}

	key.containerName = c.Name

	if c.StartupProbe != nil {
		key.probeType = prober.Startup
		startupProbe, ok := m.workers[key]
		if !ok {
			startupProbe = newWorker(m, prober.Startup, pod, c)
			m.workers[key] = startupProbe
		}
		w := newWorker(m, prober.Startup, pod, c)
		m.workers[key] = w
		go w.run(ctx)
	}

	if c.ReadinessProbe != nil {
		key.probeType = prober.Readiness
		if _, ok := m.workers[key]; ok {
			logger.V(8).Info("Readiness probe already exists for container",
				"pod", klog.KObj(pod), "containerName", c.Name)
			return
		}
		w := newWorker(m, prober.Readiness, pod, c)
		m.workers[key] = w
		go w.run(ctx)
	}

	if c.LivenessProbe != nil {
		key.probeType = prober.Liveness
		if _, ok := m.workers[key]; ok {
			logger.V(8).Info("Liveness probe already exists for container",
				"pod", klog.KObj(pod), "containerName", c.Name)
			return
		}
		w := newWorker(m, prober.Liveness, pod, c)
		m.workers[key] = w
		go w.run(ctx)
	}
}

func (m *manager) RemovePod(pod *v1.Pod) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()

	key := probeKey{podUID: pod.UID}
	for c := range podutil.ContainerIter(&pod.Spec, podutil.InitContainers|podutil.Containers) {
		key.containerName = c.Name
		for _, probeType := range [...]probeType{prober.Readiness, prober.Liveness, prober.Startup} {
			key.probeType = probeType
			if worker, ok := m.workers[key]; ok {
				worker.stop()
			}
		}
	}
}

func (m *manager) CleanupPods(desiredPods map[types.UID]sets.Empty) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()

	for key, worker := range m.workers {
		if _, ok := desiredPods[key.podUID]; !ok {
			worker.stop()
		}
	}
}

func (m *manager) getWorker(podUID types.UID, containerName string, probeType probeType) (*worker, bool) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	worker, ok := m.workers[probeKey{podUID, containerName, probeType}]
	return worker, ok
}

// Called by the worker after exiting.
func (m *manager) removeWorker(podUID types.UID, containerName string, probeType probeType) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	delete(m.workers, probeKey{podUID, containerName, probeType})
}

// workerCount returns the total number of probe workers. For testing.
func (m *manager) workerCount() int {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	return len(m.workers)
}

// kubeletRestartGracePeriod returns a time point that is 10 seconds before the kubelet start time.
// This grace period is used to determine if a container was already running before kubelet restarted.
// If a container's start time is before this grace period, it indicates the container was running
// prior to kubelet restart and should not be immediately marked as failed to avoid unnecessary
// status changes for containers that were previously ready.
func kubeletRestartGracePeriod(start time.Time) time.Time {
	return start.Add(-time.Second * 10)
}
