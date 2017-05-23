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

package aggregator

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/apis/audit"
)

// FIXME - REMOVE THIS vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv

type Sink interface {
	ProcessEvents(events ...*audit.Event)
}

type Backend interface {
	Sink

	Run(stopCh <-chan struct{}) error
}

// FIXME - REMOVE THIS ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

type aggregator struct {
	// The maximum amount of time to hold an event in cache before it is flushed to the delegate
	// backend.
	ttl   time.Duration
	clock Clock

	buffer          chan *audit.Event
	cache           map[types.UID]*cacheEntry
	cacheHead       *cacheEntry
	cacheTail       *cacheEntry
	expirationTimer time.Timer

	delegate Backend
}

// TODO: make these parameters
const (
	BufferSize = 100
	CacheSize  = 1000
	TTL        = 5 * time.Second
)

func NewAggregatorBackend(delegate Backend) Backend {
	return &aggregator{
		buffer:   make(chan *audit.Event, BufferSize),
		cache:    make(map[string]*cacheEntry),
		delegate: delegate,
	}
}

func (a *aggregator) ProcessEvents(events ...*audit.Event) {
	for _, ev := range events {
		buffer <- ev
	}
}

func (a *aggregator) Run(stopCh <-chan struct{}) error {
	delegate.Run(stopCh)

	go a.run(stopCh)

	return nil
}

func (a *aggregator) run(stopCh <-chan struct{}) error {
	a.expirationTimer = time.Timer(a.ttl)
	for {
		select {
		case <-stopCh:
			glog.V(2).Infof("Received stop: shutting down aggregator audit backend")
			return
		case ev := <-buffer:
			a.injest(ev)
		case <-a.expirationTimer:
			a.expire()
		}
	}
}

func (a *aggregator) injest(ev *audit.Event) {
	if entry, ok := a.cache[ev.AuditID]; ok {
		entry.events = append(entry.events, ev)
		if entry.complete() {
			a.send(entry)
		}
		return
	}
	entry = &cacheEntry{events: []*audit.Event{ev}}
	if entry.Complete() {
		a.send(entry)
	} else {
		a.insert(entry)
	}
}

func (a *aggregator) insert(entry *cacheEntry) {
	cache[entry.id()] = entry
	if a.cacheHead == nil {
		a.cacheHead = entry
		a.cacheTail = entry
		return
	}

	// Assume events come in roughly the right order.
	for pos := a.cacheTail; pos != nil; pos = pos.prev {
		if pos.timestamp().Before(entry.timestamp()) {
			entry.next = pos.next
			pos.next = entry
			entry.prev = pos
			if entry.next != nil {
				entry.next.prev = entry
			} else {
				a.cacheTail = entry
			}
			return
		}
	}
	entry.next = a.cacheHead
	a.cacheHead = entry
	if entry.next != nil {
		entry.next.prev = entry
	}
}

func (a *aggregator) send(entry *cacheEntry) {
	// Remove entry from cache.
	delete(a.cache, entry.id())
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
	if a.cacheHead == entry {
		a.cacheHead = entry.next
	}
	if a.cacheTail == entry {
		a.cacheTail = entry.prev
	}

	a.delegate.ProcessEvents(entry.aggregate())
}

func (a *aggregator) expire() {
	for a.cacheHead != nil && a.cacheHead.timestamp().Add(a.ttl).Before(a.clock.Now()) {
		a.send(a.cacheHead)
	}
	// Reset expiration timer.
	if a.cacheHead != nil {
		a.expirationTimer.Reset(a.cacheHead.timestamp().Add(a.ttl).Sub(a.clock.Now()))
	} else {
		a.expirationTimer.Reset(a.ttl)
	}
}

type cacheEntry struct {
	prev, next *cacheEntry
	events     []*audit.Event
}

func (e *cacheEntry) aggregate() *audit.Event {
	// TODO - intelligently combine if information is missing
	return e.events[len(e.events)-1] // FIXME - return the final stage
}

func (e *cacheEntry) complete() bool {
	// FIXME - event is complete when it receives the final stage
	return len(events) > 1
}

func (e *cacheEntry) timestamp() time.Time {
	return e.events[0].Timestamp.Time
}

func (e *cacheEntry) id() types.UID {
	return e.events[0].AuditID
}
