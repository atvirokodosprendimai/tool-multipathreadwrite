# ADR-072 Tasks

Implementation tasks for ADR-072: the exit code and the receipt agree with the tree. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T4 | none |
| 2 | T3 | T1 |
| 3 | T2 | T1, T3 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T4 | none |
| 2 | T1 | none |
| 3 | T3 | T1 |
| 4 | T2 | T1, T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The harness is read before anything is written | done | — | `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree/tasks/T1-harness-read-before-any-write.md` fence |
| T2 | The receipt is printed before the check runs | done | — | `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree/tasks/T2-receipt-printed-before-check.md` fence |
| T3 | `--json` is JSON on every refusal after the plan is named | done | — | `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree/tasks/T3-json-is-json-on-every-refusal.md` fence |
| T4 | A check is stopped with its descendants | done | — | `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree/tasks/T4-a-check-is-stopped-with-its-descendants.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-072 owns `internal/check`. `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`, `internal/iter` and `internal/rooted` stay byte-identical.
