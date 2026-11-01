package scripts

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// interpreters run each kind of script; a file with no extension that can be executed runs by itself (its #! line says how).
var interpreters = map[string][]string{
	".sh":          {"/bin/bash"},
	".zsh":         {"/bin/zsh"},
	".py":          {"python3"},
	".js":          {"node"},
	".rb":          {"ruby"},
	".applescript": {"osascript"},
}

func runnable(name string, mode fs.FileMode) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := interpreters[ext]
	return ok || (ext == "" && mode&0o111 != 0)
}

func command(ctx context.Context, path, input string) (*exec.Cmd, error) {
	var args []string
	if input != "" {
		args = []string{input}
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return exec.CommandContext(ctx, path, args...), nil
	}
	run, ok := interpreters[ext]
	if !ok {
		return nil, fmt.Errorf("%s is not a kind of script Omni can run", filepath.Base(path))
	}
	return exec.CommandContext(ctx, run[0], append(append(run[1:len(run):len(run)], path), args...)...), nil
}
