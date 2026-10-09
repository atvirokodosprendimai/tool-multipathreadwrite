# ADR-134 Tasks

Implementation tasks for ADR-134: a hard link to mrw's own state is not served. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a file that is the same file as a state file is refused at the boundary | done | none | `docs/adr/ADR-134-a-hard-link-to-mrws-state-is-not-served/tasks/T1-hardlink.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches one engine package: `internal/rooted` (the boundary), which the record owns.
- Contract section §236.
