# Task ADR-152-T1: the check names scoped_check when a whole-project run was slow

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `check.Result.Hint`, `check.SlowAfter` in `internal/check/check.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a slow unscoped run hints`, `a scoped run does not`, `a fast or unnamed run does not`, `the fallback names its template`, `the CLI prints it`, `the MCP text prints it`, `the receipt key is listed`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Result.Hint`, `SlowAfter`, set in `Run` |
| `internal/check/slow152_test.go` | add | the engine test |
| `cmd/mrw/main.go` | edit | `reportCheck` prints the line |
| `cmd/mrw/slow152_test.go` | add | the CLI line |
| `internal/mcp/tools.go` | edit | `checkReport` prints the line |
| `internal/mcp/slow152_test.go` | add | the MCP line |
| `docs/receipts.txt` | edit | `check hint`, `mcp_write check.hint`, `write check.hint` |
| `scripts/contract.sh` | edit | §256 |
| `README.md`, `AGENTS.md`, `docs/adr/BACKLOG.md` | edit | one sentence each; the survey entry marked |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestASlowUnscopedCheckSaysHowToScopeIt` (with `SlowAfter` 0: a declared `check` and no `scoped_check`, run for a named path, carries a hint naming `scoped_check` and `{dirs}`; a `scoped_check` of `{files}` carries none; a `{packages}` template on a non-Go path carries the fallback words naming the template; no paths carries none; with `SlowAfter` an hour, none); `TestTheCLIPrintsTheSlowCheckHint` (a write whose declared check is `true`, `SlowAfter` 0: stdout holds one `hint: ` line naming `scoped_check`; the JSON receipt's `check.hint` is set); `TestTheMCPTextPrintsTheSlowCheckHint` (`checkReport` with a hinted result puts `hint: ` in the tail, none without).
2. [S2] `Result.Hint` and `SlowAfter` in `check.Run`. [proof: mutation]
3. [S3] `reportCheck` prints it. [proof: mutation]
4. [S4] `checkReport` prints it. [proof: mutation]
5. [S5] `docs/receipts.txt`, README, AGENTS, BACKLOG. [proof: mutation]
6. [S6] Contract §256: through the built binary, a fast whole-project check for a write prints no `hint:` line and its JSON has no `hint`, and `mrw check --full` none either; the slow half is driven by the Go tests, since the binary takes no seam for the threshold. [proof: human: the executor runs ./scripts/contract.sh once and reads the §256 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestASlowUnscopedCheckSaysHowToScopeIt|TestTheCLIPrintsTheSlowCheckHint|TestTheMCPTextPrintsTheSlowCheckHint|TestNoShippedReceiptFieldDisappears|TestNoShippedMCPReceiptFieldDisappears' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestASlowUnscopedCheckSaysHowToScopeIt \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLIPrintsTheSlowCheckHint \(' "$out" \
  && grep -qE '^--- PASS: TestTheMCPTextPrintsTheSlowCheckHint \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && grep -q '^# 256\. ' scripts/contract.sh \
  && grep -qx 'write check.hint' docs/receipts.txt \
  && grep -qF 'declare `"scoped_check"` in `.quality-harness.json`' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASlowUnscopedCheckSaysHowToScopeIt` | `internal/check/slow152_test.go` | the hint's conditions and its two wordings | none | S1, S2 |
| `TestTheCLIPrintsTheSlowCheckHint` | `cmd/mrw/slow152_test.go` | the receipt line and the JSON key | none | S1, S3 |
| `TestTheMCPTextPrintsTheSlowCheckHint` | `internal/mcp/slow152_test.go` | the `mrw_write` text line | none | S1, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Result.Hint`, `SlowAfter` |
| 2 — something selects it | `check.Run`, which every write's check and `mrw check` call; `reportCheck` and `checkReport` print it; contract §256 drives the built binary |
| 3 — the caller can discover it | the line itself; README and AGENTS |
| 4 — it is used | nothing measures this; built because four survey sessions reported `--no-check` on every call (2026-10-09) |

## Mutation Log
(empty until execute)
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · Run never sets the hint · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · a scoped run hints too · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI receipt drops the hint line · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/mcp/tools.go` · the mrw_write text drops the hint line · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `docs/receipts.txt` · the receipt key check hint is not listed · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `README.md` · README stops naming the hint · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · Run never sets the hint · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:a slow unscoped run hints
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · a scoped run hints too · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:a scoped run does not
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · a run for no paths (--full) hints too · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:a fast or unnamed run does not
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · a fast run hints too · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:a fast or unnamed run does not
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/check/check.go` · a fallen-back scoped_check gets the undeclared words · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:the fallback names its template
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI receipt drops the hint line · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:the CLI prints it
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `internal/mcp/tools.go` · the mrw_write text drops the hint line · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:the MCP text prints it
- 2026-10-14 · 83c8042* · mutant killed · exit 1 · `docs/receipts.txt` · the receipt key check hint is not listed · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · covers:the receipt key is listed

## Verification Log
(empty until execute)
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:c149243a8fc916b6b0cc9d52dbd4bb46bc3b88bbf645d40755b30d12041195d9 · ms:335 · test-lock-sha256:dbe63fbca5c26563213de90c457abd196fae1b8fc120e4fb9665dc1ca5ed7d4b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zbG93MTUyX3Rlc3QuZ28JVGVzdFRoZUNMSVByaW50c1RoZVNsb3dDaGVja0hpbnQJMWJjMWM4Yjg4ZjkyNTYwNzZiYmIyYmI2Mjk1Y2EzMmE0MmEwMGRjNDBiNDZjYTBhY2JkOTIzNjRmZGRmNWUzNwpib2R5CWludGVybmFsL2NoZWNrL3Nsb3cxNTJfdGVzdC5nbwlUZXN0QVNsb3dVbnNjb3BlZENoZWNrU2F5c0hvd1RvU2NvcGVJdAllMGM4ZDY1NjlhNzZmMDk1NWExYjUwMTRlYWI5MjY1YTZjMjU0OWY3NWY5MGNlNTY5ZDUyZWYyMWEyZjdmZGU4CmJvZHkJaW50ZXJuYWwvbWNwL3Nsb3cxNTJfdGVzdC5nbwlUZXN0VGhlTUNQVGV4dFByaW50c1RoZVNsb3dDaGVja0hpbnQJZDBiMGI4NjYzODU3ODg0ZTlkODJlZjgzMTBiNmE4YzAwMzFiMGRiZTY2ODdhZDY2NTI3YWE5YTkwYmU5ODQ2NQ
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [build failed]
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/slow152_test.go:13:67: unknown field Hint in struct literal of type "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check".Result
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  cmd/mrw/slow152_test.go:29:15: undefined: check.SlowAfter
  cmd/mrw/slow152_test.go:30:27: undefined: check.SlowAfter
  cmd/mrw/slow152_test.go:31:8: undefined: check.SlowAfter
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [build failed]
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1899
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.06s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.296s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.194s
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1757
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.06s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.282s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.187s
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1759
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.04s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.271s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.185s
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1803
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.04s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.273s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.186s
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1869
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.05s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.276s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.187s
  ```
- 2026-10-14 · 83c8042* · exit 1 · `set -o pipefail …` · acceptance-sha256:dd9494872329df170d7e1dce2823a6cf74675b6d91b289cd72275f014c0dad5d · ms:1652
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestTheCLIPrintsTheSlowCheckHint
  --- PASS: TestTheCLIPrintsTheSlowCheckHint (0.05s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.277s
  === RUN   TestNoShippedMCPReceiptFieldDisappears
  --- PASS: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheMCPTextPrintsTheSlowCheckHint
  --- PASS: TestTheMCPTextPrintsTheSlowCheckHint (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.189s
  ```
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2128
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:1963
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:1789
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:1741
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2725
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:3066
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2256
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:1838
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2108
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2348
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2676
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2523
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2030
- 2026-10-14 · 83c8042* · exit 0 · `set -o pipefail …` · acceptance-sha256:f154ced6e0a36f784743f66f31e2d7dd6aa0860795f8279ae46e32cf3202b88e · ms:2204
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S6 ./scripts/contract.sh exit 0 ('contract holds'); the §256 rows PASS — a fast whole-project check for a write exits 0 with no hint line, its --json check carries no hint key, and mrw check --full prints no hint; the slow half is proved in-process by TestTheCLIPrintsTheSlowCheckHint

## Invariants

- The check's command, verdict, exit code and duration are unchanged.
- No receipt key is renamed or removed (ADR-111).

## Risks

- None beyond the record's.

## Out of Scope

- A seam on the binary to lower the threshold (permanent: boundary: Decision 4; the slow half is proved in-process)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
