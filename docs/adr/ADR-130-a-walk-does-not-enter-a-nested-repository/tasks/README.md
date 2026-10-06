# ADR-130 Tasks

Implementation tasks for ADR-130: a walk does not enter a nested repository. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a walk and an ast-grep judge prune a nested repository inside a checkout, and count it | done | none | `docs/adr/ADR-130-a-walk-does-not-enter-a-nested-repository/tasks/T1-nested.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Owns `internal/read/walk.go` and `internal/read/astgrep.go`.
- Contract section §233.
