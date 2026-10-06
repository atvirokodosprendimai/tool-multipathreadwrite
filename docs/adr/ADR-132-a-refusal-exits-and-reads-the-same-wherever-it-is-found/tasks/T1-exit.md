# Task ADR-132-T1: a target-caused refusal before the first rename exits 1

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `refuseStage`, `targetCause` in `internal/apply`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a target-caused refusal before the first rename exits 1`

## Goal

The eight sites of the record's audit refuse their hunk and return no error when the cause is the target's, by Decision 2; every other cause, and the root, stays an error.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `refuseStage` beside `abortStage`; `targetCause`; the eight sites; `changedSince` returns its cause beside its words |
| `internal/apply/replace.go`, `internal/apply/replace_other.go` | edit | the open refusal keeps its error; unix names `EILSEQ`, `ENAMETOOLONG` |
| `internal/apply/refusal132_test.go` | add | the tests |
| `internal/apply/*_test.go`, `cmd/mrw/*_test.go` | edit | tests that asserted an error or exit 2 for these refusals, each named in the PR; `apply_test.go:1129`'s untyped "no space" fixture becomes a typed `ENOSPC` and keeps its error |
| `scripts/contract.sh` | edit | §119, §168 to exit 1; §231 |
| `AGENTS.md`, `README.md`, `internal/guide/guide.go` | edit | ADR-125's "exit 2, NOTHING WRITTEN"; the failure matrix; exit 1's meaning |

## Ordered Steps

1. [S1] Write `TestATargetsStateRefusesItsHunkAndReturnsNoError` — a target changed since read, one removed since read, a probe refusing a name with `EACCES`, a replaceable probe refusing with a permission, a destination under a link to nothing: each returns nil, `Failed == 1`, siblings skipped, nothing written — and `TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError` — `ENOSPC`, `EIO`, `EROFS`, `EMFILE`, an unnamed error, each wrapped, from the probe seam and the replaceable seam, and a probe left behind whose removal hit a permission: each returns an error. Then `TestAStatThatFailsAfterReadingIsClassifiedByItsCause` — `changedSince`'s stat failing as removed, as a permission, as `EIO`, through a `statFn` seam (the second Codex review of the record). Confirm RED. [proof: mutation]
2. [S2] `targetCause(err)` = not a probe left behind, and `causeOf(err) != ""` or not-exist; `changedSince` returns its cause beside its words, through `statFn`; the open refusal keeps its error; `refuseStage(path, err, target)` assigns `abortStage`'s verdicts and returns `(res, nil)` when `target`; the eight sites pass their cause. Mutants: `refuseStage` returning the error (the target test goes red); `targetCause` answering true (`ENOSPC` refuses with no error); the leftover check dropped (a probe left behind with a permission refuses with no error); `changedSince`'s cause discarded (an `EIO` stat refuses with no error). [proof: mutation]
3. [S3] Every test and contract row that asserted an error or 2 for these moves, each named in the PR; contract §231 — a create under a read-only directory exits 1 with "nothing was written" and the requested `--then-sh` step reported not run, and the pair, the same create in a writable directory, exits 0; `TestATargetRefusalOverMCPIsARefusedPlan` asserts `isError` and the tally. AGENTS.md, README, guide. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestATargetsStateRefusesItsHunkAndReturnsNoError|TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError|TestATargetRefusalOverMCPIsARefusedPlan|TestAStatThatFailsAfterReadingIsClassifiedByItsCause' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestATargetsStateRefusesItsHunkAndReturnsNoError \(' "$out" \
  && grep -qE '^--- PASS: TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError \(' "$out" \
  && grep -qE '^--- PASS: TestATargetRefusalOverMCPIsARefusedPlan \(' "$out" \
  && grep -qE '^--- PASS: TestAStatThatFailsAfterReadingIsClassifiedByItsCause \(' "$out" \
  && go test ./internal/apply/ ./internal/adversarial/ ./internal/writer/ ./internal/mcp/ ./internal/guide/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 231\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestATargetsStateRefusesItsHunkAndReturnsNoError` | `internal/apply/refusal132_test.go` | each target-caused staging refusal returns no error, one failed hunk, siblings skipped, nothing written | none | S1, S2 |
| `TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError` | `internal/apply/refusal132_test.go` | ENOSPC, EIO, EROFS, EMFILE, an unnamed error and a probe left behind still return an error | none | S1, S2 |
| `TestATargetRefusalOverMCPIsARefusedPlan` | `internal/mcp/refusal132_test.go` | over MCP the refusal is `isError` with its receipt, carries no `error`, reports a requested step `not_run`, and is tallied refused — the last three only on the new path (the second Codex review of the record) | none | S3 |
| `TestAStatThatFailsAfterReadingIsClassifiedByItsCause` | `internal/apply/refusal132_test.go` | a stat failing as removed or as a permission refuses with no error; as `EIO` stays an error | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `refuseStage`, `targetCause` |
| 2 — something selects it | the eight staging sites, on every plan that reaches staging |
| 3 — the caller can discover it | the exit code and trailer; AGENTS.md; README; `mrw instructions` |
| 4 — it is used | the Windows field test; no telemetry (ADR-009) |

## Invariants

- What is refused, and the reason's words, do not change.
- Nothing is written by any of these refusals, before or after.
- An error no cause names keeps exit 2.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a test or contract row asserting exit 2 turns out to rest on a reason other than a staging refusal.

## Out of Scope

- Commit-stage failures (permanent: boundary: the record's)

## Mutation Log
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: refuseStage returns the error — a target refusal exits 2 again · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: targetCause answers true — ENOSPC refuses with no error · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the leftover check dropped — a probe left behind with a permission refuses with no error · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d
- 2026-10-06 · d493df7* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · S2: changedSince cause discarded — an EIO stat refuses with no error · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: changedSince cause discarded — an EIO stat refuses with no error · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d

## Verification Log
- 2026-10-06 · 135eb2e* · exit 1 · `set -o pipefail …` · acceptance-sha256:3b68fbb930aa9ca22ff0ead32d12d80b2d1795a32b6eab36a8af3c999a56c8b7 · ms:994 · test-lock-sha256:6b162cf7a95b0fc2442e54827bfb249f5bd0e711642716cfd812f1657a47d616 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVmdXNhbDEzMl90ZXN0LmdvCVRlc3RBVGFyZ2V0c1N0YXRlUmVmdXNlc0l0c0h1bmtBbmRSZXR1cm5zTm9FcnJvcglmODVmMzczZjg1Mzg3NTU3OTA1ZGFlMjJkZGRhMmI0ZjgxNzVmMWYzN2E5ZDY0ZmM1OGJkMTQ1ZTQwNDllYzhlCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVmdXNhbDEzMl90ZXN0LmdvCVRlc3RBbkVudmlyb25tZW50RmFpbHVyZUJlZm9yZVRoZUZpcnN0UmVuYW1lU3RheXNBbkVycm9yCTA3YWZkZmM0ZmY5NmQ3MjEyNjkzYzRkNGJmZDI3Y2UyNjhmMmIzZDNjNTQxNzEwZGVkYTJiOTIxNjE2MmE3M2QKdW5wcm92ZW4JaW50ZXJuYWwvbWNwL3JlZnVzYWwxMzJfdGVzdC5nbwlUZXN0QVRhcmdldFJlZnVzYWxPdmVyTUNQSXNBUmVmdXNlZFBsYW4
  ```
  --- last 10 line(s) of stdout (of 50 after folding 50 raw)
      --- PASS: TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError/leftover_from_probe (0.00s)
      --- SKIP: TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError/leftover_from_replaceable (0.00s)
      --- PASS: TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError/ENOSPC_from_probe (0.00s)
      --- PASS: TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError/ENOSPC_from_replaceable (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.090s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.178s [no tests to run]
  FAIL
  ```
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:39581
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:39781
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:40946
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:39796
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:41684
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:4cfcaea98aba43319885aeb8155014d61391ed225edd9a2346228d69dca54a1d · ms:0 · test-lock-sha256:3ba805a469b8b2a9dfa73cff5821d3e4c6ee00f12ad0edc4d0a497f73f2449e4 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVmdXNhbDEzMl90ZXN0LmdvCVRlc3RBU3RhdFRoYXRGYWlsc0FmdGVyUmVhZGluZ0lzQ2xhc3NpZmllZEJ5SXRzQ2F1c2UJZGFhNjVmZTI3ZTc4NzA5MjMyZGE2M2E4NDI3ZWMzNzg1YmIxYTA2ZTZmNTMyNzIzYzhkMTEyNDdhNDI4NmZjNgpib2R5CWludGVybmFsL2FwcGx5L3JlZnVzYWwxMzJfdGVzdC5nbwlUZXN0QVRhcmdldHNTdGF0ZVJlZnVzZXNJdHNIdW5rQW5kUmV0dXJuc05vRXJyb3IJZjg1ZjM3M2Y4NTM4NzU1NzkwNWRhZTIyZGRkYTJiNGY4MTc1ZjFmMzdhOWQ2NGZjNThiZDE0NWU0MDQ5ZWM4ZQpib2R5CWludGVybmFsL2FwcGx5L3JlZnVzYWwxMzJfdGVzdC5nbwlUZXN0QW5FbnZpcm9ubWVudEZhaWx1cmVCZWZvcmVUaGVGaXJzdFJlbmFtZVN0YXlzQW5FcnJvcglmNDY3YWFhYWJiM2Q2ODkxMmRhM2FhMzBhMTEyNDk2ZTNhNDk3NDYxMmEyZjZkNGQ5ZmQ0Mjg5Mzc4MTA4NmMzCmJvZHkJaW50ZXJuYWwvbWNwL3JlZnVzYWwxMzJfdGVzdC5nbwlUZXN0QVRhcmdldFJlZnVzYWxPdmVyTUNQSXNBUmVmdXNlZFBsYW4JZTIzOTAxNTEyYmQ4ODIwZjE2ZGQ2MzFjN2U5YThjNWY4ZWI0ODI0YmU5ZjgyMGE0Zjk5YzBmM2VhNzRlZTQxYQ · test-lock-kind:replace
