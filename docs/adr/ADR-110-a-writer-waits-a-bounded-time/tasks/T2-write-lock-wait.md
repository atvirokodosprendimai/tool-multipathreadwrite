# Task ADR-110-T2: a writer waits a bounded time for the write lock

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `MRW_WRITE_LOCK_TIMEOUT`; the write-lock refusal
**Consumes:** `state.HoldWithin` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a writer waits a bounded time for the write lock`

## Goal

`seen.LockWrites` waits through `HoldWithin` for `MRW_WRITE_LOCK_TIMEOUT` seconds, 120 when unset; past it the write is refused, exit 2, saying nothing was applied and naming the holder, and a value that is not a whole number of seconds is refused the same way. AGENTS.md and the README say so.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `LockWrites`, `writeLockWait` |
| `internal/seen/lock110_test.go` | add | the test |
| `AGENTS.md` | edit | the writers-take-turns sentence |
| `README.md` | edit | the same |
| `scripts/contract.sh` | edit | §209 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAWriterWaitsABoundedTimeForTheWriteLock`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal; contract §209 through `$MRW`. Mutants: `LockWrites` back to `state.Hold`; the variable ignored (always 120 s). [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/seen/ -count=1 -timeout 300s -run 'TestAWriterWaitsABoundedTimeForTheWriteLock' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriterWaitsABoundedTimeForTheWriteLock \(' "$out" \
  && grep -q '^# 209\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriterWaitsABoundedTimeForTheWriteLock` | `internal/seen/lock110_test.go` | with the write lock held, `LockWrites` under `MRW_WRITE_LOCK_TIMEOUT=0` refuses within 5 s naming the write lock and saying nothing was applied; `soon` is refused as not a whole number of seconds; unset is 120 s; after release it takes the lock | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, through `writer.Apply` |
| 3 — the caller can discover it | the refusal names the lock, the holder and the variable |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B3) |

## Mutation Log
- 2026-10-01 · 94e279c* · mutant killed · exit 1 · `internal/seen/seen.go` · LockWrites back to state.Hold: a held write lock is waited on for ever · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616
- 2026-10-01 · 94e279c* · mutant killed · exit 1 · `internal/seen/seen.go` · the variable ignored: every wait is 120 s and a bad value is accepted · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616
- 2026-10-01 · e7f79fe* · mutant inconclusive · exit 1 · `internal/seen/seen.go` · the overflow guard removed: 9223372037 seconds wraps to a negative wait · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-01 · e7f79fe* · mutant killed · exit 1 · `internal/seen/seen.go` · the overflow guard made unreachable: 9223372037 seconds wraps to a negative wait · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616

## Invariants

- An uncontended lock is taken as before; the other locks keep `Hold`.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-110 task, in its own file.

## Verification Log
- 2026-10-01 · 94e279c* · exit 1 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:219 · test-lock-sha256:09f620e589c6e9efe8ad468eb7ab5b1fb899a0a2f828e8c26188261817d8e09d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvc2Vlbi9sb2NrMTEwX3Rlc3QuZ28JVGVzdEFXcml0ZXJXYWl0c0FCb3VuZGVkVGltZUZvclRoZVdyaXRlTG9jawk5MzA3YWQ1ODQ0NjBhOWNlMWNlNzg0N2VmOGQxOTVhODQ0ZTI3YTFhMWY0MDQ5MzEyYTdhNTQ0MDBmOWU2MTA2
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen.test]
  internal/seen/lock110_test.go:46:15: undefined: writeLockWait
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [build failed]
  FAIL
  ```
- 2026-10-01 · 94e279c* · exit 0 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:525
- 2026-10-01 · 94e279c* · exit 0 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:547
- 2026-10-01 · 94e279c* · exit 0 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:456
- 2026-10-01 · e7f79fe* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:0 · test-lock-sha256:4faae6e32ac86fb8aac9e8fc50e77ed000b97d2ca633919c5d7493c4d7cc27f0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvc2Vlbi9sb2NrMTEwX3Rlc3QuZ28JVGVzdEFXcml0ZXJXYWl0c0FCb3VuZGVkVGltZUZvclRoZVdyaXRlTG9jawlhNGY5Y2EwYWY2ZDRkMjI3NTljM2E3N2Y2YWEzNDRmY2UyZDk3MzYwYWFmZDc0YWI0Njc0ZmU3ZTM5ODM4ZDc2 · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red the test gained the Codex review of #307's overflow boundary (9223372037 seconds refused, 9223372036 accepted); the earlier assertions are unchanged
- 2026-10-01 · e7f79fe* · exit 0 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:852
- 2026-10-01 · e7f79fe* · exit 0 · `set -o pipefail …` · acceptance-sha256:2e533d81f8d2f706db22f0fe93390159b9f1e08754ca0fdb3462f3c305d61616 · ms:605
