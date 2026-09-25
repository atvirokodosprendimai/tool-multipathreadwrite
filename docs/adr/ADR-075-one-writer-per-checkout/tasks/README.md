# ADR-075 Tasks

Implementation tasks for ADR-075: one writer per checkout. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | One writer per checkout; contract §150 | done | — | `docs/adr/ADR-075-one-writer-per-checkout/tasks/T1-one-writer-per-checkout.md` fence |
| T2 | The surfaces and the records say what T1 made true | done | — | `docs/adr/ADR-075-one-writer-per-checkout/tasks/T2-the-surfaces-say-what-t1-made-true.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-075 owns `internal/seen/seen.go` and the new `internal/writer`. `internal/apply`, `internal/plan`, `internal/read`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted`, `internal/subproc` and the rest of `internal/seen` stay byte-identical.
