package palette

import (
	"slices"
	"strings"
	"time"

	"omni/internal/config"
	"omni/internal/suggest"
)

// QuickOpen is the row of apps under the empty bar, in slot order: the apps set by hand in their slots, and the
// other slots filled with suggestions (internal/suggest), worked out here from the launch times kept for ranking.
// An app that is gone is skipped. Off, or with nothing to show, it is empty.
func (s *Service) QuickOpen() []Item {
	if s.x.QuickOpen == nil {
		return []Item{}
	}
	q := s.x.QuickOpen()
	if !q.Enabled {
		return []Item{}
	}
	slots := make([]*Item, q.Count)
	taken := map[string]bool{}
	for i, id := range q.Apps {
		if i >= q.Count || id == "" {
			continue
		}
		if item, ok := s.itemFor(id); ok && item.Kind == "app" {
			slots[i] = &item
			taken[id] = true
		}
	}
	if q.Suggest {
		for _, id := range suggest.Rank(s.appUses(), s.now(), q.Count+len(taken)) {
			free := slices.Index(slots, nil)
			if free < 0 {
				break
			}
			if taken[id] {
				continue
			}
			if item, ok := s.itemFor(id); ok && item.Kind == "app" {
				slots[free] = &item
				taken[id] = true
			}
		}
	}
	if q.Suggest && slices.Contains(slots, nil) {
		s.fillWithStarters(slots, taken)
	}
	items := []Item{}
	for _, item := range slots {
		if item != nil {
			items = append(items, *item)
		}
	}
	return items
}

// starterApps fill the slots that neither the person nor the history has filled yet (a fresh install, or just after
// forgetting the history), in this order, where they are installed. They give way to real suggestions as the history grows.
var starterApps = []string{"Safari", "Notes", "Mail", "Messages", "Calendar", "Music", "Microsoft Edge", "File Explorer", "Notepad"}

func (s *Service) fillWithStarters(slots []*Item, taken map[string]bool) {
	byName := map[string]string{}
	for _, app := range s.catalog.Index.All() {
		byName[app.Name] = "app:" + app.ID
	}
	for _, name := range starterApps {
		free := slices.Index(slots, nil)
		if free < 0 {
			return
		}
		id, ok := byName[name]
		if !ok || taken[id] {
			continue
		}
		if item, ok := s.itemFor(id); ok {
			slots[free] = &item
			taken[id] = true
		}
	}
}

// appUses is every app launch Omni remembers, by palette id.
func (s *Service) appUses() map[string][]time.Time {
	uses := map[string][]time.Time{}
	for _, use := range s.history.Since(time.Time{}) {
		if strings.HasPrefix(use.ID, "app:") {
			uses[use.ID] = append(uses[use.ID], use.At)
		}
	}
	return uses
}

// QuickOpenSettings is what the quick-open row is set to, for the bar to know how many number keys open a slot.
func (s *Service) QuickOpenSettings() config.QuickOpen {
	if s.x.QuickOpen == nil {
		return config.QuickOpen{}
	}
	return s.x.QuickOpen()
}
