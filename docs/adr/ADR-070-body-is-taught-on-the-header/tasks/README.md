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
| 4 | T4 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | `mrw instructions` and the handshake show `body=` on a header; contract §132 | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T1-teach-body-on-the-header.md` fence |
| T2 | An uncounted hunk whose first body line is `body=` is refused; contract §133 | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T2-refuse-a-misplaced-body-count.md` fence |
| T3 | The blind scorer reads every BACKLOG shape; minimum one mrw call | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T3-fix-the-blind-scorer.md` fence |
| T4 | The scorer reads wrapper modes, wrapper help and expandable heredocs | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T4-scorer-wrappers-and-heredocs.md` fence |
| T5 | The scorer honours a backslash escape in an unquoted heredoc body | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T5-escaped-heredoc-substitutions.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-070 owns `internal/plan` (T2). `internal/read`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `internal/iter` stay byte-identical.
