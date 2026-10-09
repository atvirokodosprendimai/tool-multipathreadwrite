# ADR-137 Tasks

Implementation tasks for ADR-137: a read can cut a long line to a window around the match. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | `--max-cols` cuts a long line to a window and a cut line licenses nothing | pending | none | `docs/adr/ADR-137-a-read-can-cut-a-long-line-to-a-window/tasks/T1-maxcols.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/read` and `cmd/mrw`, which the record owns.
- Contract section §238.
