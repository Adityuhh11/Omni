package aitools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"omni/internal/chat"
)

const (
	clipResults   = 10
	clipChars     = 500
	maxDays       = 14
	recentFiles   = 25
	recentOpened  = 15
	recentCopies  = 10
	activityStamp = "Mon 2 Jan 15:04"
)

func (d Deps) clipboardTool() chat.Local {
	return chat.Local{
		Group: "clipboard", GroupTitle: "Clipboard history", Busy: "Searching what you copied",
		Def: chat.ObjectTool("search_clipboard", "Search what the person copied recently: the copies that contain every word of the query, "+
			"newest first (the newest when it is empty).", map[string]string{"query": "words the copy contains, or empty"}),
		Prompt: "You can look through what the person copied with search_clipboard.",
		Run: func(_ context.Context, args map[string]string) (string, []chat.Source, error) {
			if !d.Clips.Enabled() {
				return "The clipboard history is off, so there is nothing to search. It can be switched on in Settings, Clipboard.", nil, nil
			}
			found := d.Clips.Search(args["query"], clipResults)
			if len(found) == 0 {
				return "Nothing copied matches “" + args["query"] + "”.", nil, nil
			}
			var b strings.Builder
			for _, e := range found {
				writeCopy(&b, e.At, e.Source, e.Text)
			}
			return b.String(), nil, nil
		},
	}
}

func writeCopy(b *strings.Builder, at time.Time, source, text string) {
	b.WriteString("- " + at.Local().Format(activityStamp))
	if source != "" {
		b.WriteString(", from " + source)
	}
	b.WriteString(": " + strconv.Quote(cut(text, clipChars)) + "\n")
}

func (d Deps) activityTool() chat.Local {
	return chat.Local{
		Group: "activity", GroupTitle: "Recent activity", Busy: "Looking at what you did",
		Def: chat.ObjectTool("recent_activity", "What the person did on this computer lately: files that changed in their indexed folders, "+
			"apps they opened from Omni, and what they copied, newest first.", map[string]string{"days": "how many days back, 1 to 14 (1 is the last 24 hours)"}),
		Prompt: "You can see what the person worked on lately with recent_activity: use it for questions like “what was I doing yesterday”.",
		Run:    d.activity,
	}
}

func (d Deps) activity(ctx context.Context, args map[string]string) (string, []chat.Source, error) {
	days, err := strconv.Atoi(strings.Trim(args["days"], `" `))
	if err != nil || days < 1 {
		days = 1
	}
	days = min(days, maxDays)
	since := d.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var b strings.Builder
	fmt.Fprintf(&b, "Since %s (the last %d day(s)), newest first. Times are local.\n", since.Local().Format(activityStamp), days)

	if d.Files != nil {
		b.WriteString("\nFiles that changed:\n")
		files, err := d.Files.Recent(ctx, since, recentFiles)
		switch {
		case err != nil:
			b.WriteString("(not available: " + err.Error() + ")\n")
		case len(files) == 0:
			b.WriteString("(none)\n")
		}
		for _, f := range files {
			b.WriteString("- " + f.Modified.Local().Format(activityStamp) + "  " + f.Path + "\n")
		}
	}
	if d.Launches != nil {
		b.WriteString("\nOpened from Omni:\n")
		opened := d.Launches(since)
		if len(opened) == 0 {
			b.WriteString("(none)\n")
		}
		for _, l := range opened[:min(len(opened), recentOpened)] {
			b.WriteString("- " + l.At.Local().Format(activityStamp) + "  " + l.Name + "\n")
		}
	}
	if d.Clips != nil && d.Clips.Enabled() {
		b.WriteString("\nCopied:\n")
		n := 0
		for _, e := range d.Clips.Search("", recentCopies) {
			if e.At.After(since) {
				writeCopy(&b, e.At, e.Source, e.Text)
				n++
			}
		}
		if n == 0 {
			b.WriteString("(none)\n")
		}
	}
	return b.String(), nil, nil
}
