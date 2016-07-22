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

package e2e_node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/stats"
	"k8s.io/kubernetes/test/e2e/framework"
	m "k8s.io/kubernetes/test/matchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = framework.KubeDescribe("Summary API", func() {
	f := framework.NewDefaultFramework("summary-test")
	Context("when querying /stats/summary", func() {
		It("it should report resource usage through the stats api", func() {
			const pod0 = "stats-busybox-0"
			const pod1 = "stats-busybox-1"

			By("Creating test pods")
			createSummaryTestPods(f, pod0, pod1)

			// // Setup expectations
			// lower := lowerBound
			// lower.Pods = []stats.PodStats{
			// 	namedPod(f.Namespace.Name, pod0, podLower),
			// 	namedPod(f.Namespace.Name, pod1, podLower),
			// }
			// upper := upperBound
			// upper.Pods = []stats.PodStats{
			// 	namedPod(f.Namespace.Name, pod0, podUpper),
			// 	namedPod(f.Namespace.Name, pod1, podUpper),
			// }

			// Setup expectations.
			fsCapacityBounds := bounded(100*mb, 10*tb)
			match := m.StrictStruct(m.Fields{
				"Node": m.StrictStruct(m.Fields{
					"NodeName":  m.Ignore(),
					"StartTime": m.Recent(time.Hour * 24 * 365), // 1 year
					"SystemContainers": m.StrictSlice(summaryObjectID, m.Elements{
						"kubelet": m.StrictStruct(m.Fields{
							"Name":      m.Ignore(),
							"StartTime": m.Recent(time.Hour * 24 * 365), // 1 year
							"CPU": structP(m.Fields{
								"Time":                 m.Recent(time.Minute),
								"UsageNanoCores":       bounded(100000, 2E9),
								"UsageCoreNanoSeconds": bounded(10000000, 1E15),
							}),
							"Memory": structP(m.Fields{
								"Time":            m.Recent(time.Minute),
								"AvailableBytes":  bounded(100*mb, 100*gb),
								"UsageBytes":      bounded(10*mb, 10*gb),
								"WorkingSetBytes": bounded(10*mb, 1*gb),
								"RSSBytes":        bounded(10*mb, 1*gb),
								"PageFaults":      bounded(1000, 1E9),
								"MajorPageFaults": bounded(0, 100000),
							}),
							"Rootfs": structP(m.Fields{
								"AvailableBytes": fsCapacityBounds,
								"CapacityBytes":  fsCapacityBounds,
								"UsedBytes":      bounded(0, 0), // Kubelet doesn't write.
								"InodesFree":     bounded(1E4, 1E6),
							}),
							"Logs": structP(m.Fields{
								"AvailableBytes": fsCapacityBounds,
								"CapacityBytes":  fsCapacityBounds,
								"UsedBytes":      bounded(kb, 10*gb),
								"InodesFree":     bounded(1E4, 1E6),
							}),
						}),
						"runtime": m.StrictStruct(m.Fields{
							"Name":      m.Ignore(),
							"StartTime": m.Recent(time.Hour * 24 * 365), // 1 year
							"CPU": structP(m.Fields{
								"Time":                 m.Recent(time.Minute),
								"UsageNanoCores":       bounded(100000, 2E9),
								"UsageCoreNanoSeconds": bounded(10000000, 1E15),
							}),
							"Memory": structP(m.Fields{
								"Time":            m.Recent(time.Minute),
								"AvailableBytes":  bounded(100*mb, 100*gb),
								"UsageBytes":      bounded(100*mb, 10*gb),
								"WorkingSetBytes": bounded(10*mb, 1*gb),
								"RSSBytes":        bounded(10*mb, 1*gb),
								"PageFaults":      bounded(100000, 1E9),
								"MajorPageFaults": bounded(0, 100000),
							}),
							"Rootfs": structP(m.Fields{
								"AvailableBytes": fsCapacityBounds,
								"CapacityBytes":  fsCapacityBounds,
								"UsedBytes":      bounded(0, 10*gb),
								"InodesFree":     bounded(1E4, 1E6),
							}),
							"Logs": structP(m.Fields{
								"AvailableBytes": fsCapacityBounds,
								"CapacityBytes":  fsCapacityBounds,
								"UsedBytes":      bounded(kb, 10*gb),
								"InodesFree":     bounded(1E4, 1E6),
							}),
						}),
					}),
					"CPU": structP(m.Fields{
						"Time":                 m.Recent(time.Minute),
						"UsageNanoCores":       bounded(100E3, 2E9),
						"UsageCoreNanoSeconds": bounded(1E9, 1E15),
					}),
					"Memory": structP(m.Fields{
						"Time":            m.Recent(time.Minute),
						"AvailableBytes":  bounded(100*mb, 100*gb),
						"UsageBytes":      bounded(10*mb, 100*gb),
						"WorkingSetBytes": bounded(10*mb, 100*gb),
						"RSSBytes":        bounded(1*mb, 100*gb),
						"PageFaults":      bounded(1000, 1E9),
						"MajorPageFaults": bounded(0, 100000),
					}),
					// TODO: Handle non-eth0 network interface names.
					"Network": m.NilOr(
						structP(m.Fields{
							"Time":     m.Recent(time.Minute),
							"RxBytes":  bounded(1*mb, 100*gb),
							"RxErrors": bounded(0, 100000),
							"TxBytes":  bounded(10*kb, 10*gb),
							"TxErrors": bounded(0, 100000),
						}),
					),
					"Fs": structP(m.Fields{
						"AvailableBytes": fsCapacityBounds,
						"CapacityBytes":  fsCapacityBounds,
						"UsedBytes":      bounded(kb, 10*gb),
						"InodesFree":     bounded(1E4, 1E6),
					}),
					"Runtime": structP(m.Fields{
						"ImageFs": structP(m.Fields{
							"AvailableBytes": fsCapacityBounds,
							"CapacityBytes":  fsCapacityBounds,
							"UsedBytes":      bounded(kb, 10*gb),
							"InodesFree":     bounded(1E4, 1E6),
						}),
					}),
				}),
				"Pods": m.Ignore(),
			})

			By("Returning stats summary")
			Eventually(func() (stats.Summary, error) {
				summary := stats.Summary{}
				resp, err := http.Get(*kubeletAddress + "/stats/summary")
				if err != nil {
					return summary, fmt.Errorf("Failed to get /stats/summary - %v", err)
				}
				contentsBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return summary, fmt.Errorf("Failed to read /stats/summary - %+v", resp)
				}
				contents := string(contentsBytes)
				decoder := json.NewDecoder(strings.NewReader(contents))
				err = decoder.Decode(&summary)
				if err != nil {
					return summary, fmt.Errorf("Failed to parse /stats/summary to go struct: %+v", resp)
				}

				return summary, nil
			}, /*1*time.Minute FIXME */ 30*time.Second, time.Second*15).Should(match)
		})
	})
})

func createSummaryTestPods(f *framework.Framework, names ...string) {
	pods := make([]*api.Pod, 0, len(names))
	for _, name := range names {
		pods = append(pods, &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: name,
			},
			Spec: api.PodSpec{
				// Don't restart the Pod since it is expected to exit
				RestartPolicy: api.RestartPolicyNever,
				Containers: []api.Container{
					{
						Name:    "busybox-container",
						Image:   ImageRegistry[busyBoxImage],
						Command: []string{"sh", "-c", "while true; do echo 'hello world' | tee /test-empty-dir-mnt/file ; sleep 1; done"},
						Resources: api.ResourceRequirements{
							Limits: api.ResourceList{
								// Must set memory limit to get MemoryStats.AvailableBytes
								api.ResourceMemory: resource.MustParse("10M"),
							},
						},
						VolumeMounts: []api.VolumeMount{
							{MountPath: "/test-empty-dir-mnt", Name: "test-empty-dir"},
						},
					},
				},
				SecurityContext: &api.PodSecurityContext{
					SELinuxOptions: &api.SELinuxOptions{
						Level: "s0",
					},
				},
				Volumes: []api.Volume{
					// TODO(#28393): Test secret volumes
					// TODO(#28394): Test hostpath volumes
					{Name: "test-empty-dir", VolumeSource: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}},
				},
			},
		})
	}
	f.PodClient().CreateBatch(pods)
}

const (
	kb = 1000
	mb = 1000 * kb
	gb = 1000 * mb
	tb = 1000 * gb
)

// var (
// 	podLower = stats.PodStats{
// 		Containers: []stats.ContainerStats{
// 			{
// 				Name: "busybox-container",
// 				CPU: &stats.CPUStats{
// 					UsageNanoCores:       val(100000),
// 					UsageCoreNanoSeconds: val(10000000),
// 				},
// 				Memory: &stats.MemoryStats{
// 					AvailableBytes:  val(1 * mb),
// 					UsageBytes:      val(10 * kb),
// 					WorkingSetBytes: val(10 * kb),
// 					RSSBytes:        val(1 * kb),
// 					PageFaults:      val(100),
// 					MajorPageFaults: val(0),
// 				},
// 				Rootfs: &stats.FsStats{
// 					AvailableBytes: val(100 * mb),
// 					CapacityBytes:  val(100 * mb),
// 					UsedBytes:      val(kb),
// 				},
// 				Logs: &stats.FsStats{
// 					AvailableBytes: val(100 * mb),
// 					CapacityBytes:  val(100 * mb),
// 					UsedBytes:      val(kb),
// 				},
// 			},
// 		},
// 		Network: &stats.NetworkStats{
// 			RxBytes:  val(10),
// 			RxErrors: val(0),
// 			TxBytes:  val(10),
// 			TxErrors: val(0),
// 		},
// 		VolumeStats: []stats.VolumeStats{{
// 			Name: "test-empty-dir",
// 			FsStats: stats.FsStats{
// 				AvailableBytes: val(100 * mb),
// 				CapacityBytes:  val(100 * mb),
// 				UsedBytes:      val(kb),
// 			},
// 		}},
// 	}

// 	lowerBound = stats.Summary{
// 		Node: stats.NodeStats{
// 			SystemContainers: []stats.ContainerStats{
// 				{
// 					Name: "kubelet",
// 					CPU: &stats.CPUStats{
// 						UsageNanoCores:       val(100000),
// 						UsageCoreNanoSeconds: val(10000000),
// 					},
// 					Memory: &stats.MemoryStats{
// 						AvailableBytes:  val(100 * mb),
// 						UsageBytes:      val(10 * mb),
// 						WorkingSetBytes: val(10 * mb),
// 						RSSBytes:        val(10 * mb),
// 						PageFaults:      val(1000),
// 						MajorPageFaults: val(0),
// 					},
// 					Rootfs: &stats.FsStats{
// 						AvailableBytes: val(100 * mb),
// 						CapacityBytes:  val(100 * mb),
// 						UsedBytes:      val(0),
// 					},
// 					Logs: &stats.FsStats{
// 						AvailableBytes: val(100 * mb),
// 						CapacityBytes:  val(100 * mb),
// 						UsedBytes:      val(kb),
// 					},
// 				},
// 				{
// 					Name: "runtime",
// 					CPU: &stats.CPUStats{
// 						UsageNanoCores:       val(100000),
// 						UsageCoreNanoSeconds: val(10000000),
// 					},
// 					Memory: &stats.MemoryStats{
// 						AvailableBytes:  val(100 * mb),
// 						UsageBytes:      val(100 * mb),
// 						WorkingSetBytes: val(10 * mb),
// 						RSSBytes:        val(10 * mb),
// 						PageFaults:      val(100000),
// 						MajorPageFaults: val(0),
// 					},
// 					Rootfs: &stats.FsStats{
// 						AvailableBytes: val(100 * mb),
// 						CapacityBytes:  val(100 * mb),
// 						UsedBytes:      val(0),
// 					},
// 					Logs: &stats.FsStats{
// 						AvailableBytes: val(100 * mb),
// 						CapacityBytes:  val(100 * mb),
// 						UsedBytes:      val(kb),
// 					},
// 				},
// 			},
// 			CPU: &stats.CPUStats{
// 				UsageNanoCores:       val(100000),
// 				UsageCoreNanoSeconds: val(1000000000),
// 			},
// 			Memory: &stats.MemoryStats{
// 				AvailableBytes:  val(100 * mb),
// 				UsageBytes:      val(10 * mb),
// 				WorkingSetBytes: val(10 * mb),
// 				RSSBytes:        val(1 * mb),
// 				PageFaults:      val(1000),
// 				MajorPageFaults: val(0),
// 			},
// 			Network: &stats.NetworkStats{
// 				RxBytes:  val(1 * mb),
// 				RxErrors: val(0),
// 				TxBytes:  val(10 * kb),
// 				TxErrors: val(0),
// 			},
// 			Fs: &stats.FsStats{
// 				AvailableBytes: val(100 * mb),
// 				CapacityBytes:  val(100 * mb),
// 				UsedBytes:      val(kb),
// 				InodesFree:     val(1E4),
// 			},
// 			Runtime: &stats.RuntimeStats{
// 				ImageFs: &stats.FsStats{
// 					AvailableBytes: val(100 * mb),
// 					CapacityBytes:  val(100 * mb),
// 					UsedBytes:      val(kb),
// 					InodesFree:     val(1E4),
// 				},
// 			},
// 		},
// 	}

// 	podUpper = stats.PodStats{
// 		Containers: []stats.ContainerStats{
// 			{
// 				Name: "busybox-container",
// 				CPU: &stats.CPUStats{
// 					UsageNanoCores:       val(100000000),
// 					UsageCoreNanoSeconds: val(1000000000),
// 				},
// 				Memory: &stats.MemoryStats{
// 					AvailableBytes:  val(10 * mb),
// 					UsageBytes:      val(mb),
// 					WorkingSetBytes: val(mb),
// 					RSSBytes:        val(mb),
// 					PageFaults:      val(100000),
// 					MajorPageFaults: val(10),
// 				},
// 				Rootfs: &stats.FsStats{
// 					AvailableBytes: val(100 * gb),
// 					CapacityBytes:  val(100 * gb),
// 					UsedBytes:      val(10 * mb),
// 				},
// 				Logs: &stats.FsStats{
// 					AvailableBytes: val(100 * gb),
// 					CapacityBytes:  val(100 * gb),
// 					UsedBytes:      val(10 * mb),
// 				},
// 			},
// 		},
// 		Network: &stats.NetworkStats{
// 			RxBytes:  val(10 * mb),
// 			RxErrors: val(1000),
// 			TxBytes:  val(10 * mb),
// 			TxErrors: val(1000),
// 		},
// 		VolumeStats: []stats.VolumeStats{{
// 			Name: "test-empty-dir",
// 			FsStats: stats.FsStats{
// 				AvailableBytes: val(100 * gb),
// 				CapacityBytes:  val(100 * gb),
// 				UsedBytes:      val(1 * mb),
// 			},
// 		}},
// 	}

// 	upperBound = stats.Summary{
// 		Node: stats.NodeStats{
// 			SystemContainers: []stats.ContainerStats{
// 				{
// 					Name: "kubelet",
// 					CPU: &stats.CPUStats{
// 						UsageNanoCores:       val(2E9),
// 						UsageCoreNanoSeconds: val(10E12),
// 					},
// 					Memory: &stats.MemoryStats{
// 						AvailableBytes:  val(100 * gb),
// 						UsageBytes:      val(10 * gb),
// 						WorkingSetBytes: val(1 * gb),
// 						RSSBytes:        val(1 * gb),
// 						PageFaults:      val(1E9),
// 						MajorPageFaults: val(100000),
// 					},
// 					Rootfs: &stats.FsStats{
// 						AvailableBytes: val(100 * gb),
// 						CapacityBytes:  val(100 * gb),
// 						UsedBytes:      val(0), // Kubelet doesn't write.
// 					},
// 					Logs: &stats.FsStats{
// 						AvailableBytes: val(100 * gb),
// 						CapacityBytes:  val(100 * gb),
// 						UsedBytes:      val(10 * gb),
// 					},
// 				},
// 				{
// 					Name: "runtime",
// 					CPU: &stats.CPUStats{
// 						UsageNanoCores:       val(2E9),
// 						UsageCoreNanoSeconds: val(10E12),
// 					},
// 					Memory: &stats.MemoryStats{
// 						AvailableBytes:  val(100 * gb),
// 						UsageBytes:      val(10 * gb),
// 						WorkingSetBytes: val(1 * gb),
// 						RSSBytes:        val(1 * gb),
// 						PageFaults:      val(1E9),
// 						MajorPageFaults: val(100000),
// 					},
// 					Rootfs: &stats.FsStats{
// 						AvailableBytes: val(100 * gb),
// 						CapacityBytes:  val(100 * gb),
// 						UsedBytes:      val(10 * gb),
// 					},
// 					Logs: &stats.FsStats{
// 						AvailableBytes: val(100 * gb),
// 						CapacityBytes:  val(100 * gb),
// 						UsedBytes:      val(10 * gb),
// 					},
// 				},
// 			},
// 			CPU: &stats.CPUStats{
// 				UsageNanoCores:       val(2E9),
// 				UsageCoreNanoSeconds: val(10E12),
// 			},
// 			Memory: &stats.MemoryStats{
// 				AvailableBytes:  val(100 * gb),
// 				UsageBytes:      val(10 * gb),
// 				WorkingSetBytes: val(1 * gb),
// 				RSSBytes:        val(1 * gb),
// 				PageFaults:      val(1E9),
// 				MajorPageFaults: val(100000),
// 			},
// 			Network: &stats.NetworkStats{
// 				RxBytes:  val(100 * gb),
// 				RxErrors: val(100000),
// 				TxBytes:  val(10 * gb),
// 				TxErrors: val(100000),
// 			},
// 			Fs: &stats.FsStats{
// 				AvailableBytes: val(100 * gb),
// 				CapacityBytes:  val(100 * gb),
// 				UsedBytes:      val(10 * gb),
// 				InodesFree:     val(1E6),
// 			},
// 			Runtime: &stats.RuntimeStats{
// 				ImageFs: &stats.FsStats{
// 					AvailableBytes: val(100 * gb),
// 					CapacityBytes:  val(100 * gb),
// 					UsedBytes:      val(10 * gb),
// 					InodesFree:     val(1E6),
// 				},
// 			},
// 		},
// 	}

// 	ignoredFields = sets.NewString(
// 		"Name",
// 		"NodeName",
// 		"PodRef",
// 		"StartTime",
// 		"UserDefinedMetrics",
// 	)

// 	allowedNils = sets.NewString(
// 		".Node.SystemContainers[kubelet].Memory.AvailableBytes",
// 		".Node.SystemContainers[runtime].Memory.AvailableBytes",
// 		// TODO(#28395): Figure out why UsedBytes is nil on ubuntu-trusty-docker10 and coreos-stable20160622
// 		".Node.SystemContainers[kubelet].Rootfs.UsedBytes",
// 		".Node.SystemContainers[kubelet].Logs.UsedBytes",
// 		".Node.SystemContainers[runtime].Rootfs.UsedBytes",
// 		".Node.SystemContainers[runtime].Logs.UsedBytes",
// 		// TODO: Handle non-eth0 network interface names.
// 		".Node.Network",
// 	)
// )

// func checkSummary(actual, lower, upper stats.Summary) []error {
// 	return checkValue("", reflect.ValueOf(actual), reflect.ValueOf(lower), reflect.ValueOf(upper))
// }

func summaryObjectID(element interface{}) string {
	switch el := element.(type) {
	case stats.PodStats:
		return fmt.Sprintf("%s::%s", el.PodRef.Namespace, el.PodRef.Name)
	case stats.ContainerStats:
		return el.Name
	case stats.VolumeStats:
		return el.Name
	case stats.UserDefinedMetric:
		return el.Name
	default:
		framework.Failf("Unknown type: %T", el)
		return "???"
	}
}

func namedPod(namespace, name string, pod stats.PodStats) stats.PodStats {
	pod.PodRef.Name = name
	pod.PodRef.Namespace = namespace
	return pod
}

// Convenience functions for common matcher combinations.
func structP(fields m.Fields) types.GomegaMatcher {
	return m.Ptr(m.StrictStruct(fields))
}

func bounded(lower, upper interface{}) types.GomegaMatcher {
	return m.Ptr(m.InRange(lower, upper))
}
