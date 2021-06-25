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

package policy

import (
	"errors"
	"fmt"
	"sort"

	"k8s.io/pod-security-admission/api"
)

// Registry holds the Checks that are used to validate a policy.
type Registry interface { // FIXME: Consider renaming this with the new interface?
	// CheckPod checks the given pod against all the checks registered for the given level & version.
	CheckPod(lv LevelVersion, podMetadata *metav1.ObjectMeta, podSpec *corev1.PodSpec) []CheckResult
}

// CheckRegistry provides a default implementation of a Registry.
type CheckRegistry struct {
	// The checks are a map of check_ID -> sorted slice of versioned checks, newest first
	baselineChecks, restrictedChecks map[string][]versionedCheck
}

type versionedCheck struct {
	firstVersion api.Version
	Check
}

func NewCheckRegistry(baselineChecks, restrictedChecks []VersionedCheck) (*CheckRegistry, error) {
	r := &CheckRegistry{
		baselineChecks:   map[string][]versionedCheck{},
		restrictedChecks: map[string][]versionedCheck{},
	}
	for _, check := range baselineChecks {
		if err := r.AddCheck(api.LevelBaseline, check); err != nil {
			return nil, err
		}
	}
	for _, check := range restrictedChecks {
		if err := r.AddCheck(api.LevelRestricted, check); err != nil {
			return nil, err
		}
	}
	return r
}

func (r *CheckRegistry) CheckForIDAndVersion(id string, version api.Version) (Check, error) {
	checks, ok := r.baselineChecks[id]
	if !ok {
		checks, ok = r.restrictedChecks[id]
		if !ok {
			return nil, fmt.Errorf("check %s not found", id)
		}
	}
	for _, check := range checks {
		if !version.Older(&check.firstVersion) {
			return check, nil
		}
	}
	firstVersion := checks[len(checks)-1].firstVersion
	return nil, fmt.Errorf("version %s is older than the first version %s of check %s", version.String(), firstVersion.String(), id)
}

func (r *CheckRegistry) ChecksForLevelAndVersion(level api.Level, version api.Version) ([]Check, error) {
	if level == api.LevelPrivileged {
		return nil, nil
	} else if !level.Valid() {
		return nil, fmt.Errorf("invalid level %s", level)
	}
	var checks []Check

	// baseline checks are included for both the baseline & restricted levels
	for _, versionedChecks := range r.baselineChecks {
		for _, check := range versionedChecks {
			if !version.Older(&check.firstVersion) {
				checks = append(checks, check)
				break
			}
		}
	}
	if level == api.LevelBaseline {
		return checks, nil
	}

	for _, versionedChecks := range r.restrictedChecks {
		for _, check := range versionedChecks {
			if !version.Older(&check.firstVersion) {
				checks = append(checks, check)
				break
			}
		}
	}
	return checks, nil
}

// AddCheck registers a Check at the given level. The checks are represented as a map of version ->
// Check, where the version represents the first version that the associated check should be used.
// All checks must answer the same ID. If the id is already registered, an error is returned.
// Checks can only be added for baseline and restricted levels.
func (r *CheckRegistry) AddCheck(level api.Level, check VersionedCheck) error {

	// FIXME: update this logic
	id := ""
	versionedChecks := make([]versionedCheck, 0, len(checks))
	for v, check := range checks {
		if check.ID() == "" {
			return fmt.Errorf("missing ID for check version %s", v)
		}
		if id == "" {
			id = check.ID()
		} else if id != check.ID() {
			return fmt.Errorf("check ID mismatch: %s != %s (%s)", id, check.ID(), v)
		}
		version, err := api.VersionToEvaluate(v)
		if err != nil {
			return fmt.Errorf("failed to parse version %s: %w", v, err)
		} else if version.Latest() {
			return errors.New("cannot register add a check for the 'latest' version")
		}
		versionedChecks = append(versionedChecks, versionedCheck{version, check})
	}

	if _, ok := r.restrictedChecks[id]; ok {
		return fmt.Errorf("check %s already registered as under restricted", id)
	}
	if _, ok := r.baselineChecks[id]; ok {
		return fmt.Errorf("check %s already registered as under baseline", id)
	}

	sort.Slice(versionedChecks, func(i, j int) bool {
		// Newest checks first
		return !versionedChecks[i].firstVersion.Older(&versionedChecks[j].firstVersion)
	})

	switch level {
	case api.LevelRestricted:
		r.restrictedChecks[id] = versionedChecks
	case api.LevelBaseline:
		r.baselineChecks[id] = versionedChecks
	case api.LevelPrivileged:
		return errors.New("cannot register checks for the privileged level")
	default:
		return fmt.Errorf("unknown level: %s", level)
	}

	return nil
}

var (
	defaultBaselineChecks, defaultRestrictedChecks []func()VersionedCheck
)

func DefaultChecks() (baseline, restricted []VersionedCheck) {
	var baseline, restricted []VersionedCheck
	for _, fn := range defaultBaselineChecks{
		baseline = append(baseline, fn())
	}
	for _, fn := range defaultRestrictedChecks{
		restricted = append(restricted, fn())
	}
	return baseline, restricted
}

func DefaultRegistry() Registry {
	baselineChecks, restrictedChecks := policy.DefaultChecks()
	registry, err := policy.NewCheckRegistry(baselineChecks, restrictedChecks)
	if err != nil {
		panic(err) // FIXME: should this just return the error instead?
	}
	return registry
}

func registerCheck(level api.Level, checkFn func()VersionedCheck) {
	// FIXME: verify that check.ID is not already registered.
	switch level {
	case api.LevelBaseline:
		defaultBaselineChecks = append(defaultBaselineChecks, checkFn)
	case api.LevelRestricted:
		defaultRestrictedChecks = append(defaultRestrictedChecks, checkFn)
	case api.LevelPrivileged:
		panic("Cannot register checks for the privileged level.")
	default:
		panic(fmt.Sprintf("Unknown level %s", level))
	}
}

type checkSpec struct {
	id              string
	name            string
	podFields       []string
	containerFields []string
}
