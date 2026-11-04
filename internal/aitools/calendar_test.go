package aitools

import (
	"context"
	"strings"
	"testing"
	"time"

	"omni/internal/calendar"
)

type fakeCalendar []calendar.Event

func (f fakeCalendar) Between(from, to time.Time) []calendar.Event {
	var out []calendar.Event
	for _, e := range f {
		if e.Start.Before(to) && e.End.After(from) {
			out = append(out, e)
		}
	}
	return out
}

// Wednesday 4 November 2026, 10:20.
var calNow = time.Date(2026, 11, 4, 10, 20, 0, 0, time.Local)

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 11, day, hour, minute, 0, 0, time.Local)
}

func TestCalendarSpanReadsWhatAModelWrites(t *testing.T) {
	for _, tc := range []struct {
		when  string
		first time.Time
		days  int
	}{
		{"", at(4, 0, 0), 1},
		{"Today", at(4, 0, 0), 1},
		{"tomorrow", at(5, 0, 0), 1},
		{"this week", at(4, 0, 0), 5}, // Wednesday to Sunday
		{"next week", at(9, 0, 0), 7},
		{"friday", at(6, 0, 0), 1},
		{"wednesday", at(4, 0, 0), 1},
		{"Monday", at(9, 0, 0), 1},
		{"2026-11-20", at(20, 0, 0), 1},
	} {
		first, days, ok := calendarSpan(calNow, tc.when)
		if !ok || !first.Equal(tc.first) || days != tc.days {
			t.Errorf("%q: %v %d %v, want %v %d", tc.when, first, days, ok, tc.first, tc.days)
		}
	}
	if _, _, ok := calendarSpan(calNow, "someday"); ok {
		t.Error("someday was read as a day")
	}
}

func TestFreeTimeIsTheGapsInWorkingHoursFromNowOn(t *testing.T) {
	ev := func(from, to time.Time) calendar.Event { return calendar.Event{Start: from, End: to} }
	for _, tc := range []struct {
		name   string
		day    time.Time
		events []calendar.Event
		want   string
	}{
		{"an empty day", at(5, 0, 0), nil, "09:00–18:00"},
		{"today starts now", at(4, 0, 0), nil, "10:20–18:00"},
		{"overlapping meetings are one block", at(5, 0, 0), []calendar.Event{ev(at(5, 8, 0), at(5, 10, 0)), ev(at(5, 9, 30), at(5, 11, 0)), ev(at(5, 13, 0), at(5, 14, 0))}, "11:00–13:00, 14:00–18:00"},
		{"a gap under 15 minutes is not free", at(5, 0, 0), []calendar.Event{ev(at(5, 9, 10), at(5, 17, 50))}, "none"},
		{"an evening event does not hide the afternoon", at(5, 0, 0), []calendar.Event{ev(at(5, 9, 0), at(5, 12, 0)), ev(at(5, 19, 0), at(5, 20, 0))}, "12:00–18:00"},
		{"a day that is over", at(3, 0, 0), nil, ""},
	} {
		if got := freeTime(tc.day, calNow, tc.events); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCalendarEventsListsDaysAndGivesTheAgenda(t *testing.T) {
	d := Deps{Calendar: fakeCalendar{
		{ID: "a", Title: "Standup", Calendar: "Work", Start: at(5, 9, 0), End: at(5, 9, 30), Join: "https://meet.google.com/x"},
		{ID: "b", Title: "Trip", Calendar: "Home", AllDay: true, Start: at(5, 0, 0), End: at(7, 0, 0)},
	}, Now: func() time.Time { return calNow }}
	text, sources, err := d.calendarEvents(context.Background(), map[string]string{"when": "tomorrow", "days": "2"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Thursday 5 November:\n- 09:00–09:30 Standup (Work), online meeting\n- all day Trip (Home)\nFree: 09:30–18:00\n",
		"Friday 6 November:\n- all day Trip (Home)\nFree: 09:00–18:00\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	if len(sources) != 2 || sources[0].Event == nil || sources[0].Event.Join == "" || sources[1].Title != "Trip" {
		t.Errorf("sources = %+v, want each event once", sources)
	}
	if text, _, _ := d.calendarEvents(context.Background(), map[string]string{"when": "this week", "days": "1"}); !strings.Contains(text, "Sunday 8 November:") {
		t.Errorf("days shortened a week:\n%s", text)
	}
	if text, _, _ := d.calendarEvents(context.Background(), map[string]string{"when": "soon"}); !strings.HasPrefix(text, "Error:") {
		t.Errorf("an unknown day: %q", text)
	}
}
