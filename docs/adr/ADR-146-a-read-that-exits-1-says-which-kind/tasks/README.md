# ADR-146 Tasks

Implementation tasks for ADR-146: a read that exits 1 says which kind. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | The exit-1 line of a read starts with its kind | pending | none | `docs/adr/ADR-146-a-read-that-exits-1-says-which-kind/tasks/T1-prefix.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `cmd/mrw`.
- Contract section §250.
