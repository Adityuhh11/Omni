// Package calendar shows the person's calendars, read only. Each calendar is a subscription address (iCalendar, the "secret
// address" Google gives, an iCloud shared calendar, Outlook's published one) kept in the system credential store, because
// anyone with the address can read the calendar. The feeds are fetched in the background every 15 minutes and kept on disk,
// so the bar and the window read memory and never wait for the network, and a calendar still shows when the computer is offline.
package calendar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // feeds name their zones ("Europe/Berlin"); Windows has no zone database Go can read

	"omni/internal/atomicfile"
	"omni/internal/config"
)

// Every is how often the feeds are fetched. Google itself refreshes a secret address only every few hours.
const Every = 15 * time.Minute

// maxFeed is the largest feed read: years of a busy calendar are a few megabytes.
const maxFeed = 32 << 20

// Status is how a feed's last fetch went.
type Status struct {
	ID, Name string
	// Synced is when it was last fetched well; zero if never.
	Synced time.Time
	Events int
	Error  string
}

// Calendar holds the feeds' events.
type Calendar struct {
	// Feeds are the calendars in the settings; Address finds the subscription address of one.
	Feeds   func() []config.CalendarFeed
	Address func(id string) (string, error)
	// Dir keeps each feed's last copy as <id>.ics.
	Dir    string
	Client *http.Client
	Logf   func(format string, args ...any)

	sync sync.Mutex // one fetch of everything at a time

	mu     sync.Mutex
	events map[string][]vevent
	status map[string]Status
}

// New returns a calendar that keeps its copies in dir.
func New(dir string, feeds func() []config.CalendarFeed, address func(id string) (string, error), logf func(string, ...any)) *Calendar {
	return &Calendar{Feeds: feeds, Address: address, Dir: dir, Client: &http.Client{Timeout: 30 * time.Second}, Logf: logf,
		events: map[string][]vevent{}, status: map[string]Status{}}
}

// Run reads the copies on disk, then fetches every feed now and every 15 minutes until ctx ends.
func (c *Calendar) Run(ctx context.Context) {
	c.load()
	tick := time.NewTicker(Every)
	defer tick.Stop()
	for {
		c.Sync(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// load reads the copy of each feed kept from its last fetch.
func (c *Calendar) load() {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		c.Logf("calendar: %v", err) // the copies are not kept; the calendars still show while the program runs
	}
	for _, f := range c.Feeds() {
		path := c.copyOf(f.ID)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // never fetched yet
		}
		info, _ := os.Stat(path)
		c.keep(f, parse(data, time.Local), info.ModTime(), "")
	}
}

// Sync fetches every feed once; a feed that fails keeps the events it had, and says why.
func (c *Calendar) Sync(ctx context.Context) {
	c.sync.Lock()
	defer c.sync.Unlock()
	feeds := c.Feeds()
	for _, f := range feeds {
		data, err := c.fetch(ctx, f.ID)
		if err != nil {
			c.mu.Lock()
			before := c.status[f.ID].Error
			c.mu.Unlock()
			if err.Error() != before { // a feed that stays down is logged once, not every quarter hour
				c.Logf("calendar %q: %v", f.Name, err)
			}
			c.keep(f, nil, time.Time{}, err.Error())
			continue
		}
		if err := atomicfile.Write(c.copyOf(f.ID), data); err != nil {
			c.Logf("calendar %q: keep a copy: %v", f.Name, err)
		}
		c.keep(f, parse(data, time.Local), time.Now(), "")
	}
	c.forget(feeds)
}

// Forget drops what is held for calendars no longer in the settings, without fetching anything.
func (c *Calendar) Forget() { c.forget(c.Feeds()) }

func (c *Calendar) fetch(ctx context.Context, id string) ([]byte, error) {
	address, err := c.Address(id)
	if err != nil {
		return nil, fmt.Errorf("read its address from the credential store: %w", err)
	}
	address = WebAddress(address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("the address: %w", err)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		// The address is a secret and never goes into the log. A *url.Error names the address it failed at, which after a redirect
		// is not the one asked for, so only what went wrong is kept.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("fetch the calendar: %s", scrub(err.Error(), address))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the server answered %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFeed+1))
	if err != nil {
		return nil, fmt.Errorf("read the feed: %w", err)
	}
	if len(data) > maxFeed {
		return nil, fmt.Errorf("the feed is larger than %d MB", maxFeed>>20)
	}
	if !strings.Contains(string(data[:min(len(data), 4096)]), "BEGIN:VCALENDAR") {
		return nil, errors.New("that address does not give a calendar (no BEGIN:VCALENDAR)")
	}
	return data, nil
}

// WebAddress turns webcal:// (what calendar sites hand out for subscribing) into https://.
func WebAddress(address string) string {
	address = strings.TrimSpace(address)
	if rest, ok := strings.CutPrefix(strings.ToLower(address), "webcal://"); ok {
		return "https://" + address[len(address)-len(rest):]
	}
	return address
}

func scrub(message, address string) string {
	return strings.ReplaceAll(message, address, "the address")
}

// keep stores what a fetch found. events nil (a failed fetch) keeps the events there were.
func (c *Calendar) keep(f config.CalendarFeed, events []vevent, synced time.Time, problem string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.status[f.ID]
	st.ID, st.Name, st.Error = f.ID, f.Name, problem
	if events != nil {
		c.events[f.ID] = events
		st.Synced, st.Events = synced, len(events)
	}
	c.status[f.ID] = st
}

// forget drops what is held for feeds that are no longer in the settings, and their copies (also those of a feed removed while
// the program was not running).
func (c *Calendar) forget(feeds []config.CalendarFeed) {
	c.mu.Lock()
	for id := range c.status {
		if !hasFeed(feeds, id) {
			delete(c.status, id)
			delete(c.events, id)
		}
	}
	c.mu.Unlock()
	copies, _ := filepath.Glob(filepath.Join(c.Dir, "*.ics"))
	for _, path := range copies {
		if id := strings.TrimSuffix(filepath.Base(path), ".ics"); !hasFeed(feeds, id) {
			if err := os.Remove(path); err != nil {
				c.Logf("calendar: remove the copy of a removed feed: %v", err)
			}
		}
	}
}

func hasFeed(feeds []config.CalendarFeed, id string) bool {
	for _, f := range feeds {
		if f.ID == id {
			return true
		}
	}
	return false
}

func (c *Calendar) copyOf(id string) string { return filepath.Join(c.Dir, id+".ics") }

// Between lists every calendar's occurrences that overlap [from, to), first to last.
func (c *Calendar) Between(from, to time.Time) []Event {
	feeds := c.Feeds()
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Event
	for i, f := range feeds {
		out = append(out, expand(c.events[f.ID], f.Name, i, from, to)...)
	}
	sortEvents(out)
	return out
}

// Statuses is how each feed's last fetch went, in the order of the settings.
func (c *Calendar) Statuses() []Status {
	feeds := c.Feeds()
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Status, 0, len(feeds))
	for _, f := range feeds {
		st := c.status[f.ID]
		st.ID, st.Name = f.ID, f.Name
		out = append(out, st)
	}
	return out
}
