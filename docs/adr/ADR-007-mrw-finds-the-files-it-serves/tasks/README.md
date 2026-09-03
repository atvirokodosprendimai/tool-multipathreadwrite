# ADR-007 Tasks

Implementation tasks for ADR-007: mrw finds the files it serves. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

## Execution Order

| Order | Task | Depends-on | Outcome |
|-------|------|------------|---------|
| 1 | T1 | none | **withdrawn 2026-09-03** — see the ADR's Amendment |
| 2 | T2 | none (was T1) | done |
| 3 | T3 | T2 | done |

T3 consumes `read.Walk` from T2, so T3 is last and is the only task that changes
the served path: nothing a caller can see moves until the walk it depends on is
proved.

T1 produced `rooted.Descendable` and was withdrawn after T2's own reachability
mutation showed the function unreachable from its only caller. The function and
its tests are deleted. The full reasoning, including the two mutation attempts
that failed to make it observable, is the Amendment at the end of the parent
ADR — read that before reintroducing a descend guard.

T2 also carries the ADR's go/no-go conditions. If it records a failing one, T3
ships `--files-from` alone and `--grep` is withdrawn — which is why the runner-up
is inside this ADR rather than deferred to another. **Measured 2026-09-03: 0.76×
against the baseline, threshold 2×. `--grep` ships.**

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Decide what a walk may descend into | withdrawn | — | n/a — the deliverable is deleted |
| T2 | Turn a walk and a pattern into specs | done | — | `go test ./internal/read/ -run 'TestWalk'` |
| T3 | Make the walk reachable, and ship the runner-up | done | — | `go test ./cmd/mrw/ -run 'TestGrep\|TestFilesFrom\|TestExclude\|TestNoArguments\|TestTheDocumented'` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

Status: `pending` | `partial` | `blocked` | `done`.
