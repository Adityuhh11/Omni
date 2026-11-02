package aitools

import (
	"context"
	"fmt"
	"strings"

	"omni/internal/chat"
	"omni/internal/search"
)

const (
	// lookupFiles and lookupMatches are what one search_files call returns; matchChars is the most of one match (about 300
	// tokens), so 4 files of 2 matches take about 2,500 tokens of an 8k context.
	lookupFiles   = 4
	lookupMatches = 2
	matchChars    = 900
)

func (d Deps) filesTool() chat.Local {
	return chat.Local{
		Group: "files", GroupTitle: "Your files", Busy: "Searching your files",
		Def: chat.ObjectTool("search_files", "Search the person's own files on this computer (documents, notes, PDFs, code) by meaning and by "+
			"words. Returns up to 4 files, each with its path and the parts that matched.",
			map[string]string{"query": "a few specific words, not a sentence"}, "query"),
		Prompt: "You can search the person's own files with search_files: use it when they ask about something they wrote, saved or " +
			"received, and name the file you answer from.",
		Run: d.searchFiles,
	}
}

// searchFiles gives the model the best files' matching parts. A search the index is not sure of returns nothing: a small model
// handed the nearest noise answers from it.
func (d Deps) searchFiles(ctx context.Context, args map[string]string) (string, []chat.Source, error) {
	query := args["query"]
	if query == "" {
		return `Error: "query" is empty.`, nil, nil
	}
	res := d.Files.Lookup(ctx, query, lookupFiles, lookupMatches)
	switch {
	case res.State != search.StateResults:
		return "", nil, fmt.Errorf("file search is not available: %s", res.Message)
	case len(res.Rows) == 0 || !res.Confident:
		return "Nothing in the person's files matched “" + query + "” well. Try other words, or say you found nothing.", nil, nil
	}
	var b strings.Builder
	var sources []chat.Source
	b.WriteString("Files matching “" + query + "”:\n")
	for i, row := range res.Rows {
		fmt.Fprintf(&b, "%d. %s\n   %s\n", i+1, row.Name, row.Path)
		for _, m := range row.Matches {
			text := m.Text
			if text == "" { // a picture, a file found by its name, or an engine from before full text
				text = m.Snippet
			}
			b.WriteString("   ")
			if m.Location != "" {
				b.WriteString("[" + m.Location + "] ")
			}
			b.WriteString(cut(strings.TrimSpace(text), matchChars) + "\n")
		}
		sources = append(sources, chat.Source{Title: row.Name, Path: row.Path})
	}
	return b.String(), sources, nil
}
