# Task ADR-008-T1: A delete receipt names the first and last line it removed

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** unassigned
**Produces:** `HunkResult.RemovedFirst`, `HunkResult.RemovedLast` (`removed_first`, `removed_last`)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Make a delete say what it removed, in two trimmed strings, so a wrong range is
visible at write time rather than at build time.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | record the bounds on a delete verdict; reuse the existing `trim` |
| `internal/apply/apply_test.go` | edit | its tests |
| `cmd/mrw/main.go` | edit | print the bounds on the human receipt line — this is what SELECTS the new fields |
| `README.md` | edit | the receipt example in the write section |
| `scripts/contract.sh` | edit | a row driving the real binary, printing the bounds of the incident that produced this ADR |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a delete records the trimmed
   first and last line it removed; a one-line delete records the same line in
   both; `replace` and the insertions record neither.
2. [S2] Add `RemovedFirst`/`RemovedLast` to `HunkResult`, populated in the
   `delete` branch of the splice from `orig[start-1]` and `orig[end-1]`, both
   through the existing `trim`.
3. [S3] Print them on the human receipt line for a delete hunk, and only for a
   delete — the field is empty elsewhere and an empty `from "" to ""` is noise.
   The contract row in S4 is what proves this reached the CLI: a unit test on
   `HunkResult` cannot see whether anything prints it. [proof: acceptance]
4. [S4] Update the README's receipt example, and add the contract row. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -run 'Delete.*Bounds|OneLineDelete' -v 2>&1 | tee /tmp/adr008-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr008-t1.out \
  && grep -q 'removed_first' README.md \
  && go test ./... && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDeleteRecordsItsBounds` | `internal/apply/apply_test.go` | the first and last removed line are recorded and trimmed | — | S1, S2 |
| `TestDeleteBoundsAreTrimmed` | `internal/apply/apply_test.go` | the bounds go through the existing `trim`, so the receipt stays bounded | — | S1, S2 |
| `TestAOneLineDeleteRecordsTheSameLineTwice` | `internal/apply/apply_test.go` | the degenerate range | — | S1, S2 |
| `TestOnlyDeleteRecordsBounds` | `internal/apply/apply_test.go` | replace and the insertions leave the fields empty | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | `cmd/mrw/main.go`'s receipt line; deleting it leaves the contract row red |
| 3 — the caller can discover it | the README receipt example and the contract row, both asserted by the fence |
| 4 — it is used | nothing measures this yet |

## Mutation Log

- 2026-09-01 · 7b31d36* · mutant killed · exit 1 · `cmd/mrw/main.go` · nothing prints the bounds: proves the CLI receipt line, not just HunkResult, is what selects the new fields · acceptance-sha256:99ef9465fd24d520dade9618b79b62212565818e2680cd69fb7be4c9571a3c02
- 2026-09-01 · 7b31d36* · mutant killed · exit 1 · `cmd/mrw/main.go` · nothing prints the bounds: proves the CLI receipt line, not just HunkResult, is what selects the new fields · acceptance-sha256:67ac8151714a48d72624f4063e59c052dad1688b0d7f83cdbdea424d1c447306

## Invariants

- Output stays bounded: two trimmed strings per delete hunk whatever the range.
- No other op gains fields, and no existing field changes meaning.
- A plan that applies today applies identically; only the receipt is wider.

## Risks

- The bounds become noise a reader skips. The contract row prints the real
  receipt from the ADR's incident, which is the check on that.

## Stop Condition

Stop if the bounds cannot be rendered without the line becoming unreadable at
80 columns — that is the signal the runner-up (requiring a guard) is the better
answer.

## Out of Scope

- The expected body — T2 owns it.
- Requiring a guard on every delete (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-01 · 7b31d36* · exit 1 · `set -o pipefail …` · acceptance-sha256:99ef9465fd24d520dade9618b79b62212565818e2680cd69fb7be4c9571a3c02 · ms:202
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/apply/apply_test.go:566:31: res.Hunks[0].RemovedLast undefined (type HunkResult has no field or method RemovedLast)
  internal/apply/apply_test.go:569:22: res.Hunks[0].RemovedLast undefined (type HunkResult has no field or method RemovedLast)
  internal/apply/apply_test.go:570:92: res.Hunks[0].RemovedLast undefined (type HunkResult has no field or method RemovedLast)
  internal/apply/apply_test.go:586:18: res.Hunks[0].RemovedFirst undefined (type HunkResult has no field or method RemovedFirst)
  internal/apply/apply_test.go:586:54: res.Hunks[0].RemovedLast undefined (type HunkResult has no field or method RemovedLast)
  internal/apply/apply_test.go:587:60: res.Hunks[0].RemovedFirst undefined (type HunkResult has no field or method RemovedFirst)
  internal/apply/apply_test.go:587:87: res.Hunks[0].RemovedLast undefined (type HunkResult has no field or method RemovedLast)
  internal/apply/apply_test.go:587:87: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-09-01 · 7b31d36* · exit 0 · `set -o pipefail …` · acceptance-sha256:99ef9465fd24d520dade9618b79b62212565818e2680cd69fb7be4c9571a3c02 · ms:3346
- 2026-09-01 · 7b31d36* · exit 0 · `set -o pipefail …` · acceptance-sha256:67ac8151714a48d72624f4063e59c052dad1688b0d7f83cdbdea424d1c447306 · ms:2654
