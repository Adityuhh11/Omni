package scripts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAShellScriptGetsItsInputAndItsOutputComesBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "echo.sh")
	text := "# @omni.title Echo\necho \"arg=$1 env=$OMNI_INPUT\"\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := parse(path, []byte(text))
	out, err := Run(context.Background(), s, `a "b" ; c`)
	if err != nil || out != `arg=a "b" ; c env=a "b" ; c` {
		t.Fatalf("out %q, %v", out, err)
	}
}

func TestAFailingScriptSaysWhatItPrintedLast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fail.sh")
	text := "# @omni.title Fail\necho 'nope' >&2\nexit 3\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := parse(path, []byte(text))
	if _, err := Run(context.Background(), s, ""); err == nil || !strings.Contains(err.Error(), "Fail failed: nope") {
		t.Fatalf("err = %v", err)
	}
}
