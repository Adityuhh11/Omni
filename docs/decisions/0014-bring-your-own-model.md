# 0014: Omni AI can ask a model somebody else runs

Date: 2026-11-02. Status: accepted. Extends 0008.

## Decision

Settings > Omni AI > **Which model** lets the person point Omni AI at any OpenAI-style server instead of the bundled model: LM Studio
or Ollama on this computer, or a cloud service with an API key. Three settings in `[chat]` (`custom`, `custom_url`,
`custom_model`) say which; the **key is in the system credential store** (Windows Credential Manager, macOS Keychain) under
`Omni/chat-custom-key`, never in a file (`internal/secrets`). The bundled model stays the default and is not touched.

- **One switch at the model.** `chat.Model` already hands the chat an `llm.Endpoint` (address and key). `llm.Switch` returns the
  bundled server's, or the custom one (`Endpoint.Model` set). Nothing else in the chat knows the difference, except two places:
  - the request leaves out what only llama.cpp knows (`chat_template_kwargs`, the `</think>` stop) and `max_tokens`, which newer
    OpenAI models refuse; it adds `model`;
  - tokens are estimated (three characters to a token), since only llama.cpp has `/tokenize`.
- **The page says where the question goes.** A loopback address (localhost, 127.x, ::1) says nothing leaves the computer; any
  other says that the questions, and what the tools hand the model, go to that host, under its terms and charges. The lede, the
  Tools section and the privacy table (README) say the same.
- **Test before use.** "Test the connection" asks `GET /v1/models` with the typed or saved key, and offers the server's own model
  names; "Use this model" saves, switches Omni AI on (nothing to download) and unloads the bundled model's memory.
- **The key is entered once.** The page is told only whether one exists, and can forget it.

## Why

- A 1.7B model on a 6 GB card is a floor, not what people have: many already run LM Studio or Ollama, and some have a key.
- The seam already existed; the work is the key's safe storage, the request's shape, and being honest about the network.

## Alternatives not taken

- A key in `config.toml`: AGENTS.md forbids it, and the file is plain text beside the person's synced settings.
- A provider list (OpenAI, Anthropic, Gemini with their own request shapes): three APIs to keep in step. The OpenAI shape is what
  LM Studio, Ollama, OpenRouter, Groq and OpenAI all take; a provider that does not can be put behind such an address.
- Tools only for local addresses: the person chose the tools and the model; the page says what goes where. Revisit if it surprises.
- A catalog of local models sized to the graphics memory, or downloads from Hugging Face: each model brings its own template, thinking switch
  and tool-call behavior (0008), so Omni would own the flags, the memory fit and the support for every one. LM Studio and Ollama already
  show what a card can run, and this decision connects to them. Dropped by the owner on 2026-11-03 before any model was measured.
- Counting tokens exactly with a tokenizer library: a dependency for a number the context fit only needs roughly (the fit keeps room).

## Consequences

- The README privacy table gains a row: with this switched on, questions go to the address the person typed.
- The reply's length is the service's own limit; "How much it can read at once" still sets how much history is sent.
- A model that does not support tool calls will not use the tools; nothing is hidden, it just answers.
- On macOS the key goes through the `security` command, with the secret on standard input and not in its arguments. CI runs the
  round-trip test on a Mac; it was not run by hand on one.
