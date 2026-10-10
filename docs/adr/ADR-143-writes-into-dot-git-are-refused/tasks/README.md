# ADR-143 Tasks

Implementation tasks for ADR-143: writes into `.git` are refused. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | A write whose path lands in `.git` is refused | done | none | `docs/adr/ADR-143-writes-into-dot-git-are-refused/tasks/T1-refuse.md` fence |
| T2 | A path reopened by name is judged for `.git` again | pending | T1 | `docs/adr/ADR-143-writes-into-dot-git-are-refused/tasks/T2-reopen.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/rooted` and `internal/apply`.
- Contract section §246.
