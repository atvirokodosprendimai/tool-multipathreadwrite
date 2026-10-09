# ADR-138 Tasks

Implementation tasks for ADR-138: a walk counts the links to directories it does not follow. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the walk counts a link to a directory and says so | done | none | `docs/adr/ADR-138-a-walk-counts-the-links-to-directories-it-does-not-follow/tasks/T1-count.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/read` and the read schema the receipt test holds, which the record owns.
- Contract section §239.
