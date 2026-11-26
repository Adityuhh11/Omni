package sidecar

import (
	"context"
	"sync"
	"time"
)

// Runner is what OnDemand turns on and off; *Switch is one.
type Runner interface {
	On()
	Off()
}

// OnDemand is for a person who keeps files out of the default search: the engine, and the video memory its model holds, starts
// when file search is first used and ends after it has sat idle, instead of running from the moment the launcher starts.
type OnDemand struct {
	run     Runner
	active  func() bool // on demand is what the person chose, and file search can run at all
	running func() bool // the engine is up or coming up
	busy    func() bool // it has work that must not be cut off: indexing
	idle    time.Duration
	now     func() time.Time

	mu   sync.Mutex
	used time.Time
}

// NewOnDemand returns an OnDemand that stops the engine after it was not used for idle.
func NewOnDemand(run Runner, active, running, busy func() bool, idle time.Duration) *OnDemand {
	return &OnDemand{run: run, active: active, running: running, busy: busy, idle: idle, now: time.Now}
}

// Use says that file search is wanted now. It starts the engine if it is off and reports whether it did, so the caller can say
// the answer is on its way. When on demand is not chosen it does nothing: the launcher runs the engine itself.
func (o *OnDemand) Use() (started bool) {
	if !o.active() {
		return false
	}
	o.mu.Lock()
	o.used = o.now()
	o.mu.Unlock()
	if o.running() {
		return false
	}
	o.run.On()
	return true
}

// Check stops the engine when it has been idle for the chosen time and is not indexing.
func (o *OnDemand) Check() {
	if !o.active() || !o.running() || o.busy() {
		return
	}
	o.mu.Lock()
	idle := o.now().Sub(o.used)
	o.mu.Unlock()
	if idle >= o.idle {
		o.run.Off()
	}
}

// Watch checks once a minute until ctx ends.
func (o *OnDemand) Watch(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			o.Check()
		}
	}
}
