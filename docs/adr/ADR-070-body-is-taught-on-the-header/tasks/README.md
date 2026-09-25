# ADR-070 Tasks

Implementation tasks for ADR-070: `body=` is taught on the header, and a body line that is a `body=` is refused. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

T1–T3 are independent. T4 amends the T3 scorer after the Codex review of v1.25.0; T5, T6 and T7
each amend the one before after a round of the Codex review of PR #222.

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3, T4 | none |
| 2 | T5 | T4 |
| 3 | T6 | T5 |
| 4 | T7 | T6 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | none |
| 5 | T5 | T4 |
| 6 | T6 | T5 |
| 7 | T7 | T6 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | `mrw instructions` and the handshake show `body=` on a header; contract §132 | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T1-teach-body-on-the-header.md` fence |
| T2 | An uncounted hunk whose first body line is `body=` is refused; contract §133 | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T2-refuse-a-misplaced-body-count.md` fence |
| T3 | The blind scorer reads every BACKLOG shape; minimum one mrw call | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T3-fix-the-blind-scorer.md` fence |
| T4 | The scorer reads wrapper modes, wrapper help and expandable heredocs | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T4-scorer-wrappers-and-heredocs.md` fence |
| T5 | The scorer honours a backslash escape in an unquoted heredoc body | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T5-escaped-heredoc-substitutions.md` fence |
| T6 | The scorer masks an escaped pair without joining its neighbours, and reads `env -S` as a command line | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T6-escape-masking-and-env-split-string.md` fence |
| T7 | `env -S` is expanded before help detection in every spelling; the escape mask keeps a substitution's interior | done | — | `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/T7-split-string-expansion-and-a-same-length-mask.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-070 owns `internal/plan` (T2). `internal/read`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `internal/iter` stay byte-identical.
