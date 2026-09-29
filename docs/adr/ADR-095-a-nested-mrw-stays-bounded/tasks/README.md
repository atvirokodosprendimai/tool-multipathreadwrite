# ADR-095 Tasks

Implementation tasks for ADR-095: a nested mrw stays bounded. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A check counts a level, and at the limit none is started | done | — | `docs/adr/ADR-095-a-nested-mrw-stays-bounded/tasks/T1-the-check-counts-a-level.md` fence |
| T2 | A group is stopped with TERM first | done | — | `docs/adr/ADR-095-a-nested-mrw-stays-bounded/tasks/T2-a-group-hears-term-first.md` fence |
| T3 | Every surface teaches the bounds and what escapes them | done | — | `docs/adr/ADR-095-a-nested-mrw-stays-bounded/tasks/T3-teach-the-bounds.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `check.DepthRefusal`, the check's `MRW_STEP_DEPTH` | T3 | T1 before T3 |
| T2 | `stopGroup` | T3 | T2 before T3 |

## Notes

- `internal/check`'s code changes in T1 only; T2 adds one test file there. `internal/read`, `apply`,
  `plan`, `seen` and `state` stay byte-identical against the merge-base in every task.
- Each fence checks formatting with `gofmt -l cmd internal`, not `.`: `.claude/worktrees/` holds other
  sessions' checkouts, which `gofmt -l .` would walk, and `git ls-files` would miss a task's own new,
  untracked test files.
- T1 and T2 touch different files and may land in either order.
- The machine is shared: run each fence stand-alone and unpiped, never two at once.
