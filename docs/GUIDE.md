# Omni user guide

Everything Omni does, the keys, file search, settings, and how it works. The [README](../README.md) has the short version and the install steps.

- [First start](#first-start)
- [Commands](#commands)
- [Keys](#keys)
- [File search](#file-search)
- [Settings and where things are](#settings-and-where-things-are)
- [Updating and removing](#updating-and-removing)
- [How it works](#how-it-works)

## First start

The first start opens a short setup guide, about a minute, every part of it optional. Open it again from the tray icon ("Set up Omni...") or Settings > About.

1. **Meet Omni.** A bar types examples by itself (apps, web and quicklinks, maths, clipboard, notes, timer, files); the chips under it jump to one.
2. **Your hotkey.** Record another if you like, and press it to see the bar open (**Try it now**). Optionally start Omni when you sign in.
3. **Your look.** Theme and accent, shown on a bar and on the guide itself as you choose.
4. **What you want.** What works already, and the extras to switch on: clipboard history (instant), Omni AI and file search (each a download of about 1 GB, started by a button that says so).
5. **Your files**, only if you turned file search on: the folders to read, and how thorough (chosen for your computer).

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
| `Ctrl+1` to `Ctrl+4` (`⌘1` to `⌘4` on a Mac) | Opens that quick-open app beside the bar, whether or not something is typed |
| `→` `←` | With nothing typed, walk through the quick-open apps; `Enter` opens the one picked, `Esc` goes back to the bar |
| `Esc` | Closes the bar |

### Quick open

Beside the bar are three or four round shortcuts to apps (Settings > General > Quick open). You can set each slot to an app or leave it as
"Suggested for you": then Omni fills it with the apps you usually open from the bar at this time of day, on this kind of day (a weekday or
the weekend). The suggestion is worked out on your computer from the launch times Omni already keeps to rank results (at most 20 per app):
nothing new is recorded and nothing is sent anywhere. "Forget my usage history" empties those launch times. With no history yet the slots show
common apps (Safari, Notes, Mail on a Mac; Edge, File Explorer, Notepad on Windows).

In file search, `code:` `img:` `pdf:` `doc:` `vid:` before a query limit it to one kind of file.

## Hotkeys for anything

**Settings > Hotkeys** gives any app, script, settings page or the timer its own global hotkey. Press **Add hotkey**, search for what to run (the same rows the
bar lists), then press **Set** and the keys. The hotkey does what Enter does on that row. A combination another program already owns is refused with
the reason, and a combination used for another Omni job (the bar, the chat, the notes, the clipboard) cannot be used twice.

```toml
[[shortcut]]
keys   = 'Ctrl+Alt+T'
target = 'app:...'       # the row's id; Settings fills it in
title  = 'Terminal'
```

## File search

File search is the part that needs real resources, so it is a choice you make in **Settings > File search**.

1. **Download and turn on.** The page says what it will download and from where before you press the button: a search engine (llama.cpp), its Python libraries
   (pinned, through `uv`) and the embedding model, about 1 GB in all. Every file is checked against a pinned SHA-256.
2. **Choose folders and what to index** (PDF pages, photos, videos, audio) and **how hard it works** (Fast, Balanced, Gentle). Nothing is read until you
   press **Start indexing**, and each option says how long it will take on this computer.
3. **It stays current.** A watcher follows your folders; a new file is searchable a few seconds after it appears.
4. **When it runs.** By default the engine starts with Omni and holds about 1.1 GB of video memory while Omni runs. Choose **Start the search engine: When I
   search for files** to free that memory until you search for a file; the first search waits while it starts (about 6 s on a laptop RTX 4050) and it stops
   again after 10 idle minutes, never in the middle of indexing.

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

[quick_open]
enabled = true
count   = 3              # 3 or 4
suggest = true           # fill the open slots from your launch times (on this computer only)
apps    = ['app:...']    # slots set by hand, in order; '' leaves a slot to the suggestions

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
