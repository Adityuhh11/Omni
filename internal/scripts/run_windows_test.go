package scripts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, name, text string) Script {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	s, ok := parse(path, []byte(text))
	if !ok {
		t.Fatal("not a command")
	}
	return s
}

func TestAPowerShellScriptGetsItsInputAndItsOutputComesBack(t *testing.T) {
	s := script(t, "echo.ps1", "# @omni.title Echo\nWrite-Output \"arg=$($args[0]) env=$env:OMNI_INPUT\"\n")
	out, err := Run(context.Background(), s, `it's "quoted" & more`)
	if err != nil || out != `arg=it's "quoted" & more env=it's "quoted" & more` {
		t.Fatalf("out %q, %v", out, err)
	}
}

func TestABatchFileGetsItsInputOnlyThroughTheEnvironment(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "made")
	s := script(t, "echo.bat", "@echo off\r\nREM @omni.title Echo\r\necho [%1]\r\n")
	out, err := Run(context.Background(), s, "x & echo hi > "+marker)
	if err != nil || out != "[]" {
		t.Fatalf("out %q, %v", out, err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the input ran as a command")
	}
}

func TestAFailingScriptSaysWhatItPrintedLast(t *testing.T) {
	s := script(t, "fail.ps1", "# @omni.title Fail\n[Console]::Error.WriteLine('port 3000 is free')\nexit 3\n")
	if _, err := Run(context.Background(), s, ""); err == nil || !strings.Contains(err.Error(), "Fail failed: port 3000 is free") {
		t.Fatalf("err = %v", err)
	}
}

func TestAScriptThatLeavesSomethingRunningStillReturns(t *testing.T) {
	s := script(t, "start.bat", "@echo off\r\nREM @omni.title Start\r\nstart /b ping -n 30 127.0.0.1\r\necho started\r\n")
	began := time.Now()
	out, err := Run(context.Background(), s, "")
	if took := time.Since(began); took > 10*time.Second {
		t.Fatalf("Run waited %v for what the script left running", took)
	}
	if err != nil || !strings.Contains(out, "started") {
		t.Fatalf("out %q, %v", out, err)
	}
}
