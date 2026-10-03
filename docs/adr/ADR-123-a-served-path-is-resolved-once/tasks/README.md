# ADR-123 Tasks

Implementation tasks for ADR-123: a served path is resolved once. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the state base cached, Resolve's real path reused | pending | none | `docs/adr/ADR-123-a-served-path-is-resolved-once/tasks/T1-resolve.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-123 owns `internal/rooted`.
