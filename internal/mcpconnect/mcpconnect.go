// Package mcpconnect adds Omni to, and removes it from, the MCP settings of the AI apps on this computer (Claude Desktop, Cursor,
// VS Code, Windsurf), so the person presses a button instead of finding and editing each app's file
// (docs/specs/2026-11-02-mcp-connect.md). It only ever touches the "omni" entry; every other server and key stays.
package mcpconnect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"omni/internal/atomicfile"
)

// entryName is the name Omni goes by in the other app's list of servers.
const entryName = "omni"

// Target is one AI app: where its MCP settings are and what to call the list of servers in them.
type Target struct {
	ID, Name string
	// Found is a path that exists when the app is installed.
	Found string
	// File is the settings file, which need not exist yet.
	File string
	// Key is the name of the list of servers in File.
	Key string
}

// App is a Target as the settings page shows it.
type App struct {
	ID, Name, File string
	// State is "off" (not connected), "on", "other" (connected to an Omni at another path) or "problem" (see Problem).
	State   string
	Problem string
}

// Apps returns the AI apps found on this computer and whether each one reaches the Omni at self.
func Apps(self string) []App { return apps(list(), self) }

// Connect adds the Omni at self to the app with this id.
func Connect(id, self string) error {
	return withTarget(id, func(t Target) error { return connect(t, self) })
}

// Disconnect removes Omni from the app with this id.
func Disconnect(id string) error { return withTarget(id, disconnect) }

func withTarget(id string, do func(Target) error) error {
	for _, t := range list() {
		if t.ID == id {
			return do(t)
		}
	}
	return fmt.Errorf("no AI app called %q", id)
}

func apps(targets []Target, self string) []App {
	var out []App
	for _, t := range targets {
		if _, err := os.Stat(t.Found); err != nil {
			continue
		}
		a := App{ID: t.ID, Name: t.Name, File: t.File, State: "off"}
		servers, _, err := read(t)
		switch entry, ok := servers[entryName]; {
		case err != nil:
			a.State, a.Problem = "problem", err.Error()
		case !ok:
		case commandOf(entry) == self:
			a.State = "on"
		default:
			a.State = "other"
		}
		out = append(out, a)
	}
	return out
}

func connect(t Target, self string) error {
	servers, top, err := read(t)
	if err != nil {
		return err
	}
	e := map[string]any{"command": self, "args": []string{"mcp"}}
	if t.Key == "servers" { // VS Code asks for the kind of server
		e["type"] = "stdio"
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode the entry: %w", err)
	}
	servers[entryName] = raw
	return write(t, top, servers)
}

func disconnect(t Target) error {
	servers, top, err := read(t)
	if err != nil {
		return err
	}
	if _, ok := servers[entryName]; !ok {
		return nil
	}
	delete(servers, entryName)
	return write(t, top, servers)
}

// read returns the app's servers and the whole file, both as raw JSON so that nothing Omni does not touch is changed. A file that
// is missing or empty is an empty object; one that is not JSON (VS Code allows comments) is refused rather than rewritten.
func read(t Target) (servers, top map[string]json.RawMessage, err error) {
	top, servers = map[string]json.RawMessage{}, map[string]json.RawMessage{}
	b, err := os.ReadFile(t.File)
	switch {
	case errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(b)) == 0):
		return servers, top, nil
	case err != nil:
		return nil, nil, fmt.Errorf("read %s: %w", t.File, err)
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, nil, fmt.Errorf("%s is not plain JSON (comments?), so Omni will not rewrite it; add the line by hand: %w", t.File, err)
	}
	if raw, ok := top[t.Key]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, nil, fmt.Errorf("%q in %s is not a list of servers: %w", t.Key, t.File, err)
		}
	}
	return servers, top, nil
}

// write saves the file with the new servers. The first time, the original is kept beside it as <file>.omni-backup.
func write(t Target, top, servers map[string]json.RawMessage) error {
	list, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("encode servers: %w", err)
	}
	top[t.Key] = list
	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", t.File, err)
	}
	if err := os.MkdirAll(filepath.Dir(t.File), 0o755); err != nil {
		return fmt.Errorf("create the folder of %s: %w", t.File, err)
	}
	if err := backup(t.File); err != nil {
		return err
	}
	if err := atomicfile.Write(t.File, out); err != nil {
		return fmt.Errorf("write %s: %w", t.File, err)
	}
	return nil
}

func backup(file string) error {
	b, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s for its backup: %w", file, err)
	}
	dst := file + ".omni-backup"
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if err := os.WriteFile(dst, b, 0o600); err != nil {
		return fmt.Errorf("back up %s: %w", file, err)
	}
	return nil
}

func commandOf(entry json.RawMessage) string {
	var e struct{ Command string }
	_ = json.Unmarshal(entry, &e) // an entry that is not an object has no command, so it reads as "other"
	return e.Command
}
