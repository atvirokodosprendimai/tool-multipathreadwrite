# ADR-007 Tasks

Implementation tasks for ADR-007: mrw finds the files it serves. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

The order is forced rather than chosen: T2 consumes `rooted.Descendable` from
T1, and T3 consumes `read.Walk` from T2. T3 is last and is the only task that
changes the served path, so nothing a caller can see moves until the walk it
depends on is proved.

T2 also carries the ADR's go/no-go conditions. If it records a failing one, T3
ships `--files-from` alone and `--grep` is withdrawn — which is why the runner-up
is inside this ADR rather than deferred to another.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Decide what a walk may descend into | pending | — | `go test ./internal/rooted/ -run 'TestDescend'` |
| T2 | Turn a walk and a pattern into specs | pending | — | `go test ./internal/read/ -run 'TestWalk'` |
| T3 | Make the walk reachable, and ship the runner-up | pending | — | `go test ./cmd/mrw/ -run 'TestGrep\|TestFilesFrom\|TestExclude'` |

Status: `pending` | `partial` | `blocked` | `done`.
