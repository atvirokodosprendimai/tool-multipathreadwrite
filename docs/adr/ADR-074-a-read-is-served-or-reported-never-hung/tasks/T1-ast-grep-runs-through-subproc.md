# Task ADR-074-T1: ast-grep runs through subproc, and an interrupt stops it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `subproc.Signals`, `subproc.Interruptible`; `read.AstGrep` through `subproc.Command`
**Consumes:** `subproc.Command` (ADR-072)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the bound holds past a held pipe`, `the grandchild dies with ast-grep`, `a signal to mrw stops ast-grep`, `the check keeps ADR-072's handling`, `a contract row drives the binary`, `the packages vet for Windows`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A wrapper that leaves a grandchild holding ast-grep's stdout made the 2 s bound take 30 s and left
the grandchild running. Run ast-grep through `subproc`, and move the check's signal listening into
`subproc` so a ^C stops ast-grep as it stops a check.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/subproc/subproc.go` | edit | `Signals` and `Interruptible`, moved from the check |
| `internal/check/check.go` | edit | the check listens through `subproc.Interruptible` |
| `internal/read/astgrep.go` | edit | `subproc.Command` under `Interruptible`; "interrupted" named |
| `internal/subproc/interrupt_unix_test.go` | new | a signal cancels the child's context |
| `cmd/mrw/astgrep_grandchild_unix_test.go` | new | the bound and the interrupt, through the CLI |
| `scripts/contract.sh` | edit | §147 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; ADR-072's check tests stay green. [proof: mutation]
   Mutants: ast-grep back on `exec.CommandContext`; `Setpgid` dropped; `WaitDelay` 0; `Interruptible` returns the context it was given; the interrupted case dropped.
3. [S3] Contract §147: a forking ast-grep wrapper returns within 4 s and its grandchild is gone; the pair, a hanging ast-grep that forks nothing, still says timed out (§111). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/subproc/ ./internal/check/ ./cmd/mrw/ -count=1 -timeout 180s -run 'TestASignalToMrwCancelsTheChildsContext|TestTheCheckStopsOnHangupUnlessHangupIsIgnored|TestAnInterruptedCheckSaysSo|TestATimedOutCheckLeavesNoGrandchild|TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound|TestAnInterruptStopsAstGrepAndItsGrandchild|TestAHangingAstGrepTimesOut' -v 2>&1 | tee /tmp/adr074-T1.out \
  && missing=$(for t in TestASignalToMrwCancelsTheChildsContext TestTheCheckStopsOnHangupUnlessHangupIsIgnored TestAnInterruptedCheckSaysSo TestATimedOutCheckLeavesNoGrandchild TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound TestAnInterruptStopsAstGrepAndItsGrandchild TestAHangingAstGrepTimesOut; do grep -qE "^--- PASS: $t \(" /tmp/adr074-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 147\. ' scripts/contract.sh \
  && GOOS=windows go vet ./internal/subproc/ ./internal/check/ ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASignalToMrwCancelsTheChildsContext` | `internal/subproc/interrupt_unix_test.go` | INT, TERM and HUP each cancel the context | — | S1, S2 |
| `TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound` | `cmd/mrw/astgrep_grandchild_unix_test.go` | timed out within 5 s, grandchild dead within 2 s | — | S1, S2 |
| `TestAnInterruptStopsAstGrepAndItsGrandchild` | `cmd/mrw/astgrep_grandchild_unix_test.go` | TERM to mrw → "interrupted", grandchild dead | — | S1, S2 |
| `TestAHangingAstGrepTimesOut` | `cmd/mrw/astgrep_timeout_test.go` | ADR-058's bound, unchanged | — | S2 |
| `TestTheCheckStopsOnHangupUnlessHangupIsIgnored` | `internal/check/group_unix_test.go` | ADR-072's list, now subproc's | — | S2 |
| `TestAnInterruptedCheckSaysSo` | `internal/check/group_unix_test.go` | ADR-072's interrupt, unchanged | — | S2 |
| `TestATimedOutCheckLeavesNoGrandchild` | `internal/check/group_unix_test.go` | ADR-072's group kill, unchanged | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `subproc.Interruptible` |
| 2 — something selects it | every `--ast-grep` read, CLI and MCP; every check |
| 3 — the caller can discover it | the read returns at the bound, or says interrupted |
| 4 — it is used | the round's wrapper held a read for 30 s |

## Verification Log
(empty until execute)
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:6237 · test-lock-sha256:f1f40925aa91fd061dc46549055ba3f133873791b5cc742df300ffe69fea29f5 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF9ncmFuZGNoaWxkX3VuaXhfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwR3JhbmRjaGlsZEhvbGRpbmdTdGRvdXREb2VzTm90RGVmZWF0VGhlQm91bmQJMTYwZDE3YmQ1MWQwYWZjNzcwMTBjZmI4MDgzYjg1Y2IyNTM0OGJkMGJjN2ViMGVhNzBhZTk0YjBhOTljOTJhMApib2R5CWNtZC9tcncvYXN0Z3JlcF9ncmFuZGNoaWxkX3VuaXhfdGVzdC5nbwlUZXN0QW5JbnRlcnJ1cHRTdG9wc0FzdEdyZXBBbmRJdHNHcmFuZGNoaWxkCTM1ZmIwMTUxZjk3ZTMxNWYyYTdlMTlkYWQ2MjIwOTRlNzRiNDRjOGZlOGI0YzA2MWFhMDFlZTFhMTRmZjdhYzcKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGltZW91dF90ZXN0LmdvCVRlc3RBSGFuZ2luZ0FzdEdyZXBUaW1lc091dAk3ZTc0MWQ5YzYyY2EyZjc2NGYxYmMzNThjODMxNmZiOTEwM2VmMjU2YWE5MjE5MDg0YmM4NWUzYTJkZjNkZDg2CmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0xlYXZlc05vR3JhbmRjaGlsZAkzMzRiYjUwMzlhNjYxZTE0YzdiZDE0Njk5OWM2ZmIwMzFkNWU3M2M3NjYwZDhmMzIyZjM4ZWM2NWRmMjBlZmMwCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBbkludGVycnVwdGVkQ2hlY2tTYXlzU28JNGYwOWU2YjY5ZGI2YTY3NWM2YWFmNjVlZGRiYmMzYjNiYTI3ZTk2M2M5MzA2NWE5NDcxNjQxZjQ1YTdlZDQwYQpib2R5CWludGVybmFsL2NoZWNrL2dyb3VwX3VuaXhfdGVzdC5nbwlUZXN0VGhlQ2hlY2tTdG9wc09uSGFuZ3VwVW5sZXNzSGFuZ3VwSXNJZ25vcmVkCWE0NjkyYTI0MWZmZGEyZmNmOTNkNmMwNGVjMDFjYjAwZWQ3MDlhYWEwZWVkZTVjZTMxNWQzOTBkZWZjNTA0MTUKYm9keQlpbnRlcm5hbC9zdWJwcm9jL2ludGVycnVwdF91bml4X3Rlc3QuZ28JVGVzdEFTaWduYWxUb01yd0NhbmNlbHNUaGVDaGlsZHNDb250ZXh0CTRkMzk2MjE5ODEwMDgxMTYyN2I0ZjMyNWZhZGMyYWZkNGY2MThkNjUyOTg0OGVkOTZlN2MxZjFmYTA3NTFjNmY
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  --- PASS: TestTheCheckStopsOnHangupUnlessHangupIsIgnored (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	1.492s
  === RUN   TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound
      astgrep_grandchild_unix_test.go:48: the read is still running at 5 s: a grandchild holding stdout defeated the 2 s bound
  --- FAIL: TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound (5.00s)
  === RUN   TestAnInterruptStopsAstGrepAndItsGrandchild
  signal: terminated
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	5.354s
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:5084
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	1.482s
  === RUN   TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound
  --- PASS: TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound (2.00s)
  === RUN   TestAnInterruptStopsAstGrepAndItsGrandchild
  --- PASS: TestAnInterruptStopsAstGrepAndItsGrandchild (0.15s)
  === RUN   TestAHangingAstGrepTimesOut
  --- PASS: TestAHangingAstGrepTimesOut (2.11s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	4.675s
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:5036
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:4968
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:4842
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · ms:4674

## Mutation Log
(empty until execute)
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/read/astgrep.go` · ast-grep runs without a process group or a bounded wait, so a grandchild holding stdout keeps the read for 30 s · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · covers:the bound holds past a held pipe
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/subproc/subproc.go` · a signal sent to mrw does not reach a child in its own process group · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · covers:a signal to mrw stops ast-grep
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/read/astgrep.go` · an interrupted ast-grep is reported as timed out · acceptance-sha256:3f74b42470ba28b910f1a769b45ee3466099c36e1547e740f93026e84c2cc578 · covers:a signal to mrw stops ast-grep

## Invariants

- Every read that served before this record serves the same bytes.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-074 Out of Scope)

## Stop Condition

Stop if the change needs a package the record does not govern.
