// Package search turns what the user types into the rows the search bar shows: it asks the memory
// engine for files, places each one on disk, and describes where in the file it matched.
//
// The service is bound to the frontend. Every request carries a token that grows with each
// keystroke; an answer that arrives after a newer request was made is marked stale so the page
// never replaces fresh results with old ones, and the older request is cancelled in the engine
// client.
package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"omni/internal/sidecar"
)

const (
	maxFiles       = 10
	matchesPerFile = 3
	// spareFiles are asked for on top of maxFiles: files deleted since the last sync are left out of
	// the answer, and the list should still be full.
	spareFiles    = 5
	searchTimeout = 10 * time.Second
)

// States of an answer.
const (
	StateResults     = "results"     // Rows holds the answer (possibly none)
	StateWarming     = "warming"     // the engine is starting or loading its model
	StateUnavailable = "unavailable" // the engine cannot run; Message says why
	StateOff         = "off"         // file search is switched off; there is nothing to wait for
	StateError       = "error"       // this one search failed; Message says why
)

// Engine is the part of the memory engine client that search needs.
type Engine interface {
	SearchFiles(ctx context.Context, req sidecar.SearchFilesRequest) (sidecar.SearchFilesResponse, error)
	RecentFiles(ctx context.Context, sinceNS int64, limit int) ([]sidecar.RecentFile, error)
}

// Changed is an indexed file and when it last changed.
type Changed struct {
	Path     string
	Modified time.Time
}

// Recent lists the indexed files changed after since, newest first, at most limit; files of folders no longer configured and
// files deleted since the last sync are left out.
func (s *Service) Recent(ctx context.Context, since time.Time, limit int) ([]Changed, error) {
	if st := s.status(); st.State != sidecar.StateReady {
		return nil, errors.New("file search is not running")
	}
	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()
	var sinceNS int64
	if !since.IsZero() {
		sinceNS = since.UnixNano()
	}
	files, err := s.engine.RecentFiles(ctx, sinceNS, limit+spareFiles)
	var apiErr *sidecar.APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
		return nil, errors.New("file search needs its update first: press Update in Settings, File search")
	}
	if err != nil {
		return nil, fmt.Errorf("list recent files: %w", err)
	}
	roots := s.roots()
	var out []Changed
	listed := map[string]bool{} // nested searched folders index a file twice
	for _, f := range files {
		root, ok := roots[f.SourceID]
		path := filepath.Join(root, filepath.FromSlash(f.URI))
		if ok && !listed[path] && s.exists(path) && len(out) < limit {
			listed[path] = true
			out = append(out, Changed{Path: path, Modified: time.Unix(0, f.MtimeNS)})
		}
	}
	return out, nil
}

// Match is one place in a file that matched.
type Match struct {
	// Location is "lines 12-52", "page 3" or empty (an image).
	Location string
	Snippet  string
	// Line is where the match starts in a code or text file (0 if unknown), for opening an editor there.
	Line int
	// Text is the whole matching piece, for Lookup only.
	Text string
	// Timed says the match is a moment in a sound or a video, Seconds from its start: a player can open it there.
	Timed   bool
	Seconds float64
}

// Row is one file in the result list.
type Row struct {
	Path   string // absolute path on disk
	Name   string
	Folder string
	// Kind is what the file is, for the icon and for how it is opened: code, text, pdf or image.
	Kind string
	// Line is the first line of the best match in a code or text file (0 if unknown), so an editor
	// can open the file there.
	Line int
	// Timed and Seconds are those of the best match: the moment a player opens the file at.
	Timed   bool
	Seconds float64
	Score   float64
	Matches []Match // best first; at most three
}

// Results is the answer to one Search call.
type Results struct {
	Token int
	State string
	// Message explains the Warming, Unavailable and Error states.
	Message string
	// Stale means a newer search was made meanwhile: ignore this answer.
	Stale bool
	// Confident is false when the best row is probably noise: show "no match".
	Confident bool
	// Scope names the kind of file searched when the query began with "code:", "img:" and the like.
	Scope string
	Rows  []Row
}

// Service is bound to the frontend; its exported methods become TypeScript functions.
type Service struct {
	engine Engine
	roots  func() map[string]string // source id -> folder on disk, as configured now
	status func() sidecar.Status
	exists func(path string) bool // false for a file deleted since the last sync; tests replace it

	mu     sync.Mutex
	latest int
	cancel context.CancelFunc
}

// New returns a search service. roots maps each source id to the folder it was indexed from (asked
// on every search, so folders added or removed in the settings count at once); status reports
// whether the engine can answer.
func New(engine Engine, roots func() map[string]string, status func() sidecar.Status) *Service {
	return &Service{engine: engine, roots: roots, status: status, exists: fileExists}
}

// Search finds files for query. token identifies the request; use a larger one for each new one.
func (s *Service) Search(query string, token int) Results {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel() // the previous request is no longer wanted
	}
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	s.cancel, s.latest = cancel, token
	s.mu.Unlock()
	defer cancel()

	out := s.find(ctx, query, sidecar.SearchFilesRequest{K: maxFiles + spareFiles, PerFile: matchesPerFile})
	out.Token = token
	if s.isStale(token) || errors.Is(ctx.Err(), context.Canceled) {
		return Results{Token: token, Stale: true}
	}
	return out
}

// Lookup is a search for Omni AI: the best few files with their best matches' whole text. It runs beside the bar's searches and
// cancels none of them.
func (s *Service) Lookup(ctx context.Context, query string, files, perFile int) Results {
	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()
	return s.find(ctx, query, sidecar.SearchFilesRequest{K: files + spareFiles, PerFile: perFile, FullText: true})
}

// find runs req for query (a scope word before it narrows the kinds) and places the files on disk, at most req.K - spareFiles.
func (s *Service) find(ctx context.Context, query string, req sidecar.SearchFilesRequest) Results {
	out := Results{State: StateResults}
	query = strings.TrimSpace(query)
	if query == "" {
		return out
	}
	var filters *sidecar.Filters
	if sc, rest, scoped := parseScope(query); scoped {
		out.Scope, query, filters = sc.label, rest, &sidecar.Filters{Kinds: sc.kinds}
		if query == "" {
			out.Message = "Searching " + strings.ToLower(sc.label) + ". Type what to look for."
			return out
		}
	}
	switch st := s.status(); st.State {
	case sidecar.StateReady:
	case sidecar.StateOff:
		return Results{State: StateOff, Message: "File search is off. Turn it on in Settings."}
	case sidecar.StateFailed:
		return Results{State: StateUnavailable, Message: st.Message}
	default:
		return Results{State: StateWarming, Message: "Search is starting up..."}
	}

	limit := req.K - spareFiles
	req.Query, req.Filters = query, filters
	resp, err := s.engine.SearchFiles(ctx, req)
	var apiErr *sidecar.APIError
	if req.FullText && errors.As(err, &apiErr) && apiErr.Status == http.StatusUnprocessableEntity {
		req.FullText = false // an engine from before full_text, not yet updated: snippets are better than no answer
		resp, err = s.engine.SearchFiles(ctx, req)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return out
	}
	if err != nil {
		if sidecar.NotReady(err) {
			return Results{State: StateWarming, Message: "Search is starting up..."}
		}
		log.Printf("search %q failed: %v", query, err)
		return Results{State: StateError, Message: sidecar.Explain(err)}
	}
	out.Confident = resp.Confident
	listed := map[string]bool{} // a folder inside another searched folder is indexed under both; the engine's best-first order keeps the better copy
	for _, file := range resp.Files {
		if row, ok := s.row(file); ok && !listed[row.Path] && s.exists(row.Path) {
			listed[row.Path] = true
			out.Rows = append(out.Rows, row)
		}
	}
	if len(out.Rows) > limit {
		out.Rows = out.Rows[:limit]
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Service) isStale(token int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return token != s.latest
}

// row places a file found by the engine on disk. A file of a source that is no longer
// configured has no known folder and is left out.
func (s *Service) row(file sidecar.File) (Row, bool) {
	root, ok := s.roots()[file.SourceID]
	if !ok {
		return Row{}, false
	}
	path := filepath.Join(root, filepath.FromSlash(file.URI))
	row := Row{
		Path:   path,
		Name:   filepath.Base(path),
		Folder: filepath.Dir(path),
		Kind:   displayKind(file.Kind),
		Score:  file.Score,
	}
	for _, chunk := range file.Chunks {
		match := Match{Location: location(chunk.Locator), Snippet: chunk.Snippet, Text: chunk.Text, Line: firstLine(chunk.Locator)}
		if t := parseLocator(chunk.Locator).T; t != nil {
			match.Timed, match.Seconds = true, *t
		}
		if len(row.Matches) == 0 {
			row.Timed, row.Seconds = match.Timed, match.Seconds
		}
		row.Matches = append(row.Matches, match)
		if row.Line == 0 {
			row.Line = match.Line
		}
	}
	return row, true
}

// displayKind folds the engine's chunk kinds into the six things the UI tells apart.
func displayKind(kind string) string {
	switch kind {
	case "pdf_text", "pdf_page":
		return "pdf"
	case "image":
		return "image"
	case "video", "video_frame":
		return "video"
	case "audio", "audio_window":
		return "audio"
	case "code":
		return "code"
	case "file":
		return "file"
	default: // markdown and text
		return "text"
	}
}

type locatorJSON struct {
	Lines []int    `json:"lines"`
	Page  int      `json:"page"`
	T     *float64 `json:"t"` // seconds into a video
}

func parseLocator(locator string) locatorJSON {
	var parsed locatorJSON
	_ = json.Unmarshal([]byte(locator), &parsed) // an unreadable locator just shows no location
	return parsed
}

// location describes where in a file a chunk is: "lines 12-52", "page 3" or, in a video, "at 1:23".
func location(locator string) string {
	parsed := parseLocator(locator)
	switch {
	case len(parsed.Lines) == 2 && parsed.Lines[0] == parsed.Lines[1]:
		return fmt.Sprintf("line %d", parsed.Lines[0])
	case len(parsed.Lines) == 2:
		return fmt.Sprintf("lines %d-%d", parsed.Lines[0], parsed.Lines[1])
	case parsed.Page > 0:
		return fmt.Sprintf("page %d", parsed.Page)
	case parsed.T != nil:
		return "at " + clock(*parsed.T)
	}
	return ""
}

// clock writes seconds as a video player does: 0:07, 1:23, 1:02:03.
func clock(seconds float64) string {
	s := int(seconds)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func firstLine(locator string) int {
	if lines := parseLocator(locator).Lines; len(lines) == 2 {
		return lines[0]
	}
	return 0
}
