# ADR-136 Tasks

Implementation tasks for ADR-136: reads say less. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the neighbour note once, the stale notice short, silent on commands without a ledger | done | none | `docs/adr/ADR-136-reads-say-less/tasks/T1-quiet.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/read` (one local), `internal/seen` (one constant) and `cmd/mrw` (one condition), which the record owns.
- Contract section §237.
