# Task ADR-002-T2: An edit to an unseen or externally-changed file is refused, naming both SHAs

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `apply.Options.Seen` (T2), `apply.Options.Force` (T2), `mrw write --force` (T2)
**Consumes:** `seen.Ledger` (T1), `seen.Load` (T1)
**Data dependency:** hermetic

## Goal

`apply.Apply` refuses to edit an existing file that is absent from the ledger or
whose current hash differs from the recorded one, reporting the reason through
the normal per-hunk receipt and aborting the whole run.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options{Seen, Force}` and the check in `planFile`, before any hunk validates |
| `internal/apply/apply_test.go` | edit | Unseen, changed, seen-and-editable, create-exempt, force-bypass, nil-disables |
| `cmd/mrw/main.go` | edit | `writeCmd` loads the ledger and adds `--force` — the flag parser is what makes the escape hatch reachable |
| `scripts/contract.sh` | edit | Row 6 asserts both refusals against the built binary |
| `README.md` | edit | The behaviour table — the declared interface a caller reads |

## Ordered Steps

1. Confirm the failing tests exist and can go red. **Retrofit note:** the
   original red run is historical; the substitute proof was performed on
   2026-08-31 — replacing the guard condition with `if false` turned
   `TestAFileNeverSeenCannotBeEdited` and
   `TestAFileChangedBehindMrwsBackCannotBeEdited` red, and running
   `scripts/contract.sh` against a binary built from that mutant reported 4
   failed assertions and exit 1. Re-record it with `adr-verify --mutant`.
2. Give `Apply` an `Options` struct rather than a fourth positional bool.
3. Check once per file, before any hunk: skip when the file does not exist
   (`create` has no existing content to be stale about), when `Seen` is nil (the
   caller is not using the guard), or when `Force` is set.
4. Report the refusal THROUGH the first hunk's verdict, so it travels in the
   same receipt as every other failure and aborts the run identically.
5. Word both messages so they name the remedy: `mrw read <path>` for unseen,
   and both short SHAs for changed.
6. Add `--force`, documented as the escape hatch rather than the habit.

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -run 'TestAFileNeverSeenCannotBeEdited|TestAFileChangedBehindMrwsBackCannotBeEdited|TestASeenFileIsEditable|TestCreateNeedsNoPriorObservation|TestForceBypassesTheGuard|TestNilLedgerDisablesTheGuard' -v 2>&1 | tee /tmp/adr002t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr002t2.out \
  && go test ./internal/apply/ \
  && ./scripts/contract.sh
```

The six new units run alone first — none of them can be satisfied by an
already-passing sibling — then the package, then the end-to-end assertions.

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestAFileNeverSeenCannotBeEdited` | `internal/apply/apply_test.go` | An empty ledger refuses, the reason says "has not been read", the file is unchanged | — |
| `TestAFileChangedBehindMrwsBackCannotBeEdited` | `internal/apply/apply_test.go` | A recorded hash that no longer matches refuses, the reason says "changed since" | — |
| `TestASeenFileIsEditable` | `internal/apply/apply_test.go` | A matching hash applies normally | — |
| `TestCreateNeedsNoPriorObservation` | `internal/apply/apply_test.go` | `create` is exempt | — |
| `TestForceBypassesTheGuard` | `internal/apply/apply_test.go` | `Force` applies despite an empty ledger | — |
| `TestNilLedgerDisablesTheGuard` | `internal/apply/apply_test.go` | A nil ledger is distinct from an empty one | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAFileNeverSeenCannotBeEdited` |
| 2 — something selects it | `cmd/mrw/main.go` `writeCmd` passes `seen.Load(root)` into `apply.Options`; `scripts/contract.sh` row 6 fails if it passes nil, which is exactly what the `if false` mutation demonstrated |
| 3 — the caller can discover it | `README.md`'s behaviour table and `mrw write --help`'s `--force` usage string; the refusal message itself names the remedy |
| 4 — it is used | It fired three times during its own construction on real mistakes; nothing counts refusals |

## Mutation Log

- 2026-08-31 · f679354* · mutant killed · exit 1 · `internal/apply/apply.go` · Disables the read-before-modify guard entirely. A security control that cannot fail is worse than none, and this is the exact mutation performed by hand on 2026-08-31 — recorded here by the tool instead. · acceptance-sha256:1a164fa00b6b0a3cafe657d5e120f4917184af27b7678ef7452d851a2daf6897

## Invariants

- A nil `Seen` map disables the check; an EMPTY map enforces it against nothing
  seen. The two must stay distinct — collapsing them silently disarms the guard
  for every caller that passes no options.
- `create` is never blocked by this check.
- The refusal aborts the whole run, like every other hunk failure — it is not a
  per-file skip.

## Risks

- A caller who hits the refusal reaches for `--force` instead of `--stat`.
  Mitigated by message wording and by the skill; not mechanically preventable,
  and deliberately so.

## Stop Condition

Stop if the check needs to become advisory-by-default (warn rather than refuse).
That inverts the decision in the parent ADR and needs a superseding record, not
a flag flip.

## Out of Scope

- Locking or concurrent-writer protection (permanent: detection after the fact
  is the decision; prevention is a different tool)
- `mrw forget <path>` to drop an entry (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:1a164fa00b6b0a3cafe657d5e120f4917184af27b7678ef7452d851a2daf6897
