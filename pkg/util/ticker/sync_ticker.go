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

package ticker

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

type Synchronizer struct {
	tickerLock sync.RWMutex
	tickers    map[time.Duration]*Ticker
}

// TODO: Make this a parameter to Synchronizer as necessary.
const granularity = time.Second

func (s *Synchronizer) NewTicker(period time.Duration) *Ticker {
	if period%granularity != 0 {
		// FIXME - what should the failure mode be?
		glog.Fatalf("Period (%d) must be divisible by synchronizer granularity (%d)", period, granularity)
	}

	c := make(chan Tick) // FIXME - maybe this should be buffered?
	t := &Ticker{C: c, c: c}

	s.tickerLock.Lock()
	defer s.tickerLock.Unlock()
	t.next = s.tickers[period]
	s.tickers[period] = t
}

func (s *Synchronizer) Run() {
	var count uint32
	for _ = range time.Tick(granularity) {
		count++
		s.tick(count)
	}
}

func (s *Synchronizer) tick(count uint32) {
	defer HandleCrash()
	tickerLock.Lock()
	defer tickerLock.Unlock()

	for period := range s.tickers {
		if count%period != 0 {
			continue
		}

		// Find fist non-stopped ticker.
		t := s.tickers[period]
		for ; t != nil && t.stopped == stopped; t = t.next {
		}
		s.tickers[period] = t
		for t != nil {
			// Don't block if there is no receiver.
			select {
			case t.c <- Tick{}:
			default:
			}

			// Find next non-stopped ticker.
			next := t.next
			for ; next != nil && next.stopped == stopped; next = next.next {
			}
			t.next = next
			t = next
		}
	}
}

const (
	stopped = 1
)

type Ticker struct {
	C <-chan Tick

	// Same channel as C, used for sending Ticks.
	c       chan Tick
	stopped int32
	next    *Ticker // Linked list
}

func (t *Ticker) Stop() {
	atomic.StoreInt32(&t.stopped, stopped)
}

type Tick struct{}
