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

package nodeinfo

import (
	"errors"
	"fmt"
	"sync/atomic"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Provider interface {
	// GetNode returns the node object, or an approximation when the real API
	// object is unavailable.
	GetNode() (*v1.Node, error)

	// GetObjectRef returns an ObjectReference for this node in the API, or an error
	// if an accurate node reference cannot be provided.
	GetObjectRef() (*v1.ObjectReference, error)

	// GetEventRef gets a node ObjectReference to use when recording events.
	// Specifically, it spoofs the node UID when the real UID is unavailable.
	// See https://github.com/kubernetes/kubernetes/issues/42701 for background.
	GetEventRef() *v1.ObjectReference
}

type provider struct {
	nodeName   string
	nodeGetter NodeGetter

	// initalNodeFn approximates a node object from the available information when the API server
	// can't be reached.  This function must be idempotent.
	initialNodeFn func() (*v1.Node, error)
	initialNode   atomic.Value // type *v1.Node

	// Cached UID value
	nodeUID atomic.Value // type types.UID
}

var _ Provider = &provider{}

// NodeGetter is the subset of "k8s.io/client-go/listers/core/v1".NodeLister
// used by the nodeinfo Provider.
type NodeGetter interface {
	// Get retrieves the Node from the index for a given name.
	Get(name string) (*v1.Node, error)
}

// NewProvider instantiates a new node info provider.
func NewProvider(nodeName string, nodeGetter NodeGetter) Provider {
	return &provider{
		nodeName:   nodeName,
		nodeGetter: nodeGetter,
	}
}

// SetInitialNodeFn sets the function that's used to construct the initial node
// object.
// TODO: This function is only necessary because of a circular dependency with
// the kubelet initialization. Ideally the dependency initialization would be
// reordered so this can be passed in to NewProvider instead.
func (p *provider) SetInitialNodeFn(initialNodeFn func() (*v1.Node, error)) {
	p.initialNodeFn = initialNodeFn
}

// GetNode implements Provider.
// When the real node object cannot be fetched, the initialNodeFn function is
// used to generate (and cache) an approximate node.
func (p *provider) GetNode() (*v1.Node, error) {
	n, err := p.nodeGetter.Get(p.nodeName)
	if err == nil {
		return n, nil
	}

	return p.getInitialNode()
}

// GetObjectRef implements Provider.
func (p *provider) GetObjectRef() (*v1.ObjectReference, error) {
	uid, err := p.getUID()
	if err != nil {
		return nil, fmt.Errorf("cannot get UID: %v", err)
	}
	return &v1.ObjectReference{
		APIVersion: v1.SchemeGroupVersion.String(),
		Kind:       "Node",
		Name:       p.nodeName,
		UID:        uid,
	}, nil
}

// GetEventRef implements provider.
func (p *provider) GetEventRef() *v1.ObjectReference {
	uid, _ := p.getUID()
	if uid == "" {
		uid = types.UID(p.nodeName)
	}
	return &v1.ObjectReference{
		APIVersion: v1.SchemeGroupVersion.String(),
		Kind:       "Node",
		Name:       p.nodeName,
		UID:        uid,
	}
}

func (p *provider) getUID() (types.UID, error) {
	cached := p.nodeUID.Load()
	if cached != nil {
		return cached.(types.UID), nil
	}

	n, err := p.nodeGetter.Get(p.nodeName)
	if err != nil {
		return "", err
	}

	p.nodeUID.Store(n.UID)
	return n.UID, nil
}

func (p *provider) getInitialNode() (*v1.Node, error) {
	if p.initialNodeFn == nil {
		return nil, errors.New("initialNodeFn is unset")
	}

	cached := p.initialNode.Load()
	if cached != nil {
		return cached.(*v1.Node).DeepCopy(), nil
	}

	// This doesn't need to be synchronized since we require the function to be idempotent.
	// We don't use sync.Once here because we want to retry on error.
	n, err := p.initialNodeFn()
	if err == nil && n != nil {
		p.initialNode.Store(n)
	}
	return n.DeepCopy(), err
}
