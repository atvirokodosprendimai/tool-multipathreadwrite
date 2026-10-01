# ADR-110 Tasks

Implementation tasks for ADR-110: a writer waits a bounded time. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a lock wait is bounded and names the holder | done | none | `docs/adr/ADR-110-a-writer-waits-a-bounded-time/tasks/T1-bounded-hold.md` fence |
| T2 | a writer waits a bounded time for the write lock | done | T1 | `docs/adr/ADR-110-a-writer-waits-a-bounded-time/tasks/T2-write-lock-wait.md` fence |
| T3 | a refusal before any hunk survives the smallest ceiling | done | T2 | `docs/adr/ADR-110-a-writer-waits-a-bounded-time/tasks/T3-small-ceiling.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-110 owns `internal/state` and `internal/seen`; `internal/read`, `internal/apply`, `internal/plan`, `internal/check`, `internal/lines` and `internal/rooted` stay byte-identical.
- Contract section §209 (T2).
