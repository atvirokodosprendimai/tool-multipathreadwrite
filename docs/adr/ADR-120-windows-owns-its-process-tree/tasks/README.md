# ADR-120 Tasks

Implementation tasks for ADR-120: Windows owns its process tree. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the job sequence, kernel32 behind it, and the Windows grandchild tests | pending | none | `docs/adr/ADR-120-windows-owns-its-process-tree/tasks/T1-job.md` fence |
| T2 | the docs say a Windows child stops with mrw | pending | T1 | `docs/adr/ADR-120-windows-owns-its-process-tree/tasks/T2-docs.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-120 owns no engine package; `internal/check` changes in a test file only.
- No contract section: see the record's Out of Scope.
