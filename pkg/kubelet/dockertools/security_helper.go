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

package dockertools

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"k8s.io/client-go/pkg/api"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/security/apparmor"
)

type OptsHelper interface {
	// Get the security options for the container named ctrName according to the annotations.
	// TODO: Rethink this API once security features are moved out of annotations.
	GetSecurityOpts(annotations map[string]string, ctrName string) ([]DockerOpt, error)
	FmtDockerOpts([]DockerOpt) ([]string, error)
}

func NewOptsHelper(
	apiVersion kubecontainer.Version,
	appArmorValidator apparmor.Validator,
	seccompProfileRoot string,
) {
	return &optsHelper{
		apiVersion:         apiVersion,
		appArmorValidator:  appArmorValidator,
		seccompProfileRoot: seccompProfileRoot,
	}
}

type DockerOpt struct {
	// The key-value pair passed to docker.
	Key, Value string
	// The alternative value to use in log/event messages.
	Msg string
}

const (
	minSeccompAPIVersion = dockerV110APIVersion

	// Docker changed the API for specifying options in v1.11
	optSeparatorChangeVersion = "1.23" // Corresponds to docker 1.11.x
	optSeparatorOld           = ':'
	optSeparatorNew           = '='
)

var (
	// Default set of seccomp security options.
	defaultSeccompOpt = []DockerOpt{{"seccomp", "unconfined", ""}}
)

type optsHelper struct {
	apiVersion         kubecontainer.Version
	appArmorValidator  apparmor.Validator
	seccompProfileRoot string
}

func (h *optsHelper) FmtDockerOpts(opts []DockerOpt) ([]string, error) {
	sep := optSeparatorNew
	if result, err := h.apiVersion.Compare(optSeparatorChangeVersion); err != nil {
		return nil, err
	} else if result < 0 {
		sep = optSeparatorOld
	}

	fmtOpts := make([]string, len(opts))
	for i, opt := range opts {
		fmtOpts[i] = fmt.Sprintf("%s%c%s", opt.key, sep, opt.value)
	}
	return fmtOpts, nil
}

func (h *optsHelper) GetSecurityOpts(annotations map[string]string, ctrName string) ([]DockerOpt, error) {
	var securityOpts []DockerOpt
	if seccompOpts, err := h.getSeccompOpts(annotations, ctrName); err != nil {
		return nil, err
	} else {
		securityOpts = append(securityOpts, seccompOpts...)
	}

	if appArmorOpts, err := h.getAppArmorOpts(annotations, ctrName); err != nil {
		return nil, err
	} else {
		securityOpts = append(securityOpts, appArmorOpts...)
	}

	return securityOpts, nil
}

// Get the docker security options for seccomp.
func (h *optsHelper) getSeccompOpts(annotations map[string]string, ctrName string) ([]DockerOpt, error) {
	// seccomp is only on docker versions >= v1.10
	if result, err := h.apiVersion.Compare(minSeccompAPIVersion); err != nil {
		return nil, err
	} else if result < 0 {
		return nil, nil // return early for Docker < 1.10
	}

	profile, profileOK := annotations[api.SeccompContainerAnnotationKeyPrefix+ctrName]
	if !profileOK {
		// try the pod profile
		profile, profileOK = annotations[api.SeccompPodAnnotationKey]
		if !profileOK {
			// return early the default
			return defaultSeccompOpt, nil
		}
	}

	if profile == "unconfined" {
		// return early the default
		return defaultSeccompOpt, nil
	}

	if profile == "docker/default" {
		// return nil so docker will load the default seccomp profile
		return nil, nil
	}

	if !strings.HasPrefix(profile, "localhost/") {
		return nil, fmt.Errorf("unknown seccomp profile option: %s", profile)
	}

	name := strings.TrimPrefix(profile, "localhost/") // by pod annotation validation, name is a valid subpath
	fname := filepath.Join(h.seccompProfileRoot, filepath.FromSlash(name))
	file, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("cannot load seccomp profile %q: %v", name, err)
	}

	b := bytes.NewBuffer(nil)
	if err := json.Compact(b, file); err != nil {
		return nil, err
	}
	// Rather than the full profile, just put the filename & md5sum in the event log.
	msg := fmt.Sprintf("%s(md5:%x)", name, md5.Sum(file))

	return []DockerOpt{{"seccomp", b.String(), msg}}, nil
}

// Get the docker security options for AppArmor.
func (h *optsHelper) getAppArmorOpts(annotations map[string]string, ctrName string) ([]DockerOpt, error) {
	profile := apparmor.GetProfileNameFromPodAnnotations(annotations, ctrName)
	if profile == "" || profile == apparmor.ProfileRuntimeDefault {
		// The docker applies the default profile by default.
		return nil, nil
	}

	if err := h.appArmorValidator.ValidateProfile(profile); err != nil {
		return nil, err
	}

	profileName := strings.TrimPrefix(profile, apparmor.ProfileNamePrefix)
	return []DockerOpt{{"apparmor", profileName, ""}}, nil
}
