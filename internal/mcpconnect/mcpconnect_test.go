package mcpconnect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const self = `C:\Omni\omni.exe`

func target(t *testing.T, content string) Target {
	t.Helper()
	dir := t.TempDir()
	tg := Target{ID: "app", Name: "App", Found: dir, File: filepath.Join(dir, "sub", "mcp.json"), Key: "mcpServers"}
	if content != "" {
		if err := os.MkdirAll(filepath.Dir(tg.File), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tg.File, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return tg
}

func stateOf(t *testing.T, tg Target) App {
	t.Helper()
	got := apps([]Target{tg}, self)
	if len(got) != 1 {
		t.Fatalf("apps = %+v", got)
	}
	return got[0]
}

func TestConnectKeepsEveryOtherServerAndKeyAndBacksUpTheOriginalOnce(t *testing.T) {
	orig := `{"theme":"dark","count":12345678901234567890,"mcpServers":{"other":{"command":"x","args":["a"]}}}`
	tg := target(t, orig)
	if err := connect(tg, self); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	b, _ := os.ReadFile(tg.File)
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if string(got["theme"]) != `"dark"` || string(got["count"]) != "12345678901234567890" {
		t.Fatalf("other keys changed: %s", b)
	}
	var servers map[string]struct {
		Command string
		Args    []string
		Type    string
	}
	if err := json.Unmarshal(got["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	if servers["other"].Command != "x" || servers["omni"].Command != self || servers["omni"].Args[0] != "mcp" || servers["omni"].Type != "" {
		t.Fatalf("servers = %+v", servers)
	}
	if s := stateOf(t, tg); s.State != "on" {
		t.Fatalf("state = %+v", s)
	}
	if err := connect(tg, `D:\elsewhere\omni.exe`); err != nil { // a second write must not replace the first backup
		t.Fatal(err)
	}
	if bk, _ := os.ReadFile(tg.File + ".omni-backup"); string(bk) != orig {
		t.Fatalf("backup = %q", bk)
	}
}

func TestConnectCreatesAMissingFileAndFolder(t *testing.T) {
	tg := target(t, "")
	if err := connect(tg, self); err != nil {
		t.Fatal(err)
	}
	if s := stateOf(t, tg); s.State != "on" {
		t.Fatalf("state = %+v", s)
	}
	if _, err := os.Stat(tg.File + ".omni-backup"); err == nil {
		t.Fatal("a backup of nothing was made")
	}
}

func TestVSCodeGetsTheStdioTypeUnderItsOwnKey(t *testing.T) {
	tg := target(t, "")
	tg.Key = "servers"
	if err := connect(tg, self); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(tg.File); !strings.Contains(string(b), `"type": "stdio"`) || !strings.Contains(string(b), `"servers"`) {
		t.Fatalf("file = %s", b)
	}
}

func TestAFileThatIsNotPlainJSONIsRefusedAndLeftAlone(t *testing.T) {
	orig := "{\n // my servers\n \"servers\": {}\n}"
	tg := target(t, orig)
	if err := connect(tg, self); err == nil || !strings.Contains(err.Error(), "not plain JSON") {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(tg.File); string(b) != orig {
		t.Fatalf("file was changed: %s", b)
	}
	if s := stateOf(t, tg); s.State != "problem" || s.Problem == "" {
		t.Fatalf("state = %+v", s)
	}
}

func TestAnOmniAtAnotherPathReadsAsOtherAndConnectUpdatesIt(t *testing.T) {
	tg := target(t, `{"mcpServers":{"omni":{"command":"D:\\old\\omni.exe","args":["mcp"]}}}`)
	if s := stateOf(t, tg); s.State != "other" {
		t.Fatalf("state = %+v", s)
	}
	if err := connect(tg, self); err != nil {
		t.Fatal(err)
	}
	if s := stateOf(t, tg); s.State != "on" {
		t.Fatalf("state = %+v", s)
	}
}

func TestDisconnectRemovesOnlyOmni(t *testing.T) {
	tg := target(t, `{"mcpServers":{"other":{"command":"x"},"omni":{"command":"y"}}}`)
	if err := disconnect(tg); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(tg.File)
	if !strings.Contains(string(b), `"other"`) || strings.Contains(string(b), `"omni"`) {
		t.Fatalf("file = %s", b)
	}
	if err := disconnect(target(t, "")); err != nil { // nothing to remove is fine
		t.Fatal(err)
	}
}

func TestAnAppThatIsNotInstalledIsNotListed(t *testing.T) {
	tg := target(t, "")
	tg.Found = filepath.Join(tg.Found, "missing")
	if got := apps([]Target{tg}, self); len(got) != 0 {
		t.Fatalf("apps = %+v", got)
	}
}

func TestTheTableNamesEachAppsFileUnderItsFolder(t *testing.T) {
	home, support := filepath.Join("h"), filepath.Join("h", "support")
	for _, tg := range targets(home, support) {
		if !strings.HasPrefix(tg.File, tg.Found) || tg.Key == "" || tg.ID == "" {
			t.Errorf("target = %+v", tg)
		}
	}
}
