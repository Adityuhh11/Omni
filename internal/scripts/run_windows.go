package scripts

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// createNoWindow keeps a console window from flashing up for a script.
const createNoWindow = 0x08000000

// interpreters run each kind of script; an .exe runs by itself.
var interpreters = map[string][]string{
	".ps1": {"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File"},
	".py":  {"py", "-3"},
	".js":  {"node"},
	".sh":  {"bash"},
	".bat": {"cmd.exe", "/d", "/c"},
	".cmd": {"cmd.exe", "/d", "/c"},
}

func runnable(name string, _ fs.FileMode) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := interpreters[ext]
	return ok || ext == ".exe"
}

func command(ctx context.Context, path, input string) (*exec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var args []string
	if input != "" && ext != ".bat" && ext != ".cmd" {
		args = []string{input}
	}
	var cmd *exec.Cmd
	if ext == ".exe" {
		cmd = exec.CommandContext(ctx, path, args...)
	} else if run, ok := interpreters[ext]; ok {
		cmd = exec.CommandContext(ctx, run[0], append(append(run[1:len(run):len(run)], path), args...)...)
	} else {
		return nil, fmt.Errorf("%s is not a kind of script Omni can run", filepath.Base(path))
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd, nil
}
