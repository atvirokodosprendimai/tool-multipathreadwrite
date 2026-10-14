# ADR-128 Tasks

Implementation tasks for ADR-128: the MCP surface. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a BOM read, force advice dropped, unknown acks named, a cancel that stops a check | done | none | `docs/adr/ADR-128-the-mcp-surface/tasks/T1-surface.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Owns `internal/apply` for `Options.NoForce` and the refusal words only.
- Contract section §226.
