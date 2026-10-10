# ADR-144 Tasks

Implementation tasks for ADR-144: a damaged read ledger is told once. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | The CLI tells a ledger it had to ignore lines of | done | none | `docs/adr/ADR-144-a-damaged-read-ledger-is-told-once/tasks/T1-notice.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/seen` and `cmd/mrw`.
- Contract section §248.
