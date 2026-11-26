package palette

import (
	"testing"
	"time"

	"omni/internal/apps"
)

const winSettingsID = "windows.immersivecontrolpanel_cw5n1h2txyewy!microsoft.windows.immersivecontrolpanel"

func TestTypingSettingsPutsWindowsSettingsFirstWhateverElseIsUsedMore(t *testing.T) {
	s, _ := newService(t)
	s.catalog.Index.Set([]apps.App{
		{ID: "id:Settings Sync Helper", Name: "Settings Sync Helper"},
		{ID: "id:Settings Backup", Name: "Settings Backup"},
		{ID: winSettingsID, Name: "Settings"},
	})
	for i := range 20 { // the person opens the helper all the time
		if err := s.history.Record("app:id:Settings Sync Helper", "settings", s.now().Add(-time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	for _, typed := range []string{"settings", "setti", "sett", "Settings"} {
		items := s.Query(typed, ModeAll).Items
		if len(items) == 0 || items[0].ID != "app:"+winSettingsID {
			t.Errorf("typing %q put %+v first, want Windows Settings", typed, items[:min(len(items), 1)])
		}
	}
}
