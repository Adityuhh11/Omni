package chat

import (
	"os"
	"strings"
	"testing"
	"time"

	"omni/internal/config"
	"omni/internal/llm"
	"omni/internal/websearch"
)

// TestRealModelUsesTheWeb runs the real model with the real web: it asks a question that needs a lookup and one that does not. It
// runs only when OMNI_CHAT_DIR is set, and needs the network and an installed Omni AI (for the person who changes this package).
func TestRealModelUsesTheWeb(t *testing.T) {
	if os.Getenv("OMNI_CHAT_DIR") == "" {
		t.Skip("OMNI_CHAT_DIR is not set")
	}
	layout, err := llm.Locate("")
	if err != nil || !layout.Installed() {
		t.Fatalf("no install: %v", err)
	}
	cfg := config.Default().Chat
	cfg.Enabled, cfg.WebSearch = true, true
	if v := os.Getenv("OMNI_CHAT_THINKING"); v != "" {
		cfg.Thinking = true
	}
	model := llm.NewServer(llm.Options{
		Layout: func() llm.Layout { return layout }, ContextTokens: func() int { return cfg.ContextTokens },
		IdleMinutes: func() int { return 0 }, Logf: t.Logf,
	})
	t.Cleanup(model.Unload)
	ev := &events{done: make(chan Done, 8)}
	svc := NewService(Options{
		Model: model, Web: websearch.New(), Settings: func() config.Chat { return cfg }, Installed: func() bool { return true },
		Emit: ev.emit, Logf: t.Logf, SetThinking: func(bool) error { return nil }, SetWebSearch: func(bool) error { return nil },
	})
	for _, question := range []string{
		"What is the latest stable version of Go, and when was it released?",
		"What is 12 times 7?",
		"Who won the most recent Formula 1 drivers' championship?",
	} {
		svc.New()
		began := time.Now()
		if _, err := svc.Send(question); err != nil {
			t.Fatal(err)
		}
		var done Done
		select {
		case done = <-ev.done:
		case <-time.After(3 * time.Minute):
			t.Fatal("no answer")
		}
		last := done.View.Messages[len(done.View.Messages)-1]
		t.Logf("Q: %s\nA (%v): %s\nsources: %+v\nerror: %q", question, time.Since(began).Round(100*time.Millisecond), strings.TrimSpace(last.Text), last.Sources, done.Error)
		if strings.Contains(question, "latest stable") && len(last.Sources) == 0 {
			t.Errorf("%q needs the web and the answer has no sources: the model did not look", question)
		}
		if done.Error != "" || strings.TrimSpace(last.Text) == "" {
			t.Errorf("no answer to %q: %+v", question, done)
		}
	}
}

// TestRealServerAsAnotherModel asks a question through llm.Switch as if the bundled llama.cpp server were somebody else's
// OpenAI-style server (the person's LM Studio, a cloud service): the request must be the plain kind, the tokens estimated,
// and nothing may start or hold the bundled model for it. Same conditions as the test above, without the network.
func TestRealServerAsAnotherModel(t *testing.T) {
	if os.Getenv("OMNI_CHAT_DIR") == "" {
		t.Skip("OMNI_CHAT_DIR is not set")
	}
	layout, err := llm.Locate("")
	if err != nil || !layout.Installed() {
		t.Fatalf("no install: %v", err)
	}
	cfg := config.Default().Chat
	cfg.Enabled = true
	bundled := llm.NewServer(llm.Options{
		Layout: func() llm.Layout { return layout }, ContextTokens: func() int { return cfg.ContextTokens },
		IdleMinutes: func() int { return 0 }, Logf: t.Logf,
	})
	t.Cleanup(bundled.Unload)
	stand, err := bundled.Ensure(t.Context()) // started by the test only to have a server to ask; the switch below does not own it
	if err != nil {
		t.Fatal(err)
	}
	useCustom := false
	asks := llm.Switch{Bundled: bundled, Key: func() string { return stand.Key },
		Custom: func() (string, string, bool) { return stand.URL + "/v1", "some-model", useCustom }}
	ev := &events{done: make(chan Done, 8)}
	svc := NewService(Options{
		Model: asks, Settings: func() config.Chat { return cfg }, Installed: func() bool { return true },
		Emit: ev.emit, Logf: t.Logf, SetThinking: func(bool) error { return nil }, SetWebSearch: func(bool) error { return nil },
	})
	for _, custom := range []bool{true, false} {
		useCustom = custom
		svc.New()
		if _, err := svc.Send("What is 12 times 7?"); err != nil {
			t.Fatal(err)
		}
		var done Done
		select {
		case done = <-ev.done:
		case <-time.After(3 * time.Minute):
			t.Fatal("no answer")
		}
		last := done.View.Messages[len(done.View.Messages)-1]
		t.Logf("custom=%v: %s (error %q)", custom, strings.TrimSpace(last.Text), done.Error)
		if done.Error != "" || !strings.Contains(last.Text, "84") {
			t.Errorf("custom=%v: %+v", custom, done)
		}
	}
}
