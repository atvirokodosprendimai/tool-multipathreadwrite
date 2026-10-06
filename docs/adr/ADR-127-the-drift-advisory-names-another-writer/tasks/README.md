# ADR-127 Tasks

Implementation tasks for ADR-127: the drift advisory names another writer's write. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | every landed write is counted, and a check that saw others says how many | done | none | `docs/adr/ADR-127-the-drift-advisory-names-another-writer/tasks/T1-writes.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No engine package changes.
- Contract section §225.
