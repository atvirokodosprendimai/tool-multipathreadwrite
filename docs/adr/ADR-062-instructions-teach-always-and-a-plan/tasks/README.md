# ADR-062 Tasks

Implementation tasks for ADR-062: `mrw instructions` teaches always and a plan. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | Shared() always + plan; CLI() cookbook; §114 | done | — | `docs/adr/ADR-062-instructions-teach-always-and-a-plan/tasks/T1-always-and-a-plan.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | always + plan on Shared(); cookbook on CLI() only | — | sole task |

## Notes

- Continue on `adr-061-files-when-unmapped` (061 unmerged).
- Engine go/no-go: this record owns `internal/guide` and the MCP teaching strings. apply / plan / seen / check / state stay byte-identical.
- Do not put §114 in the Tests table (ADR-054 first-red lock: a contract section is UNPROVEN forever).
- Do not list `internal/mcp/mcp_test.go` in the Tests table (leftover hasher would lock every Test* in that file).
