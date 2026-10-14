# ADR-153 Tasks

Implementation tasks for ADR-153: a grep can be counted without being served. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | `--count` on the CLI and `count` on `mrw_read`, and `--max-lines` taught in place of a pipe into head | done | none | `docs/adr/ADR-153-a-grep-can-be-counted/tasks/T1-count.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine package `internal/read` changes; this record owns it.
- Contract section §257.
