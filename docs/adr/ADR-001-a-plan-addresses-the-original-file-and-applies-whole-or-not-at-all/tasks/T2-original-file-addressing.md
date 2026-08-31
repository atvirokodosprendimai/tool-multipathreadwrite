# Task ADR-001-T2: Every address resolves against the original file, so earlier hunks never shift later ones

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `apply.Apply` (T2), `apply.Input` (T2), `apply.Result` (T2)
**Consumes:** `plan.Hunk` (T1)
**Data dependency:** hermetic

## Goal

`apply.Apply` splices a file once, resolving every hunk's address against the
file as it was read — so a plan is written top-to-bottom and no hunk's address
depends on what the hunks before it did.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | add | The splicer: resolve, order, walk the original once with a cursor |
| `internal/apply/apply_test.go` | add | The no-shift property, EOF addressing, newline and permission preservation |
| `cmd/mrw/main.go` | edit | `writeCmd` maps `plan.Hunk` to `apply.Input` — the call site that SELECTS this engine |

## Ordered Steps

1. Confirm the failing test for the no-shift property exists and can go red.
   **Retrofit note:** the original red run is historical (`676296e`); the
   accepted substitute is `adr-verify --mutant` recording `killed` against the
   cursor arithmetic in `internal/apply/apply.go`.
2. Resolve `EOF` sentinels against the file's real length.
3. Normalise every op to a position in the ORIGINAL line numbering: an
   insertion becomes a zero-width edit before line `p`, a replace/delete
   consumes `[start,end]`.
4. Sort by that position — insertions before consuming ops at the same
   position, ties broken by plan order — then walk the original once with a
   cursor, emitting untouched lines, bodies, and skipping consumed ranges.
5. Reject a hunk whose start is behind the cursor as an overlap.
6. Preserve the file's trailing-newline state and its permission bits; write
   through a temp file and `rename` so a crash cannot leave a half-written file.

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -run 'TestEarlierHunksNeverShiftLaterAddresses|TestAddressesResolveAgainstTheOriginalFile|TestEOFAddressing|TestOverlappingHunksAreRejected|TestTrailingNewlineIsPreserved|TestFilePermissionsSurvive|TestMultipleFilesInOneRun' -v 2>&1 | tee /tmp/adr001t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr001t2.out \
  && go test ./internal/apply/
```

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestEarlierHunksNeverShiftLaterAddresses` | `internal/apply/apply_test.go` | Two inserts with a DELETE between them — the case that actually moves lines — each land after the original line named | — |
| `TestAddressesResolveAgainstTheOriginalFile` | `internal/apply/apply_test.go` | Replace 3-6 and insert-after 2 in one pass | — |
| `TestEOFAddressing` | `internal/apply/apply_test.go` | `$` resolves to the real last line | — |
| `TestOverlappingHunksAreRejected` | `internal/apply/apply_test.go` | Two overlapping replaces fail and write nothing | — |
| `TestTrailingNewlineIsPreserved` | `internal/apply/apply_test.go` | A file without a trailing newline does not gain one | — |
| `TestFilePermissionsSurvive` | `internal/apply/apply_test.go` | 0755 survives the temp-file-and-rename | — |
| `TestMultipleFilesInOneRun` | `internal/apply/apply_test.go` | Delete, insert and create across three paths in one call | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAddressesResolveAgainstTheOriginalFile` |
| 2 — something selects it | `cmd/mrw/main.go` `writeCmd` calls `apply.Apply`; `scripts/contract.sh` row 2 fails if that call is removed |
| 3 — the caller can discover it | `mrw write --help` states that addresses resolve against the original file |
| 4 — it is used | Every multi-site edit in this repository's own development; nothing counts them |

## Mutation Log

- 2026-08-31 · f679354* · mutant killed · exit 1 · `internal/apply/apply.go` · Off-by-one in the cursor walk. If the no-shift tests do not bind to the splice arithmetic, this survives and every multi-hunk edit drops a line. · acceptance-sha256:2aeb304873631a58f1652db1b9ca6b3d8ba28e5f1fba51db927a5ffa6c0cf5d3
- 2026-08-31 · f679354* · mutant killed · exit 1 · `internal/apply/apply.go` · Off-by-one in the cursor walk. If the no-shift tests do not bind to the splice arithmetic, this survives and every multi-hunk edit drops a line. Re-recorded after the Acceptance filter was widened, which invalidated the prior digest. · acceptance-sha256:968937937de36696feb3d9b891d9859d44c51286165144273b1756e3f83ddb07

## Invariants

- `internal/apply` does not import `internal/plan` — the engine is testable with
  hand-built hunks and the format can change without touching the write path.
- A file's trailing-newline state and permission bits survive an edit.
- A write is atomic per file: temp file plus `rename`, never an in-place truncate.

## Risks

- Ordering two hunks that begin at the same position is a judgement, not a law.
  Mitigated by the documented rule (insertions first, then plan order) and by
  rejecting a true overlap rather than guessing.

## Stop Condition

Stop if a caller needs hunks to observe each other's effects — that is a
different execution model (sequential rewrite), it contradicts this ADR's first
decision point, and it needs its own record.

## Out of Scope

- Refusing to edit a file that was not read first — that is ADR-002.
- Running the project's tests after a write — that is ADR-003.

## Verification Log
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aeb304873631a58f1652db1b9ca6b3d8ba28e5f1fba51db927a5ffa6c0cf5d3
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:968937937de36696feb3d9b891d9859d44c51286165144273b1756e3f83ddb07
