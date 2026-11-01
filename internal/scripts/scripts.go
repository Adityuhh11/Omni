// Package scripts finds the person's script commands in a folder and runs them. A file with an `@omni.title` line near its
// top is a command in the bar; with an `@omni.tool` line it is also a tool Omni AI may use (docs/SCRIPTS.md).
package scripts

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// headerLines is how far down a file its @omni lines are looked for.
	headerLines = 30
	// RunTimeout is the longest a script may run; one that must run longer can start something and return.
	RunTimeout = 60 * time.Second
	// maxOutput is the most of a script's output that is kept.
	maxOutput = 64 << 10
)

// Outputs are what may be done with what a script prints: nothing, put on the clipboard, or typed into the program the bar
// came from.
var Outputs = []string{"silent", "copy", "paste"}

// Script is one command.
type Script struct {
	Path string
	// Name is the file's name without its extension: it names the script in [chat] tools ("script:<name>") and in the bar.
	Name  string
	Title string
	// Keyword, typed with a space and text after it, runs the script with that text (`port 3000`); it is the name in lower
	// case when the file sets none.
	Keyword string
	// Argument is what the script takes, as the bar and the model are told ("a port number"); empty when it takes nothing.
	Argument string
	Output   string
	// Tool describes the script to Omni AI; empty means the model is not offered it.
	Tool string
}

var header = regexp.MustCompile(`@omni\.([a-z]+)[ \t]*(.*)$`)

// parse reads the @omni lines at the top of a file. ok is false when the file has no title: it is not a command (a helper the
// commands use, a README).
func parse(path string, top []byte) (Script, bool) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	s := Script{Path: path, Name: name, Output: "silent"}
	lines := bufio.NewScanner(bytes.NewReader(top))
	for n := 0; n < headerLines && lines.Scan(); n++ {
		m := header.FindStringSubmatch(lines.Text())
		if m == nil {
			continue
		}
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m[2]), "-->"))
		switch m[1] {
		case "title":
			s.Title = value
		case "keyword":
			s.Keyword = strings.ToLower(strings.Join(strings.Fields(value), ""))
		case "argument":
			s.Argument = value
		case "output":
			if slices.Contains(Outputs, value) {
				s.Output = value
			}
		case "tool":
			s.Tool = value
		}
	}
	if s.Keyword == "" {
		s.Keyword = strings.ToLower(strings.Join(strings.Fields(name), ""))
	}
	return s, s.Title != ""
}

// Folder is the scripts folder. Headers are read again only for files that changed since they were last read.
type Folder struct {
	dir string

	mu    sync.Mutex
	known map[string]cached
}

type cached struct {
	mod    time.Time
	size   int64
	script Script
	ok     bool
}

// Open returns the folder dir, made if it is missing so it can be opened and filled.
func Open(dir string) (*Folder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("make the scripts folder: %w", err)
	}
	return &Folder{dir: dir, known: map[string]cached{}}, nil
}

// Dir is the folder.
func (f *Folder) Dir() string { return f.dir }

// List returns the commands in the folder, by title.
func (f *Folder) List() []Script {
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]bool{}
	var out []Script
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || !runnable(e.Name(), info.Mode()) {
			continue
		}
		path := filepath.Join(f.dir, e.Name())
		seen[path] = true
		c, ok := f.known[path]
		if !ok || !c.mod.Equal(info.ModTime()) || c.size != info.Size() {
			c = cached{mod: info.ModTime(), size: info.Size()}
			if top, err := readTop(path); err == nil {
				c.script, c.ok = parse(path, top)
			}
			f.known[path] = c
		}
		if c.ok {
			out = append(out, c.script)
		}
	}
	for path := range f.known {
		if !seen[path] {
			delete(f.known, path)
		}
	}
	slices.SortFunc(out, func(a, b Script) int { return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)) })
	return out
}

// Find returns the command with this name.
func (f *Folder) Find(name string) (Script, bool) {
	for _, s := range f.List() {
		if s.Name == name {
			return s, true
		}
	}
	return Script{}, false
}

func readTop(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	top := make([]byte, 4096)
	n, err := file.Read(top)
	if n == 0 && err != nil {
		return nil, err
	}
	return top[:n], nil
}

// Run runs the script with input (empty for none) and returns what it printed. The input is the first argument and also
// OMNI_INPUT in the environment (a batch file gets only the environment: cmd.exe would read its quotes as commands). It runs
// in the home folder, for at most RunTimeout.
func Run(ctx context.Context, s Script, input string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, RunTimeout)
	defer cancel()
	cmd, err := command(ctx, s.Path, input)
	if err != nil {
		return "", err
	}
	cmd.Dir, _ = os.UserHomeDir()
	cmd.Env = append(os.Environ(), "OMNI_INPUT="+input)
	var stdout, stderr limited
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	out := strings.TrimRight(stdout.String(), "\r\n")
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s ran for longer than %s and was stopped", s.Title, RunTimeout)
	}
	if err != nil {
		var exit *exec.ExitError
		if msg := lastLine(stderr.String()); errors.As(err, &exit) && msg != "" {
			return out, fmt.Errorf("%s failed: %s", s.Title, msg)
		}
		return out, fmt.Errorf("%s failed: %w", s.Title, err)
	}
	return out, nil
}

// lastLine is the last line of a script's error output with text on it: usually the one that says what went wrong.
func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	line := strings.TrimSpace(lines[len(lines)-1])
	if len(line) > 300 {
		line = line[:300] + "…"
	}
	return line
}

// limited keeps the first maxOutput bytes written to it and drops the rest.
type limited struct{ bytes.Buffer }

func (l *limited) Write(p []byte) (int, error) {
	if room := maxOutput - l.Len(); room > 0 {
		l.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
