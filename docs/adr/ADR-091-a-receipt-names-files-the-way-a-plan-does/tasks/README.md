# ADR-091 Tasks

Implementation tasks for ADR-091: a receipt names files the way a plan does. See the parent ADR for
the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The read receipt's keys are slash-spelled | done | — | `docs/adr/ADR-091-a-receipt-names-files-the-way-a-plan-does/tasks/T1-slash-keys.md` fence |
| T2 | The write receipt's paths are slash-spelled | done | — | `docs/adr/ADR-091-a-receipt-names-files-the-way-a-plan-does/tasks/T2-write-receipt-paths.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No engine package changes; `internal/read`, `apply`, `plan`, `seen`, `check` and `state` stay
  byte-identical against the merge-base.
