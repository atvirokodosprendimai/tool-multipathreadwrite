# ADR-084 Tasks

Implementation tasks for ADR-084: what blind reading 03 taught. See the parent ADR for the decision.

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
| T1 | The -M refusal names the form; the text teaches a write's exits and --exclude pruning | done | — | `docs/adr/ADR-084-what-blind-reading-03-taught/tasks/T1-refusal-and-teaching.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-084 owns `internal/plan` only. `internal/read`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` and `internal/subproc` stay byte-identical.
