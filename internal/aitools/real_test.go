package aitools

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"omni/internal/browsers"
	"omni/internal/chat"
	"omni/internal/config"
	"omni/internal/llm"
	"omni/internal/notes"
	"omni/internal/scripts"
	"omni/internal/search"
)

// TestRealModelChoosesLocalTools runs the installed Omni AI model with the real tools: does it call the right one, and only
// when it should? Each question is asked with only its own group on, then with every group on (the hard case for a 1.7B model).
// It runs only when OMNI_CHAT_DIR is set (the person who changes this package, with Omni AI installed).
func TestRealModelChoosesLocalTools(t *testing.T) {
	if os.Getenv("OMNI_CHAT_DIR") == "" {
		t.Skip("OMNI_CHAT_DIR is not set")
	}
	layout, err := llm.Locate("")
	if err != nil || !layout.Installed() {
		t.Fatalf("no install: %v", err)
	}
	store, err := notes.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("", "# Groceries\nmilk, eggs, sourdough, coffee beans"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wifi.ps1"), []byte("# @omni.title Wi-Fi status\n# @omni.tool Shows the Wi-Fi network this computer is on and its signal strength.\nWrite-Output 'SSID: HomeNet  Signal: 87%'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder, err := scripts.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	lease := search.Results{State: search.StateResults, Confident: true, Rows: []search.Row{{Name: "lease-2026.pdf", Path: `C:\Docs\lease-2026.pdf`,
		Matches: []search.Match{{Location: "page 4", Text: "12. Pets. The tenant may keep one cat or one dog under 15 kg with the landlord's written consent and a pet deposit of 300 euros."}}}}}
	pages := []browsers.Page{{Title: "Rust by Example", URL: "https://doc.rust-lang.org/rust-by-example/", Browser: "Zen", At: time.Now().Add(-26 * time.Hour)}}
	deps := Deps{Files: fakeFiles{res: lease}, Notes: store, Scripts: folder,
		Browser: func(string, int, int) ([]browsers.Page, error) { return pages, nil }}
	locals := deps.Tools()

	cfg := config.Default().Chat
	cfg.Enabled = true
	model := llm.NewServer(llm.Options{
		Layout: func() llm.Layout { return layout }, ContextTokens: func() int { return cfg.ContextTokens },
		IdleMinutes: func() int { return 0 }, Logf: func(string, ...any) {},
	})
	t.Cleanup(model.Unload)

	var mu sync.Mutex
	done := make(chan chat.Done, 4)
	var svc *chat.Service
	svc = chat.NewService(chat.Options{
		Model: model, Settings: func() config.Chat { mu.Lock(); defer mu.Unlock(); return cfg }, Installed: func() bool { return true },
		Locals: func() []chat.Local { return locals }, UpdateTools: func(func(on, always []string) ([]string, []string)) error { return nil },
		Emit: func(name string, data any) {
			switch d := data.(type) {
			case chat.Done:
				done <- d
			case chat.Confirm:
				go svc.Answer(d.ID, true)
			}
		},
		SetThinking: func(bool) error { return nil }, SetWebSearch: func(bool) error { return nil },
	})

	cases := []struct {
		group, question, want, inAnswer string
	}{
		{"files", "What does my lease say about having a pet?", "lease-2026.pdf", "300"},
		{"notes", "What is on my groceries note?", "", "sourdough"},
		{"script:wifi.ps1", "Which Wi-Fi network am I on right now?", "", "HomeNet"},
		{"browser", "Which Rust tutorial page did I look at yesterday?", "", "Rust by Example"},
		{"", "What is 12 times 7?", "", "84"},
	}
	all := []string{"files", "notes", "script:wifi.ps1", "browser"}
	right := map[string]int{}
	for _, setup := range []string{"own", "all"} {
		for _, c := range cases {
			mu.Lock()
			cfg.Tools = []string{c.group}
			if setup == "all" {
				cfg.Tools = all
			}
			mu.Unlock()
			svc.New()
			began := time.Now()
			if _, err := svc.Send(c.question); err != nil {
				t.Fatal(err)
			}
			var d chat.Done
			select {
			case d = <-done:
			case <-time.After(3 * time.Minute):
				t.Fatal("no answer")
			}
			last := d.View.Messages[len(d.View.Messages)-1]
			ok := strings.Contains(strings.ToLower(last.Text), strings.ToLower(c.inAnswer))
			if c.want != "" {
				ok = ok && slices.ContainsFunc(last.Sources, func(s chat.Source) bool { return strings.HasSuffix(s.Path, c.want) })
			}
			if ok {
				right[setup]++
			}
			t.Logf("[%s] Q: %s\n  A (%v, ok=%v): %s\n  sources: %+v %s", setup, c.question, time.Since(began).Round(100*time.Millisecond), ok,
				strings.TrimSpace(last.Text), last.Sources, d.Error)
		}
	}
	t.Logf("right: own group %d/%d, every group on %d/%d", right["own"], len(cases), right["all"], len(cases))
	if right["own"] < len(cases)-1 {
		t.Errorf("with only its own tools on, the model got %d of %d right", right["own"], len(cases))
	}
}
