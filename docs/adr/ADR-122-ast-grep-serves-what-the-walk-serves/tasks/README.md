# ADR-122 Tasks

Implementation tasks for ADR-122: ast-grep serves what the walk serves. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | ast-grep's hits pass the walk's rules | done | none | `docs/adr/ADR-122-ast-grep-serves-what-the-walk-serves/tasks/T1-filter.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-122 owns `internal/read`.
- Contract section §221.
