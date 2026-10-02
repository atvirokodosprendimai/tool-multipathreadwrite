# ADR-119 Tasks

Implementation tasks for ADR-119: a write says when its shape looks wrong. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the closer hint and its receipt keys | done | none | `docs/adr/ADR-119-a-write-says-when-its-shape-looks-wrong/tasks/T1-hints.md` fence |
| T2 | the measurement against the pre-registered bar | done | T1 | `docs/adr/ADR-119-a-write-says-when-its-shape-looks-wrong/tasks/T2-measure.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-119 owns `internal/apply`.
- Contract section §219.
