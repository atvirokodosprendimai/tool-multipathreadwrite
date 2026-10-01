# ADR-112 Tasks

Implementation tasks for ADR-112: a check reports what changed under it. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the files a write touched that changed since are named | done | none | `docs/adr/ADR-112-a-check-reports-what-changed-under-it/tasks/T1-drift.md` fence |
| T2 | a write reports what changed while its check ran | done | T1 | `docs/adr/ADR-112-a-check-reports-what-changed-under-it/tasks/T2-write-reports-drift.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-112 owns no engine package; every one stays byte-identical.
- Contract section §211 (T2).
