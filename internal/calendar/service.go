package calendar

import (
	"context"
	"time"
)

// Service is the calendar as its window uses it; its exported methods become TypeScript functions.
type Service struct {
	c            *Calendar
	open         func(address string) error
	openSettings func()
}

// NewService returns the window's service. open opens an address in the browser (a meeting link); openSettings shows the
// calendar page of the settings.
func NewService(c *Calendar, open func(string) error, openSettings func()) *Service {
	return &Service{c: c, open: open, openSettings: openSettings}
}

// Between lists the events that overlap the days from start (milliseconds since 1970, as the page counts) for days days.
func (s *Service) Between(startMillis int64, days int) []Event {
	from := time.UnixMilli(startMillis)
	return s.c.Between(from, from.AddDate(0, 0, max(1, min(days, 62))))
}

// Calendars is each calendar's name and how its last fetch went, in the order that picks its colour.
func (s *Service) Calendars() []Status { return s.c.Statuses() }

// Refresh fetches every calendar now; the page asks for the events again after it.
func (s *Service) Refresh() { s.c.Sync(context.Background()) }

// Join opens an event's meeting address. Only an address that is a meeting link is opened: the page passes what an event says.
func (s *Service) Join(address string) string {
	if !IsMeetingLink(address) {
		return "that is not a meeting link"
	}
	if err := s.open(address); err != nil {
		return err.Error()
	}
	return ""
}

// OpenSettings shows where calendars are added.
func (s *Service) OpenSettings() { s.openSettings() }
