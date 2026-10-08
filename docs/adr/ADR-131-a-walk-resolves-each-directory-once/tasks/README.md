# ADR-131 Tasks

Implementation tasks for ADR-131: a walk resolves each directory once. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a per-walk Resolver | partial | none | `docs/adr/ADR-131-a-walk-resolves-each-directory-once/tasks/T1-resolver.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-131 owns `internal/rooted` and `internal/read`.
