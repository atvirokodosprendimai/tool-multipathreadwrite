# Task ADR-001-T3: One failed hunk aborts the run, and every hunk and every addressed file is reported

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `apply.HunkResult` (T3), `apply.FileResult` (T3), `mrw write --json` receipt (T3)
**Consumes:** `plan.Hunk` (T1), `apply.Apply` (T2)
**Data dependency:** hermetic

## Goal

Nothing is written unless every hunk validates; each hunk reports `ok`,
`failed` with a reason, or `skipped`; and every file the plan addressed appears
in the receipt whether or not it was written.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The validate-all pass, the per-hunk verdict map, and the addressed-file accounting |
| `internal/apply/apply_test.go` | edit | The abort behaviour, the verdicts, and the receipt's file list |
| `cmd/mrw/main.go` | edit | `report` renders the verdicts and `--json` emits them — the surface that makes the verdict reachable |
| `scripts/contract.sh` | add | Asserts the abort end-to-end against the built binary, in a throwaway repo |

## Ordered Steps

1. Confirm the failing tests exist and can go red. **Retrofit note:** the
   original red run is historical; the accepted substitute is
   `adr-verify --mutant` recording `killed` against the abort branch. This was
   performed by hand on 2026-08-31 for the addressed-file accounting: stubbing
   `failed = append(failed, fr)` turned `TestFailedFilesStillAppearInTheReceipt`
   and `TestAFailedFileIsReportedOnceNotPerHunk` red with the exact symptom.
2. Split validation from writing: compute every file's new content and verdicts
   first, write only if `Failed == 0`.
3. Record a verdict for every hunk, echoing the op and address the CALLER wrote
   rather than the resolved form, so a report line matches the plan line.
4. Downgrade surviving `ok` verdicts to `skipped` when any sibling failed.
5. Report every ADDRESSED file, in plan order, including those whose hunks
   failed — carrying the state observed about them so "could not validate" is
   distinguishable from "never looked at".
6. Add `scripts/contract.sh` and wire it into CI beside `go test`.

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -run 'TestOneBadHunkAbortsEverythingAndSaysWhich|TestFailedFilesStillAppearInTheReceipt|TestAFailedFileIsReportedOnceNotPerHunk|TestDryRunWritesNothingButReportsTheOutcome|TestPreconditions' -v 2>&1 | tee /tmp/adr001t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr001t3.out \
  && go test ./internal/apply/ \
  && ./scripts/contract.sh
```

The unit tests run alone first, then the package, then the end-to-end contract
script — chained with `&&` so none can carry the verdict alone.

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestOneBadHunkAbortsEverythingAndSaysWhich` | `internal/apply/apply_test.go` | One bad anchor of three: the offender is `failed` with a reason, siblings are `skipped`, both files unchanged | — |
| `TestFailedFilesStillAppearInTheReceipt` | `internal/apply/apply_test.go` | A two-file plan with one failing file reports BOTH files, in plan order, none written | — |
| `TestAFailedFileIsReportedOnceNotPerHunk` | `internal/apply/apply_test.go` | Three hunks on one file, two failing: one file result, two failures | — |
| `TestDryRunWritesNothingButReportsTheOutcome` | `internal/apply/apply_test.go` | `--dry-run` computes the resulting sha without writing | — |
| `TestPreconditions` | `internal/apply/apply_test.go` | Each of sha/lines/anchor/bounds/missing-file/create-exists fails its hunk | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestOneBadHunkAbortsEverythingAndSaysWhich` |
| 2 — something selects it | `cmd/mrw/main.go` `report`/`--json` render the verdicts; `scripts/contract.sh` asserts "the offender is named" and "siblings report skip" against the binary, so deleting the rendering fails CI |
| 3 — the caller can discover it | The `--json` receipt schema is the declared interface; `scripts/contract.sh` reads `files[]` and `hunks[]` by name |
| 4 — it is used | The receipt is designed for a hook to read; no hook consumes it yet — nothing measures this |

## Mutation Log

- 2026-08-31 · f679354* · mutant killed · exit 1 · `internal/apply/apply.go` · Drops the addressed-but-failed file from the receipt. This is the defect found in review on 2026-08-31: a two-file plan reported one file and --json omitted the failing one, under-reporting blast radius. · acceptance-sha256:a238f1be1299a9e7197c07858dc5cb53d98d31c5413aa610ceb0c033cf203635
- 2026-08-31 · f679354* · mutant killed · exit 1 · `internal/apply/apply.go` · Drops the addressed-but-failed file from the receipt — the defect found in review on 2026-08-31. Re-recorded after the Acceptance filter was widened, which invalidated the prior digest. · acceptance-sha256:0e5796d88bde0be19d848ca0689963c885d8b33547904ebbff29147f90a32c7b

## Invariants

- `Applied` is false whenever any hunk failed.
- A hunk is never reported `ok` unless its file was actually written (or the run
  was a dry run, which says so).
- Every path named by any hunk appears exactly once in `Files`.

## Risks

- The receipt is a public contract the moment a hook reads it; changing a field
  name silently breaks that consumer. Mitigated by `scripts/contract.sh` reading
  the JSON by field name, so a rename fails CI.

## Stop Condition

Stop if a caller needs partial application ("apply what you can") — that
directly contradicts this ADR's second decision point and needs its own record,
not a flag.

## Out of Scope

- Reverting a write after a failed check — ADR-003 decides that, and decides
  against it.
- Refusing to edit an unseen file — ADR-002.

## Verification Log
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:a238f1be1299a9e7197c07858dc5cb53d98d31c5413aa610ceb0c033cf203635
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:0e5796d88bde0be19d848ca0689963c885d8b33547904ebbff29147f90a32c7b
