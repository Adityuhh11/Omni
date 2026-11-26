package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFolder(t *testing.T) *Folder {
	t.Helper()
	f, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAddWritesAScriptTheFolderThenListsUnderItsCustomName(t *testing.T) {
	f := newFolder(t)
	ext := Languages[0].Ext
	path, err := f.Add(New{Title: "Kill port!", Ext: ext, Output: "copy", Body: "echo hi\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "kill-port" + ext; filepath.Base(path) != want {
		t.Fatalf("file %q, want %q", filepath.Base(path), want)
	}
	got, ok := f.Find(filepath.Base(path))
	if !ok || got.Title != "Kill port!" || got.Output != "copy" || got.Keyword != "kill-port" {
		t.Fatalf("listed %+v (found %v)", got, ok)
	}
	if text, _ := os.ReadFile(path); !strings.Contains(string(text), "echo hi") {
		t.Fatalf("the code was lost: %q", text)
	}
}

func TestAddRefusesWhatCannotWork(t *testing.T) {
	f := newFolder(t)
	ext := Languages[0].Ext
	for name, n := range map[string]New{
		"no name":             {Title: "  ", Ext: ext, Body: "x"},
		"a name on two lines": {Title: "a\nb", Ext: ext, Body: "x"},
		"unknown language":    {Title: "A", Ext: ".exe", Body: "x"},
		"no code":             {Title: "A", Ext: ext, Body: " \n"},
		"unknown output":      {Title: "A", Ext: ext, Output: "shout", Body: "x"},
	} {
		if _, err := f.Add(n); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if entries, _ := os.ReadDir(f.Dir()); len(entries) != 0 {
		t.Fatalf("a refused script left files behind: %v", entries)
	}
}

func TestAddNeverOverwritesAndRefusesASecondScriptWithTheSameName(t *testing.T) {
	f := newFolder(t)
	ext := Languages[0].Ext
	first, err := f.Add(New{Title: "Backup", Ext: ext, Body: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Add(New{Title: "backup", Ext: ext, Body: "two"}); err == nil {
		t.Fatal("two scripts called Backup")
	}
	// A different name that makes the same file name gets a number, and the first file is untouched.
	second, err := f.Add(New{Title: "Backup?", Ext: ext, Body: "three"})
	if err != nil {
		t.Fatal(err)
	}
	if second == first || !strings.Contains(filepath.Base(second), "backup-2") {
		t.Fatalf("second script went to %q", second)
	}
	if text, _ := os.ReadFile(first); !strings.Contains(string(text), "one") {
		t.Fatalf("the first script was changed: %q", text)
	}
}

func TestAddKeepsAShebangFirstAndReplacesATitleTheCodeBrought(t *testing.T) {
	f := newFolder(t)
	ext := Languages[0].Ext
	path, err := f.Add(New{Title: "Mine", Ext: ext, Body: "#!/usr/bin/env bash\n# @omni.title Theirs\n# @omni.keyword go\necho hi\n"})
	if err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(text), "#!/usr/bin/env bash") {
		t.Fatalf("the #! line is not first: %q", text)
	}
	if strings.Contains(string(text), "Theirs") {
		t.Fatalf("the old title stayed: %q", text)
	}
	got, ok := f.Find(filepath.Base(path))
	if !ok || got.Title != "Mine" || got.Keyword != "go" {
		t.Fatalf("listed %+v (found %v)", got, ok)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Kill port":       "kill-port",
		"  Wi-Fi: status": "wi-fi-status",
		"日本語":             "script",
		"a  b__c":         "a-b-c",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}
