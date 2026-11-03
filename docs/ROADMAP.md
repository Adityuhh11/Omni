# Roadmap

Omni is at 1.x: Windows 11 and macOS 13+ (Apple Silicon), used daily by its author and a few friends. This is what comes next, in order.
It is a direction, not a promise; the [issues](https://github.com/Adityuhh11/Omni/issues) are where work is agreed.

## Next

1. **A trusted install.** Windows warns about an unsigned program and Defender sometimes guesses wrong about a launcher that registers a
   hotkey and watches the clipboard. The plan is code signing (SignPath Foundation or Azure Artifact Signing), a Defender submission and a
   VirusTotal link with each release, and a winget manifest. On the Mac: Apple signing and notarization, which also brings the
   one-click update there.
2. **A stability pass.** A clean install on a computer with another graphics vendor and on one with none, the engine restarting after
   sleep, long runs (days) watching memory and handles, and the paste targets that behave differently (terminals, elevated windows).
3. **Search quality from real use.** Collect queries where the right result was not first, add them to the golden set in
   `sidecar/eval/`, and only then consider a reranker, full-text trigram search or tree-sitter chunking.
4. **Linux.** Planned, not started. The engine and the page already test on Linux; the launcher's system code does not exist yet.

## Where help is most useful

- Trying Omni on hardware the maintainer does not have, and reporting what works and what breaks.
- The Mac build, which is newer and has had less use.
- Anything in the list above, after an issue says go ([`CONTRIBUTING.md`](../CONTRIBUTING.md)).

## Not planned

- Telemetry, accounts or cloud services of Omni's own.
- A model catalog or model downloads from Hugging Face: choosing a local chat model is left to LM Studio and Ollama, which show what fits a
  given graphics card. Omni AI can ask a model they serve.
- Shut down and restart commands in the bar.
