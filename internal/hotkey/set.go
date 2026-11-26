package hotkey

import (
	"fmt"
	"log"
	"sync"
)

// Binding is a hotkey and what it runs: the id of a row of the bar.
type Binding struct{ Keys, Target string }

// Set is the hotkeys the person made for apps, scripts and the like: any number, changed together. Apply brings the registered
// hotkeys to a new list, touching only the ones that differ, so a combination another program owns fails alone and leaves
// every other hotkey working.
type Set struct {
	keys Keys
	run  func(target string)

	apply sync.Mutex // one Apply at a time; Register may wait for the main thread, so mu is never held across it

	mu      sync.Mutex
	targets map[string]string // what each registered combination runs
	errs    map[string]string // why a combination of the last Apply could not be used
}

// NewSet returns a Set whose hotkeys call run with their target, on the thread that receives the hotkey: run must not block.
func NewSet(keys Keys, run func(target string)) *Set {
	return &Set{keys: keys, run: run, targets: map[string]string{}, errs: map[string]string{}}
}

// Apply registers the hotkeys of want that are not yet, releases the ones no longer wanted, and points the kept ones at their
// current target. A combination that failed before is tried again.
func (s *Set) Apply(want []Binding) {
	s.apply.Lock()
	defer s.apply.Unlock()
	wanted := make(map[string]string, len(want))
	for _, b := range want {
		wanted[b.Keys] = b.Target
	}
	var drop, add []string
	s.mu.Lock()
	for keys := range s.targets {
		if _, ok := wanted[keys]; !ok {
			drop = append(drop, keys)
		}
	}
	for keys, target := range wanted {
		if _, ok := s.targets[keys]; ok {
			s.targets[keys] = target
		} else {
			add = append(add, keys)
		}
	}
	s.mu.Unlock()

	for _, keys := range drop {
		if err := s.keys.Unregister(keys); err != nil {
			log.Printf("hotkey: %s could not be released: %v", keys, err)
		}
		s.mu.Lock()
		delete(s.targets, keys)
		s.mu.Unlock()
	}
	errs := map[string]string{}
	for _, keys := range add {
		s.mu.Lock()
		s.targets[keys] = wanted[keys] // before the registration: a press right after it must find its target
		s.mu.Unlock()
		if err := s.keys.Register(keys, func() { s.fire(keys) }); err != nil {
			s.mu.Lock()
			delete(s.targets, keys)
			s.mu.Unlock()
			errs[keys] = fmt.Sprintf("the hotkey %s cannot be used: %v", keys, err)
		}
	}
	s.mu.Lock()
	s.errs = errs
	s.mu.Unlock()
}

func (s *Set) fire(keys string) {
	s.mu.Lock()
	target := s.targets[keys]
	s.mu.Unlock()
	if target != "" {
		s.run(target)
	}
}

// Errors says why each combination of the last Apply could not be used, by combination; empty when all work.
func (s *Set) Errors() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.errs))
	for k, v := range s.errs {
		out[k] = v
	}
	return out
}
