# ADR-034 Tasks

Implementation tasks for ADR-034: State that names a checkout nobody has is removable, and only when
asked. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T4 | T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The state base can describe itself, and drop only its dead | done | — | `go test ./internal/state/ -run 'TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned…' …` |
| T2 | `mrw seen` reports the base, and prunes it when asked | done | — | `go test ./cmd/mrw/ -run 'TestSeenPruneRemovesOnlyTheDeadEntries…' …` |
| T3 | The contract drives the prune, and the docs say it exists | done | — | `grep -q '^# 71\. ' scripts/contract.sh && ./scripts/contract.sh` |
| T4 | The gate stops producing what the prune removes | done | — | a full `contract.sh` run leaves the real state base's entry count unchanged |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `state.Entry`, `state.Entries()`, `state.Prune()` | T2 | T1 before T2 — the flag has nothing to call otherwise |
| T2 | `mrw seen --prune`, `--dry-run`, the count line | T3 | T2 before T3 — the row drives the built binary |
| T3 | §71, which pins its own `XDG_STATE_HOME` under `$WORK` | T4 | T3 before T4 — T4 pins the same variable for the whole script and must not move §71's verdict |

## Notes

- ⚠ **A one-entry fixture is green against a prune that deletes everything.** Every fixture in T1,
  T2 and §71 holds a LIVE entry, a DEAD entry and an UNIDENTIFIABLE entry, and asserts the first and
  third survive. This is the parent record's High-likelihood risk and it is pre-registered because
  it is the default way to get this wrong.
- ⚠ **Assert the filesystem, not the report.** A command that prints the right sentence and deletes
  nothing passes an output check. T2's S1 says so explicitly.
- ⚠ **The state directory stays the FIRST line of `mrw seen`.** `scripts/contract.sh:1590` reads it
  with `head -1`; the count line goes after it and §71 pins that.
- Nothing but `--prune` reaches `state.Prune`. There is no automatic reaper, which is the objection
  `docs/adr/BACKLOG.md:112` raised when ADR-004 deferred this and which this record honours rather
  than overturns.
- This record owns `internal/state` and `cmd/mrw`. `internal/read`, `internal/apply`,
  `internal/plan`, `internal/seen` and `internal/check` stay byte-identical against the merge base —
  which is why `internal/state` is the one package absent from each fence's go/no-go clause.
