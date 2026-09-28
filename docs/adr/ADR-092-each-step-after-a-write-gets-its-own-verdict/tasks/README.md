# ADR-092 Tasks

Implementation tasks for ADR-092: each step after a write gets its own verdict. See the parent ADR
for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T4 | T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `internal/check` runs an ordered list of steps | done | — | `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict/tasks/T1-run-steps-in-order.md` fence |
| T2 | `write` and `check` take `--then` and `--then-sh` | done | — | `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict/tasks/T2-then-on-write-and-check.md` fence |
| T3 | Every surface teaches the steps | done | — | `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict/tasks/T3-teach-then.md` fence |
| T4 | Every refusal and every kept log reaches the receipt | done | — | `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict/tasks/T4-review-of-266.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `check.RunSteps`, `check.Step`, `check.StepResult`, `Config.Steps` | T2 | T1 before T2 |
| T2 | `--then`, `--then-sh`, the `then` receipt | T3 | T2 before T3 |

## Notes

- `internal/check` changes in T1 only; `internal/read`, `apply`, `plan`, `seen` and `state` stay
  byte-identical against 8ecb059 in every task.
