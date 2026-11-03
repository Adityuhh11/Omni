// Package browsers reads the history and bookmarks of the web browsers on this computer, read-only, so Omni AI can answer "what
// was that page I read about X". Firefox and Zen keep them in places.sqlite; Chrome,
// Edge and Brave in a History database and a Bookmarks file. A running browser locks its database, so a copy is read.
package browsers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure Go, so the Windows build needs no C compiler
)

// Page is a page the person visited or bookmarked.
type Page struct {
	Title, URL, Browser string
	// At is the last visit; zero for a bookmark never visited.
	At         time.Time
	Visits     int
	Bookmarked bool
}

type kind int

const (
	firefox kind = iota
	chromium
)

// source is one browser's folder of profiles: Firefox keeps <dir>/<profile>/places.sqlite, Chromium <dir>/<profile>/History.
type source struct {
	browser, dir string
	kind         kind
}

// Search returns up to limit pages of every browser found whose title or address contains every word, newest first. days limits
// how far back visits count (0 is no limit; a bookmark is found whenever it was added). One browser that cannot be read does not
// stop the others; the error is returned only when none could be.
func Search(words string, days, limit int) ([]Page, error) {
	return search(sources(), words, days, limit)
}

func search(srcs []source, words string, days, limit int) ([]Page, error) {
	var since time.Time
	if days > 0 {
		since = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	}
	w := strings.Fields(strings.ToLower(words))
	var all []Page
	var errs []error
	read := 0
	for _, s := range srcs {
		profiles, _ := filepath.Glob(filepath.Join(s.dir, "*", map[kind]string{firefox: "places.sqlite", chromium: "History"}[s.kind]))
		for _, db := range profiles {
			pages, err := readProfile(s, db, w, since, limit)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", s.browser, err))
				continue
			}
			read++
			all = append(all, pages...)
		}
	}
	if read == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return newestFirst(all, limit), nil
}

func readProfile(s source, db string, words []string, since time.Time, limit int) (pages []Page, err error) {
	query := firefoxPages
	if s.kind == chromium {
		query = chromiumPages
	}
	err = withCopy(db, func(path string) (e error) { pages, e = query(path, words, since, limit); return e })
	if err == nil && s.kind == chromium {
		if b, berr := chromiumBookmarks(filepath.Join(filepath.Dir(db), "Bookmarks"), words); berr == nil { // no file: no bookmarks
			pages = append(pages, b...)
		}
	}
	for i := range pages {
		pages[i].Browser = s.browser
	}
	return pages, err
}

// withCopy gives do a private copy of the database (and its write-ahead log, where the newest visits still are).
func withCopy(db string, do func(path string) error) error {
	dir, err := os.MkdirTemp("", "omni-browser-")
	if err != nil {
		return fmt.Errorf("make a folder for the copy: %w", err)
	}
	defer os.RemoveAll(dir)
	dst := filepath.Join(dir, "db")
	if err := copyFile(db, dst); err != nil {
		return err
	}
	if err := copyFile(db+"-wal", dst+"-wal"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return do(dst)
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("open %s: %w", from, err)
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("create %s: %w", to, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", from, err)
	}
	return out.Close()
}

// likeAll is "(title LIKE ? OR url LIKE ?) AND ..." for the words, with its arguments.
func likeAll(titleCol, urlCol string, words []string) (string, []any) {
	var clauses []string
	var args []any
	for _, w := range words {
		w = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(w)
		clauses = append(clauses, fmt.Sprintf(`(%s LIKE ? ESCAPE '\' OR %s LIKE ? ESCAPE '\')`, titleCol, urlCol))
		args = append(args, "%"+w+"%", "%"+w+"%")
	}
	if len(clauses) == 0 {
		return "1", nil
	}
	return strings.Join(clauses, " AND "), args
}

func queryPages(path, query string, args []any, scan func(*sql.Rows) (Page, error)) ([]Page, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open the copy: %w", err)
	}
	defer db.Close()
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("read a row: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// firefoxPages: moz_places holds every address with its last visit in microseconds since 1970; a bookmark points at one.
// A bookmark is found whenever it was visited, so the date limit does not apply to it.
func firefoxPages(path string, words []string, since time.Time, limit int) ([]Page, error) {
	where, args := likeAll("p.title", "p.url", words)
	args = append(args, since.UnixMicro(), limit)
	return queryPages(path, `SELECT p.url, COALESCE(p.title, ''), COALESCE(p.last_visit_date, 0), p.visit_count,
		EXISTS(SELECT 1 FROM moz_bookmarks b WHERE b.fk = p.id AND b.type = 1) AS bookmarked
		FROM moz_places p WHERE p.url LIKE 'http%' AND `+where+` AND (p.last_visit_date >= ? OR bookmarked)
		ORDER BY COALESCE(p.last_visit_date, 0) DESC LIMIT ?`, args,
		func(r *sql.Rows) (Page, error) {
			var p Page
			var at int64
			err := r.Scan(&p.URL, &p.Title, &at, &p.Visits, &p.Bookmarked)
			if at > 0 {
				p.At = time.UnixMicro(at)
			}
			return p, err
		})
}

// chromiumEpoch is 1601-01-01 in microseconds before 1970, the zero of Chromium's timestamps.
const chromiumEpoch = 11644473600000000

func chromiumPages(path string, words []string, since time.Time, limit int) ([]Page, error) {
	where, args := likeAll("title", "url", words)
	args = append(args, since.UnixMicro()+chromiumEpoch, limit)
	return queryPages(path, `SELECT url, COALESCE(title, ''), last_visit_time, visit_count FROM urls
		WHERE url LIKE 'http%' AND `+where+` AND last_visit_time >= ? ORDER BY last_visit_time DESC LIMIT ?`, args,
		func(r *sql.Rows) (Page, error) {
			var p Page
			var at int64
			err := r.Scan(&p.URL, &p.Title, &at, &p.Visits)
			if at > 0 {
				p.At = time.UnixMicro(at - chromiumEpoch)
			}
			return p, err
		})
}

// chromiumBookmarks walks the Bookmarks file's folders for the bookmarks that contain every word.
func chromiumBookmarks(file string, words []string) ([]Page, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	type node struct {
		Name, URL, Type string
		Children        []node
	}
	var doc struct{ Roots map[string]node }
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	var out []Page
	var walk func(n node)
	walk = func(n node) {
		if n.Type == "url" && strings.HasPrefix(n.URL, "http") && containsAll(strings.ToLower(n.Name+" "+n.URL), words) {
			out = append(out, Page{Title: n.Name, URL: n.URL, Bookmarked: true})
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, root := range doc.Roots {
		walk(root)
	}
	return out, nil
}

func containsAll(text string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

// newestFirst merges the same address seen in several profiles (keeping the latest visit and any bookmark) and cuts to limit.
func newestFirst(pages []Page, limit int) []Page {
	byURL := map[string]Page{}
	for _, p := range pages {
		if old, ok := byURL[p.URL]; ok {
			p.Bookmarked = p.Bookmarked || old.Bookmarked
			p.Visits += old.Visits
			if old.At.After(p.At) {
				p.At, p.Browser = old.At, old.Browser
			}
		}
		byURL[p.URL] = p
	}
	out := make([]Page, 0, len(byURL))
	for _, p := range byURL {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		return out[i].URL < out[j].URL
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Plain shortens an address to what names the page: no query or fragment, which is where tokens and session ids live.
func Plain(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.RawQuery, u.Fragment, u.User = "", "", nil
	return u.String()
}
