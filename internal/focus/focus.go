// Package focus keeps the sites and programs the person listed out of the way while a focus period runs. Once a second it looks at
// the window in front (its program, and the address a browser shows) and hides it when it is on the list; the caller then shows the
// timer, so what is left of the period is what the person sees instead.
//
// It asks for no extra permission on Windows (the address comes from UI Automation, as a screen reader reads it). On a Mac it asks
// System Events and the browser through AppleScript, which macOS confirms once per app. What is in front never leaves the computer.
//
// What is in front and how to hide it are in focus_windows.go and focus_darwin.go.
package focus

import (
	"context"
	"net/url"
	"runtime"
	"strings"
	"time"

	"omni/internal/config"
)

// front is the window in front: its program ("discord.exe" on Windows, "Discord" on a Mac), the address it shows if it is a
// browser, and what hide needs to find it again.
type front struct {
	app    string
	url    string
	handle uintptr
}

// probe looks at the window in front and hides it. One is made on the goroutine that uses it, which keeps it to one thread.
type probe interface {
	front() (front, error)
	hide(front) error
	close()
}

// Guard hides what is listed while Focusing says a focus period runs.
type Guard struct {
	Focusing func() bool
	Rules    func() config.Timer
	// OnBlock is told what rule hid a window ("youtube.com"), after it was hidden.
	OnBlock func(rule string)
	Logf    func(format string, args ...any)

	newProbe func() (probe, error) // the system's, unless a test sets it
}

// interval is how often the window in front is looked at during a focus period. ponytail: polling; an event hook on the
// foreground window would react at once, if a second ever matters.
const interval = time.Second

// Run watches until ctx ends. Outside a focus period it does nothing but read the clock.
func (g *Guard) Run(ctx context.Context) {
	runtime.LockOSThread() // UI Automation is a COM object, bound to the thread that made it
	defer runtime.UnlockOSThread()
	open := g.newProbe
	if open == nil {
		open = newProbe
	}
	var p probe
	defer func() {
		if p != nil {
			p.close()
		}
	}()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		rules := g.Rules()
		if !rules.Block || !g.Focusing() {
			continue
		}
		if p == nil {
			var err error
			if p, err = open(); err != nil {
				g.Logf("focus: cannot watch the window in front, so nothing is blocked: %v", err)
				return
			}
		}
		f, err := p.front()
		if err != nil {
			continue // a window that closed while it was read: the next look settles it
		}
		rule, ok := blocked(f, rules)
		if !ok {
			continue
		}
		if err := p.hide(f); err != nil {
			g.Logf("focus: hide %s: %v", f.app, err)
			continue
		}
		if g.OnBlock != nil {
			g.OnBlock(rule)
		}
	}
}

// blocked says which rule, if any, the window in front breaks.
func blocked(f front, rules config.Timer) (string, bool) {
	for _, app := range rules.BlockApps {
		if programName(app) != "" && programName(app) == programName(f.app) {
			return app, true
		}
	}
	host := hostOf(f.url)
	if host == "" {
		return "", false
	}
	for _, site := range rules.BlockSites {
		s := hostOf(site)
		if s != "" && (host == s || strings.HasSuffix(host, "."+s)) {
			return site, true
		}
	}
	return "", false
}

// programName compares programs as people write them: any case, with or without ".exe".
func programName(s string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".exe")
}

// hostOf is the site of an address as a browser shows it or a person types it ("youtube.com/watch?v=1", "https://www.x.com/"),
// without "www.". A word with no dot (what is being typed in the address bar) is no site.
func hostOf(address string) string {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" || strings.ContainsAny(address, " \t") {
		return ""
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	if !strings.Contains(host, ".") {
		return ""
	}
	return host
}
