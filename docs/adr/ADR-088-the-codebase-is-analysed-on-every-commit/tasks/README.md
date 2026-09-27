# ADR-088 Tasks

Implementation tasks for ADR-088: the codebase is analysed on every commit, and production holds nothing only a test reaches. See the parent ADR for the decision.

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
|----|-------|--------|--------|-------------|
| T1 | Production holds nothing only a test reaches | done | — | `docs/adr/ADR-088-the-codebase-is-analysed-on-every-commit/tasks/T1-nothing-only-tests-reach.md` fence |
| T2 | The static gate, clean | done | — | `docs/adr/ADR-088-the-codebase-is-analysed-on-every-commit/tasks/T2-the-static-gate.md` fence |
| T3 | The analysis runs after every commit | done | — | `docs/adr/ADR-088-the-codebase-is-analysed-on-every-commit/tasks/T3-after-every-commit.md` fence |
| T4 | A read whose answer did not reach the caller records nothing | done | — | `docs/adr/ADR-088-the-codebase-is-analysed-on-every-commit/tasks/T4-an-unwritten-read-records-nothing.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-088 owns `internal/check` (T1); `internal/read`, `internal/state`, `internal/seen`, `internal/apply`, `internal/iter` and `internal/subproc`, for errors wrapped with `%w` and discards written `_ =` (T2). `internal/plan` and `internal/lines` stay byte-identical.
