# ADR-083 Tasks

Implementation tasks for ADR-083: a plan refused after it parsed is counted on both surfaces. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The CLI counts a plan refused after it parsed | done | — | `docs/adr/ADR-083-a-refused-plan-is-counted-on-both-surfaces/tasks/T1-the-cli-counts-a-refused-plan.md` fence |
| T2 | `mrw_write` counts the same refusals and a ledger-failed landing | done | — | `docs/adr/ADR-083-a-refused-plan-is-counted-on-both-surfaces/tasks/T2-mrw-write-counts-the-same.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-083 owns `cmd/mrw` (T1) and `internal/mcp` (T2). `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted`, `internal/subproc` and `internal/authoring` stay byte-identical.
