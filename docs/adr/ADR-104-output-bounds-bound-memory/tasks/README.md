# ADR-104 Tasks

Implementation tasks for ADR-104: output bounds bound memory. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | the check tail in bounded memory; contract §200 | done | — | `docs/adr/ADR-104-output-bounds-bound-memory/tasks/T1-check-tail.md` fence |
| T2 | an MCP request line is capped; contract §199 | done | — | `docs/adr/ADR-104-output-bounds-bound-memory/tasks/T2-mcp-request-cap.md` fence |
| T3 | a file over the read cap is refused by name | done | — | `docs/adr/ADR-104-output-bounds-bound-memory/tasks/T3-file-cap.md` fence |
| T4 | an ast-grep answer over the cap is refused | done | — | `docs/adr/ADR-104-output-bounds-bound-memory/tasks/T4-ast-grep-cap.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | independent |

## Notes

- Engine go/no-go: ADR-104 owns `internal/check` and `internal/read`; `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` stay byte-identical.
- Contract sections §199 and §200: §196–§197 are ADR-102's and §198 ADR-103's (PR #294, open when this record was written).
