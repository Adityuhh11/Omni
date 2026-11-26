package settingsui

import (
	"strings"
	"testing"

	"omni/internal/config"
	"omni/internal/hotkey"
)

// fakeShortcuts plays hotkey.Set: the combinations in `taken` fail to register.
type fakeShortcuts struct {
	taken   map[string]bool
	applied [][]hotkey.Binding
	errs    map[string]string
}

func (f *fakeShortcuts) Apply(list []hotkey.Binding) {
	f.applied = append(f.applied, list)
	f.errs = map[string]string{}
	for _, b := range list {
		if f.taken[b.Keys] {
			f.errs[b.Keys] = "the hotkey " + b.Keys + " cannot be used: taken"
		}
	}
}

func (f *fakeShortcuts) Errors() map[string]string { return f.errs }

func TestSavingShortcutsRegistersThemAndShowsThemInTheView(t *testing.T) {
	h := newHarness(t)
	set := &fakeShortcuts{}
	h.svc.o.Shortcuts = set
	list := []config.Shortcut{{Keys: "Ctrl+Alt+T", Target: "app:terminal", Title: "Terminal"}}
	if msg := h.svc.SaveShortcuts(list); msg != "" {
		t.Fatal(msg)
	}
	if len(set.applied) != 1 || set.applied[0][0] != (hotkey.Binding{Keys: "Ctrl+Alt+T", Target: "app:terminal"}) {
		t.Fatalf("applied %v", set.applied)
	}
	if got := h.svc.View().Shortcuts; len(got) != 1 || got[0].Title != "Terminal" {
		t.Fatalf("view shortcuts = %+v", got)
	}
}

func TestAShortcutOnATakenCombinationIsRefusedAndTheOldListStays(t *testing.T) {
	h := newHarness(t)
	set := &fakeShortcuts{taken: map[string]bool{"Ctrl+Alt+Z": true}}
	h.svc.o.Shortcuts = set
	keep := []config.Shortcut{{Keys: "Ctrl+Alt+T", Target: "app:terminal"}}
	if msg := h.svc.SaveShortcuts(keep); msg != "" {
		t.Fatal(msg)
	}
	msg := h.svc.SaveShortcuts(append(keep, config.Shortcut{Keys: "Ctrl+Alt+Z", Target: "app:other"}))
	if !strings.Contains(msg, "cannot be used") {
		t.Fatalf("message = %q", msg)
	}
	if got := h.svc.View().Shortcuts; len(got) != 1 || got[0].Keys != "Ctrl+Alt+T" {
		t.Fatalf("the refused shortcut was kept: %+v", got)
	}
	if last := set.applied[len(set.applied)-1]; len(last) != 1 {
		t.Fatalf("the registrations were not put back: %v", last)
	}
}

func TestAShortcutThatFailedBeforeDoesNotBlockSavingAnother(t *testing.T) {
	h := newHarness(t)
	set := &fakeShortcuts{taken: map[string]bool{"Ctrl+Alt+Z": true}}
	h.svc.o.Shortcuts = set
	// Saved by hand-editing the file, or when the other program still let go: it is in the list but could not be registered.
	h.store.Update(func(c *config.Config) { c.Shortcuts = []config.Shortcut{{Keys: "Ctrl+Alt+Z", Target: "app:x"}} })
	set.Apply([]hotkey.Binding{{Keys: "Ctrl+Alt+Z", Target: "app:x"}})
	list := []config.Shortcut{{Keys: "Ctrl+Alt+Z", Target: "app:x"}, {Keys: "Ctrl+Alt+Y", Target: "app:y"}}
	if msg := h.svc.SaveShortcuts(list); msg != "" {
		t.Fatalf("message = %q", msg)
	}
	if v := h.svc.View(); len(v.Shortcuts) != 2 || v.ShortcutErrors["Ctrl+Alt+Z"] == "" {
		t.Fatalf("view = %+v", v)
	}
}

func TestSavingAShortcutOnAnotherJobsHotkeyIsRefusedByTheConfig(t *testing.T) {
	h := newHarness(t)
	h.svc.o.Shortcuts = &fakeShortcuts{}
	msg := h.svc.SaveShortcuts([]config.Shortcut{{Keys: h.store.Get().Hotkey, Target: "app:x"}})
	if !strings.Contains(msg, "launcher's own") {
		t.Fatalf("message = %q", msg)
	}
}
