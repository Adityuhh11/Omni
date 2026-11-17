# Omni user guide

Everything Omni does, the keys, file search, settings, and how it works. The [README](../README.md) has the short version and the install steps.

- [Commands](#commands)
- [Keys](#keys)
- [File search](#file-search)
- [Settings and where things are](#settings-and-where-things-are)
- [Updating and removing](#updating-and-removing)
- [How it works](#how-it-works)

## Commands

Press **Alt+Shift+Space** (Mac: **⌥⇧Space**) and type.

| | |
|---|---|
| **Apps** | Type a few letters (`vsc`, `chr`), the first letters of each word, or part of a name. Frequently used apps rise; an app that is already open is brought to the front. |
| **Files, by meaning** *(optional)* | Describe what you want: "the diagram with the three boxes", "how I hash passwords", "t distribution table". Finds code, PDF pages, notes, photos (by what they show), videos and, if you choose, sound (audio files and the soundtrack of videos, by what is said or played). Results show where in the file the match is. |
| **Web and quicklinks** | `?` or `web:` searches in the browser you choose. `gh rust lifetimes` searches GitHub; add your own quicklinks. |
| **Clipboard history** *(off until you switch it on)* | `clip:` lists what you copied, newest first. Enter pastes it into the program you came from. Skips copies marked private, password managers and programs you list. |
| **Snippets** | Text you paste often, with `{date}`, `{time}` and `{clipboard}` filled in. `snip:` or a keyword. |
| **Omni AI** *(optional)* | `ask:` opens a small chat window. It answers with a language model on this computer (Liquid AI's LFM3-1.7B, 1 GB, downloaded once), or with a model you already run (LM Studio, Ollama) or a service with your own API key. It can use tools you switch on: your files, notes, clipboard history, browser history, your calendars (with your free time worked out), your [scripts](SCRIPTS.md), and web search. Other AI apps (Claude Desktop, Cursor, VS Code) can use those tools too. |
| **Notes** | `note:` lists your notes; `note: buy milk` makes one. They are plain markdown files. |
| **Focus, timer and pomodoro** | `focus 50` starts a focus period that keeps the sites and apps you list out of the way (minimized on Windows, hidden on a Mac) until it ends; a pomodoro's focus periods do the same. `timer 25`, `timer 1h30 tea`, `pomodoro`: a small timer sits at the edge of the screen. |
| **Calendar** *(read only)* | Add Google, iCloud or Outlook calendars by their subscription address. `cal` lists what is coming today and tomorrow; `join` opens the next meeting's Zoom, Meet or Teams link; the calendar window shows the week. |
| **Calculator and conversions** | `12*7+3`, `15% of 80`, `5 km in miles`, `72 f in c`. Enter copies the answer. |
| **System** | `lock`, `sleep`, `hibernate`, `turn off the display`, `sign out`, `restart`, `shut down`, `task manager`, `recycle bin`, `windows settings`. The three that end the session take a second Enter. |
| **Tags and pins** | `Ctrl+T` tags an app or file, `#work` lists everything tagged. `Ctrl+P` pins up to eight rows that show the moment the bar opens. |

## Keys

On a Mac they are drawn as `⌃ ⌥ ⇧ ⌘`.

| Key | Does |
|---|---|
| `Tab` | Switches between All / Apps / Files / Web |
| `Ctrl+K` | Shows what you can do with the selected row |
| `Ctrl+Enter` | Reveals a file in Explorer or Finder |
| `Alt+Enter` | Searches the web from any row |
| `Ctrl+,` | Opens Settings |
| `Esc` | Closes the bar |

In file search, `code:` `img:` `pdf:` `doc:` `vid:` before a query limit it to one kind of file.

## File search

File search is the part that needs real resources, so it is a choice you make in **Settings > File search**.

1. **Download and turn on.** The page says what it will download and from where before you press the button: a search engine (llama.cpp), its Python libraries
   (pinned, through `uv`) and the embedding model, about 1 GB in all. Every file is checked against a pinned SHA-256.
2. **Choose folders and what to index** (PDF pages, photos, videos, audio) and **how hard it works** (Fast, Balanced, Gentle). Nothing is read until you
   press **Start indexing**, and each option says how long it will take on this computer.
3. **It stays current.** A watcher follows your folders; a new file is searchable a few seconds after it appears.

A graphics card makes it fast (NVIDIA, AMD or Intel Arc, through Vulkan, or Metal on a Mac). Without one it still works, only slower to index: choose
*Words only* for PDFs and *names* for photos. On a laptop with an RTX 4050, text indexes at about 53 chunks a second, and pictures and PDF pages at
about 6 a second (about 0.2 a second on the CPU). Without a graphics card, a 96-file folder took 8 minutes instead of 16 seconds; a search still takes about 40 ms.

Files are read only from folders you add, and secrets (`.env*`, keys and similar) are never opened.

## Settings and where things are

Everything is in Settings (`Ctrl+,`, or the tray icon). The file behind it is `config.toml`; Settings edits it and takes effect without a restart, and a
mistyped key is reported rather than ignored.

```toml
hotkey  = 'Alt+Shift+Space'
roots   = ['C:\Users\you\Documents', 'C:\dev']      # folders file search indexes (one inside another is refused)
exclude = ['C:\dev\big-generated-folder']           # inside a root, left out
off     = ['C:\dev']                                # still indexed, left out of results (the switch in Settings)
browser = 'firefox'                                 # empty = the system default

[[quicklink]]
trigger = 'gh'
title   = 'GitHub'
url     = 'https://github.com/search?q={q}'

[search]
enabled  = true
pace     = 'balanced'    # fast | balanced | gentle
pdf_pages = 'auto'       # auto | all | text
photos   = 'content'     # content | name
videos   = 'name'        # name | content
audio    = 'name'        # name | content: audio files and the soundtrack of videos
```

| What | Windows | macOS |
|---|---|---|
| Settings, notes, chats, launch history | `%APPDATA%\Omni` | `~/Library/Application Support/Omni` |
| Search engine, models, the index, clipboard history | `%LOCALAPPDATA%\Omni` | `~/Library/Application Support/Omni` |
| Log (Settings > About opens it) | `%LOCALAPPDATA%\Omni\logs\omni.log` | `~/Library/Logs/Omni/omni.log` |

API keys are kept in the system credential store (Windows Credential Manager, macOS Keychain), never in `config.toml`.

## Updating and removing

**Windows:** update from Settings > About (it checks the release's SHA-256 first). Uninstall from Settings > Apps; it asks whether to delete what Omni
saved too. Your own files are never touched.

**Mac:** run the install line again to update; [INSTALL-MAC.md](INSTALL-MAC.md) has that, the permission macOS may forget after an update, and how to remove it.

## How it works

```
 hotkey ──▶ [ Go launcher (Wails v3) ] ──▶ Svelte page in a web view (the bar, Settings)
                  │  apps · web · tags · pins · clipboard · snippets: answered in-process, microseconds
                  │
                  │  authenticated HTTP on 127.0.0.1                         (only when file search is on)
                  ▼
        [ Python memory engine ] ──▶ llama.cpp server ──▶ EmbeddingGemma 2 (Q8_0), on the GPU or the CPU
                  │
                  └── index: SQLite (metadata, keyword + path search) + one memory-mapped vector file, searched on the CPU
```

- **Instant things are answered by Go itself**, from memory, so the bar never waits.
- **File search is hybrid.** Files are split into chunks (code by lines, Markdown by section, PDFs by page, photos and video frames as pictures, audio in
  30-second windows). Each chunk becomes a 768-number vector, and a query is turned into one the same way. Results blend vector similarity, keyword (BM25)
  matches, and the query appearing in the file's path. The engine also says when nothing is a *confident* match, using a noise ceiling it measures from its
  own index, so a query with no real answer returns "closest files" rather than a confident wrong one.
- **Only the models use the graphics card.** Storage and search run on the processor.

For contributors: [ARCHITECTURE.md](ARCHITECTURE.md) (modules, data flow, on-disk format, invariants) and [api.md](api.md) (the engine's HTTP contract).
