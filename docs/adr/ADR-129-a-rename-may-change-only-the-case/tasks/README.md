# ADR-129 Tasks

Implementation tasks for ADR-129: a rename may change only the case of a name. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a case-only rename applies; a hard link and a directory-only respelling stay refused | done | none | `docs/adr/ADR-129-a-rename-may-change-only-the-case/tasks/T1-respell.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Owns `internal/apply/pathop.go`.
- Contract section §230.
