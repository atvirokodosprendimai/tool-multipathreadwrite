# ADR-068 Tasks

Implementation tasks for ADR-068: the read ledger keeps a path exactly as it was served. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | `parseLine` keeps a path's edge spaces; contract §127 | done | — | `docs/adr/ADR-068-the-read-ledger-keeps-a-path-exactly/tasks/T1-parse-line-keeps-the-path.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-068 owns `internal/seen`. `internal/read`, `internal/apply`, `internal/plan`, `internal/check`, `internal/state`, `internal/lines` and `internal/iter` stay byte-identical.
