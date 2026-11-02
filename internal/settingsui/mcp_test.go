package settingsui

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"omni/internal/chat"
	"omni/internal/settings"
)

func mcpPage(t *testing.T) (*MCP, *settings.Store) {
	t.Helper()
	store, err := settings.Open(filepath.Join(t.TempDir(), "Omni", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(context.Context, map[string]string) (string, []chat.Source, error) { return "", nil, nil }
	tools := func() []chat.Local {
		return []chat.Local{
			{Group: "notes", GroupTitle: "Notes", Def: chat.ObjectTool("search_notes", "short", nil), Apps: chat.ObjectTool("search_notes", "the full account", nil), Run: run},
			{Group: "notes", GroupTitle: "Notes", Def: chat.ObjectTool("add_note", "Add.", nil), Writes: true, Run: run},
			{Group: "script:x.ps1", GroupTitle: "X", Def: chat.ObjectTool("x", "Runs X.", nil), Writes: true, Run: run},
		}
	}
	return NewMCP(MCPOptions{Store: store, Tools: tools, Self: "omni.exe"}), store
}

func TestTheAIAppsPageListsOnlyWhatReadsWithTheFullDescriptionAndNothingIsOnAtFirst(t *testing.T) {
	page, _ := mcpPage(t)
	v := page.View()
	if v.Enabled || len(v.Groups) != 1 || v.Groups[0].On {
		t.Fatalf("view = %+v", v)
	}
	g := v.Groups[0]
	if g.Group != "notes" || len(g.Tools) != 1 || g.Tools[0].Name != "search_notes" || g.Tools[0].Description != "the full account" {
		t.Fatalf("group = %+v", g)
	}
	if v.Config == "" {
		t.Fatal("no config to copy")
	}
}

func TestTheSwitchesForOtherAppsAreApartFromOmniAIs(t *testing.T) {
	page, store := mcpPage(t)
	for _, step := range []func() string{
		func() string { return page.SetEnabled(true) },
		func() string { return page.SetTool("notes", true) },
		func() string { return page.SetTool("notes", true) }, // twice stays once
	} {
		if r := step(); r != "" {
			t.Fatal(r)
		}
	}
	cfg := store.Get().Chat
	if !cfg.MCP || !slices.Equal(cfg.MCPTools, []string{"notes"}) || len(cfg.Tools) != 0 {
		t.Fatalf("config = %+v", cfg)
	}
	if !page.View().Groups[0].On {
		t.Fatal("the page does not show the group on")
	}
	if r := page.SetTool("notes", false); r != "" || len(store.Get().Chat.MCPTools) != 0 {
		t.Fatalf("off: %q %+v", r, store.Get().Chat.MCPTools)
	}
}
