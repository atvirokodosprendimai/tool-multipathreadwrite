# ADR-090 Tasks

Implementation tasks for ADR-090: the shipped examples run on a tree they did not write. See the
parent ADR for the decision.

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
| T1 | The examples run on a fixture | done | — | `docs/adr/ADR-090-the-shipped-examples-run-on-a-tree-they-did-not-write/tasks/T1-the-examples-run-on-a-fixture.md` fence |
| T2 | The description walk names its sentinels | done | — | `docs/adr/ADR-090-the-shipped-examples-run-on-a-tree-they-did-not-write/tasks/T2-the-walk-names-its-sentinels.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No engine package changes; `internal/read`, `apply`, `plan`, `seen`, `check` and `state` stay
  byte-identical against the merge-base.
