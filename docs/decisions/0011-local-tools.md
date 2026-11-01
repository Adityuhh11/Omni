# 0011: Omni AI uses Omni: local tools, scripts as tools, and an MCP server

Date: 2026-11-01. Status: accepted.

## Decision

Omni AI may call **local tools**: searches of what Omni already keeps (the file index, the clipboard history, the notes, recent
activity) and the person's own **scripts**. A tool is a `chat.Local` (a definition, a run function, whether it changes something);
`internal/aitools` builds them from the other packages and `main` hands them to the chat. Tools come in groups (`files`,
`clipboard`, `notes`, `activity`, `script:<name>`), each off until switched on (`[chat] tools`). A tool that changes something
(a new note, any script) asks in the chat window before each run unless its group is in `[chat] always_run`.

A file in the scripts folder with an `@omni.title` line is a command in the bar; with an `@omni.tool` line it is also a tool
(`docs/SCRIPTS.md`). The same tools that only read are offered to other AI apps through MCP when `[chat] mcp` is on.

## Why

- **The data is already here.** Omni has a semantic index of the person's folders, their copies, notes and launches. A launcher's
  chat that cannot see them answers like any chatbot. Search plus whole matching chunks (`full_text` on `/search/files`; snippets
  are 240 characters, too short to answer from) is retrieval over the person's disk with nothing leaving it.
- **Groups, not one switch.** The bundled model has 1.7B parameters: two tools it handles, fifteen it confuses. The person picks
  the few they want; a stronger model later (BYOK) can be given all of them.
- **Ask before changing anything.** A script can do anything, and a model can be talked into calling one by text in a file or a
  page. So every script and every write asks by default; "run without asking" is per group and the person's choice.
- **One file is a command and a tool.** No SDK and no manifest: a few comment lines in any language the computer runs. The bar
  passes text after the keyword; the model passes `input`. Both also arrive as `OMNI_INPUT`, and a batch file gets only that,
  because `cmd.exe` would read quotes in an argument as commands.
- **MCP over a relay, not a port in the config.** The AI apps that matter start a command and talk on its standard input and
  output. `omni mcp` relays each line to the running Omni at a loopback address with a token made for the run, both in
  `mcp.json` in the data folder (rewritten at each start, removed at stop). The server refuses requests without the token or with
  an `Origin` header (a web page could reach a loopback port; it cannot read the file). Only tools that read are offered: the
  other app cannot show Omni's question, so nothing that writes can be confirmed there.

## Alternatives not taken

- A tool per data source always on: too many tools for the bundled model, and consent should be per kind of data.
- `read_file` for any path: the index already holds the text, and a path the model composes is how a page talks a model into
  reading `~/.ssh`. The model sees what search returned.
- A plugin SDK (JavaScript in the page, or Go plugins): a framework to learn and to keep stable. A script with comment lines needs
  neither.
- The official MCP Go SDK: four methods and one transport are about 200 lines with the standard library.
- An MCP server that runs on its own: it would need the engine's per-run token and a second copy of the clipboard and notes.

## Consequences

- With web search and a local tool both on, the model could put what a tool returned into the words it searches for, which go to
  DuckDuckGo. Settings and the README say so.
- An engine older than this launcher answers `full_text` with 422: the lookup falls back to snippets, and recent files say to
  update file search.
- Scripts run in the person's home folder for at most 60 s while the bar waits; one that takes longer can start something and
  return.
