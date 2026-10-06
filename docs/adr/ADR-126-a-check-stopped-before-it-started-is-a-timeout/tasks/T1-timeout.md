# Task ADR-126-T1: a check stopped before it started by its deadline exits 3

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `check.TimedOutBeforeStart`, `check.StoppedBeforeStart`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a check stopped before it started by its deadline exits 3`

## Goal

A check whose deadline passed before it started is `TimedOutBeforeStart`; the write path and `mrw check` exit 3 for it, naming the timeout.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | the constant, the branch, `StoppedBeforeStart` |
| `internal/check/timeout126_test.go` | add | the run-level test |
| `cmd/mrw/main.go` | edit | both exit mappings ask `StoppedBeforeStart` |
| `cmd/mrw/timeout126_test.go` | add | the write and `mrw check` tests |
| `AGENTS.md` | edit | exit 3 names a check that timed out before it started |

## Ordered Steps

1. [S1] Write `TestADeadlineBeforeTheStartIsATimeoutNotACannotStart`, `TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3` and `TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3`. Confirm RED. [proof: mutation]
2. [S2] The constant, the branch, the predicate and both mappings. Mutants: the branch reverted to "timed out after"; the write path's mapping without the timeout; `mrw check`'s mapping without it. [proof: mutation]
3. [S3] AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestADeadlineBeforeTheStartIsATimeoutNotACannotStart|TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3|TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestADeadlineBeforeTheStartIsATimeoutNotACannotStart \(' "$out" \
  && grep -qE '^--- PASS: TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3 \(' "$out" \
  && grep -qE '^--- PASS: TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3 \(' "$out" \
  && go test ./internal/check/ ./internal/writer/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q 'timed out before it started' AGENTS.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestADeadlineBeforeTheStartIsATimeoutNotACannotStart` | `internal/check/timeout126_test.go` | a run whose context's deadline has passed: `Ran` false, `Skipped` the timeout-before-start reason, `StoppedBeforeStart` true; a cancel stays `Interrupted` | none | S1, S2 |
| `TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3` | `cmd/mrw/timeout126_test.go` | the write lands and exits 3 naming the timeout, not "declare one" | none | S1, S2 |
| `TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3` | `cmd/mrw/timeout126_test.go` | `mrw check` the same | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TimedOutBeforeStart`, `StoppedBeforeStart` |
| 2 — something selects it | `check.run` on a deadline before the start; both CLI mappings |
| 3 — the caller can discover it | the exit message; AGENTS.md |
| 4 — it is used | the review of #325 and the Codex review found it; no telemetry (ADR-009) |

## Invariants

- A check that could not start for any other reason stays exit 2.
- A check that timed out while running stays exit 3, `Ran: true`.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a receipt key would have to change.

## Out of Scope

- A contract row (permanent: boundary: the built binary cannot be made to pass a deadline before a start without a race)

## Mutation Log
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `internal/check/check.go` · S2: the branch reverted to "timed out after" · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the write path without the timeout · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: mrw check without the timeout · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c

## Verification Log
- 2026-10-06 · cc32ee4* · exit 1 · `set -o pipefail …` · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c · ms:1330 · test-lock-sha256:a4ddb0a65fb6e5bb5855b3930872126a2d253dc2a6d6b9d942c816d870bea490 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy90aW1lb3V0MTI2X3Rlc3QuZ28JVGVzdEFXcml0ZVdob3NlRGVhZGxpbmVQYXNzZWRCZWZvcmVJdHNDaGVja1N0YXJ0ZWRFeGl0czMJMjI3Zjg2ZDM2ZGE4NjM0MjMyNmRiNzBhMGJlMTFiYzY4YzNkZTM2M2VlYjZiMjBjMWQ4MDRkYThlNDAzNDQ1Mgpib2R5CWNtZC9tcncvdGltZW91dDEyNl90ZXN0LmdvCVRlc3RNcndDaGVja1dob3NlRGVhZGxpbmVQYXNzZWRCZWZvcmVJdFN0YXJ0ZWRFeGl0czMJZmMwMTFhMDlhZWYxOTExYjQwOGRmZDFhMzQwZGMzNjkyMmIwNmQ3NWRhZjE1ZmYyNWNmMmExMzllN2I5NjIzNApib2R5CWludGVybmFsL2NoZWNrL3RpbWVvdXQxMjZfdGVzdC5nbwlUZXN0QURlYWRsaW5lQmVmb3JlVGhlU3RhcnRJc0FUaW1lb3V0Tm90QUNhbm5vdFN0YXJ0CTIyN2I2NjYxMTQyZGE2ZDUyZmJhZWUxMDg3YzFjZDFlODU2YzBkMmFlZWE5OTgwMTllYWJiYjM5MTQ1ZjA3MmE
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
          1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
          check SKIPPED: timed out after 5m0s
  --- FAIL: TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3 (0.02s)
  === RUN   TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3
      timeout126_test.go:53: want exit 3, timed out before it started: no check could run: timed out after 5m0s — declare one in .quality-harness.json
          check SKIPPED: timed out after 5m0s
  --- FAIL: TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3 (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.305s
  FAIL
  ```
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c · ms:37479
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c · ms:37403
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:eaea67728c8290591db1055266ed4d86ab4e661c2986311fe9c5e47c2ff5e48c · ms:37624
