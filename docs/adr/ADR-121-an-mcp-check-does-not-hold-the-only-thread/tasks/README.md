# ADR-121 Tasks

Implementation tasks for ADR-121: an MCP check does not hold the only thread. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the loop released during a check, progress while a call runs | done | none | `docs/adr/ADR-121-an-mcp-check-does-not-hold-the-only-thread/tasks/T1-thread.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-121 owns no engine package.
- Contract section §220.
