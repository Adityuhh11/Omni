# Browser history tool: tool choice and cost (2026-11-02)

`OMNI_CHAT_DIR` set; RTX 4050 Laptop, 6 GB, 342 MB used by other processes; 8k context; thinking off; bundled Omni AI model.
`go test ./internal/aitools -run TestRealModelChoosesLocalTools -v` (fake file, notes, script and browser data).

| | questions | each question's own group on | every group on |
|---|---|---|---|
| before (files, notes, a script) | 4 | 4 of 4 | 4 of 4 |
| after (+ `browser`: "Which Rust tutorial page did I look at yesterday?") | 5 | 5 of 5 | 5 of 5 |

The model left the tools alone for arithmetic in both runs, and answered the browser question with the page's title and address.

**Cost of SQLite.** `go build -ldflags "-s -w" ./cmd/omni` on Windows: 16,686,592 bytes on `dev`, 20,793,856 with the browser package
(+4.1 MB). `GODEBUG=inittrace=1` on the browsers test binary: `modernc.org/libc` 0.54 ms, the other packages 0.00 ms.

**Real data.** A search over this computer's Zen, Chrome, Edge and Brave profiles (Zen open): 8 results in 0.68 s, including a
page opened a minute before (so the write-ahead log is read).
