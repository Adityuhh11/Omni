package browsers

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func num(n int64) string { return strconv.FormatInt(n, 10) }

var now = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func exec(t *testing.T, file string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func fakeFirefox(t *testing.T) source {
	dir := t.TempDir()
	us := func(d time.Duration) int64 { return now.Add(-d).UnixMicro() }
	exec(t, filepath.Join(dir, "p1", "places.sqlite"),
		`CREATE TABLE moz_places (id INTEGER PRIMARY KEY, url TEXT, title TEXT, visit_count INTEGER, last_visit_date INTEGER)`,
		`CREATE TABLE moz_bookmarks (id INTEGER PRIMARY KEY, type INTEGER, fk INTEGER)`,
		`INSERT INTO moz_places VALUES (1, 'https://go.dev/doc/effective_go?token=secret#x', 'Effective Go', 3, `+num(us(time.Hour))+`)`,
		`INSERT INTO moz_places VALUES (2, 'https://old.example/recipe', 'Old sourdough recipe', 1, `+num(us(40*24*time.Hour))+`)`,
		`INSERT INTO moz_places VALUES (3, 'https://never.example/visited', 'Kept for later', 0, NULL)`,
		`INSERT INTO moz_places VALUES (4, 'place:type=6', 'Smart folder', 0, `+num(us(time.Minute))+`)`,
		`INSERT INTO moz_bookmarks VALUES (1, 1, 2), (2, 1, 3)`)
	return source{"Zen", dir, firefox}
}

func TestFirefoxFindsByWordsNewestFirstSkipsPlaceURLsAndKeepsBookmarksPastTheDateLimit(t *testing.T) {
	s := fakeFirefox(t)
	got, err := search([]source{s}, "", 0, 10)
	if err != nil || len(got) != 3 {
		t.Fatalf("all: %+v, %v", got, err)
	}
	if got[0].Title != "Effective Go" || got[0].Browser != "Zen" || got[0].Bookmarked || got[0].Visits != 3 {
		t.Fatalf("newest = %+v", got[0])
	}
	if !got[1].Bookmarked || got[1].Title != "Old sourdough recipe" || !got[2].At.IsZero() {
		t.Fatalf("rest = %+v", got[1:])
	}
	got, _ = search([]source{s}, "SOURDOUGH recipe", 0, 10)
	if len(got) != 1 || got[0].URL != "https://old.example/recipe" {
		t.Fatalf("words = %+v", got)
	}
	got, _ = search([]source{s}, "", 7, 10) // a week: the old visit is out, but it and the unvisited page are bookmarks
	if len(got) != 3 {
		t.Fatalf("week = %+v", got)
	}
	got, _ = search([]source{s}, "effective", 7, 10)
	if len(got) != 1 {
		t.Fatalf("week, effective = %+v", got)
	}
	if got, _ = search([]source{s}, "50%_", 0, 10); len(got) != 0 { // % and _ are not wildcards
		t.Fatalf("wildcards = %+v", got)
	}
}

func TestChromiumHistoryAndBookmarksFileAreMergedAndDated(t *testing.T) {
	dir := t.TempDir()
	at := now.Add(-2*time.Hour).UnixMicro() + chromiumEpoch
	exec(t, filepath.Join(dir, "Default", "History"),
		`CREATE TABLE urls (id INTEGER PRIMARY KEY, url TEXT, title TEXT, visit_count INTEGER, last_visit_time INTEGER)`,
		`INSERT INTO urls VALUES (1, 'https://svelte.dev/docs', 'Svelte docs', 5, `+num(at)+`)`)
	if err := os.WriteFile(filepath.Join(dir, "Default", "Bookmarks"), []byte(`{"roots":{"bookmark_bar":{"type":"folder","children":[
		{"type":"url","name":"Svelte tutorial","url":"https://svelte.dev/tutorial"},
		{"type":"url","name":"Elsewhere","url":"https://example.com"}]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := search([]source{{"Chrome", dir, chromium}}, "svelte", 3, 10)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got[0].Title != "Svelte docs" || !got[0].At.Equal(now.Add(-2*time.Hour)) || !got[1].Bookmarked {
		t.Fatalf("got %+v", got)
	}
}

func TestAnUnreadableBrowserDoesNotStopTheOthersAndNoneAtAllIsAnError(t *testing.T) {
	bad := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bad, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "p", "places.sqlite"), []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	badSrc := source{"Broken", bad, firefox}
	if _, err := search([]source{badSrc}, "", 0, 5); err == nil {
		t.Fatal("no error when the only browser is unreadable")
	}
	if got, err := search([]source{badSrc, fakeFirefox(t)}, "go", 0, 5); err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got, err := search([]source{{"None", filepath.Join(bad, "missing"), firefox}}, "", 0, 5); err != nil || len(got) != 0 {
		t.Fatalf("a browser that is not installed: %+v, %v", got, err)
	}
}

func TestSamePageInTwoProfilesIsOnceWithTheLatestVisit(t *testing.T) {
	a := Page{URL: "https://x.test", At: now.Add(-time.Hour), Visits: 1, Browser: "Zen"}
	b := Page{URL: "https://x.test", At: now, Visits: 2, Bookmarked: true, Browser: "Chrome"}
	got := newestFirst([]Page{a, b}, 5)
	if len(got) != 1 || got[0].Browser != "Chrome" || got[0].Visits != 3 || !got[0].Bookmarked {
		t.Fatalf("got %+v", got)
	}
}

func TestPlainDropsTheQueryTheFragmentAndTheLogin(t *testing.T) {
	if got := Plain("https://user:pw@example.com/a/b?token=1#top"); got != "https://example.com/a/b" {
		t.Fatalf("got %q", got)
	}
}
