# ADR-135 Tasks

Implementation tasks for ADR-135: a walk counts the discovered names Windows will not keep. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the walk counts a discovered name Windows will not keep and says so | done | none | `docs/adr/ADR-135-a-walk-counts-the-names-windows-will-not-keep/tasks/T1-count.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/rooted` (a sentinel and one function) and `internal/read` (the count), which the record owns.
- Contract section §237.
