package calendar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omni/internal/config"
)

// feed is a calendar with what real feeds contain: a zone by name, UTC, a whole day, a weekly meeting with a day off, one
// occurrence moved and one cancelled, a folded line, escapes, an alarm with its own description, and a meeting link.
const feed = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
	"BEGIN:VEVENT\r\nUID:standup\r\nSUMMARY:Standup\r\nDTSTART;TZID=Europe/Berlin:20261102T093000\r\nDTEND;TZID=Europe/Berlin:20261102T094500\r\n" +
	"RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=9\r\nEXDATE;TZID=Europe/Berlin:20261104T093000\r\n" +
	"DESCRIPTION:Join: https://acme.zoom.us/j/123456?pwd=abc\\nAgenda\\, notes\r\n" +
	"BEGIN:VALARM\r\nACTION:DISPLAY\r\nDESCRIPTION:Reminder\r\nTRIGGER:-PT10M\r\nEND:VALARM\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:standup\r\nRECURRENCE-ID;TZID=Europe/Berlin:20261106T093000\r\nSUMMARY:Standup (moved)\r\n" +
	"DTSTART;TZID=Europe/Berlin:20261106T110000\r\nDTEND;TZID=Europe/Berlin:20261106T111500\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:standup\r\nRECURRENCE-ID;TZID=Europe/Berlin:20261109T093000\r\nSTATUS:CANCELLED\r\n" +
	"DTSTART;TZID=Europe/Berlin:20261109T093000\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:launch\r\nSUMMARY:Launch review with a title long enough\r\n  to be folded\r\nDTSTART:20261103T150000Z\r\nDURATION:PT1H30M\r\n" +
	"LOCATION:Room 4\\; second floor\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:holiday\r\nSUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20261105\r\nDTEND;VALUE=DATE:20261106\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:broken\r\nSUMMARY:No start\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func week(t *testing.T) (from, to time.Time) {
	t.Helper()
	from = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	return from, from.AddDate(0, 0, 8) // Monday 2 to Tuesday 10 November
}

func TestAFeedExpandsIntoTheOccurrencesOfTheWeek(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("no zone database")
	}
	from, to := week(t)
	got := expand(parse([]byte(feed), time.UTC), "Work", 0, from, to)
	var lines []string
	for _, e := range got {
		lines = append(lines, e.Start.In(berlin).Format("Mon 02 15:04")+" "+e.Title)
	}
	want := []string{
		"Mon 02 09:30 Standup",
		"Tue 03 16:00 Launch review with a title long enough to be folded",
		// Wednesday 4 is excluded.
		"Thu 05 01:00 Holiday", // midnight UTC, the zone this test reads the feed in
		"Fri 06 11:00 Standup (moved)",
		// Monday 9 is cancelled; Wednesday 11 is after the week.
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("occurrences:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
	standup, launch, holiday := got[0], got[1], got[2]
	if standup.Join != "https://acme.zoom.us/j/123456?pwd=abc" || standup.Notes != "Join: https://acme.zoom.us/j/123456?pwd=abc\nAgenda, notes" {
		t.Errorf("standup join %q, notes %q", standup.Join, standup.Notes)
	}
	if standup.End.Sub(standup.Start) != 15*time.Minute || launch.End.Sub(launch.Start) != 90*time.Minute {
		t.Errorf("lengths %v, %v", standup.End.Sub(standup.Start), launch.End.Sub(launch.Start))
	}
	if launch.Location != "Room 4; second floor" || launch.Join != "" {
		t.Errorf("launch location %q, join %q", launch.Location, launch.Join)
	}
	if !holiday.AllDay || holiday.End.Sub(holiday.Start) != 24*time.Hour {
		t.Errorf("holiday = %+v", holiday)
	}
	if standup.ID == got[len(got)-1].ID {
		t.Error("two occurrences share an id")
	}
}

func TestAnEventThatStartedBeforeTheWindowAndIsStillGoingIsListed(t *testing.T) {
	from := time.Date(2026, 11, 3, 15, 30, 0, 0, time.UTC) // half way through the launch review
	got := expand(parse([]byte(feed), time.UTC), "Work", 0, from, from.Add(time.Hour))
	if len(got) != 1 || !strings.HasPrefix(got[0].Title, "Launch review") {
		t.Errorf("got %+v", got)
	}
}

func TestDurationsAndWebcal(t *testing.T) {
	for in, want := range map[string]time.Duration{"PT1H30M": 90 * time.Minute, "P1D": 24 * time.Hour, "P1W": 7 * 24 * time.Hour, "PT45S": 45 * time.Second, "P1DT2H": 26 * time.Hour} {
		if got, ok := parseDuration(in); !ok || got != want {
			t.Errorf("%s = %v, %v", in, got, ok)
		}
	}
	if _, ok := parseDuration("P1X"); ok {
		t.Error("nonsense read as a duration")
	}
	if got := WebAddress("  WEBCAL://p01-calendars.icloud.com/published/2/abc "); got != "https://p01-calendars.icloud.com/published/2/abc" {
		t.Errorf("webcal = %q", got)
	}
}

func TestSyncFetchesKeepsACopyAndHidesTheAddressInErrors(t *testing.T) {
	served := feed
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/secret-token/basic.ics" {
			_, _ = w.Write([]byte(served))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	dir := t.TempDir()
	feeds := []config.CalendarFeed{{ID: "work", Name: "Work"}, {ID: "gone", Name: "Gone"}}
	addresses := map[string]string{"work": srv.URL + "/secret-token/basic.ics", "gone": srv.URL + "/secret-token/other.ics"}
	var logged []string
	c := New(dir, func() []config.CalendarFeed { return feeds }, func(id string) (string, error) { return addresses[id], nil },
		func(f string, a ...any) { logged = append(logged, f) })

	c.Sync(context.Background())
	from, to := week(t)
	if n := len(c.Between(from, to)); n != 4 {
		t.Fatalf("%d events after the sync", n)
	}
	st := c.Statuses()
	if st[0].Error != "" || st[0].Synced.IsZero() || st[0].Events != 5 || !strings.Contains(st[1].Error, "404") {
		t.Errorf("statuses %+v", st)
	}
	if _, err := os.Stat(filepath.Join(dir, "work.ics")); err != nil {
		t.Errorf("no copy kept: %v", err)
	}

	// The server goes away: the events stay, and the reason given does not repeat the secret address.
	srv.Close()
	c.Sync(context.Background())
	st = c.Statuses()
	if len(c.Between(from, to)) != 4 || st[0].Error == "" || strings.Contains(st[0].Error, "secret-token") {
		t.Errorf("after the server went: %d events, status %+v", len(c.Between(from, to)), st[0])
	}

	// A new start reads the copy before any fetch; a feed removed from the settings is forgotten with its copy.
	again := New(dir, func() []config.CalendarFeed { return feeds[:1] }, func(string) (string, error) { return "http://127.0.0.1:1/x", nil }, t.Logf)
	again.load()
	if n := len(again.Between(from, to)); n != 4 {
		t.Errorf("%d events from the copy", n)
	}
	again.Sync(context.Background())
	if _, err := os.Stat(filepath.Join(dir, "gone.ics")); !os.IsNotExist(err) || len(again.Statuses()) != 1 {
		t.Errorf("a removed feed is still held: %v, %d statuses", err, len(again.Statuses()))
	}
}

func TestAPageThatIsNotACalendarIsRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>sign in</html>")) }))
	defer srv.Close()
	c := New(t.TempDir(), func() []config.CalendarFeed { return []config.CalendarFeed{{ID: "a", Name: "A"}} },
		func(string) (string, error) { return srv.URL, nil }, t.Logf)
	c.Sync(context.Background())
	if st := c.Statuses()[0]; !strings.Contains(st.Error, "does not give a calendar") {
		t.Errorf("status %+v", st)
	}
}

// One invite in two calendars (same uid, same start) is two events with two ids: the page keys its lists by id.
func TestTheSameInviteInTwoCalendarsHasTwoIDs(t *testing.T) {
	evs := parse([]byte(feed), time.UTC)
	from, to := week(t)
	a, b := expand(evs, "Work", 0, from, to), expand(evs, "Home", 1, from, to)
	if len(a) == 0 || a[0].ID == b[0].ID {
		t.Fatalf("ids %q and %q", a[0].ID, b[0].ID)
	}
}

// An empty calendar is a calendar that was fetched, not a failed fetch, and a calendar emptied since is shown empty.
func TestAnEmptyFeedIsFetchedAndEmpties(t *testing.T) {
	served := feed
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(served)) }))
	defer srv.Close()
	c := New(t.TempDir(), func() []config.CalendarFeed { return []config.CalendarFeed{{ID: "a", Name: "A"}} },
		func(string) (string, error) { return srv.URL, nil }, t.Logf)
	c.Sync(context.Background())
	served = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"
	c.Sync(context.Background())
	from, to := week(t)
	if st := c.Statuses()[0]; st.Synced.IsZero() || st.Events != 0 || len(c.Between(from, to)) != 0 {
		t.Errorf("after the calendar was emptied: %+v, %d events", st, len(c.Between(from, to)))
	}
}

// An all-day event lasts whole days, also across a change of clocks, and an UNTIL with no zone is in the event's zone.
func TestAllDayRecurrenceAcrossTheClockChangeAndAnUntilWithoutAZone(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no zone database")
	}
	ics := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:w\r\nSUMMARY:Bins\r\nDTSTART;VALUE=DATE:20260301\r\nDTEND;VALUE=DATE:20260302\r\n" +
		"RRULE:FREQ=WEEKLY;UNTIL=20260315\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	got := expand(parse([]byte(ics), ny), "Home", 0, time.Date(2026, 2, 1, 0, 0, 0, 0, ny), time.Date(2026, 4, 1, 0, 0, 0, 0, ny))
	var days []string
	for _, e := range got {
		days = append(days, e.Start.In(ny).Format("01-02 15:04")+"→"+e.End.In(ny).Format("01-02 15:04"))
	}
	want := "03-01 00:00→03-02 00:00 03-08 00:00→03-09 00:00 03-15 00:00→03-16 00:00"
	if strings.Join(days, " ") != want {
		t.Errorf("occurrences %v\nwant %s", days, want)
	}
}

// The reason a fetch failed never holds the address, also when the failure was after a redirect.
func TestAFailureAfterARedirectDoesNotLogTheAddress(t *testing.T) {
	var redirect *httptest.Server
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer dead.Close()
	redirect = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dead.URL+"/published/2/SECRETTOKEN", http.StatusFound)
	}))
	defer redirect.Close()
	var logged []string
	c := New(t.TempDir(), func() []config.CalendarFeed { return []config.CalendarFeed{{ID: "a", Name: "A"}} },
		func(string) (string, error) { return redirect.URL + "/published/2/SECRETTOKEN", nil },
		func(f string, a ...any) { logged = append(logged, fmt.Sprintf(f, a...)) })
	c.Sync(context.Background())
	if st := c.Statuses()[0]; st.Error == "" || strings.Contains(st.Error+strings.Join(logged, " "), "SECRETTOKEN") {
		t.Errorf("status %q, log %v", st.Error, logged)
	}
}

// A feed that stays down is logged once, not every quarter hour, until what goes wrong changes.
func TestAFeedThatStaysDownIsLoggedOnce(t *testing.T) {
	var logged []string
	c := New(t.TempDir(), func() []config.CalendarFeed { return []config.CalendarFeed{{ID: "a", Name: "A"}} },
		func(string) (string, error) { return "http://127.0.0.1:1/x", nil },
		func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) })
	for range 3 {
		c.Sync(context.Background())
	}
	if len(logged) != 1 {
		t.Errorf("logged %d times: %q", len(logged), logged)
	}
}
