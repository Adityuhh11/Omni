package settingsui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"omni/internal/chat"
	"omni/internal/llm"
	"omni/internal/settings"
	"omni/internal/testpath"
)

type fakeChatSetup struct {
	state   llm.InstallState
	started int
	removed int
}

func (f *fakeChatSetup) State() llm.InstallState { return f.state }
func (f *fakeChatSetup) Start() error            { f.started++; return nil }
func (f *fakeChatSetup) Cancel()                 {}
func (f *fakeChatSetup) Remove() error           { f.removed++; return nil }
func (f *fakeChatSetup) Layout() llm.Layout      { return llm.Layout{} }

type fakeChatModel struct {
	status   llm.Status
	unloaded int
	rearmed  int
}

func (f *fakeChatModel) Status() llm.Status { return f.status }
func (f *fakeChatModel) Unload()            { f.unloaded++ }
func (f *fakeChatModel) Rearm()             { f.rearmed++ }

type fakeChatHistory struct{ n int }

func (f fakeChatHistory) Dir() string { return testpath.P(`C:\chats`) }
func (f fakeChatHistory) Search(string, int) ([]chat.Summary, error) {
	return make([]chat.Summary, f.n), nil
}

type omniRig struct {
	page    *OmniAI
	store   *settings.Store
	setup   *fakeChatSetup
	model   *fakeChatModel
	hotkeys []string
	cleared int
	rebind  error
}

func newOmniRig(t *testing.T) *omniRig {
	t.Helper()
	store, err := settings.Open(filepath.Join(t.TempDir(), "Omni", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	r := &omniRig{store: store, setup: &fakeChatSetup{}, model: &fakeChatModel{}}
	r.page = NewOmniAI(OmniAIOptions{
		Store: store, Setup: r.setup, Model: r.model, History: fakeChatHistory{n: 4},
		ClearHistory: func() error { r.cleared++; return nil },
		RebindHotkey: func(a string) error { r.hotkeys = append(r.hotkeys, a); return r.rebind },
		HotkeyError:  func() string { return "" },
		OpenFolder:   func(string) error { return errors.New("no explorer") },
	})
	return r
}

func TestTheViewShowsTheStateTheChoicesAndTheDefaults(t *testing.T) {
	r := newOmniRig(t)
	r.model.status = llm.Status{State: llm.Ready, Device: "gpu", UnloadsAt: time.UnixMilli(5000)}
	v := r.page.View()
	if v.Enabled || v.Installed || v.Load != "when_asked" || v.UnloadAfterMinutes != 10 || v.ContextTokens != 8192 || !v.KeepHistory || v.HistoryDays != 90 {
		t.Fatalf("defaults = %+v", v)
	}
	if v.ModelState != "ready" || v.Device != "gpu" || v.UnloadsAt != 5000 || v.Conversations != 4 || v.HistoryFolder != testpath.P(`C:\chats`) {
		t.Fatalf("model and history = %+v", v)
	}
	if len(v.Hosts) != 2 || v.DownloadMB < 1000 || v.LicenseURL == "" || len(v.ContextChoices) != 4 || len(v.Loads) != 2 {
		t.Fatalf("what the page tells before the download = %+v", v)
	}
}

func TestTurnOnInstallsFirstAndOnlySwitchesOnWhenInstalled(t *testing.T) {
	r := newOmniRig(t)
	if msg := r.page.TurnOn(); msg != "" || r.setup.started != 1 || r.store.Get().Chat.Enabled {
		t.Fatalf("not installed: msg %q, started %d, enabled %v; the download starts and main switches on when it ends", msg, r.setup.started, r.store.Get().Chat.Enabled)
	}
	r.setup.state = llm.InstallState{Installed: true}
	if msg := r.page.TurnOn(); msg != "" || !r.store.Get().Chat.Enabled || r.setup.started != 1 {
		t.Fatalf("installed: msg %q, enabled %v, started %d", msg, r.store.Get().Chat.Enabled, r.setup.started)
	}
	r.setup.state = llm.InstallState{Installed: true, NeedsUpdate: true}
	if r.page.TurnOn(); r.setup.started != 2 || r.store.Get().Chat.Enabled {
		t.Fatalf("an out-of-date install is updated first: started %d", r.setup.started)
	}
	r.setup.state = llm.InstallState{Installing: true}
	if msg := r.page.TurnOn(); msg == "" {
		t.Fatal("a second download was started")
	}
}

func TestTurnOffUnloadsAndRemoveDeletesTheFiles(t *testing.T) {
	r := newOmniRig(t)
	r.setup.state = llm.InstallState{Installed: true}
	r.page.TurnOn()
	if msg := r.page.TurnOff(); msg != "" || r.store.Get().Chat.Enabled || r.model.unloaded != 1 {
		t.Fatalf("TurnOff: %q, enabled %v, unloaded %d", msg, r.store.Get().Chat.Enabled, r.model.unloaded)
	}
	if msg := r.page.Remove(); msg != "" || r.setup.removed != 1 || r.model.unloaded != 2 {
		t.Fatalf("Remove: %q, removed %d, unloaded %d", msg, r.setup.removed, r.model.unloaded)
	}
}

func TestBehaviourIsValidatedAndANewContextSizeUnloadsTheModel(t *testing.T) {
	r := newOmniRig(t)
	if msg := r.page.SetBehaviour("at_start", 30, 16384); msg != "" {
		t.Fatal(msg)
	}
	got := r.store.Get().Chat
	if got.Load != "at_start" || got.UnloadAfterMinutes != 30 || got.ContextTokens != 16384 || r.model.unloaded != 1 {
		t.Fatalf("chat = %+v, unloaded %d", got, r.model.unloaded)
	}
	if r.page.SetBehaviour("at_start", 5, 16384); r.model.unloaded != 1 {
		t.Fatal("the model was unloaded though the context size did not change")
	}
	for name, msg := range map[string]string{
		"load":    r.page.SetBehaviour("always", 5, 8192),
		"minutes": r.page.SetBehaviour("when_asked", 7, 8192),
		"context": r.page.SetBehaviour("when_asked", 5, 5000),
		"days":    r.page.SetHistory(true, 400),
	} {
		if msg == "" {
			t.Errorf("%s: an invalid value was accepted", name)
		}
	}
	if got := r.store.Get().Chat; got.Load != "at_start" || got.ContextTokens != 16384 {
		t.Fatalf("a refused change must leave the settings alone: %+v", got)
	}
}

func TestTheHotkeyIsBoundBeforeItIsSavedAndARefusedOneChangesNothing(t *testing.T) {
	r := newOmniRig(t)
	if msg := r.page.SetHotkey(" Ctrl+Alt+A "); msg != "" || r.store.Get().Chat.Hotkey != "Ctrl+Alt+A" || len(r.hotkeys) != 1 {
		t.Fatalf("msg %q, hotkey %q, bound %v", msg, r.store.Get().Chat.Hotkey, r.hotkeys)
	}
	r.rebind = errors.New("another program owns it")
	if msg := r.page.SetHotkey("Ctrl+Alt+B"); msg == "" || r.store.Get().Chat.Hotkey != "Ctrl+Alt+A" {
		t.Fatalf("a hotkey that cannot be bound was saved: %q", r.store.Get().Chat.Hotkey)
	}
	r.rebind = nil
	before := len(r.hotkeys)
	if msg := r.page.SetHotkey("alt+shift+space"); msg == "" || len(r.hotkeys) != before {
		t.Fatalf("the launcher's own hotkey must be refused before it is registered: %q, bound %v", msg, r.hotkeys)
	}
	if msg := r.page.SetHotkey(""); msg != "" || r.store.Get().Chat.Hotkey != "" {
		t.Fatalf("turning it off: %q", msg)
	}
}

func TestHistorySettingsAndClearing(t *testing.T) {
	r := newOmniRig(t)
	if msg := r.page.SetHistory(false, 30); msg != "" || r.store.Get().Chat.KeepHistory || r.store.Get().Chat.HistoryDays != 30 {
		t.Fatalf("SetHistory: %q %+v", msg, r.store.Get().Chat)
	}
	if msg := r.page.ClearHistory(); msg != "" || r.cleared != 1 {
		t.Fatalf("ClearHistory: %q, cleared %d", msg, r.cleared)
	}
	if msg := r.page.OpenHistoryFolder(); msg != "no explorer" {
		t.Fatalf("OpenHistoryFolder = %q", msg)
	}
	if msg := r.page.SetThinking(true); msg != "" || !r.store.Get().Chat.Thinking {
		t.Fatalf("SetThinking: %q", msg)
	}
	if !r.page.View().OnTop {
		t.Fatal("the chat window stays on top by default")
	}
	if msg := r.page.SetOnTop(false); msg != "" || r.store.Get().Chat.OnTop || r.page.View().OnTop {
		t.Fatalf("SetOnTop: %q", msg)
	}
}

func TestUsingAnotherModelTurnsOmniAIOnFreesTheBundledOneAndGoingBackFollowsTheInstall(t *testing.T) {
	r := newOmniRig(t)
	key := ""
	r.page.o.CustomKey = func() string { return key }
	r.page.o.SetCustomKey = func(k string) error { key = k; return nil }

	if msg := r.page.UseCustom(" http://localhost:1234/v1 ", " qwen "); msg != "" {
		t.Fatal(msg)
	}
	c := r.store.Get().Chat
	if !c.Custom || !c.Enabled || c.CustomURL != "http://localhost:1234/v1" || c.CustomModel != "qwen" || r.model.unloaded != 1 {
		t.Fatalf("chat = %+v, unloaded %d", c, r.model.unloaded)
	}
	if v := r.page.View().Custom; !v.On || !v.Local || v.Host != "localhost:1234" || v.HasKey {
		t.Fatalf("view = %+v", v)
	}
	if msg := r.page.SetCustomKey(" sk-1 "); msg != "" || key != "sk-1" || !r.page.View().Custom.HasKey {
		t.Fatalf("key %q, %q", key, msg)
	}
	if msg := r.page.TurnOff(); msg != "" || r.page.TurnOn() != "" || !r.store.Get().Chat.Enabled || r.setup.started != 0 {
		t.Fatal("turning on with another model must not start the download")
	}
	if r.page.UseCustom("ftp://nope", "m") == "" {
		t.Fatal("a bad address was accepted")
	}

	if msg := r.page.UseBundled(); msg != "" {
		t.Fatal(msg)
	}
	if c := r.store.Get().Chat; c.Custom || c.Enabled { // the bundled model is not installed, so Omni AI cannot stay on
		t.Fatalf("chat = %+v", c)
	}
	r.setup.state.Installed = true
	r.page.UseCustom("https://api.example.com/v1", "m")
	r.page.UseBundled()
	if c := r.store.Get().Chat; c.Custom || !c.Enabled {
		t.Fatalf("chat = %+v", c)
	}
}

func TestTestCustomListsTheModelsWithTheTypedKeyOrTheSavedOne(t *testing.T) {
	r := newOmniRig(t)
	var used llm.Endpoint
	r.page.o.CustomKey = func() string { return "saved" }
	r.page.o.ListModels = func(_ context.Context, ep llm.Endpoint) ([]string, error) {
		used = ep
		if ep.URL == "http://down" {
			return nil, errors.New("reach http://down: refused")
		}
		return []string{"a", "b"}, nil
	}
	if got := r.page.TestCustom("http://host:1/v1", ""); got.Error != "" || len(got.Models) != 2 || used.Key != "saved" || used.URL != "http://host:1" {
		t.Fatalf("got %+v, used %+v", got, used)
	}
	if r.page.TestCustom("http://host:1/v1", " typed "); used.Key != "typed" {
		t.Fatalf("used %+v", used)
	}
	if got := r.page.TestCustom("http://down", ""); got.Error == "" {
		t.Fatalf("got %+v", got)
	}
	if got := r.page.TestCustom("not an address", ""); got.Error == "" {
		t.Fatalf("got %+v", got)
	}
}
