# AGENTS.md

Instructions for coding agents working in this repository (Claude Code reads this through `CLAUDE.md`). They are the rules a human
contributor follows too; [`CONTRIBUTING.md`](CONTRIBUTING.md) covers the process around them (issue first, setup, pull requests).

Omni is a local-first launcher for Windows 11 and macOS 13+ (Apple Silicon): a Go core, Wails v3 with a Svelte + TypeScript page for the
UI, and a Python sidecar (the "memory engine") that runs local models for file search.

## Orient first

Read [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) before a non-trivial change: modules, data flow, invariants, and which file is for which
system. [`DESIGN.md`](DESIGN.md) comes first for work in `frontend/`. The engine's HTTP contract is [`docs/api.md`](docs/api.md);
[`docs/README.md`](docs/README.md) indexes the rest.

## Checks

Run what you touched before you say it is done; CI runs all of it on Windows and macOS.

```sh
gofmt -l .                       # prints nothing when clean
go vet ./... && go test ./...    # launcher; add -race for concurrent code
cd frontend && npm ci && npm test && npm run check     # page
cd sidecar && uv sync && uv run ruff check && uv run pyright && uv run pytest   # engine
```

Build: `.\scripts\build.ps1` (Windows) or `./scripts/build-mac.sh` (Mac). Tests that need a model or a large index skip themselves.
Retrieval quality: `uv run python -m omni_memory.eval` in `sidecar/` (needs `OMNI_LLAMA_SERVER` and `OMNI_MODEL_DIR`; see `sidecar/eval/README.md`).

## Code rules

- **Modular:** one package = one job. `internal/<name>/` packages expose a small API; their internals stay private.
- **Two systems, kept apart:** shared logic in `x.go`, Windows code only in `x_windows.go`, Mac code only in `x_darwin.go` (tests likewise);
  no system API in a shared file. The table is in `docs/ARCHITECTURE.md`; keep it current in the same commit.
- **Small files.** If a file needs scrolling to understand, split it by responsibility.
- **Names over comments.** Comment the *why*, never the *what*. Every Go package has a package doc comment.
- **Errors:** wrap with context (`fmt.Errorf("load config: %w", err)`); never swallow.
- **Go:** standard library first; `gofmt` and `go vet` clean. **Python:** `uv` for the environment and dependencies; `ruff` and `pyright` clean.
  **Frontend:** Svelte + TypeScript.
- **Tests:** Go `testing` package, table tests for logic; `pytest` for the engine; the page has its own. Trivial glue needs no test.
  Fix a bug test-first.

## Constraints

- Windows 11 and macOS 13+ on Apple Silicon. Linux is not a target (the engine and page tests run there; the launcher does not build).
- Models share 6 GB of graphics memory on a laptop GPU and must fit together. Only models use the GPU; index storage and search run on the CPU.
- The hotkey → panel path never waits on a model. The launcher opens in under 100 ms.
- Sharp at every display scale: the Windows process is per-monitor DPI aware (`build/windows/omni.manifest`) and `scripts/build.ps1` fails a
  build that is not. Look at any new window at 125%.
- Privacy: nothing leaves the machine except explicit calls to a configured LLM provider and requests the user starts or switches on. No
  telemetry. The README's privacy table is the full list; do not add to it casually.
- Settings live in `config.toml` in the per-user app folder. API keys go in the system credential store (Windows Credential Manager, macOS
  Keychain), never in files or git.

## Working rules

- **Stay in scope.** Do what the issue asks. No drive-by rewrites, renames or "improvements" across files nobody mentioned.
- **No slop.** Every file, function and dependency earns its place: no speculative abstractions, no "for later" scaffolding, no unused
  configuration. A new dependency needs a reason and goes into `THIRD_PARTY_NOTICES.md`.
- **Measure, don't guess.** A change that affects speed, memory or search quality gets numbers before and after, on the same machine
  (`cmd/panelbench` for the hotkey path; the retrieval eval in `sidecar`). Put them in the pull request.
- **Docs follow code.** Update the doc that describes the code in the same commit.

## Branches and commits

- Work on a branch cut from `dev` and open the pull request into `dev`. `main` changes only by a pull request the maintainer merges; never push to it.
- Conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`), small and focused.
- Commits are signed off with the [DCO](CONTRIBUTING.md#sign-your-commits) by the human who submits them (`git commit -s`). An agent does not
  sign off for them and adds no `Co-Authored-By` or "Generated with" trailers to commits or pull request descriptions.
