# ADR-079 Tasks

Implementation tasks for ADR-079: mrw's bookkeeping survives racing processes and says what happened. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | One lock per state file; contract §160 | done | — | `docs/adr/ADR-079-bookkeeping-survives-racing-processes/tasks/T1-one-lock-per-state-file.md` fence |
| T2 | A dry run is not a landing; contract §161 | done | — | `docs/adr/ADR-079-bookkeeping-survives-racing-processes/tasks/T2-a-dry-run-is-not-a-landing.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-079 owns `internal/state` (the lock) and `internal/seen/seen.go` with its lock files. `internal/apply`, `internal/plan`, `internal/check`, `internal/lines`, `internal/subproc`, `internal/rooted` and `internal/read` stay byte-identical.
