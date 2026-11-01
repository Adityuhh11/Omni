package scripts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseReadsTheOmniLines(t *testing.T) {
	for _, tc := range []struct {
		name, top string
		want      Script
		ok        bool
	}{
		{"all", "# @omni.title Kill port\n# @omni.keyword  kp\n# @omni.argument a port number\n# @omni.output copy\n# @omni.tool Ends what listens on a port.\n",
			Script{Name: "kill-port", Title: "Kill port", Keyword: "kp", Argument: "a port number", Output: "copy", Tool: "Ends what listens on a port."}, true},
		{"defaults", "REM @omni.title Hello\r\n", Script{Name: "kill-port", Title: "Hello", Keyword: "kill-port", Output: "silent"}, true},
		{"unknown output stays silent", "// @omni.title X\n// @omni.output shout\n", Script{Name: "kill-port", Title: "X", Keyword: "kill-port", Output: "silent"}, true},
		{"no title is not a command", "# helper\n# @omni.tool something\n", Script{}, false},
		{"too far down", string(make([]byte, 0)) + repeat("#\n", headerLines) + "# @omni.title Late\n", Script{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parse(filepath.Join("dir", "kill-port.ps1"), []byte(tc.top))
			if ok != tc.ok {
				t.Fatalf("ok = %v", ok)
			}
			if !ok {
				return
			}
			tc.want.Path = filepath.Join("dir", "kill-port.ps1")
			if got != tc.want {
				t.Fatalf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}

func TestAFolderListsCommandsAndSeesChanges(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "scripts")
	f, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string, mod time.Time) {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	write("b.py", "# @omni.title Beta\n", at)
	write("a.py", "# @omni.title alpha\n", at)
	write("notes.txt", "@omni.title not runnable\n", at)
	write("helper.py", "# no title\n", at)
	if got := f.List(); len(got) != 2 || got[0].Title != "alpha" || got[1].Title != "Beta" {
		t.Fatalf("list = %+v", got)
	}
	write("a.py", "# @omni.title Gamma!\n", at.Add(time.Second))
	if s, ok := f.Find("a"); !ok || s.Title != "Gamma!" {
		t.Fatalf("an edited script: %+v", s)
	}
	if err := os.Remove(filepath.Join(dir, "b.py")); err != nil {
		t.Fatal(err)
	}
	if got := f.List(); len(got) != 1 {
		t.Fatalf("after a removal: %+v", got)
	}
}

func TestOutputBeyondTheLimitIsDropped(t *testing.T) {
	var l limited
	big := make([]byte, maxOutput+10)
	if n, err := l.Write(big); n != len(big) || err != nil || l.Len() != maxOutput {
		t.Fatalf("wrote %d %v, kept %d", n, err, l.Len())
	}
}
