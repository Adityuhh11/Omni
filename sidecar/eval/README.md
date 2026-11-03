# Retrieval eval

Checks that file search still finds the right file. It indexes a small public corpus with the real embedding model, runs the golden
queries, and fails if quality drops.

| File | What |
|---|---|
| `testdata/` | 21 small files (code in Python, Go and JavaScript; notes) and a `.env.example` that must never be indexed |
| `golden.yaml` | 29 queries. `expect`: a hit counts if its path contains one of these. `none: true`: nothing answers it, so the engine must not claim a confident match. `forbid`: no top-10 hit may contain this. |
| `baseline.json` | The accepted metrics. The run fails if a metric drops more than 0.02, or a `forbid` file shows up. |

## Run it

It needs the real model. Turn on File search in Omni once (that installs the engine and models), then, from `sidecar/`:

```powershell
$e = "$env:LOCALAPPDATA\Omni\engine"
$env:OMNI_LLAMA_SERVER = "$e\llama\llama-server.exe"
$env:OMNI_MODEL_DIR    = "$e\models"
uv run python -m omni_memory.eval
```

On a Mac the engine is under `~/Library/Application Support/Omni/engine`. A report is written to `eval/<date>-baseline.md` (git-ignored).
Use `--cpu` to embed on the processor. A run on another device than the baseline's is measured but not gated, because bf16 on a graphics card
and fp32 on a processor can differ by a query.

## Change it

- **A new query** goes in `golden.yaml` with the file it should find. Add files to `testdata/` only when a query needs one; keep them small.
- **The baseline** moves only when quality improves or the golden set changes. Run with `--write-baseline`, and say in the pull request what changed
  and why. The baseline records its device and the number of queries; a baseline from a different golden set or model must be re-recorded.
- **Your own folder:** `--data <folder> --golden <file> --report-dir <elsewhere>` runs the same checks on files you choose. The baseline is read from the
  report directory, so there is none to gate against there; only a `forbid` hit fails the run. Read the report.
