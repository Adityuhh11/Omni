package palette

import (
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

func TestCalendarListsWhatIsComingAndJoinOpensTheNextMeeting(t *testing.T) {
	s, dismissed := newService(t)
	now := time.Date(2026, 11, 3, 10, 5, 0, 0, time.Local)
	s.now = func() time.Time { return now }
	at := func(h, m int) time.Time { return time.Date(2026, 11, 3, h, m, 0, 0, time.Local) }
	cal := fakeCalendar{
		{ID: "a", Calendar: "Work", Title: "Holiday", Start: at(0, 0), End: at(24, 0), AllDay: true},
		{ID: "b", Calendar: "Work", Title: "Standup", Start: at(10, 0), End: at(10, 15)},
		{ID: "c", Calendar: "Work", Title: "Design review", Start: at(10, 30), End: at(11, 0), Join: "https://meet.google.com/abc-defg-hij"},
		{ID: "d", Calendar: "Home", Title: "Dentist", Start: at(33, 0), End: at(34, 0)},
		{ID: "e", Calendar: "Home", Title: "Party", Start: at(9, 0).AddDate(0, 0, 3), End: at(11, 0).AddDate(0, 0, 3)},
	}
	var joined []string
	opened := 0
	With(s, Extras{Calendar: cal, OpenCalendar: func() { opened++ }, Join: func(a string) error { joined = append(joined, a); return nil }})

	var rows []string
	for _, item := range s.Query("cal", ModeAll).Items {
		if item.Kind == "calendar" {
			rows = append(rows, item.Title+" | "+item.Subtitle)
		}
	}
	want := []string{
		"Holiday | All day · Work",
		"Standup | 10:00–10:15 · Work · now",
		"Design review | 10:30–11:00 · Work · in 25 min · Enter joins",
		"Dentist | Tomorrow 09:00–10:00 · Home",
		"Open the calendar | Calendar",
	}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Fatalf("cal rows:\n%s\nwant:\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}

	join := s.Query("join", ModeAll).Items
	if len(join) == 0 || join[0].Title != "Join Design review" {
		t.Fatalf("join = %+v", join)
	}
	if err := s.Do(join[0].ID, "open", "join"); err != nil || len(joined) != 1 || joined[0] != "https://meet.google.com/abc-defg-hij" || *dismissed != 1 {
		t.Errorf("joined %v, err %v, dismissed %d", joined, err, *dismissed)
	}
	if err := s.Do("cal:open", "open", "cal"); err != nil || opened != 1 {
		t.Errorf("opened %d, err %v", opened, err)
	}
	for _, item := range s.Query("calendars", ModeAll).Items {
		if strings.HasPrefix(item.ID, "cal:") {
			t.Errorf("a longer word gave the calendar row %s", item.ID)
		}
	}
}

func TestWithoutCalendarsTheBarOffersNoCalendarRows(t *testing.T) {
	s, _ := newService(t)
	for _, item := range s.Query("cal", ModeAll).Items {
		if item.Kind == "calendar" {
			t.Errorf("row %+v without a calendar", item)
		}
	}
}
