# Task ADR-088-T4: a read whose answer did not reach the caller records nothing

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw read` flushes and checks its answer before recording it; contract §171
**Consumes:** errcheck without the blanket preset (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a failed answer records nothing`, `a delivered answer still records`, `the built binary exits 2`

## Goal

`mrw read` recorded the served lines, then flushed its buffered answer on return with the error unchecked, so a read whose output could not be written licensed a write to lines nobody saw.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | flush and check before `seen.Record`; exit 2 naming the failure |
| `cmd/mrw/unwritten_read_test.go` | new | drives the command with a stdout that refuses writes |
| `scripts/contract.sh` | edit | §171 on the built binary, `/dev/full` |

## Ordered Steps

1. [S1] Write `TestAReadWhoseAnswerCannotBeWrittenRecordsNothing` and contract §171; the test fails on `56d4480`, where the read exits 0 and records. [proof: mutation]
2. [S2] Flush and check the answer before recording it. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAReadWhoseAnswerCannotBeWrittenRecordsNothing' -v 2>&1 | tee /tmp/adr088-T4.out \
  && grep -qE '^--- PASS: TestAReadWhoseAnswerCannotBeWrittenRecordsNothing \(' /tmp/adr088-T4.out \
  && grep -q '^# 171\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadWhoseAnswerCannotBeWrittenRecordsNothing` | `cmd/mrw/unwritten_read_test.go` | a refused stdout gives exit 2 and no ledger entry; a working one records | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the checked flush in the read action |
| 2 — something selects it | every `mrw read` |
| 3 — the caller can discover it | the exit code and the message name it |
| 4 — it is used | every CLI read; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 56d4480* · exit 1 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:860 · test-lock-sha256:72e0fef6d29596eae41f53989c2000437089097e77b24cc5ecb8367b56f449be · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdW53cml0dGVuX3JlYWRfdGVzdC5nbwlUZXN0QVJlYWRXaG9zZUFuc3dlckNhbm5vdEJlV3JpdHRlblJlY29yZHNOb3RoaW5nCTA0NWYzOWJjOWI0YTMwNGVmM2E5MDkyNTIwMTlhZGM0NzZmMzk3ZTM0NjE1ODg1YmI3NTgyMmQzZjdkMzk2MGQ
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  panic({0x10461a6d0?, 0x104675940?})
  	/opt/homebrew/Cellar/go/1.27.1/libexec/src/runtime/panic.go:859 +0x120
  github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.TestAReadWhoseAnswerCannotBeWrittenRecordsNothing(0x362c7657c248)
  	~/GolandProjects/tool-multipathreadwrite/cmd/mrw/unwritten_read_test.go:55 +0x208
  testing.tRunner(0x362c7657c248, 0x104657010)
  	/opt/homebrew/Cellar/go/1.27.1/libexec/src/testing/testing.go:2193 +0xc4
  created by testing.(*T).Run in goroutine 1
  	/opt/homebrew/Cellar/go/1.27.1/libexec/src/testing/testing.go:2258 +0x3b8
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.103s
  FAIL
  ```
- 2026-09-27 · 56d4480* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:0 · test-lock-sha256:7e59493105aa3b23d671a7a324c8c981ed41b2f539531187aaa42304b14744b9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdW53cml0dGVuX3JlYWRfdGVzdC5nbwlUZXN0QVJlYWRXaG9zZUFuc3dlckNhbm5vdEJlV3JpdHRlblJlY29yZHNOb3RoaW5nCTFlNWU1MmQ3YWFjMDQ1NDM4ZmJiMGRjYmRiYzQ0MTE3ZGVjYTBiYWYyNWRhNWQ2ZTE3Zjg0ZWZiZjVjNmU1NGQ · test-lock-kind:replace
- 2026-09-27 · human-observed · Zy's session, 2026-09-27: the first red panicked on a nil error before its assertion; the test now fails cleanly first (exit 0 is the defect) and asserts the same things.
- 2026-09-27 · 56d4480* · exit 0 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:781
- 2026-09-27 · 56d4480* · exit 0 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:419
- 2026-09-27 · 56d4480* · exit 0 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:335
- 2026-09-27 · 56d4480* · exit 0 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:379
- 2026-09-27 · 56d4480* · exit 0 · `set -o pipefail …` · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · ms:944

## Mutation Log
(empty until execute)
- 2026-09-27 · 56d4480* · mutant killed · exit 1 · `cmd/mrw/main.go` · the answer is no longer checked before it is recorded · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · covers:a failed answer records nothing
- 2026-09-27 · 56d4480* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · a delivered read records nothing · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · covers:a delivered answer still records
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-27 · 56d4480* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · a delivered read records nothing · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · covers:a delivered answer still records
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-27 · 56d4480* · mutant killed · exit 1 · `cmd/mrw/main.go` · a delivered read records nothing · acceptance-sha256:cfcd7167d120c6dfa9351f0c95e6182c1962b734876f9bc41d78d6577f494278 · covers:a delivered answer still records

## Invariants

- A read that reaches its caller records exactly what it recorded before.

## Risks

- None beyond the record's.

## Out of Scope

- The same check on the write receipt (permanent: boundary: see the record's Out of Scope)

## Stop Condition

The fence exits 0 and contract §171 passes on Linux.
