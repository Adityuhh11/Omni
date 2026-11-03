package palette

import (
	"fmt"
	"strings"
	"time"

	"omni/internal/calendar"
)

// CalendarBox is the calendar as the bar reads it; *calendar.Calendar satisfies it. It answers from memory.
type CalendarBox interface {
	Between(from, to time.Time) []calendar.Event
}

// calendarWords list what is coming; joinWords offer the next meeting with a link.
var (
	calendarWords = map[string]bool{"cal": true, "calendar": true, "agenda": true}
	joinWords     = map[string]bool{"join": true}
)

// calendarAhead is how far the bar looks: the rest of today and tomorrow morning. upcomingRows is how many it lists.
const (
	calendarAhead = 36 * time.Hour
	upcomingRows  = 5
)

// calendarItems are the rows for "cal" (what is coming, then the window) and "join" (the next meeting with a link).
func (s *Service) calendarItems(text string) []Item {
	if s.x.Calendar == nil {
		return nil
	}
	word := strings.ToLower(strings.TrimSpace(text))
	now := s.now()
	if !calendarWords[word] && !joinWords[word] {
		return nil
	}
	var events []calendar.Event
	for _, e := range s.x.Calendar.Between(now, now.Add(calendarAhead)) {
		if !e.AllDay || sameDay(e.Start, now) {
			events = append(events, e)
		}
	}
	if joinWords[word] {
		for _, e := range events {
			if e.Join != "" && !e.AllDay {
				item := eventItem(e, now)
				item.Title = "Join " + e.Title
				return []Item{item}
			}
		}
		return nil
	}
	var items []Item
	for _, e := range events[:min(len(events), upcomingRows)] {
		items = append(items, eventItem(e, now))
	}
	return append(items, Item{ID: "cal:open", Kind: "calendar", Title: "Open the calendar", Subtitle: "Calendar"})
}

// eventItem is an event as a row: Enter joins its meeting, or opens the calendar. Every row has its own id, because the page keys
// its list by id and two rows with one id break it: "cal:show:<event>", or "cal:join:<event> <link>" (neither holds a space).
func eventItem(e calendar.Event, now time.Time) Item {
	id := "cal:show:" + e.ID
	if e.Join != "" {
		id = "cal:join:" + e.ID + " " + e.Join
	}
	when := "All day"
	if !e.AllDay {
		when = e.Start.Local().Format("15:04") + "–" + e.End.Local().Format("15:04")
	}
	if !sameDay(e.Start, now) {
		when = "Tomorrow " + when
	}
	parts := []string{when, e.Calendar}
	if soon := untilStart(e, now); soon != "" {
		parts = append(parts, soon)
	}
	if e.Join != "" {
		parts = append(parts, "Enter joins")
	}
	return Item{ID: id, Kind: "calendar", Title: e.Title, Subtitle: strings.Join(parts, " · ")}
}

// untilStart says when a timed event begins, while that is near: "now", "in 25 min", "in 2 h".
func untilStart(e calendar.Event, now time.Time) string {
	switch d := e.Start.Sub(now); {
	case e.AllDay:
		return ""
	case d <= 0:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("in %d min", int(d.Minutes()+0.5))
	case d < 6*time.Hour:
		return fmt.Sprintf("in %d h", int(d.Hours()+0.5))
	}
	return ""
}

func sameDay(a, b time.Time) bool {
	a, b = a.Local(), b.Local()
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// doCalendar joins a meeting or opens the calendar, and closes the bar.
func (s *Service) doCalendar(rest, action string) error {
	if action != "open" {
		return fmt.Errorf("a calendar row can only be opened, not %q", action)
	}
	if join, ok := strings.CutPrefix(rest, "join:"); ok {
		_, address, _ := strings.Cut(join, " ")
		if s.x.Join == nil {
			return fmt.Errorf("joining is not available")
		}
		if err := s.x.Join(address); err != nil {
			return err
		}
		s.dismiss()
		return nil
	}
	if s.x.OpenCalendar == nil {
		return fmt.Errorf("the calendar is not available")
	}
	s.dismiss()
	s.x.OpenCalendar()
	return nil
}
