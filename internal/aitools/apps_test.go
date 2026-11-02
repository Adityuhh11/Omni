package aitools

import (
	"encoding/json"
	"sort"
	"testing"

	"omni/internal/browsers"
	"omni/internal/chat"
	"omni/internal/notes"
)

// arguments reduces a tool to its argument names and which are required, as JSON, to compare two descriptions of it.
func arguments(t *testing.T, def chat.Tool) string {
	t.Helper()
	params := def.Function.Parameters
	names := []string{}
	for name := range params["properties"].(map[string]any) {
		names = append(names, name)
	}
	sort.Strings(names)
	b, err := json.Marshal([]any{names, params["required"]})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEveryToolThatReadsHasAFullDescriptionForOtherAppsWithTheSameArguments(t *testing.T) {
	store, err := notes.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := Deps{Files: fakeFiles{}, Clips: fakeClips{}, Notes: store, Browser: func(string, int, int) ([]browsers.Page, error) { return nil, nil }}
	for _, l := range d.Tools() {
		if l.Writes {
			continue
		}
		name := l.Def.Function.Name
		apps := l.ForApps()
		if apps.Function.Name != name || l.Apps.Function.Name == "" {
			t.Errorf("%s has no description for other apps", name)
			continue
		}
		if arguments(t, apps) != arguments(t, l.Def) {
			t.Errorf("%s: the arguments differ between the two descriptions", name)
		}
		if len(apps.Function.Description) <= len(l.Def.Function.Description) {
			t.Errorf("%s: the description for other apps is not fuller than the short one", name)
		}
	}
}
