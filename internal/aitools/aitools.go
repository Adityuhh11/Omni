// Package aitools turns what Omni already keeps (the file index, the clipboard history, the notes, what was opened from the
// bar, the calendars) and the person's scripts into tools Omni AI may use. Every group is off until the
// person switches it on; the chat package offers and runs them.
package aitools

import (
	"context"
	"time"

	"omni/internal/chat"
	"omni/internal/clipboard"
	"omni/internal/notes"
	"omni/internal/scripts"
	"omni/internal/search"
)

// FileSearch is the file index as the tools use it; *search.Service satisfies it.
type FileSearch interface {
	Lookup(ctx context.Context, query string, files, perFile int) search.Results
	Recent(ctx context.Context, since time.Time, limit int) ([]search.Changed, error)
}

// Clips is the clipboard history; *clipboard.Manager satisfies it.
type Clips interface {
	Enabled() bool
	Search(words string, limit int) []clipboard.Entry
}

// Notes is the notes folder; *notes.Store satisfies it.
type Notes interface {
	Dir() string
	Search(words string, limit int) ([]notes.Note, error)
	Get(id string) (notes.Note, error)
	Save(id, text string) (notes.Note, error)
}

// Scripts is the scripts folder; *scripts.Folder satisfies it.
type Scripts interface {
	List() []scripts.Script
}

// Launch is something opened from the bar.
type Launch struct {
	Name string
	At   time.Time
}

// Deps are the parts of Omni the tools reach. A nil part leaves its tools out.
type Deps struct {
	Files   FileSearch
	Clips   Clips
	Notes   Notes
	Scripts Scripts
	// Calendar is the calendars; nil leaves the calendar tool out.
	Calendar Calendar
	// Browser is browsers.Search; nil leaves the browser tool out.
	Browser Browser
	// Launches lists what was opened from the bar after since, newest first, by name.
	Launches func(since time.Time) []Launch
	// RunScript runs a script (scripts.Run); tests replace it.
	RunScript func(ctx context.Context, s scripts.Script, input string) (string, error)
	Now       func() time.Time
}

// Tools are the tools there are now. Scripts are listed again on each call, so a script added to the folder is offered at once.
func (d Deps) Tools() []chat.Local {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.RunScript == nil {
		d.RunScript = scripts.Run
	}
	var out []chat.Local
	if d.Files != nil {
		out = append(out, d.filesTool())
	}
	if d.Clips != nil {
		out = append(out, d.clipboardTool())
	}
	if d.Notes != nil {
		out = append(out, d.notesTools()...)
	}
	if d.Files != nil || d.Launches != nil || d.Clips != nil {
		out = append(out, d.activityTool())
	}
	if d.Browser != nil {
		out = append(out, d.browserTool())
	}
	if d.Calendar != nil {
		out = append(out, d.calendarTool())
	}
	if d.Scripts != nil {
		out = append(out, d.scriptTools()...)
	}
	return withAppDescriptions(out)
}

// cut shortens text to at most n bytes, on a whole character, marking that it was cut.
func cut(text string, n int) string {
	if len(text) <= n {
		return text
	}
	for n > 0 && text[n]&0xC0 == 0x80 { // not in the middle of a character
		n--
	}
	return text[:n] + "…"
}
