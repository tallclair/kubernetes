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

package dockershim

import (
	"bytes"
	"io"
	"net/url"
	"time"

	"k8s.io/kubernetes/pkg/kubelet/dockertools"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
	"k8s.io/kubernetes/pkg/util/term"
)

type streamingRuntime struct {
	client dockertools.DockerInterface
}

var _ streaming.Runtime = &streamingRuntime{}

func (ds *streamingRuntime) Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan term.Size) error {
	// FIXME - implemnet this.
	return nil
}

func (ds *streamingRuntime) Attach(containerID string, in io.Reader, out, err io.WriteCloser, resize <-chan term.Size) error {
	// FIXME - implemnet this.
	return nil
}

func (ds *streamingRuntime) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	// FIXME - implement this.
	return nil
}

// ExecSync executes a command in the container, and returns the stdout output.
// If command exits with a non-zero exit code, an error is returned.
func (ds *dockerService) ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error) {
	var stdoutBuffer, stderrBuffer bytes.Buffer

}

// Exec prepares a streaming endpoint to execute a command in the container, and returns the address.
func (ds *dockerService) Exec(containerID string, cmd []string, tty, stdin bool) (*url.URL, error) {
	if ds.streamingServer == nil {
		return nil, streaming.ErrorStreamingDisabled
	}
}

// Attach prepares a streaming endpoint to attach to a running container, and returns the address.
func (ds *dockerService) Attach(containerID string, stdin bool) (*url.URL, error) {
	if ds.streamingServer == nil {
		return nil, streaming.ErrorStreamingDisabled
	}
}

// PortForward prepares a streaming endpoint to forward ports from a PodSandbox, and returns the address.
func (ds *dockerService) PortForward(podSandboxID string, ports []int32) (*url.URL, error) {
	if ds.streamingServer == nil {
		return nil, streaming.ErrorStreamingDisabled
	}
}
