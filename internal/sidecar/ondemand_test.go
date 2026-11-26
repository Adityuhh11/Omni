package sidecar

import (
	"testing"
	"time"
)

type fakeRunner struct {
	up        bool
	ons, offs int
}

func (f *fakeRunner) On()  { f.up = true; f.ons++ }
func (f *fakeRunner) Off() { f.up = false; f.offs++ }

func TestOnDemandStartsOnUseAndStopsWhenIdle(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	run := &fakeRunner{}
	active, busy := true, false
	d := NewOnDemand(run, func() bool { return active }, func() bool { return run.up }, func() bool { return busy }, 10*time.Minute)
	d.now = func() time.Time { return now }

	if !d.Use() || run.ons != 1 {
		t.Fatalf("the first use must start the engine (started %d times)", run.ons)
	}
	if d.Use() || run.ons != 1 {
		t.Fatal("a use while the engine runs must not start it again")
	}

	now = now.Add(9 * time.Minute)
	d.Check()
	if !run.up {
		t.Fatal("the engine stopped before the idle time")
	}

	now = now.Add(2 * time.Minute)
	busy = true
	d.Check()
	if !run.up {
		t.Fatal("the engine stopped while it was indexing")
	}

	busy = false
	d.Check()
	if run.up || run.offs != 1 {
		t.Fatal("the engine kept running after the idle time")
	}

	active = false
	if d.Use() || run.ons != 1 {
		t.Fatal("with on-demand off, Use must leave the engine to the launcher")
	}
}

func TestOnDemandUseKeepsTheEngineAlive(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	run := &fakeRunner{}
	d := NewOnDemand(run, func() bool { return true }, func() bool { return run.up }, func() bool { return false }, 10*time.Minute)
	d.now = func() time.Time { return now }
	d.Use()
	for range 5 { // a search every 8 minutes, for 40 minutes
		now = now.Add(8 * time.Minute)
		d.Use()
		d.Check()
	}
	if !run.up || run.offs != 0 {
		t.Fatal("the engine stopped while it was in use")
	}
}
