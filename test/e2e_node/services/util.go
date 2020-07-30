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

package services

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"
)

// terminationSignals are signals that cause the program to exit in the
// supported platforms (linux, darwin, windows).
var terminationSignals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

var insecureHTTPClient = &http.Client{}

func init() {
	insecureTransport := http.DefaultTransport.(*http.Transport).Clone()
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	insecureHTTPClient.Transport = insecureTransport
}

// waitForTerminationSignal waits for termination signal.
func waitForTerminationSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, terminationSignals...)
	<-sig
}

// readinessCheck checks whether services are ready via the supplied health
// check URLs. Once there is an error in errCh, the function will stop waiting
// and return the error.
func readinessCheck(name string, urls []string, errCh <-chan error) error {
	klog.Infof("Running readiness check for service %q", name)
	endTime := time.Now().Add(*serverStartTimeout)
	blockCh := make(chan error)
	defer close(blockCh)
	for endTime.After(time.Now()) {
		select {
		// We *always* want to run the health check if there is no error on the channel.
		// With systemd, reads from errCh report nil because cmd.Run() waits
		// on systemd-run, rather than the service process. systemd-run quickly
		// exits with status 0, causing the channel to be closed with no error. In
		// this case, you want to wait for the health check to complete, rather
		// than returning from readinessCheck as soon as the channel is closed.
		case err, ok := <-errCh:
			if ok { // The channel is not closed, this is a real error
				if err != nil { // If there is an error, return it
					return err
				}
				// If not, keep checking readiness.
			} else { // The channel is closed, this is only a zero value.
				// Replace the errCh with blockCh to avoid busy loop,
				// and keep checking readiness.
				errCh = blockCh
			}
		case <-time.After(time.Second):
			ready := true
			for _, url := range urls {
				if !healthCheck(url) {
					ready = false
					break
				}
			}
			if ready {
				return nil
			}
		}
	}
	return fmt.Errorf("e2e service %q readiness check timeout %v", name, *serverStartTimeout)
}

// Perform a health check. Anything other than a 200-response is treated as a failure.
// Skip verification of server certs.
func healthCheck(url string) bool {
	resp, err := insecureHTTPClient.Head(url)
	return err == nil && resp.StatusCode == http.StatusOK
}
