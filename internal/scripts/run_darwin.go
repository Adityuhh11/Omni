package scripts

import (
	"context"
	"fmt"
	"io/fs"
	"os"
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

// homebrew are where Homebrew puts interpreters: an app started from the Finder or at login has only the system's folders on
// its PATH, so node, ruby or a newer python3 would not be found.
var homebrew = []string{"/opt/homebrew/bin", "/usr/local/bin"}

// environ is the environment a script runs in, with Homebrew's folders on the PATH (a script's #!/usr/bin/env node needs it).
func environ() []string {
	env := os.Environ()
	for i, kv := range env {
		if path, ok := strings.CutPrefix(kv, "PATH="); ok {
			env[i] = "PATH=" + path + ":" + strings.Join(homebrew, ":")
			return env
		}
	}
	return append(env, "PATH=/usr/bin:/bin:/usr/sbin:/sbin:"+strings.Join(homebrew, ":"))
}

// lookPath finds an interpreter on the PATH or in Homebrew's folders.
func lookPath(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, dir := range homebrew {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return filepath.Join(dir, name)
		}
	}
	return name // not found: running it says so
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
	return exec.CommandContext(ctx, lookPath(run[0]), append(append(run[1:len(run):len(run)], path), args...)...), nil
}
