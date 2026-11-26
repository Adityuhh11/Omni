package settingsui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omni/internal/scripts"
)

func TestAddScriptPassesWhatTheDialogCollectedAndReportsWhyItWasRefused(t *testing.T) {
	r := newOmniRig(t)
	var got scripts.New
	r.page.o.AddScript = func(n scripts.New) (string, error) {
		got = n
		if strings.TrimSpace(n.Title) == "" {
			return "", os.ErrInvalid
		}
		return "x", nil
	}
	if msg := r.page.AddScript("Kill port", ".ps1", "copy", "echo 1"); msg != "" || got != (scripts.New{Title: "Kill port", Ext: ".ps1", Output: "copy", Body: "echo 1"}) {
		t.Fatalf("message %q, got %+v", msg, got)
	}
	if msg := r.page.AddScript("", ".ps1", "", "echo 1"); msg == "" {
		t.Fatal("a refusal came back as success")
	}
}

func TestTheViewOffersTheLanguagesThisSystemRuns(t *testing.T) {
	if got := newOmniRig(t).page.View().ScriptLanguages; len(got) == 0 || got[0].Ext != scripts.Languages[0].Ext {
		t.Fatalf("languages = %+v", got)
	}
}

func TestLoadScriptReadsAFileAndGuessesItsNameAndLanguage(t *testing.T) {
	r := newOmniRig(t)
	ext := scripts.Languages[0].Ext
	dir := t.TempDir()
	file := filepath.Join(dir, "Clean temp"+ext)
	os.WriteFile(file, []byte("echo hi\n"), 0o644)
	big := filepath.Join(dir, "big.txt")
	os.WriteFile(big, make([]byte, 300<<10), 0o644)

	var pick string
	r.page.o.PickScript = func() (string, error) { return pick, nil }

	pick = file
	if got := r.page.LoadScript(); got != (ScriptFile{Name: "Clean temp", Ext: ext, Text: "echo hi\n"}) {
		t.Fatalf("got %+v", got)
	}
	pick = "" // the dialog was cancelled
	if got := r.page.LoadScript(); got != (ScriptFile{}) {
		t.Fatalf("a cancelled dialog gave %+v", got)
	}
	pick = big
	if got := r.page.LoadScript(); got.Error == "" || got.Text != "" {
		t.Fatalf("a file too big to be a script gave %+v", got)
	}
}
