# 0013: Browser history and bookmarks are a read-only tool group

Date: 2026-11-02. Status: accepted. Extends 0011 and 0012.

## Decision

A tool group `browser` with one tool, `search_browser(query, days)`: the pages the person visited or bookmarked in Zen, Firefox,
Chrome, Edge and Brave, newest first, up to 10, each with browser, title, address and time. It is off until switched on, in Omni AI
(`[chat] tools`) and for other AI apps (`[chat] mcp_tools`) separately, like every group.

- **Read a copy.** A running browser locks its database. Omni copies `places.sqlite` (Firefox, Zen) or `History` (Chromium) and
  the `-wal` file, where the newest visits still are, to a temporary folder, reads the copy with pure-Go SQLite and deletes it.
  Nothing in a browser's folder is opened for writing or changed. Chromium's bookmarks are its `Bookmarks` JSON file.
- **Words in the title or address.** Not the page's text, which no browser keeps. Every word must match, as in the notes tool.
- **No query or fragment in the address.** They are where tokens, session ids and search terms live, and the page is findable
  without them. Page titles are labelled as information, never instructions, as web text is.
- **Safari is not read.** Its files need Full Disk Access on macOS; revisit if asked.

## Why

- Browsing is where most reading happens; "which page was that" is a question files cannot answer.
- A copy is the only way that works with the browser open, and a person asking this question has it open.

## Alternatives not taken

- A browser extension: a second thing to install, ship and keep in step with each browser's store; the files are already there.
- The engine's Python `sqlite3`: the engine runs only when file search is installed, and this should not need it.
- Indexing page text into the file index: heavier and a privacy decision of its own; later, if it is wanted.
- `database/sql` with cgo SQLite (`mattn/go-sqlite3`): needs a C compiler on the Windows build.

## Consequences

- A new dependency, `modernc.org/sqlite` and its libraries (`THIRD_PARTY_NOTICES.md`). The executable grows from 16.7 to 20.8 MB;
  the packages add 0.54 ms to program start (`eval/2026-11-02-browser-tool.md`).
- A search copies every profile's database: about 0.7 s over four browsers on this computer. It runs only when the tool is called.
- Browsing history is sensitive. Over MCP what an app reads goes wherever that app sends it; the group's switch on the AI apps
  page is the person's consent, and the tool's description says to use only what the question needs.
