# ADR-155 Tasks

Implementation tasks for ADR-155: a read can address a JSON value by its path. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a JSON path serves its value's lines, fails as a missed range, and a plan refuses it | done | none | `docs/adr/ADR-155-a-read-can-address-a-json-value/tasks/T1-json-path.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine packages `internal/read` and `internal/plan` change; this record owns both.
- Contract section §259.
