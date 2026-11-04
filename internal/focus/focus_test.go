package focus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"omni/internal/config"
)

func TestWhatIsBlocked(t *testing.T) {
	rules := config.Timer{Block: true, BlockSites: []string{"youtube.com", "https://www.reddit.com/", "x.com", "  "}, BlockApps: []string{"Discord.exe", "Steam"}}
	tests := []struct {
		name string
		f    front
		want string
	}{
		{"a listed site", front{app: "chrome.exe", url: "youtube.com/watch?v=1"}, "youtube.com"},
		{"its subdomain", front{app: "chrome.exe", url: "https://m.youtube.com/"}, "youtube.com"},
		{"www is the same site", front{app: "firefox.exe", url: "https://www.youtube.com"}, "youtube.com"},
		{"a rule written as an address", front{app: "msedge.exe", url: "reddit.com/r/golang"}, "https://www.reddit.com/"},
		{"a short name is not a suffix of a longer one", front{app: "chrome.exe", url: "https://netflix.com"}, ""},
		{"another site that ends the same way", front{app: "chrome.exe", url: "notyoutube.com"}, ""},
		{"a site named in the path", front{app: "chrome.exe", url: "google.com/search?q=youtube.com"}, ""},
		{"words typed in the address bar", front{app: "chrome.exe", url: "youtube"}, ""},
		{"a search typed in the address bar", front{app: "chrome.exe", url: "x com news"}, ""},
		{"a program, any case and without .exe", front{app: "discord.exe"}, "Discord.exe"},
		{"a Mac program", front{app: "steam"}, "Steam"},
		{"anything else", front{app: "code.exe"}, ""},
	}
	for _, tt := range tests {
		got, ok := blocked(tt.f, rules)
		if got != tt.want || ok != (tt.want != "") {
			t.Errorf("%s: blocked = %q, %v; want %q", tt.name, got, ok, tt.want)
		}
	}
}

type fakeProbe struct {
	mu     sync.Mutex
	in     front
	hidden []string
}

func (p *fakeProbe) front() (front, error) { p.mu.Lock(); defer p.mu.Unlock(); return p.in, nil }
func (p *fakeProbe) hide(f front) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hidden = append(p.hidden, f.app)
	p.in = front{app: "explorer.exe"} // what is in front once the window is out of the way
	return nil
}
func (p *fakeProbe) close() {}

func TestTheGuardHidesOnlyDuringFocusAndOnlyWhenBlockingIsOn(t *testing.T) {
	p := &fakeProbe{in: front{app: "discord.exe"}}
	var mu sync.Mutex
	focusing, rules, told := false, config.Timer{Block: true, BlockApps: []string{"discord"}}, []string{}
	g := &Guard{
		Focusing: func() bool { mu.Lock(); defer mu.Unlock(); return focusing },
		Rules:    func() config.Timer { mu.Lock(); defer mu.Unlock(); return rules },
		OnBlock:  func(rule string) { mu.Lock(); told = append(told, rule); mu.Unlock() },
		Logf:     t.Logf,
		newProbe: func() (probe, error) { return p, nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { g.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	time.Sleep(interval + 300*time.Millisecond)
	p.mu.Lock()
	early := len(p.hidden)
	p.mu.Unlock()
	if early != 0 {
		t.Fatalf("hid %d windows outside a focus period", early)
	}
	mu.Lock()
	focusing = true
	mu.Unlock()
	time.Sleep(2*interval + 300*time.Millisecond)
	p.mu.Lock()
	hidden := len(p.hidden)
	p.mu.Unlock()
	mu.Lock()
	if hidden != 1 || len(told) != 1 || told[0] != "discord" {
		t.Errorf("hidden %d, told %v", hidden, told)
	}
	mu.Unlock()
}

func TestAGuardThatCannotWatchStopsAndSaysSo(t *testing.T) {
	var said string
	g := &Guard{
		Focusing: func() bool { return true },
		Rules:    func() config.Timer { return config.Timer{Block: true} },
		Logf:     func(format string, _ ...any) { said = format },
		newProbe: func() (probe, error) { return nil, errors.New("no UI Automation") },
	}
	done := make(chan struct{})
	go func() { g.Run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * interval):
		t.Fatal("the guard kept running without a way to watch")
	}
	if said == "" {
		t.Error("nothing was logged")
	}
}
