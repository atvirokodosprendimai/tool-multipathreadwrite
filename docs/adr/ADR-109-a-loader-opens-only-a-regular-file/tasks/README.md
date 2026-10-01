# ADR-109 Tasks

Implementation tasks for ADR-109: a loader opens only a regular file. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the helper, and read opens through it | done | none | `docs/adr/ADR-109-a-loader-opens-only-a-regular-file/tasks/T1-helper-and-read.md` fence |
| T2 | apply's load asks the descriptor | done | T1 | `docs/adr/ADR-109-a-loader-opens-only-a-regular-file/tasks/T2-apply-load.md` fence |
| T3 | the compilers, body files and MCP ask the descriptor | done | T1 | `docs/adr/ADR-109-a-loader-opens-only-a-regular-file/tasks/T3-compilers-and-mcp.md` fence |
| T4 | a FIFO config is refused | done | T1 | `docs/adr/ADR-109-a-loader-opens-only-a-regular-file/tasks/T4-fifo-config.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-109 owns `internal/read`, `internal/apply`, `internal/plan` and `internal/check`; `internal/state`, `internal/lines`, `internal/rooted`, `internal/seen` stay byte-identical.
- Contract section §207 (T4).
