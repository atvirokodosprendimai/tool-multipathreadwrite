# ADR-054 Tasks

Implementation tasks for ADR-054: A write that applied can still leave a broken tree. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | T1, T2, T3 |

T1 first: it is the arm that would have caught the three Zeus `.rs` breakages in the turn. Markdown plans do not spawn the check. T2 and T3 are additive and may run in parallel after T1 starts. T4 teaches once the three surfaces exist.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | CLI write runs the check by default; `--no-check`; contract §89 | pending | — | `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree/tasks/T1-write-check-by-default-and-contract-89.md` fence |
| T2 | Delimiter-balance delta; contract §90 | pending | — | `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree/tasks/T2-delimiter-balance-delta-and-contract-90.md` fence |
| T3 | stats always prints `failed_check`; landed line; contract §91 | pending | — | `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree/tasks/T3-stats-failed-check-row-and-contract-91.md` fence |
| T4 | Teach `--no-check` / balance / stats; BACKLOG 054 | pending | — | `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree/tasks/T4-teach-and-backlog.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | default check / `--no-check` | T4 | T1 before T4 |
| T2 | `HunkResult.Balance` | T4 | T2 before T4 |
| T3 | stats five names + landed line | T4 | T3 before T4 |

## Notes

- Branch from `origin/main` (ADR-053 lives there).
- Engine go/no-go: `internal/apply` is owned by T2; `internal/check` stays byte-identical; `internal/authoring` vocabulary stays five names (T3 is rendering in `cmd/mrw`).
- Do not start T1 while the parent is Proposed.
