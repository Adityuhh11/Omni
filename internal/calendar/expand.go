package calendar

import (
	"cmp"
	"regexp"
	"slices"
	"time"

	"github.com/teambition/rrule-go"
)

// Event is one occurrence in a window of time, as the calendar window and the bar show it.
type Event struct {
	// ID is the feed's uid and the occurrence's start, so each repetition of a meeting is its own event.
	ID       string
	Calendar string // the name of the feed it came from
	Color    int    // the feed's place in the list, which picks its colour on the page
	Title    string
	Location string
	Notes    string
	Start    time.Time
	End      time.Time
	AllDay   bool
	// Join is the address of the video call the event names (Zoom, Meet, Teams, Webex), or empty.
	Join string
}

// expand lists the occurrences of a feed's events that overlap [from, to), first to last.
func expand(events []vevent, calendar string, color int, from, to time.Time) []Event {
	// A changed or cancelled occurrence of a recurring event replaces the one the rule makes.
	replaced := map[string][]time.Time{}
	for _, e := range events {
		if !e.recurrenceID.IsZero() {
			replaced[e.uid] = append(replaced[e.uid], e.recurrenceID)
		}
	}
	var out []Event
	add := func(e vevent, start time.Time) {
		end := start.Add(e.end.Sub(e.start))
		if !start.Before(to) || !end.After(from) || e.status == "CANCELLED" {
			return
		}
		out = append(out, Event{
			ID: e.uid + "@" + start.UTC().Format(time.RFC3339), Calendar: calendar, Color: color, Title: e.summary, Location: e.location,
			Notes: e.description, Start: start, End: end, AllDay: e.allDay, Join: joinLink(e.url, e.location, e.description),
		})
	}
	for _, e := range events {
		if e.rrule == "" || !e.recurrenceID.IsZero() {
			add(e, e.start)
			continue
		}
		opt, err := rrule.StrToROption(e.rrule)
		if err != nil {
			add(e, e.start) // a rule it cannot read: at least the first occurrence shows
			continue
		}
		opt.Dtstart = e.start
		rule, err := rrule.NewRRule(*opt)
		if err != nil {
			add(e, e.start)
			continue
		}
		set := rrule.Set{}
		set.RRule(rule)
		for _, t := range append(e.exdates, replaced[e.uid]...) {
			set.ExDate(t)
		}
		// Occurrences that started before from but are still going count too.
		for _, start := range set.Between(from.Add(-e.end.Sub(e.start)), to, true) {
			add(e, start.In(e.start.Location()))
		}
	}
	sortEvents(out)
	return out
}

// sortEvents puts events first to last; two that start together go by title, then id, so the order never flickers.
func sortEvents(events []Event) {
	slices.SortFunc(events, func(a, b Event) int {
		return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.Title, b.Title), cmp.Compare(a.ID, b.ID))
	})
}

// meeting finds the address of a video call. The domains are the services' own; anything after them up to a space or a quote
// is the address.
var meeting = regexp.MustCompile(`https://[a-zA-Z0-9.-]*(zoom\.us/(j|my)/|meet\.google\.com/|teams\.microsoft\.com/l/meetup-join/|teams\.live\.com/meet/|webex\.com/)[^\s"<>]*`)

func joinLink(fields ...string) string {
	for _, f := range fields {
		if m := meeting.FindString(f); m != "" {
			return m
		}
	}
	return ""
}
