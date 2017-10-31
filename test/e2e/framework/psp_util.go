/*
Copyright 2017 The Kubernetes Authors.

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

package framework

import (
	"fmt"
	"sync"

	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
)

const (
	PodSecurityPolicyPrivileged     = "gce.privileged"
	PodSecurityPolicyPrivilegedRole = "gce:podsecuritypolicy:privileged"
)

var (
	isPSPEnabledOnce sync.Once
	isPSPEnabled     bool
)

func IsPodSecurityPolicyEnabled(f *Framework) bool {
	isPSPEnabledOnce.Do(func() {
		psps, err := f.ClientSet.ExtensionsV1beta1().PodSecurityPolicies().List(metav1.ListOptions{})
		if err != nil {
			Logf("Error listing PodSecurityPolicies; assuming PodSecurityPolicy is disabled: %v", err)
			isPSPEnabled = false
		} else if psps == nil || len(psps.Items) == 0 {
			Logf("No PodSecurityPolicies found; assuming PodSecurityPolicy is disabled.")
			isPSPEnabled = false
		} else {
			Logf("Found PodSecurityPolicies; assuming PodSecurityPolicy is enabled.")
			isPSPEnabled = true
		}
	})
	return isPSPEnabled
}

func CreateDefaultPSPBinding(f *Framework, namespace string) {
	By(fmt.Sprintf("Binding the %s PodSecurityPolicy to the default service account in %s",
		PodSecurityPolicyPrivileged, namespace))
	BindClusterRoleInNamespace(f.ClientSet.RbacV1beta1(),
		PodSecurityPolicyPrivilegedRole,
		namespace,
		rbacv1beta1.Subject{
			Kind:      rbacv1beta1.ServiceAccountKind,
			Namespace: namespace,
			Name:      "default",
		})
}
