# ADR-139 Tasks

Implementation tasks for ADR-139: `mrw write --create`. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | `--create PATH` compiles standard input to a create plan | done | none | `docs/adr/ADR-139-write-create-makes-one-file-from-stdin/tasks/T1-create.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/ingest` (a new function) and `cmd/mrw` (a flag), which the record owns.
- Contract section §240.
