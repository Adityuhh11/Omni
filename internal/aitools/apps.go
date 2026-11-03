package aitools

import "omni/internal/chat"

// appTools are the tools as other AI apps see them over MCP. The model Omni AI runs
// is small and gets short descriptions; Claude and its kind can use a full account of what a tool covers and how to ask it.
// Each has the name and the arguments of its short twin.
var appTools = map[string]chat.Tool{
	"search_files": chat.ObjectTool("search_files",
		"Search the person's own files on their computer: documents, PDFs, notes, code, pictures and the other files in the folders "+
			"they have chosen to index in Omni (not the whole disk). It is retrieval for answering questions, matching by meaning and "+
			"by words together, so ask it the way you would ask a person: a question, or a description of what the file says, is "+
			"better than a keyword; exact names, numbers and identifiers also work. Returns up to 4 files, each with its path and its "+
			"2 best passages in full (with the page or place they came from), enough to answer from and to name the file. When "+
			"nothing matches well it says so rather than guessing; then try other wording or a narrower question. To see more of a "+
			"file, search again with words from it.",
		map[string]string{"query": "What to find: a question or a description of the content in plain language, or specific words, names or identifiers."},
		"query"),
	"search_notes": chat.ObjectTool("search_notes",
		"Find notes in the person's Omni notes (the short notes they keep in Omni, not their other documents; use search_files for "+
			"those). Returns the notes that contain every word of the query, newest first, up to 8, each with its id, title and the "+
			"start of its text. It matches words, not meaning, so use a few distinctive words the note would contain. An empty query "+
			"lists the newest notes. Read a whole note with read_note.",
		map[string]string{"query": "Words the note must contain (all of them), or empty to list the newest notes."}),
	"read_note": chat.ObjectTool("read_note",
		"Read one of the person's Omni notes in full, by the id that search_notes returned.",
		map[string]string{"id": "The note's id, as given by search_notes."}, "id"),
	"search_clipboard": chat.ObjectTool("search_clipboard",
		"Search the person's recent clipboard history, which Omni keeps if they have switched it on: what they copied lately, with "+
			"when and from which app. Returns up to 10 copies that contain every word of the query, newest first; an empty query "+
			"returns the newest. It can hold anything they copied, including private text, so use only what the question needs.",
		map[string]string{"query": "Words the copy must contain (all of them), or empty for the newest copies."}),
	"search_browser": chat.ObjectTool("search_browser",
		"Search the web pages the person visited or bookmarked in the browsers on this computer (Zen, Firefox, Chrome, Edge and "+
			"Brave), from their history and bookmarks. Returns up to 10 pages, newest first, each with browser, title, address "+
			"(without the query part) and when it was last visited, and whether it is bookmarked. It matches words in the title "+
			"or address, not the page's text, so use a few distinctive words; an empty query returns the newest pages. Private "+
			"windows are not recorded. Use it for “which page was that”, “what did I read about X last week”; this is the "+
			"person's browsing, so use only what the question needs.",
		map[string]string{"query": "Words the title or address must contain (all of them), or empty for the newest pages.", "days": "Only visits in the last N days (a bookmark counts whenever it was visited). Empty for any time."}),
	"recent_activity": chat.ObjectTool("recent_activity",
		"What the person did on their computer lately, from the last 1 to 14 days, newest first: files that changed in the folders "+
			"Omni indexes, apps they opened with Omni, and what they copied. Use it for questions such as “what was I working on "+
			"yesterday?” or “which files did I change this week?”; then search_files can read the files it names.",
		map[string]string{"days": "How many days back, 1 to 14 (1 is the last 24 hours). Defaults to 1."}),
}

// withAppDescriptions gives each tool that has one its description for other AI apps.
func withAppDescriptions(tools []chat.Local) []chat.Local {
	for i, t := range tools {
		if apps, ok := appTools[t.Def.Function.Name]; ok {
			tools[i].Apps = apps
		}
	}
	return tools
}
