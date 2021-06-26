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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/pod-security-admission/api"
	"k8s.io/pod-security-admission/policy"
)

// Options hold configuration for running integration tests against an existing server.
type Options struct {
	// Client is a client interface with sufficient permission to create, update, and delete
	// namespaces, pods, and pod-template-containing objects.
	// Required.
	Client kubernetes.Interface

	// CreateNamespace is an optional stub for creating a namespace with the given name and labels.
	// Returning an error fails the test.
	// If nil, DefaultCreateNamespace is used.
	CreateNamespace func(client kubernetes.Interface, name string, labels map[string]string) (*corev1.Namespace, error)

	// These are the check ids/starting versions to exercise.
	// If unset, policy.DefaultChecks() are used.
	Checks []policy.LevelCheck

	// ExemptClient is an optional client interface to exercise behavior of an exempt client.
	ExemptClient kubernetes.Interface
	// ExemptNamespaces are optional namespaces not expected to have PodSecurity controls enforced.
	ExemptNamespaces []string
	// ExemptRuntimeClasses are optional runtimeclasses not expected to have PodSecurity controls enforced.
	ExemptRuntimeClasses []string
}

func toJSON(pod *corev1.Pod) string {
	data, _ := json.Marshal(pod)
	return string(data)
}

// checksForLevelAndVersion returns the set of check IDs that apply when evaluating the given level and version.
// checks are assumed to be well-formed and valid to pass to policy.NewCheckRegistry().
// level must be api.LevelRestricted or api.LevelBaseline
func checksForLevelAndVersion(checks []policy.LevelCheck, level api.Level, version api.Version) ([]string, error) {
	retval := []string{}
	for _, check := range checks {
		checkVersion, err := api.VersionToEvaluate(check.Versions[0].MinimumVersion)
		if err != nil {
			return nil, err
		}
		if !version.Older(checkVersion) && (level == check.Level || level == api.LevelRestricted) {
			retval = append(retval, check.ID)
		}
	}
	return retval, nil
}

// maxMinorVersionToTest returns the maximum minor version to exercise for a given set of checks.
// checks are assumed to be well-formed and valid to pass to policy.NewCheckRegistry().
func maxMinorVersionToTest(checks []policy.LevelCheck) (int, error) {
	// start with the release under development (1.22 at time of writing).
	// this can be incremented to the current version whenever is convenient.
	maxTestMinor := 22
	for _, check := range checks {
		lastCheckVersion, err := api.VersionToEvaluate(check.Versions[len(check.Versions)-1].MinimumVersion)
		if err != nil {
			return 0, err
		}
		if lastCheckVersion.Major() != 1 {
			return 0, fmt.Errorf("expected major version 1, got ")
		}
		if lastCheckVersion.Minor() > maxTestMinor {
			maxTestMinor = lastCheckVersion.Minor()
		}
	}
	return maxTestMinor, nil
}

// and ensures pod fixtures expected to pass and fail against that level/version work as expected.
func Run(t *testing.T, opts Options) {
	if opts.Client == nil {
		t.Fatal("Client is required")
	}

	if opts.CreateNamespace == nil {
		opts.CreateNamespace = DefaultCreateNamespace
	}
	if len(opts.Checks) == 0 {
		opts.Checks = policy.DefaultChecks()
	}
	_, err := policy.NewCheckRegistry(opts.Checks)
	if err != nil {
		t.Fatalf("invalid checks: %v", err)
	}
	maxMinor, err := maxMinorVersionToTest(opts.Checks)
	if err != nil {
		t.Fatalf("invalid checks: %v", err)
	}

	for _, level := range []api.Level{api.LevelBaseline, api.LevelRestricted} {
		for minor := 0; minor <= maxMinor; minor++ {
			version := api.MajorMinorVersion(1, minor)

			// create test name
			ns := fmt.Sprintf("podsecurity-%s-1-%d", level, minor)

			// create namespace
			_, err := opts.CreateNamespace(opts.Client, ns, map[string]string{
				api.EnforceLevelLabel:   string(level),
				api.EnforceVersionLabel: fmt.Sprintf("v1.%d", minor),
			})
			if err != nil {
				t.Errorf("failed creating namespace %s: %v", ns, err)
				continue
			}
			t.Cleanup(func() {
				opts.Client.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
			})

			// create service account (to allow pod to pass serviceaccount admission)
			sa, err := opts.Client.CoreV1().ServiceAccounts(ns).Create(
				context.Background(),
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				metav1.CreateOptions{},
			)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				t.Errorf("failed creating serviceaccount %s: %v", ns, err)
				continue
			}
			t.Cleanup(func() {
				opts.Client.CoreV1().ServiceAccounts(ns).Delete(context.Background(), sa.Name, metav1.DeleteOptions{})
			})

			// create pod
			createPod := func(t *testing.T, i int, pod *corev1.Pod, expectSuccess bool, expectErrorSubstring string) {
				t.Helper()
				// avoid mutating original pod fixture
				pod = pod.DeepCopy()
				// assign pod name and serviceaccount
				pod.Name = "test"
				pod.Spec.ServiceAccountName = "default"
				// dry-run create
				_, err := opts.Client.CoreV1().Pods(ns).Create(context.Background(), pod, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
				if !expectSuccess {
					if err == nil {
						t.Errorf("%d: expected error creating %s, got none", i, toJSON(pod))
					}
					if strings.Contains(err.Error(), policy.UnknownForbiddenReason) {
						t.Errorf("%d: unexpected unknown forbidden reason creating %s: %v", i, toJSON(pod), err)
					}
					if !strings.Contains(err.Error(), expectErrorSubstring) {
						t.Errorf("%d: expected error with substring %q, got %v", i, expectErrorSubstring, err)
					}
				}
				if expectSuccess && err != nil {
					t.Errorf("%d: unexpected error creating %s: %v", i, toJSON(pod), err)
				}
			}

			minimalValidPod, err := getMinimalValidPod(level, version)
			if err != nil {
				t.Fatal(err)
			}
			t.Run(ns+"_pass_base", func(t *testing.T) {
				createPod(t, 0, minimalValidPod.DeepCopy(), true, "")
			})

			checkIDs, err := checksForLevelAndVersion(opts.Checks, level, version)
			if err != nil {
				t.Fatal(err)
			}
			if len(checkIDs) == 0 {
				t.Fatal(fmt.Errorf("no checks registered for %s/1.%d", level, minor))
			}
			for _, checkID := range checkIDs {
				checkData, err := getFixtures(fixtureKey{level: level, version: version, check: checkID})
				if err != nil {
					t.Fatal(err)
				}

				t.Run(ns+"_pass_"+checkID, func(t *testing.T) {
					for i, pod := range checkData.pass {
						createPod(t, i, pod, true, "")
					}
				})
				t.Run(ns+"_fail_"+checkID, func(t *testing.T) {
					for i, pod := range checkData.fail {
						createPod(t, i, pod, false, checkData.expectErrorSubstring)
					}
				})
			}
		}
	}
}

func DefaultCreateNamespace(client kubernetes.Interface, name string, labels map[string]string) (*corev1.Namespace, error) {
	return client.CoreV1().Namespaces().Create(
		context.Background(),
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		},
		metav1.CreateOptions{},
	)
}
