# ADR-080 Tasks

Implementation tasks for ADR-080: a check and a finder leave nothing running and name what they kept. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | Nothing mrw starts outlives the call; contract §162 | done | — | `docs/adr/ADR-080-children-and-logs/tasks/T1-reap-after-every-exit.md` fence |
| T2 | A check interrupted before it starts says so | done | — | `docs/adr/ADR-080-children-and-logs/tasks/T2-interrupted-before-it-starts.md` fence |
| T3 | A kept log is named, and old ones go; contract §163 | done | — | `docs/adr/ADR-080-children-and-logs/tasks/T3-logs-named-and-pruned.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-080 owns `internal/subproc`, `internal/check/check.go` and one call in `internal/read/astgrep.go`. `internal/apply`, `internal/plan`, `internal/lines`, `internal/iter`, `internal/seen`, `internal/state` and `internal/rooted` stay byte-identical.
