# Local tools with the bundled model (2026-11-01)

Does LFM3-1.7B (QAD Q4_0, the Omni AI default) call the right local tool, and only when it should? `internal/aitools/real_test.go`
(`OMNI_CHAT_DIR` set; RTX 4050 Laptop, 6 GB, about 1.1 GB already used by another process; 8k context; thinking off).

Four questions, each asked with only its own group on and again with every group on (files, notes, a script):

| Question | Wants | Own group | Every group |
|---|---|---|---|
| What does my lease say about having a pet? | `search_files`, the file as a source, "300" | 4 of 4 runs | 4 of 4 |
| What is on my groceries note? | `search_notes`, "sourdough" | 4 of 4 | 4 of 4 |
| Which Wi-Fi network am I on right now? | the `@omni.tool` script (confirmed), "HomeNet" | 4 of 4 | 4 of 4 |
| What is 12 times 7? | no tool, "84" | 4 of 4 | 4 of 4 |

Answers took 0.2 s (no tool) to 8.2 s (first file search, model warm-up included); 0.9 to 3.2 s otherwise.

The file index is a fixed answer in this test (the search itself is measured by the retrieval eval); the notes store and the
PowerShell script are real.

**Tried and dropped:** telling the model in the system prompt not to show note ids ("never show them") made no difference to the
ids and cost one tool choice with every group on (3 of 4) in its one run; the prompt is back as it was. The ids stay visible
now and then.

**Not measured yet:** more than three groups at once with scripts the model must tell apart, and `recent_activity` on real
history. Watch tool choice when the group list grows.
