# Task ADR-002-T1: A ledger records what mrw last observed each file to hold

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `seen.Ledger` (T1), `seen.Load` (T1), `seen.Record` (T1), `seen.SHA` (T1)
**Consumes:** none
**Data dependency:** hermetic

## Goal

`internal/seen` persists path → SHA-256 in `.mrw/seen`, merging observations
rather than replacing them, and both `mrw read` and `mrw write` record into it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | add | Load, Record (merge), Forget, SHA |
| `internal/seen/seen_test.go` | add | Round trip, merge semantics, missing-file case |
| `internal/read/read.go` | edit | `Run` returns the SHAs it served — reading a file IS the observation |
| `internal/read/read_test.go` | edit | The changed return |
| `cmd/mrw/main.go` | edit | `readCmd` and `writeCmd` call `seen.Record` — the call sites that SELECT the ledger; without them the package is dead code |
| `.gitignore` | edit | `/.mrw/` — per-developer state, anchored so it cannot swallow a like-named path component |

## Ordered Steps

1. Confirm the failing tests for the ledger exist and can go red. **Retrofit
   note:** the original red run is historical (`829aae7`); the substitute proof
   is `adr-verify --mutant` recording `killed`.
2. Define the on-disk form: `<sha>  <path>` per line, sorted, so it diffs
   cleanly and two runs produce identical bytes.
3. `Load` returns an empty ledger for a missing file — no observations yet is
   the normal starting state, not a fault.
4. `Record` MERGES: paths absent from the argument keep their recorded value, so
   one command observing two files cannot erase what another observed about a
   third.
5. Change `read.Run` to return the observed SHAs alongside its problem count,
   and record them in `readCmd` — including under `--stat`, since the hash is
   the staleness proof and content is not needed to establish it.
6. Record the resulting SHAs in `writeCmd` after a successful apply.

## Acceptance

```bash
set -o pipefail
go test ./internal/seen/ -v 2>&1 | tee /tmp/adr002t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr002t1.out \
  && go test ./internal/seen/ ./internal/read/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestRoundTrip` | `internal/seen/seen_test.go` | Record then Load returns what was recorded | — |
| `TestMissingLedgerIsEmptyNotAnError` | `internal/seen/seen_test.go` | No file means an empty ledger, not a failure | — |
| `TestRecordMergesRatherThanReplaces` | `internal/seen/seen_test.go` | Recording b.go does not drop a.go | — |
| `TestRecordOverwritesTheSamePath` | `internal/seen/seen_test.go` | A second observation of one path replaces the first, leaving one row | — |
| `TestForget` | `internal/seen/seen_test.go` | Forget drops named paths and reports the count; an unrecorded path is not a removal | — |
| `TestSHAIsStable` | `internal/seen/seen_test.go` | Deterministic, and a one-byte change changes the hash | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestRoundTrip` |
| 2 — something selects it | `cmd/mrw/main.go` calls `seen.Record` in both `readCmd` and `writeCmd`; deleting either makes `scripts/contract.sh` row 6 fail, because a file read would no longer authorise a write |
| 3 — the caller can discover it | `.mrw/seen` is plain text and documented in the skill; `mrw read --stat` is the documented cheap way to write to it |
| 4 — it is used | Every `mrw write` in this repository's development authorises through it; nothing counts them |

## Mutation Log

## Invariants

- `Record` never drops a path it was not asked about.
- The file is sorted, so the bytes are stable across runs.
- `seen.SHA` is the ONLY hash used for the ledger — two hashes of the same bytes
  disagreeing would make every write look stale.

## Risks

- `seen.Forget` is written but no CLI subcommand calls it: dead code until
  wired. Recorded in the ADR's Out of Scope as deferred, so it is visible rather
  than quietly present.

## Stop Condition

Stop if the ledger needs to hold anything other than a hash per path — a
timestamp, a tool version, a lock. Each of those is a different decision about
what staleness means, and belongs in a record rather than in a field.

## Out of Scope

- The refusal itself — that is T2.
- Pruning entries for deleted files (deferred: docs/adr/BACKLOG.md)

## Verification Log
