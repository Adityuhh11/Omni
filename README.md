<h1 align="center">Omni</h1>

<p align="center">
  A fast, private launcher for Windows and Mac.<br>
  Press a hotkey, type, and open an app, search the web, paste a snippet, or find a file by what it <em>says</em>, not just what it is called.<br>
  Everything runs on your computer.
</p>

<p align="center">
  <a href="https://github.com/Adityuhh11/Omni/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Adityuhh11/Omni/actions/workflows/ci.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg"></a>
  <a href="https://github.com/Adityuhh11/Omni/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Adityuhh11/Omni?display_name=tag"></a>
</p>

<p align="center">
  <a href="#what-it-does">What it does</a> ·
  <a href="#install">Install</a> ·
  <a href="#file-search">File search</a> ·
  <a href="#privacy-and-the-network">Privacy</a> ·
  <a href="#build-from-source">Build</a> ·
  <a href="#contributing">Contributing</a>
</p>

> **Status: 1.x, early.** Windows 11 and macOS 13+ on Apple Silicon. Used daily by its author and a few friends; the Mac build is newer and has had
> less use. Expect rough edges, and please [open an issue](https://github.com/Adityuhh11/Omni/issues/new/choose) when you meet one.
> Linux is planned, not started. See the [roadmap](docs/ROADMAP.md).

|  |  |
|---|---|
| **Instant** | The bar is painted about 33 ms after the hotkey (median) and never waits on a model. |
| **Light** | Idle, no CPU and about 270 MB including its web view. File search is optional and adds a small model and a Python process. |
| **Private** | No telemetry, no account. Your files and your index never leave the machine. The few network calls are [listed below](#privacy-and-the-network). |

## What it does

| | |
|---|---|
| **Apps** | Type a few letters (`vsc`, `chr`), the first letters of each word, or part of a name. Frequently used apps rise; an app that is already open is brought to the front. |
| **Files, by meaning** *(optional)* | Describe what you want: "the diagram with the three boxes", "how I hash passwords", "t distribution table". Finds code, PDF pages, notes, photos (by what they show), videos and, if you choose, sound (audio files and the soundtrack of videos, by what is said or played). Results show where in the file the match is. |
| **Web and quicklinks** | `?` or `web:` searches in the browser you choose. `gh rust lifetimes` searches GitHub; add your own quicklinks. |
| **Clipboard history** *(off until you switch it on)* | `clip:` lists what you copied, newest first. Enter pastes it into the program you came from. Skips copies marked private, password managers and programs you list. |
| **Snippets** | Text you paste often, with `{date}`, `{time}` and `{clipboard}` filled in. `snip:` or a keyword. |
| **Omni AI** *(optional)* | `ask:` opens a small chat window. It answers with a language model on this computer (Liquid AI's LFM3-1.7B, 1 GB, downloaded once), or with a model you already run (LM Studio, Ollama) or a service with your own API key. It can use tools you switch on: your files, notes, clipboard history, browser history, your scripts, and web search. Other AI apps (Claude Desktop, Cursor, VS Code) can use those tools too. |
| **Notes** | `note:` lists your notes; `note: buy milk` makes one. They are plain markdown files. |
| **Timer and pomodoro** | `timer 25`, `timer 1h30 tea`, `pomodoro`: a small timer sits at the edge of the screen. |
| **Calculator and conversions** | `12*7+3`, `15% of 80`, `5 km in miles`, `72 f in c`. Enter copies the answer. |
| **System** | `lock`, `sleep`, `task manager`, `recycle bin`, `windows settings`. Shut down and restart are left out on purpose. |
| **Tags and pins** | `Ctrl+T` tags an app or file, `#work` lists everything tagged. `Ctrl+P` pins up to eight rows that show the moment the bar opens. |

**Keys** (on a Mac they are drawn as `⌃ ⌥ ⇧ ⌘`): `Tab` switches between All / Apps / Files / Web, `Ctrl+K` shows what you can do with the selected row,
`Ctrl+Enter` reveals a file in Explorer or Finder, `Alt+Enter` searches the web from any row, `Ctrl+,` opens Settings, `Esc` closes.
In file search, `code:` `img:` `pdf:` `doc:` `vid:` before a query limit it to one kind of file.

## Install

### Windows 11

1. Download `omni-<version>-windows-x64-setup.exe` from the [latest release](https://github.com/Adityuhh11/Omni/releases/latest) and run it. It installs
   for you alone (no administrator rights) and adds a Start menu entry. Prefer no installer? The `.zip` holds the same `omni.exe`.
2. Windows may say **"Windows protected your PC"** because the program is not code-signed yet (signing is on the [roadmap](docs/ROADMAP.md)). Click
   **More info**, then **Run anyway**. You can check the download against the SHA-256 on the release page, or prove it was built from this repository with
   `gh attestation verify <file> --repo Adityuhh11/Omni`.
3. Omni lives in the notification area (the arrow by the clock). The first start opens a short setup guide.
4. Press **Alt+Shift+Space**. Change the hotkey in Settings > General.

Update from Settings > About (it checks the release's SHA-256 first). Uninstall from Settings > Apps; it asks whether to delete what Omni saved too.
Your own files are never touched.

### macOS 13+ (Apple Silicon)

```sh
curl -fsSL https://github.com/Adityuhh11/Omni/releases/latest/download/install.sh | sh
```

It downloads `Omni.app`, checks it against the checksum GitHub published, and opens it. Press **⌥⇧Space** for the bar. Omni is not signed by Apple, so a
zip downloaded in a browser needs one extra step; [docs/INSTALL-MAC.md](docs/INSTALL-MAC.md) has that, the permissions macOS asks for, and how to update and remove it.

## File search

File search is the part that needs real resources, so it is a choice you make in **Settings > File search**.

1. **Download and turn on.** The page says what it will download and from where before you press the button: a search engine (llama.cpp), its Python libraries
   (pinned, through `uv`) and the embedding model, about 1 GB in all. Every file is checked against a pinned SHA-256.
2. **Choose folders and what to index** (PDF pages, photos, videos, audio) and **how hard it works** (Fast, Balanced, Gentle). Nothing is read until you
   press **Start indexing**, and each option says how long it will take on this computer.
3. **It stays current.** A watcher follows your folders; a new file is searchable a few seconds after it appears.

A graphics card makes it fast (NVIDIA, AMD or Intel Arc, through Vulkan, or Metal on a Mac). Without one it still works, only slower to index: choose
*Words only* for PDFs and *names* for photos.

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

Details: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) (modules, data flow, on-disk format, invariants) and [`docs/api.md`](docs/api.md) (the engine's HTTP contract).

## Performance

Measured on a laptop with an RTX 4050 (6 GB) and a real index of 46,000 chunks.

| | |
|---|---|
| Hotkey to a painted, focused input | **31 to 36 ms** median, under 50 ms p95 |
| App search, per keystroke | **0.2 ms** |
| File search query | **16 ms** median, 21 ms p95 |
| Idle, launcher | **0 % CPU**, about **270 MB** with its web view; file search adds about 450 MB and 1 GB of video memory |
| Indexing speed | text about 53 chunks/s; pictures and PDF pages about 6/s on the GPU, about 0.2/s on the CPU |
| No graphics card | a 96-file folder indexes in 8 minutes instead of 16 seconds; a search still takes about 40 ms |
| Quality (92 hand-written queries) | right file first for 90 %, in the top three for 100 %, no false alarms |

## Privacy and the network

No telemetry. No account. Your files are read only from folders you add, and secrets (`.env*`, keys and similar) are never opened. The program
touches the network in exactly these cases, all started or switched on by you:

| When | Where | What is sent |
|---|---|---|
| You press **Download and turn on** (or **Update**) in File search | `github.com`, `pypi.org`, `files.pythonhosted.org`, `huggingface.co` | ordinary downloads of pinned files |
| You press **Download and turn on** (or **Update**) in Omni AI | `github.com`, `huggingface.co` | ordinary downloads of pinned files (llama.cpp and the model); nothing you ask is ever sent anywhere |
| You open a web search or link | your browser | what you typed after the trigger |
| You switch on **Search the web** in the Omni AI chat's + menu (off by default) | `duckduckgo.com`, or `wikipedia.org` when DuckDuckGo asks for a check; then the pages the model chooses to read | the words the model chooses to search for (it composes them from your question and, after a read, from pages; it is a small model); then ordinary requests for pages that searches returned or that you wrote. Local and private addresses are refused |
| You choose **Another model** in Settings, Omni AI, and use it (off by default) | the address you typed: LM Studio or Ollama on this computer, or a service (OpenAI and others) | your questions and the conversation, and what the tools you switched on found for them. The API key, kept in the system credential store, goes as that service's `Authorization` header. The page says whether the address is on this computer |
| You switch on **Let AI apps on this computer use these tools** in Settings, Omni AI (off by default) | the AI app you connect (Claude Desktop, Cursor, VS Code…) | what that app asks Omni's tools that read (your files, notes, clipboard history, activity) and are switched on; the app may send it to its own servers |
| You press **Check for updates**, or turned on the daily check (off by default) | `api.github.com` | a request for the latest release number, nothing about you |
| You press **Update to X** | `github.com` | a download of that release, checked against its published SHA-256 |

The engine and the Omni AI model server listen on `127.0.0.1` only, behind a random per-run token. Clipboard history is off by default and stored only on
your machine. Omni AI's tools (your files, notes, clipboard, activity, browser history and bookmarks, and your [scripts](docs/SCRIPTS.md)) run on this
computer and are off until you switch them on; with web search also on, the model could put what a tool found into the words it searches for.

## Settings and where things are

Everything is in Settings (`Ctrl+,`, or the tray icon). The file behind it is `config.toml`; Settings edits it and takes effect without a restart, and a
mistyped key is reported rather than ignored.

```toml
hotkey  = 'Alt+Shift+Space'
roots   = ['C:\Users\you\Documents', 'C:\dev']      # folders file search indexes
exclude = ['C:\dev\big-generated-folder']           # inside a root, left out
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

## Build from source

You need [Go](https://go.dev/dl/) 1.27+, [Node.js](https://nodejs.org/) 24 and npm, the [Wails v3 CLI](https://v3alpha.wails.io/)
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29`) and [uv](https://docs.astral.sh/uv/) for the engine's tests. On a Mac, also the Xcode
command line tools.

```sh
git clone https://github.com/Adityuhh11/Omni.git && cd Omni
.\scripts\build.ps1        # Windows: bindings, frontend, then bin\omni.exe
./scripts/build-mac.sh     # macOS:   bindings, frontend, then bin/omni
```

The checks CI runs on every pull request, on a Windows and a Mac machine:

```sh
gofmt -l .                                                    # prints nothing when clean
go vet ./... && go test ./...                                 # launcher
cd frontend && npm test && npm run check                      # page
cd sidecar && uv run pytest && uv run ruff check && uv run pyright   # engine
```

The Go launcher builds on Windows and macOS only; the engine and page tests also run on Linux. Retrieval quality has its own eval:
[`sidecar/eval/README.md`](sidecar/eval/README.md). Releases: [`docs/RELEASING.md`](docs/RELEASING.md).

| Path | What |
|---|---|
| `cmd/omni` | the launcher's `main`: wires everything together |
| `internal/` | the launcher's packages, one job each: the bar, Settings, apps, web, clipboard, file search from the launcher's side, Omni AI, update |
| `frontend/` | the Svelte 5 + TypeScript page (the bar and Settings) |
| `sidecar/` | the Python memory engine (`omni_memory`): chunking, embedding, store, search, eval |
| `docs/` | architecture, the engine's API, roadmap, release and install notes |

## Contributing

Bug reports, ideas and testing on hardware the maintainer does not have are welcome. Open an [issue](https://github.com/Adityuhh11/Omni/issues/new/choose)
first: pull requests are by invitation, once an issue says go. [`CONTRIBUTING.md`](CONTRIBUTING.md) has the whole process (setup, checks, the DCO sign-off,
using AI tools); the code rules, which coding agents read too, are in [`AGENTS.md`](AGENTS.md). A vulnerability goes through [`SECURITY.md`](SECURITY.md),
not a public issue.

## License

Omni is released under the [MIT License](LICENSE). The libraries it is built on, the programs its setup installs and the models it can download have their
own licenses; they are listed in [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

| Model (downloaded on request, not part of Omni) | License |
|---|---|
| EmbeddingGemma 2, for file search | Apache License 2.0 |
| LFM3-1.7B by Liquid AI, for Omni AI | Liquid AI's LFM Open License ([terms](https://huggingface.co/LiquidAI/LFM3-1.7B-GGUF/blob/main/LICENSE)) |

## Acknowledgements

Omni stands on [llama.cpp](https://github.com/ggml-org/llama.cpp) (runs both models), [Wails](https://wails.io) (the desktop shell), [Svelte](https://svelte.dev),
[uv](https://github.com/astral-sh/uv), [SQLite](https://sqlite.org), [FastAPI](https://fastapi.tiangolo.com), [PDFium](https://pdfium.googlesource.com/pdfium/) through
[pypdfium2](https://github.com/pypdfium2-team/pypdfium2), [PyAV](https://github.com/PyAV-Org/PyAV), and the Geist and Sora typefaces.
Thanks to the people who built them, and to Google and Liquid AI for releasing their models.
