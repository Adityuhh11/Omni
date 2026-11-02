package aitools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"omni/internal/chat"
)

const (
	noteResults = 8
	noteChars   = 6000
)

func (d Deps) notesTools() []chat.Local {
	group := chat.Local{Group: "notes", GroupTitle: "Notes",
		Prompt: "You can search, read and add the person's notes with search_notes, read_note and add_note."}
	searchNotes, readNote, addNote := group, group, group
	searchNotes.Busy = "Searching your notes"
	searchNotes.Def = chat.ObjectTool("search_notes", "Find the person's notes that contain every word of the query (all notes, newest first, "+
		"when it is empty). Returns each note's id, title and start.", map[string]string{"query": "words the note contains, or empty"})
	searchNotes.Run = d.searchNotes
	readNote.Busy = "Reading a note"
	readNote.Def = chat.ObjectTool("read_note", "Read one note whole, by the id search_notes gave.", map[string]string{"id": "the note's id"}, "id")
	readNote.Run = d.readNote
	addNote.Busy = "Adding a note"
	addNote.Writes = true
	addNote.Def = chat.ObjectTool("add_note", "Save a new note. Its first line is its title.", map[string]string{"text": "the note, in markdown"}, "text")
	addNote.Run = d.addNote
	return []chat.Local{searchNotes, readNote, addNote}
}

func (d Deps) notePath(id string) string { return filepath.Join(d.Notes.Dir(), id+".md") }

func (d Deps) searchNotes(_ context.Context, args map[string]string) (string, []chat.Source, error) {
	found, err := d.Notes.Search(args["query"], noteResults)
	if err != nil {
		return "", nil, err
	}
	if len(found) == 0 {
		return "No note matches “" + args["query"] + "”.", nil, nil
	}
	var b strings.Builder
	for _, n := range found {
		fmt.Fprintf(&b, "- id %s: %s", n.ID, n.Title)
		if n.Preview != "" {
			b.WriteString(" — " + n.Preview)
		}
		b.WriteString("\n")
	}
	return b.String(), nil, nil
}

func (d Deps) readNote(_ context.Context, args map[string]string) (string, []chat.Source, error) {
	n, err := d.Notes.Get(args["id"])
	if err != nil {
		return "There is no note with the id “" + args["id"] + "”. Use an id search_notes gave.", nil, nil
	}
	return cut(n.Text, noteChars), []chat.Source{{Title: n.Title, Path: d.notePath(n.ID)}}, nil
}

func (d Deps) addNote(_ context.Context, args map[string]string) (string, []chat.Source, error) {
	if strings.TrimSpace(args["text"]) == "" {
		return `Error: "text" is empty.`, nil, nil
	}
	n, err := d.Notes.Save("", args["text"])
	if err != nil {
		return "", nil, err
	}
	return "Saved the note “" + n.Title + "”.", []chat.Source{{Title: n.Title, Path: d.notePath(n.ID)}}, nil
}
