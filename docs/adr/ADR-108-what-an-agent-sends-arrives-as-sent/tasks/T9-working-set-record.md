# Task ADR-108-T9: the working set reads back every record it saves

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the working set reads back every record it saves
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the working set reads back every record it saves`

## Goal

The working set gets the record bound the ledger got in T5, with the dispositions its content needs. `iter.Save` wrote a note or an entry of any length, while `load`'s `bufio.Scanner` stopped at its ~64 KiB default: a 70,000-character `mrw iter note` saved, then every later load — every CLI and MCP write, and `mrw iter clear` itself — failed with `token too long` (Codex review of #304). `Save` now refuses a note or entry over 64 KiB, naming it: the caller typed it, so it is not dropped silently. `load` skips a longer line, which only an older binary can have written, and keeps the rest.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/iter/iter.go` | edit | `Save`, `load` |
| `internal/iter/record108_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAWorkingSetRecordTheWriterSavesTheLoaderReads`; confirm RED. [proof: mutation]
2. [S2] `Save` refuses a line past `maxEntryBytes`, the note's `# ` prefix included; `load` reads lines through a `ReadSlice` reader that keeps `bufio.ScanLines`' line ends and skips one past the bound. Mutants: the save refusal removed; the skip flag never set; a final unterminated line dropped at EOF; the trailing `\r` kept. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/iter/ -count=1 -timeout 300s -run 'TestAWorkingSetRecordTheWriterSavesTheLoaderReads' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWorkingSetRecordTheWriterSavesTheLoaderReads \(' "$out" \
  && grep -q '^# 205\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWorkingSetRecordTheWriterSavesTheLoaderReads` | `internal/iter/record108_test.go` | a note or entry over the bound is refused by `Save`; a set holding a line over it, as an older binary wrote, loads with its other entries and note, and `Update` (what `mrw iter clear` runs) succeeds | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | `mrw iter`, and every CLI and MCP write, which loads the working set |
| 3 — the caller can discover it | the refusal names the size and the limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #304 |

## Mutation Log
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/iter/iter.go` · the save refusal removed: a 70,000-byte note is saved · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/iter/iter.go` · the loader skip removed: a 70,000-byte line loads as an entry · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f
- 2026-10-01 · 98a60ba* · mutant killed · exit 1 · `internal/iter/iter.go` · the save refusal removed: a 70,000-byte note is saved · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f
- 2026-10-01 · 98a60ba* · mutant killed · exit 1 · `internal/iter/iter.go` · the skip flag never set: the tail of a 70,000-byte line loads as an entry · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f
- 2026-10-01 · 98a60ba* · mutant killed · exit 1 · `internal/iter/iter.go` · a final unterminated line dropped: EOF with data fails the load · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f
- 2026-10-01 · 98a60ba* · mutant killed · exit 1 · `internal/iter/iter.go` · the trailing CR kept: a CRLF entry loads as a.go\r · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f

## Invariants

- A working set within the bound saves and loads byte-for-byte as before; an entry keeps its edge whitespace (ADR-069).

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:c2a068db4e8000be97a501a33684a3788edf048d42b4d3af128e3c7d456738f7 · ms:900 · test-lock-sha256:38803488223460d5829ef65eba8de97f185f82e6fdce6f2210ff4252a82ee461 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2l0ZXIvcmVjb3JkMTA4X3Rlc3QuZ28JVGVzdEFXb3JraW5nU2V0UmVjb3JkVGhlV3JpdGVyU2F2ZXNUaGVMb2FkZXJSZWFkcwk2YjExZDU3MTY1YjE2NmJjYjc1NTAwZDg0YzJmZDU1NDI2MjQ0MGExOGIwMWMyMTEzNDE0ZDg4NzQ3ZWI5Yzk4
  ```
  --- last 9 line(s) of stdout
  === RUN   TestAWorkingSetRecordTheWriterSavesTheLoaderReads
      record108_test.go:20: a 70,000-byte note was saved, or refused without its size: <nil>
      record108_test.go:23: a 70,000-byte entry was saved
      record108_test.go:38: a set holding a 70,000-byte line loaded "ok" ["a.go:1"], bufio.Scanner: token too long; want note ok and a.go:1, b.go:2
      record108_test.go:41: clearing a set that held a 70,000-byte line failed: bufio.Scanner: token too long
  --- FAIL: TestAWorkingSetRecordTheWriterSavesTheLoaderReads (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.177s
  FAIL
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:306
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAWorkingSetRecordTheWriterSavesTheLoaderReads
  --- PASS: TestAWorkingSetRecordTheWriterSavesTheLoaderReads (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.065s
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:317
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAWorkingSetRecordTheWriterSavesTheLoaderReads
  --- PASS: TestAWorkingSetRecordTheWriterSavesTheLoaderReads (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.074s
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:340
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAWorkingSetRecordTheWriterSavesTheLoaderReads
  --- PASS: TestAWorkingSetRecordTheWriterSavesTheLoaderReads (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.072s
  ```
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:500
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:349
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:376
- 2026-10-01 · 98a60ba* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:0 · test-lock-sha256:7c8ab1423c9edfe32a3ba22c3b5feda9306da26c96204ca5e22c84b42c6146c9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2l0ZXIvcmVjb3JkMTA4X3Rlc3QuZ28JVGVzdEFXb3JraW5nU2V0UmVjb3JkVGhlV3JpdGVyU2F2ZXNUaGVMb2FkZXJSZWFkcwk1NTJlMGVlZjYwMGM0YTFmNzIyNzQzMTYxNmYyNGZlNTc0ZjI2ZWI4ZjM5ODVkMzdhMDRhN2NkYmRlMzNmMjk2 · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red the test gained the Codex re-review's boundary cases (a note whose prefixed line fits round-trips and one byte more is refused; an entry of exactly the bound; CRLF, edge spaces, a final unterminated line filling the buffer, a final trailing CR); the earlier assertions are unchanged
- 2026-10-01 · 98a60ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:851
- 2026-10-01 · 98a60ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:493
- 2026-10-01 · 98a60ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:483
- 2026-10-01 · 98a60ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:f67dabf9b727e34e6a545d08535bd59cc3beceb84c94154617d4e7915c93f59f · ms:483
