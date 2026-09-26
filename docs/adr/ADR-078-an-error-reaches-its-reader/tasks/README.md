# ADR-078 Tasks

Implementation tasks for ADR-078: an error reaches its reader in a form it can use. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The CLI's errors reach stderr and say what to do; contract §155, §159 | done | — | `docs/adr/ADR-078-an-error-reaches-its-reader/tasks/T1-cli-errors-reach-stderr.md` fence |
| T2 | The MCP server refuses what it cannot represent; contract §156 | done | — | `docs/adr/ADR-078-an-error-reaches-its-reader/tasks/T2-mcp-refuses-what-it-cannot-represent.md` fence |
| T3 | The MCP answer says what to do next; contract §157, §158 | done | — | `docs/adr/ADR-078-an-error-reaches-its-reader/tasks/T3-mcp-answer-says-what-next.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-078 owns `internal/read/read.go` (`Options.Stop`), `internal/read/walk.go` (`CheckExclude`) and `internal/plan/plan.go`. `internal/apply`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/seen`, `internal/subproc` and `internal/rooted` stay byte-identical.
