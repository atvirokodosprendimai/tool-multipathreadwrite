# ADR-116 Tasks

Implementation tasks for ADR-116: a walk honours .gitignore and skips binaries. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the walk honours .gitignore, skips binaries, and says so | done | none | `docs/adr/ADR-116-a-walk-honours-gitignore-and-skips-binaries/tasks/T1-walk.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-116 owns `internal/read`.
