package settingsui

import (
	"omni/internal/config"
	"omni/internal/hotkey"
)

// ShortcutSet is the registered hotkeys that run apps, scripts and the like; *hotkey.Set satisfies it.
type ShortcutSet interface {
	Apply([]hotkey.Binding)
	Errors() map[string]string
}

// ShortcutBindings are the shortcuts as hotkeys that run their targets.
func ShortcutBindings(list []config.Shortcut) []hotkey.Binding {
	out := make([]hotkey.Binding, len(list))
	for i, s := range list {
		out[i] = hotkey.Binding{Keys: s.Keys, Target: s.Target}
	}
	return out
}

// SaveShortcuts replaces the shortcuts. Like the other hotkeys, a combination that cannot be registered (another program owns it)
// is refused with the reason, and the old list stays; one that was already in the list and fails is left alone, so it cannot
// block the saving of another.
func (s *Service) SaveShortcuts(list []config.Shortcut) string {
	old := s.o.Store.Get().Shortcuts
	if msg := reason(s.o.Store.Update(func(c *config.Config) { c.Shortcuts = list })); msg != "" {
		return msg
	}
	if s.o.Shortcuts == nil {
		return ""
	}
	s.o.Shortcuts.Apply(ShortcutBindings(list))
	errs := s.o.Shortcuts.Errors()
	known := map[string]bool{}
	for _, o := range old {
		known[o.Keys] = true
	}
	for _, n := range list {
		if msg := errs[n.Keys]; msg != "" && !known[n.Keys] {
			_ = s.o.Store.Update(func(c *config.Config) { c.Shortcuts = old })
			s.o.Shortcuts.Apply(ShortcutBindings(old))
			return msg
		}
	}
	return ""
}

// shortcutErrors says why each saved combination is not working, by combination.
func (s *Service) shortcutErrors() map[string]string {
	if s.o.Shortcuts == nil {
		return nil
	}
	return s.o.Shortcuts.Errors()
}
