# ADR-004 Tasks

Implementation tasks for ADR-004: mrw leaves nothing in the working tree. See
the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

**Unlike ADR-001 to ADR-003, this one is NOT a retrofit.** Both tasks began with
a genuine TDD-red run — `go test ./internal/state/` failed to build before the
package existed — so the mutation here is the usual second proof rather than a
substitute for a red that already happened. Both are `done` as of 2026-08-31.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A per-checkout state directory resolves outside the working tree | done | — | `go test ./internal/state/ -v … && go test ./internal/state/` |
| T2 | The ledger and the working set move to the state directory, and `mrw seen` shows where | done | — | `go test ./internal/seen/ ./internal/iter/ -v … && go test ./... && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `state.Dir`, `state.Migrate` | T2 | T1 before T2 — T2 asks T1 where to store |

## Notes

- The property worth protecting is stated as one test rather than a list of
  paths: `TestNoStateIsWrittenUnderTheRepoRoot` asserts the root contains only
  what a plan wrote. A list of forbidden filenames would need updating every
  time the tool learns to store something new; the property does not.
- T2 keeps reading a legacy in-tree `.mrw/seen` when the state directory has
  none. That is deliberate compatibility, not indecision — deleting or ignoring
  data a caller already has would make the fix worse than the bug.
