package aitools

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omni/internal/chat"
	"omni/internal/clipboard"
	"omni/internal/notes"
	"omni/internal/scripts"
	"omni/internal/search"
)

type fakeFiles struct {
	res    search.Results
	recent []search.Changed
	err    error
}

func (f fakeFiles) Lookup(context.Context, string, int, int) search.Results { return f.res }
func (f fakeFiles) Recent(context.Context, time.Time, int) ([]search.Changed, error) {
	return f.recent, f.err
}

type fakeClips struct {
	on      bool
	entries []clipboard.Entry
}

func (c fakeClips) Enabled() bool { return c.on }
func (c fakeClips) Search(string, int) []clipboard.Entry {
	return c.entries
}

type fakeScripts []scripts.Script

func (s fakeScripts) List() []scripts.Script { return s }

func tool(t *testing.T, d Deps, name string) chat.Local {
	t.Helper()
	for _, l := range d.Tools() {
		if l.Def.Function.Name == name {
			return l
		}
	}
	t.Fatalf("no tool %s", name)
	return chat.Local{}
}

func TestSearchFilesGivesWholeMatchesAndNamesTheFilesAsSources(t *testing.T) {
	d := Deps{Files: fakeFiles{res: search.Results{State: search.StateResults, Confident: true, Rows: []search.Row{
		{Name: "lease.pdf", Path: `C:\Docs\lease.pdf`, Matches: []search.Match{{Location: "page 3", Text: "Pets are allowed with a deposit.", Snippet: "Pets…"}}},
		{Name: "cat.png", Path: `C:\Pics\cat.png`, Matches: []search.Match{{Snippet: "a cat"}}},
	}}}}
	text, sources, err := tool(t, d, "search_files").Run(context.Background(), map[string]string{"query": "pets lease"})
	if err != nil || !strings.Contains(text, "[page 3] Pets are allowed with a deposit.") || !strings.Contains(text, "a cat") {
		t.Fatalf("text %q, %v", text, err)
	}
	if len(sources) != 2 || sources[0].Path != `C:\Docs\lease.pdf` {
		t.Fatalf("sources %+v", sources)
	}
}

func TestSearchFilesSaysNothingMatchedWhenTheIndexIsNotSure(t *testing.T) {
	d := Deps{Files: fakeFiles{res: search.Results{State: search.StateResults, Rows: []search.Row{{Name: "noise.txt"}}}}}
	text, sources, err := tool(t, d, "search_files").Run(context.Background(), map[string]string{"query": "x"})
	if err != nil || !strings.Contains(text, "Nothing in the person's files matched") || strings.Contains(text, "noise") || sources != nil {
		t.Fatalf("text %q, %v", text, err)
	}
	d.Files = fakeFiles{res: search.Results{State: search.StateOff, Message: "File search is off."}}
	if _, _, err := tool(t, d, "search_files").Run(context.Background(), map[string]string{"query": "x"}); err == nil {
		t.Fatal("a search that cannot run is an error the model is told about")
	}
}

func TestNotesCanBeSearchedReadAndAddedAndAddingAsksFirst(t *testing.T) {
	store, err := notes.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("", "# Groceries\nmilk, eggs"); err != nil {
		t.Fatal(err)
	}
	d := Deps{Notes: store}
	ctx := context.Background()
	found, _, _ := tool(t, d, "search_notes").Run(ctx, map[string]string{"query": "milk"})
	id := strings.TrimSuffix(strings.Fields(found)[2], ":")
	text, sources, err := tool(t, d, "read_note").Run(ctx, map[string]string{"id": id})
	if err != nil || !strings.Contains(text, "milk, eggs") || len(sources) != 1 || sources[0].Path != filepath.Join(store.Dir(), id+".md") {
		t.Fatalf("read %q %+v %v (search said %q)", text, sources, err, found)
	}
	if text, _, _ := tool(t, d, "read_note").Run(ctx, map[string]string{"id": `..\..\secret`}); !strings.Contains(text, "There is no note") {
		t.Fatalf("a bad id: %q", text)
	}
	add := tool(t, d, "add_note")
	if !add.Writes || tool(t, d, "search_notes").Writes {
		t.Fatal("only adding a note changes something")
	}
	if _, _, err := add.Run(ctx, map[string]string{"text": "# Call Sam\nabout the lease"}); err != nil {
		t.Fatal(err)
	}
	if all, _ := store.Search("", 0); len(all) != 2 {
		t.Fatalf("notes = %+v", all)
	}
}

func TestRecentActivityPutsFilesLaunchesAndCopiesTogether(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local)
	var asked []time.Time
	d := Deps{
		Now:   func() time.Time { return now },
		Files: fakeFiles{recent: []search.Changed{{Path: `C:\work\budget.xlsx`, Modified: now.Add(-time.Hour)}}},
		Launches: func(since time.Time) []Launch {
			asked = append(asked, since)
			return []Launch{{Name: "Visual Studio Code", At: now.Add(-2 * time.Hour)}}
		},
		Clips: fakeClips{on: true, entries: []clipboard.Entry{{Text: "new copy", Source: "Code", At: now.Add(-time.Minute)}, {Text: "old copy", At: now.Add(-72 * time.Hour)}}},
	}
	text, _, err := tool(t, d, "recent_activity").Run(context.Background(), map[string]string{"days": "2"})
	for _, want := range []string{"budget.xlsx", "Visual Studio Code", `from Code: "new copy"`} {
		if err != nil || !strings.Contains(text, want) {
			t.Fatalf("%q lacks %q (%v)", text, want, err)
		}
	}
	if !asked[0].Equal(now.Add(-48 * time.Hour)) {
		t.Fatalf("since = %v", asked[0])
	}
	if strings.Contains(text, "old copy") {
		t.Fatalf("a copy from before the period: %q", text)
	}
	d.Files = fakeFiles{err: errors.New("file search is not running")}
	if text, _, _ := tool(t, d, "recent_activity").Run(context.Background(), map[string]string{"days": "nonsense"}); !strings.Contains(text, "not available: file search is not running") {
		t.Fatalf("%q", text)
	}
}

func TestTheClipboardToolSaysWhenTheHistoryIsOff(t *testing.T) {
	text, _, _ := tool(t, Deps{Clips: fakeClips{}}, "search_clipboard").Run(context.Background(), map[string]string{})
	if !strings.Contains(text, "history is off") {
		t.Fatalf("%q", text)
	}
}

func TestOnlyScriptsWithAToolLineAreToolsAndTheyAlwaysCountAsChanging(t *testing.T) {
	var gotInput string
	d := Deps{
		Scripts: fakeScripts{
			{Name: "Kill Port!", Title: "Kill port", Argument: "a port number", Tool: "Ends what listens on a port."},
			{Name: "kill-port", Title: "Same name", Tool: "x"},
			{Name: "plain", Title: "Not a tool"},
		},
		RunScript: func(_ context.Context, s scripts.Script, input string) (string, error) {
			gotInput = input
			return "killed 1234", nil
		},
	}
	locals := d.Tools()
	var scriptTools []chat.Local
	for _, l := range locals {
		if strings.HasPrefix(l.Group, "script:") {
			scriptTools = append(scriptTools, l)
		}
	}
	if len(scriptTools) != 1 || scriptTools[0].Def.Function.Name != "script_kill_port" || !scriptTools[0].Writes || scriptTools[0].Group != "script:Kill Port!" {
		t.Fatalf("script tools %+v", scriptTools)
	}
	text, _, err := scriptTools[0].Run(context.Background(), map[string]string{"input": "3000"})
	if err != nil || gotInput != "3000" || text != "Output of Kill port:\nkilled 1234" {
		t.Fatalf("%q %v", text, err)
	}
}

func TestCutNeverSplitsACharacter(t *testing.T) {
	if got := cut("héllo", 2); got != "h…" {
		t.Fatalf("%q", got)
	}
}
