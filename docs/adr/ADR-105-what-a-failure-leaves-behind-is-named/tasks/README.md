# ADR-105 Tasks

Implementation tasks for ADR-105: what a failure leaves behind is named. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | T2 |
| 4 | T4 | T1, T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | a failed cleanup is named in `left_behind` | done | — | `docs/adr/ADR-105-what-a-failure-leaves-behind-is-named/tasks/T1-left-behind.md` fence |
| T2 | state files are replaced whole; contract §201 | done | — | `docs/adr/ADR-105-what-a-failure-leaves-behind-is-named/tasks/T2-atomic-state.md` fence |
| T3 | the licence files are synced; the tree sync is measured | done | — | `docs/adr/ADR-105-what-a-failure-leaves-behind-is-named/tasks/T3-fsync-measured.md` fence |
| T4 | the failure matrix; the BACKLOG entries closed | done | — | `docs/adr/ADR-105-what-a-failure-leaves-behind-is-named/tasks/T4-failure-matrix.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T2 | `state.Write` | T3 | T3 builds `WriteSynced` on it |
| T1 | `Result.LeftBehind` | T4 | the matrix names the field |
| T3 | the fsync outcome | T4 | the matrix states the power-loss window |

## Notes

- Engine go/no-go: ADR-105 owns `internal/apply`, `internal/state`, `internal/seen`, `internal/iter`; `internal/plan`, `internal/lines`, `internal/rooted`, `internal/read`, `internal/check`, `internal/links` stay byte-identical.
- Contract section §201: §199–§200 are ADR-104's (PR #295, open when this record was written).
