package hotkey

import (
	"errors"
	"testing"
)

type fakeKeys struct {
	bound  map[string]func()
	taken  map[string]bool
	events []string
}

func (f *fakeKeys) Register(a string, fn func()) error {
	if f.taken[a] {
		return errors.New("already registered")
	}
	f.bound[a] = fn
	f.events = append(f.events, "register "+a)
	return nil
}

func (f *fakeKeys) Unregister(a string) error {
	delete(f.bound, a)
	f.events = append(f.events, "unregister "+a)
	return nil
}

func TestRebindRegistersTheNewKeyBeforeDroppingTheOldOne(t *testing.T) {
	keys := &fakeKeys{bound: map[string]func(){}, taken: map[string]bool{"Ctrl+Alt+Z": true}}
	slot := NewSlot(keys)
	ran := 0
	if err := slot.Rebind("Ctrl+Alt+A", func() { ran++ }); err != nil {
		t.Fatal(err)
	}
	keys.bound["Ctrl+Alt+A"]()
	if ran != 1 {
		t.Fatal("the hotkey did not run its function")
	}
	if err := slot.Rebind("Ctrl+Alt+B", func() {}); err != nil {
		t.Fatal(err)
	}
	want := []string{"register Ctrl+Alt+A", "register Ctrl+Alt+B", "unregister Ctrl+Alt+A"}
	for i, w := range want {
		if keys.events[i] != w {
			t.Fatalf("events = %v, want %v", keys.events, want)
		}
	}
	if err := slot.Rebind("Ctrl+Alt+Z", func() {}); err == nil {
		t.Fatal("a combination another program owns was accepted")
	}
	if _, ok := keys.bound["Ctrl+Alt+B"]; !ok {
		t.Fatal("a refused combination must leave the working one in place")
	}
}

func TestAnEmptyAcceleratorTurnsTheHotkeyOffAndTheSameOneIsLeftAlone(t *testing.T) {
	keys := &fakeKeys{bound: map[string]func(){}, taken: map[string]bool{}}
	slot := NewSlot(keys)
	_ = slot.Rebind("Ctrl+Alt+A", func() {})
	_ = slot.Rebind("Ctrl+Alt+A", func() {})
	if len(keys.events) != 1 {
		t.Fatalf("the same combination was registered again: %v", keys.events)
	}
	if err := slot.Rebind("", nil); err != nil || len(keys.bound) != 0 {
		t.Fatalf("turning it off: %v, still bound %v", err, keys.bound)
	}
	slot.SetError("x")
	if slot.Error() != "x" {
		t.Fatal("SetError did not keep the message")
	}
	_ = slot.Rebind("Ctrl+Alt+B", func() {})
	if slot.Error() != "" {
		t.Fatal("a working hotkey must clear the error")
	}
}

func TestASetRegistersWhatIsAskedDropsWhatIsNotAndKeepsTheRest(t *testing.T) {
	keys := &fakeKeys{bound: map[string]func(){}, taken: map[string]bool{"Ctrl+Alt+Z": true}}
	var ran []string
	set := NewSet(keys, func(target string) { ran = append(ran, target) })

	set.Apply([]Binding{{"Ctrl+Alt+A", "app:one"}, {"Ctrl+Alt+B", "app:two"}, {"Ctrl+Alt+Z", "app:taken"}})
	if len(keys.bound) != 2 {
		t.Fatalf("bound = %v", keys.bound)
	}
	if msg := set.Errors()["Ctrl+Alt+Z"]; msg == "" || set.Errors()["Ctrl+Alt+A"] != "" {
		t.Fatalf("errors = %v: only the taken combination may fail", set.Errors())
	}
	keys.bound["Ctrl+Alt+A"]()
	keys.bound["Ctrl+Alt+B"]()
	if len(ran) != 2 || ran[0] != "app:one" || ran[1] != "app:two" {
		t.Fatalf("ran %v", ran)
	}

	// B goes, A keeps its registration but now runs another thing, C arrives.
	keys.events = nil
	set.Apply([]Binding{{"Ctrl+Alt+A", "script:x"}, {"Ctrl+Alt+C", "app:three"}})
	if got := len(keys.events); got != 2 || keys.bound["Ctrl+Alt+B"] != nil || keys.bound["Ctrl+Alt+C"] == nil {
		t.Fatalf("events = %v, bound = %v", keys.events, keys.bound)
	}
	ran = nil
	keys.bound["Ctrl+Alt+A"]()
	if len(ran) != 1 || ran[0] != "script:x" {
		t.Fatalf("after a change the old target ran: %v", ran)
	}
	if len(set.Errors()) != 0 {
		t.Fatalf("errors = %v", set.Errors())
	}
}

func TestASetTriesAgainAKeyThatFailedBefore(t *testing.T) {
	keys := &fakeKeys{bound: map[string]func(){}, taken: map[string]bool{"Ctrl+Alt+Z": true}}
	set := NewSet(keys, func(string) {})
	set.Apply([]Binding{{"Ctrl+Alt+Z", "app:x"}})
	keys.taken["Ctrl+Alt+Z"] = false // the other program let go
	set.Apply([]Binding{{"Ctrl+Alt+Z", "app:x"}})
	if keys.bound["Ctrl+Alt+Z"] == nil || len(set.Errors()) != 0 {
		t.Fatalf("bound = %v, errors = %v", keys.bound, set.Errors())
	}
}
