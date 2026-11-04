package settingsui

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"omni/internal/calendar"
	"omni/internal/config"
	"omni/internal/settings"
)

func TestACalendarIsAddedWithItsAddressInTheStoreAndRemovedWithIt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:a\r\nSUMMARY:A\r\nDTSTART:20261103T100000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"))
	}))
	defer srv.Close()
	store, err := settings.Open(filepath.Join(t.TempDir(), "Omni", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	addresses := map[string]string{}
	cal := calendar.New(t.TempDir(), func() []config.CalendarFeed { return store.Get().Calendar.Feeds },
		func(id string) (string, error) { return addresses[id], nil }, t.Logf)
	page := NewCalendar(CalendarOptions{
		Store: store, Calendar: cal,
		SetAddress:    func(id, a string) error { addresses[id] = a; return nil },
		DeleteAddress: func(id string) error { delete(addresses, id); return nil },
	})

	for _, bad := range [][2]string{{"", srv.URL}, {"Work", "not an address"}, {"Work", "ftp://example.com/cal.ics"}} {
		if why := page.Add(bad[0], bad[1]); why == "" {
			t.Errorf("Add(%q, %q) was taken", bad[0], bad[1])
		}
	}
	if why := page.Add(" Work ", srv.URL); why != "" {
		t.Fatal(why)
	}
	feeds := store.Get().Calendar.Feeds
	if len(feeds) != 1 || feeds[0].Name != "Work" || addresses[feeds[0].ID] != srv.URL {
		t.Fatalf("feeds %+v, addresses %v", feeds, addresses)
	}
	if v := page.View(); len(v) != 1 || v[0].Events != 1 || v[0].Synced == "" || v[0].Error != "" {
		t.Errorf("view %+v", v)
	}
	if strings.Contains(store.Get().Calendar.Feeds[0].Name+store.Get().Calendar.Feeds[0].ID, "127.0.0.1") {
		t.Error("the address is in the settings")
	}
	if why := page.Remove(feeds[0].ID); why != "" || len(store.Get().Calendar.Feeds) != 0 || len(addresses) != 0 || len(page.View()) != 0 {
		t.Errorf("after remove: %q, %+v, %v", why, store.Get().Calendar.Feeds, addresses)
	}
}
