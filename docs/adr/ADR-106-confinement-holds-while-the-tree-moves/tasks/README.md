# ADR-106 Tasks

Implementation tasks for ADR-106: confinement holds while the tree moves. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T4 | T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | a swapped parent escapes the root today | done | — | `docs/adr/ADR-106-confinement-holds-while-the-tree-moves/tasks/T1-swapped-parent.md` fence |
| T2 | the write path goes through `os.Root` | pending | — | `docs/adr/ADR-106-confinement-holds-while-the-tree-moves/tasks/T2-through-the-root.md` fence |
| T3 | a target replaced after validation is refused | done | — | `docs/adr/ADR-106-confinement-holds-while-the-tree-moves/tasks/T3-identity-recheck.md` fence |
| T4 | what confinement holds, written down | done | — | `docs/adr/ADR-106-confinement-holds-while-the-tree-moves/tasks/T4-docs.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the swap fixtures | T2, T3 | T2 makes T1's test pass |
| T2 | `tree` | T3 | the recheck reads through the tree |

## Notes

- Engine go/no-go: ADR-106 owns `internal/apply`; every other engine package stays byte-identical.
- T2 is pushed to CI before T3 starts: the Windows shards are the only place junctions and root handles are exercised.
