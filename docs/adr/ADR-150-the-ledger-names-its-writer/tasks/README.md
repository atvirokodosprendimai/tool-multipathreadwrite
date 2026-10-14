# ADR-150 Tasks

Implementation tasks for ADR-150: the read ledger names the mrw that wrote it, and a newer one is left alone. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the header names its writer, v4 stays current, a newer ledger is left alone and named | done | none | `docs/adr/ADR-150-the-ledger-names-its-writer/tasks/T1-name-the-writer.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine package `internal/seen` changes; this record owns it.
- Contract section §254.
