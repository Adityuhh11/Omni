package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omni/internal/chat"
)

func tools() []chat.Local {
	return []chat.Local{
		{Def: chat.ObjectTool("search_notes", "Find notes.", map[string]string{"query": "words"}),
			Apps: chat.ObjectTool("search_notes", "Find notes, in full.", map[string]string{"query": "the words, in full"}),
			Run: func(_ context.Context, args map[string]string) (string, []chat.Source, error) {
				return "found " + args["query"], []chat.Source{{Title: "n", Path: `C:\notes\n.md`}}, nil
			}},
		{Def: chat.ObjectTool("add_note", "Add a note.", map[string]string{"text": "it"}), Writes: true,
			Run: func(context.Context, map[string]string) (string, []chat.Source, error) {
				panic("an app outside Omni ran a tool that writes")
			}},
	}
}

func start(t *testing.T) *Server {
	t.Helper()
	s := &Server{Tools: tools, Version: "1.3.0", File: filepath.Join(t.TempDir(), FileName)}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s
}

// talk relays lines through Relay and returns the answers, one per request.
func talk(t *testing.T, file string, lines ...string) []map[string]any {
	t.Helper()
	var out strings.Builder
	if err := Relay(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, file); err != nil {
		t.Fatal(err)
	}
	var answers []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("%q: %v", l, err)
		}
		answers = append(answers, m)
	}
	return answers
}

func TestAnAppCanListAndCallTheToolsThatOnlyRead(t *testing.T) {
	s := start(t)
	got := talk(t, s.File,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_notes","arguments":{"query":"milk"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_note","arguments":{"text":"x"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"nonsense"}`,
	)
	if len(got) != 5 {
		t.Fatalf("answers %v", got)
	}
	if v := got[0]["result"].(map[string]any)["protocolVersion"]; v != "2025-03-26" {
		t.Fatalf("protocol %v", v)
	}
	hello := got[0]["result"].(map[string]any)
	if hello["serverInfo"].(map[string]any)["title"] != "Omni" || !strings.Contains(hello["instructions"].(string), "launcher") {
		t.Fatalf("initialize %v", hello)
	}
	list := got[1]["result"].(map[string]any)["tools"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["name"] != "search_notes" {
		t.Fatalf("tools %v", list)
	}
	first := list[0].(map[string]any)
	if first["description"] != "Find notes, in full." || first["annotations"].(map[string]any)["readOnlyHint"] != true {
		t.Fatalf("the app should see the full description and that the tool only reads: %v", first)
	}
	text := got[2]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "found milk") || !strings.Contains(text, `C:\notes\n.md`) {
		t.Fatalf("call %q", text)
	}
	if got[3]["error"] == nil || got[4]["error"].(map[string]any)["code"].(float64) != -32601 {
		t.Fatalf("errors %v %v", got[3], got[4])
	}
}

func TestWithoutTheTokenOrFromABrowserNothingAnswers(t *testing.T) {
	s := start(t)
	var addr Address
	data, _ := os.ReadFile(s.File)
	if err := json.Unmarshal(data, &addr); err != nil {
		t.Fatal(err)
	}
	for name, header := range map[string]http.Header{
		"no token":   {},
		"wrong":      {"Authorization": {"Bearer nope"}},
		"web origin": {"Authorization": {"Bearer " + addr.Token}, "Origin": {"https://evil.example"}},
	} {
		req, _ := http.NewRequest(http.MethodPost, addr.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		req.Header = header
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s: %d", name, resp.StatusCode)
		}
	}
}

func TestWhenOmniIsNotRunningTheAppIsToldWhy(t *testing.T) {
	s := start(t)
	s.Stop()
	if _, err := os.Stat(s.File); !os.IsNotExist(err) {
		t.Fatal("the address outlived the server")
	}
	got := talk(t, s.File, `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if len(got) != 1 || got[0]["id"].(float64) != 7 || !strings.Contains(got[0]["error"].(map[string]any)["message"].(string), "not running") {
		t.Fatalf("answers %v", got)
	}
}
