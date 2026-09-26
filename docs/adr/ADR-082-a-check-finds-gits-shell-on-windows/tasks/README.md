# ADR-082 Tasks

Implementation tasks for ADR-082: a check finds Git's shell on Windows. See the parent ADR for the decision.

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
| T1 | A check finds Git's shell on Windows | done | — | `docs/adr/ADR-082-a-check-finds-gits-shell-on-windows/tasks/T1-git-sh-on-windows.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-082 owns `internal/check`. `internal/apply`, `internal/plan`, `internal/read`, `internal/rooted`, `internal/lines`, `internal/iter`, `internal/seen`, `internal/state` and `internal/subproc` stay byte-identical against the merge-base.
