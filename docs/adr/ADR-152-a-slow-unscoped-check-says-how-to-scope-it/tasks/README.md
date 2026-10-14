# ADR-152 Tasks

Implementation tasks for ADR-152: a slow check that ran the whole project says how to scope it. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the check names scoped_check when a whole-project run was slow | done | none | `docs/adr/ADR-152-a-slow-unscoped-check-says-how-to-scope-it/tasks/T1-hint-at-scoped-check.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine package `internal/check` changes; this record owns it.
- Contract section §256.
