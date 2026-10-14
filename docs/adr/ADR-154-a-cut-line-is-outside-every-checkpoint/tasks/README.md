# ADR-154 Tasks

Implementation tasks for ADR-154: mrw_read takes max_cols, and a line it served cut is outside every checkpoint. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | max_cols on mrw_read, and checkpoints that bracket only recorded lines | pending | none | `docs/adr/ADR-154-a-cut-line-is-outside-every-checkpoint/tasks/T1-cut-lines-outside-checkpoints.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- `internal/mcp` only; no engine package changes.
- Contract section §258.
