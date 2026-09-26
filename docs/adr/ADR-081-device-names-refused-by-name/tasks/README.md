# ADR-081 Tasks

Implementation tasks for ADR-081: a Windows device name is refused by name. See the parent ADR for the decision.

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
| T1 | Every reserved device name is refused by name on Windows | done | — | `docs/adr/ADR-081-device-names-refused-by-name/tasks/T1-refused-by-name.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-081 owns `internal/rooted` only. `internal/apply`, `internal/plan`, `internal/check`, `internal/lines`, `internal/iter`, `internal/seen`, `internal/state`, `internal/subproc` and `internal/read` stay byte-identical against the merge-base.
