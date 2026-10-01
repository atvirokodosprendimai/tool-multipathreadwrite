# ADR-111 Tasks

Implementation tasks for ADR-111: a receipt field, once shipped, stays. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the CLI's receipt keys are listed and held | done | none | `docs/adr/ADR-111-a-receipt-field-once-shipped-stays/tasks/T1-cli-receipts.md` fence |
| T2 | MCP's receipt keys are listed and held | done | T1 | `docs/adr/ADR-111-a-receipt-field-once-shipped-stays/tasks/T2-mcp-receipts.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-111 owns no engine package; every one stays byte-identical.
- Contract section §210 (T1).
