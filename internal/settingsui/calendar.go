package settingsui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"omni/internal/calendar"
	"omni/internal/config"
	"omni/internal/settings"
)

// The calendar page: the calendars shown in the calendar window, each a subscription address kept in the credential store.

// CalendarOptions are the parts of the program the page needs; main supplies them.
type CalendarOptions struct {
	Store    *settings.Store
	Calendar *calendar.Calendar
	// SetAddress and DeleteAddress keep a feed's address in the credential store (secrets, under "calendar-<id>").
	SetAddress    func(id, address string) error
	DeleteAddress func(id string) error
	// Open shows the calendar window.
	Open func()
}

// Calendar is bound to the calendar page of the settings window. Methods that change something return the reason they were
// refused, or "" when it worked.
type Calendar struct{ o CalendarOptions }

// NewCalendar returns the page's service.
func NewCalendar(o CalendarOptions) *Calendar { return &Calendar{o: o} }

// CalendarFeedView is one calendar as the page lists it.
type CalendarFeedView struct {
	ID, Name string
	// Synced is when it was last fetched, as the page says it ("3 minutes ago" is the page's to write); empty if never.
	Synced string
	Events int
	Error  string
}

// View lists the calendars.
func (c *Calendar) View() []CalendarFeedView {
	var out []CalendarFeedView
	for _, st := range c.o.Calendar.Statuses() {
		v := CalendarFeedView{ID: st.ID, Name: st.Name, Events: st.Events, Error: st.Error}
		if !st.Synced.IsZero() {
			v.Synced = st.Synced.Format(time.RFC3339)
		}
		out = append(out, v)
	}
	return out
}

// Add adds a calendar by its subscription address and fetches it at once.
func (c *Calendar) Add(name, address string) string {
	name, address = strings.TrimSpace(name), calendar.WebAddress(address)
	if name == "" {
		return "give the calendar a name"
	}
	if u, err := url.Parse(address); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "that is not a calendar address: it starts with https:// or webcal://"
	}
	id, err := newID()
	if err != nil {
		return err.Error()
	}
	if err := c.o.SetAddress(id, address); err != nil {
		return fmt.Sprintf("keep the address in the credential store: %v", err)
	}
	if err := c.o.Store.Update(func(cfg *config.Config) {
		cfg.Calendar.Feeds = append(cfg.Calendar.Feeds, config.CalendarFeed{ID: id, Name: name})
	}); err != nil {
		_ = c.o.DeleteAddress(id)
		return reason(err)
	}
	c.o.Calendar.Sync(context.Background())
	return ""
}

// Remove forgets a calendar, its address and its copy.
func (c *Calendar) Remove(id string) string {
	if err := c.o.Store.Update(func(cfg *config.Config) {
		cfg.Calendar.Feeds = slices.DeleteFunc(cfg.Calendar.Feeds, func(f config.CalendarFeed) bool { return f.ID == id })
	}); err != nil {
		return reason(err)
	}
	if err := c.o.DeleteAddress(id); err != nil {
		return fmt.Sprintf("remove the address from the credential store: %v", err)
	}
	c.o.Calendar.Forget()
	return ""
}

// Refresh fetches every calendar now.
func (c *Calendar) Refresh() { c.o.Calendar.Sync(context.Background()) }

// OpenCalendar shows the calendar window.
func (c *Calendar) OpenCalendar() { c.o.Open() }

func newID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("make an id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
