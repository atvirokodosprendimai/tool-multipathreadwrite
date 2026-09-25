# ADR-070 Tasks

Implementation tasks for ADR-070: `body=` is taught on the header, and a body line that is a `body=` is refused. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | `mrw instructions` and the handshake show `body=` on a header; contract §132 | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T1-teach-body-on-the-header.md` fence |
| T2 | An uncounted hunk whose first body line is `body=` is refused; contract §133 | pending | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T2-refuse-a-misplaced-body-count.md` fence |
| T3 | The blind scorer reads every BACKLOG shape; minimum one mrw call | pending | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T3-fix-the-blind-scorer.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-070 owns `internal/plan` (T2). `internal/read`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `internal/iter` stay byte-identical.
