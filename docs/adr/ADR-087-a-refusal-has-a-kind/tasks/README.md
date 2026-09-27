# ADR-087 Tasks

Implementation tasks for ADR-087: a refusal the parser and the engine share has a kind. See the parent ADR for the decision.

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
| T1 | Kinds on the parser, the engine and the ack remedy | done | — | `docs/adr/ADR-087-a-refusal-has-a-kind/tasks/T1-kinds.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-087 owns `internal/plan` and `internal/apply`. `internal/read`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` and `internal/subproc` stay byte-identical.
