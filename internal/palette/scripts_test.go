package palette

import (
	"context"
	"errors"
	"strings"
	"testing"

	"omni/internal/scripts"
)

type fakeScriptBox []scripts.Script

func (b fakeScriptBox) Dir() string            { return `C:\scripts` }
func (b fakeScriptBox) List() []scripts.Script { return b }
func (b fakeScriptBox) Find(name string) (scripts.Script, bool) {
	for _, s := range b {
		if s.Name == name {
			return s, true
		}
	}
	return scripts.Script{}, false
}

var killPort = scripts.Script{Name: "kill-port", Path: `C:\scripts\kill-port.ps1`, Title: "Kill port", Keyword: "port", Argument: "a port number", Output: "copy"}

type scriptRig struct {
	ran      []string
	copied   []string
	pasted   []string
	runErr   error
	dismissd *int
}

func scriptService(t *testing.T, list ...scripts.Script) (*Service, *scriptRig) {
	s, dismissed := newService(t)
	r := &scriptRig{dismissd: dismissed}
	With(s, Extras{
		Scripts: fakeScriptBox(list),
		RunScript: func(_ context.Context, sc scripts.Script, input string) (string, error) {
			r.ran = append(r.ran, sc.Name+"("+input+")")
			return "out of " + sc.Name, r.runErr
		},
		Copy:  func(text string) error { r.copied = append(r.copied, text); return nil },
		Paste: func(text string) error { r.pasted = append(r.pasted, text); return nil },
	})
	return s, r
}

func scriptRows(a Answer) []Item {
	var out []Item
	for _, it := range a.Items {
		if it.Kind == "script" {
			out = append(out, it)
		}
	}
	return out
}

func TestAKeywordAndTextRunTheScriptWithThatText(t *testing.T) {
	s, r := scriptService(t, killPort)
	rows := scriptRows(s.Query("port 3000", ModeAll))
	if len(rows) != 1 || rows[0].Title != "Kill port: 3000" || rows[0].Fill != "" {
		t.Fatalf("rows %+v", rows)
	}
	if err := s.Do(rows[0].ID, "open", "port 3000"); err != nil {
		t.Fatal(err)
	}
	if len(r.ran) != 1 || r.ran[0] != "kill-port(3000)" || len(r.copied) != 1 || r.copied[0] != "out of kill-port" || *r.dismissd != 1 {
		t.Fatalf("ran %v copied %v dismissed %d", r.ran, r.copied, *r.dismissd)
	}
}

func TestATitleMatchOfAScriptThatTakesTextFillsItsKeyword(t *testing.T) {
	s, r := scriptService(t, killPort)
	rows := scriptRows(s.Query("kill", ModeAll))
	if len(rows) != 1 || rows[0].Fill != "port " || !strings.Contains(rows[0].Subtitle, "a port number") {
		t.Fatalf("rows %+v", rows)
	}
	if err := s.Do(rows[0].ID, "open", "kill"); err == nil || len(r.ran) != 0 {
		t.Fatalf("ran without its text: %v %v", err, r.ran)
	}
}

func TestScriptOutputsAndFailures(t *testing.T) {
	silent := scripts.Script{Name: "hello", Title: "Hello", Keyword: "hello", Output: "silent"}
	paste := scripts.Script{Name: "stamp", Title: "Stamp", Keyword: "stamp", Output: "paste"}
	s, r := scriptService(t, silent, paste)
	if err := s.Do("script:hello"+scriptSep, "open", ""); err != nil || len(r.copied)+len(r.pasted) != 0 || *r.dismissd != 1 {
		t.Fatalf("silent: %v %v %v", err, r.copied, r.pasted)
	}
	if err := s.Do("script:stamp"+scriptSep, "open", ""); err != nil || len(r.pasted) != 1 || r.pasted[0] != "out of stamp" {
		t.Fatalf("paste: %v %v", err, r.pasted)
	}
	r.runErr = errors.New("Hello failed: nope")
	if err := s.Do("script:hello"+scriptSep, "open", ""); err == nil || *r.dismissd != 1 {
		t.Fatalf("a failure must stay in the bar: %v, dismissed %d", err, *r.dismissd)
	}
	if err := s.Do("script:gone"+scriptSep, "open", ""); err == nil {
		t.Fatal("a deleted script")
	}
	if rows := scriptRows(s.Query("hello there", ModeAll)); len(rows) != 0 {
		t.Fatalf("a script that takes no text matched its keyword with text after it: %+v", rows)
	}
}

func TestTheScriptsFolderRowOpensTheFolder(t *testing.T) {
	s, _ := scriptService(t)
	rows := scriptRows(s.Query("scripts", ModeAll))
	if len(rows) != 1 || rows[0].ID != scriptsFolderID {
		t.Fatalf("rows %+v", rows)
	}
	if err := s.Do(scriptsFolderID, "open", ""); err != nil || s.files.(*fakeFiles).opened[0] != `C:\scripts` {
		t.Fatalf("%v %+v", err, s.files)
	}
}
