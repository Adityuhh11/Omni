# 0012: Other AI apps get their own page, their own switches and their own descriptions

Date: 2026-11-02. Status: accepted. Amends 0011.

## Decision

Settings has a page, **AI apps**, for the MCP server of 0011. It holds the switch for the server, the line to add to the other
app, and every tool that app could be given, grouped as in Omni AI, each group with its own switch.

- **Their own switches.** `[chat] mcp_tools` lists the groups other apps may use. It is empty at first, so turning the server on
  shares nothing until the person picks. `[chat] tools` (the groups Omni AI's own model may use) no longer affects MCP, and the
  reverse. Only tools that read are ever offered, as before.
- **Their own descriptions.** A `chat.Local` may carry `Apps`, a second definition of the same tool for other AI apps; without
  one, `Def` is used. The bundled model has 1.7B parameters and gets short descriptions and one-line argument hints ("a few
  specific words, not a sentence"). Claude, GPT and the like are better served by a full description: what the tool covers and
  does not, what it returns, how to ask. `search_files` is retrieval over the person's own disk, and a capable model should write
  it a question or a description of the content, not keywords.
- **Say what Omni is.** `initialize` returns a paragraph about Omni and the data behind the tools (`instructions`), the server's
  `title`, and each tool carries `annotations.readOnlyHint: true` and `openWorldHint: false`, which apps use to group tools and to
  ask less.

## Why

- **The two audiences differ in who sees the data.** What Omni AI reads stays on the computer; what Claude Desktop reads goes to
  its servers. Letting one switch decide both made "I want the clipboard in my local chat" mean "my clipboard may go to a cloud
  model". Separate lists make each choice about one destination.
- **Nothing on by default.** The server shares the person's files and notes; the person names which, once, on the page that says
  where it goes.
- **A description is the interface.** For an external model the description is all it knows about the tool. The short form written
  for the small model made the tools sound narrower than they are.

## Alternatives not taken

- Keep one list of groups for both and add a master switch: cannot express "local only".
- A per-tool switch (not per group): the groups are what the person understands (files, notes), and a group's tools only make
  sense together (`search_notes` finds the id `read_note` needs).
- Offer tools that change something with the other app's own confirmation (Claude Desktop asks before every call): Omni cannot
  tell that the person said yes, only that a request came with the token. Unchanged from 0011.
- Move the settings out of `[chat]` into `[mcp]`: nothing released reads `[chat] mcp` yet, but the move buys nothing the person
  sees.

## Consequences

- A config from this branch's earlier commits (`mcp = true`, nothing in `mcp_tools`) offers no tools until the person switches
  groups on.
- Two descriptions of a tool can drift. They sit next to each other in `internal/aitools`, and a test checks that both name the
  same tool and the same arguments.
