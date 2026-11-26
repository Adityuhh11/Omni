package palette

import (
	"strings"
	"testing"
	"time"

	"omni/internal/config"
)

func TestQuickOpenPutsChosenAppsInTheirSlotsAndFillsTheRestFromUse(t *testing.T) {
	s, _ := newService(t, "Mail", "Music", "Calculator", "Notes", "Zed")
	q := config.QuickOpen{Enabled: true, Count: 3, Suggest: true, Apps: []string{"", "app:id:Calculator", "app:id:Gone"}}
	s.x.QuickOpen = func() config.QuickOpen { return q }
	now := s.now()
	for day := 1; day <= 3; day++ {
		for _, id := range []string{"app:id:Mail", "app:id:Calculator"} {
			if err := s.history.Record(id, "", now.AddDate(0, 0, -day)); err != nil {
				t.Fatal(err)
			}
		}
	}
	titles := func() string {
		var got []string
		for _, item := range s.QuickOpen() {
			got = append(got, item.Title)
		}
		return strings.Join(got, "|")
	}
	// Calculator is set for slot 2, Mail is suggested from use, and the gone app's slot gets the first starter app not yet shown.
	if got := titles(); got != "Mail|Calculator|Notes" {
		t.Fatalf("quick open = %s", got)
	}
	q.Suggest = false
	if got := titles(); got != "Calculator" {
		t.Fatalf("without suggestions = %s", got)
	}
	q.Enabled = false
	if got := s.QuickOpen(); len(got) != 0 {
		t.Fatalf("off, the row has %v", got)
	}
	if err := s.history.ForgetUses(); err != nil || len(s.history.Since(time.Time{})) != 0 {
		t.Fatalf("ForgetUses: %v", err)
	}
}
