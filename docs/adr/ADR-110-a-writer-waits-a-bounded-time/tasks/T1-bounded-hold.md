# Task ADR-110-T1: a lock wait is bounded and names the holder

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `state.HoldWithin`, `state.LockTimeoutError`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a lock wait is bounded and names the holder`

## Goal

`state.HoldWithin` tries the lock without blocking and retries until its wait has passed, then returns a `LockTimeoutError` naming the holder, whose pid a taker writes to `<name>.holder`; a wait of 0 tries once.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/lock.go` | edit | `HoldWithin`, `LockTimeoutError`, the holder file |
| `internal/state/lock_unix.go` | edit | `tryLock` |
| `internal/state/lock_windows.go` | edit | `tryLock` |
| `internal/state/lock110_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAWaitForAHeldLockIsBounded`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: `tryLock` blocking (no `LOCK_NB`); the holder pid not written. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/state/ -count=1 -timeout 300s -run 'TestAWaitForAHeldLockIsBounded' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWaitForAHeldLockIsBounded \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWaitForAHeldLockIsBounded` | `internal/state/lock110_test.go` | a second `HoldWithin` of a held lock returns a `LockTimeoutError` naming this process's pid after its wait and within 5 s; a wait of 0 refuses at once; after release the lock is taken | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, through `writer.Apply` |
| 3 — the caller can discover it | the refusal names the lock, the holder and the variable |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B3) |

## Mutation Log
- 2026-10-01 · 94e279c* · mutant killed · exit 1 · `internal/state/lock_unix.go` · tryLock blocking: a held lock is waited on for ever · acceptance-sha256:a466b08b082f128452f844873b96fabe8f5db38b1f12968357a8864c337a9c4b
- 2026-10-01 · 94e279c* · mutant killed · exit 1 · `internal/state/lock.go` · the holder pid not written: the refusal cannot name who holds the lock · acceptance-sha256:a466b08b082f128452f844873b96fabe8f5db38b1f12968357a8864c337a9c4b

## Invariants

- An uncontended lock is taken as before; the other locks keep `Hold`.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-110 task, in its own file.

## Verification Log
- 2026-10-01 · 94e279c* · exit 1 · `set -o pipefail …` · acceptance-sha256:a466b08b082f128452f844873b96fabe8f5db38b1f12968357a8864c337a9c4b · ms:177 · test-lock-sha256:ed0c46203188b5f7a5f19045a6737011796547e1e99c71bf39c9d7d15e6b6ffb · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvc3RhdGUvbG9jazExMF90ZXN0LmdvCVRlc3RBV2FpdEZvckFIZWxkTG9ja0lzQm91bmRlZAliZWZkMjM4MDVkODE2Mzk4ODE4NTlhMTgwYzdjZjhmMGFkMDZlY2JiMDYxY2E3NDE3OTJlYTdkZTc1MzgzZDZl
  ```
  --- last 8 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state.test]
  internal/state/lock110_test.go:19:18: undefined: HoldWithin
  internal/state/lock110_test.go:30:13: undefined: HoldWithin
  internal/state/lock110_test.go:36:10: undefined: LockTimeoutError
  internal/state/lock110_test.go:49:15: undefined: HoldWithin
  internal/state/lock110_test.go:56:15: undefined: HoldWithin
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [build failed]
  FAIL
  ```
- 2026-10-01 · 94e279c* · exit 0 · `set -o pipefail …` · acceptance-sha256:a466b08b082f128452f844873b96fabe8f5db38b1f12968357a8864c337a9c4b · ms:826
- 2026-10-01 · 94e279c* · exit 0 · `set -o pipefail …` · acceptance-sha256:a466b08b082f128452f844873b96fabe8f5db38b1f12968357a8864c337a9c4b · ms:794
