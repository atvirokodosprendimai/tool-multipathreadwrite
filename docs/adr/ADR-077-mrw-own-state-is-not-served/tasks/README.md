# ADR-077 Tasks

Implementation tasks for ADR-077: mrw's own state is never served as the caller's file. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | mrw's state is refused at the boundary; contract §154 | done | — | `docs/adr/ADR-077-mrw-own-state-is-not-served/tasks/T1-state-refused-at-the-boundary.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-077 owns `internal/state/state.go`, `internal/state/prune.go` and one hunk of `internal/read/astgrep.go`. `internal/apply`, `internal/plan`, `internal/check`, `internal/lines`, `internal/iter`, `internal/seen` and `internal/subproc` stay byte-identical.
