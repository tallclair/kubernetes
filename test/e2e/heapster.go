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

package e2e

import (
	. "github.com/onsi/ginkgo"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

var _ = Describe("Heapster", func() {
	f := NewFramework("heapster")

	BeforeEach(func() {
		SkipUnlessProviderIs("gce") // FIXME
	})

	It("should verify monitoring pods and all cluster nodes are available on influxdb using heapster.", func() {
		testMonitoringUsingHeapsterInfluxdb(f.Client) // FIXME
	})
})

func heapsterPod(f Framework, name string, source string) *api.Pod {
	return &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace.Name,
		},
		Spec: api.PodSpec{
			Containers: []api.Container{{
				Name:  "heapster",
				Image: "gcr.io/google_containers/heapster:v0.20.0-alpha8",
				Resources: api.ResourceRequirements{
					Limits: api.ResourceList{
						api.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
						api.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.DecimalSI),
					},
					Requests: api.ResourceList{
						api.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
						api.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.DecimalSI),
					},
					Command: []string{
						"/heapster",
						"--source=" + source,
						"--sink=influxdb:http://monitoring-influxdb:8086", // FIXME
						"--metric_resolution=60s",
					},
				},
			}},
		},
	}
}
