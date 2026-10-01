# ADR-113 Tasks

Implementation tasks for ADR-113: one write path, and an MCP write is checked. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the write sequence is shared, and the CLI's outcomes do not move | done | none | `docs/adr/ADR-113-one-write-path-and-mcp-checks/tasks/T1-shared-sequence.md` fence |
| T2 | an MCP write runs the check and reports it | done | T1 | `docs/adr/ADR-113-one-write-path-and-mcp-checks/tasks/T2-mcp-checks.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-113 owns no engine package; every one stays byte-identical.
- Contract section §213 (T2). T1 edits no contract row.
- One pull request, T1 and T2 as separate commits: `scripts/fence-prose.py` checks every task's fence, pending ones included.
