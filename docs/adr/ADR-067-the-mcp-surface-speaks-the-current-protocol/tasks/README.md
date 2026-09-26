# ADR-067 Tasks

Implementation tasks for ADR-067: the MCP surface speaks the current protocol revisions, reports a caller's argument mistakes to the model, and its index names every problem. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | T1, T2, T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | the index names every path the walk could not use; contract §122; BACKLOG P2 closed | done | — | `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol/tasks/T1-the-index-names-every-problem.md` fence |
| T2 | `initialize` negotiates; `serverInfo` title and description; `-32603` on an encode failure; a null id refused; contract §123 | done | — | `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol/tasks/T2-initialize-negotiates-wire-names-failure.md` fence |
| T3 | argument mistakes are tool execution errors; contract §124; §83's git row follows | done | — | `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol/tasks/T3-argument-mistakes-are-tool-execution-errors.md` fence |
| T4 | dual-era: `server/discover`, per-request `_meta`, `-32022`, `resultType`, caching hints, a per-call ceiling reserve; README; contract §125 | done | — | `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol/tasks/T4-the-server-is-dual-era.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T2 | `supportedVersions` constant and `serverInfo()` helper | T4 | T4's `-32022`, `server/discover` and result `_meta` read the same list and helper |
| T1, T3 | the legacy answers T4's golden records | T4 | captured after both land |

## Notes

- No `§NN` row in any Tests table (ADR-052/054 lesson).
- Engine go/no-go: ADR-067 owns `internal/mcp`, `scripts/contract.sh` and `README.md`. `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `cmd/mrw` stay byte-identical, and every fence asserts it.
- T1 and T3 both edit `internal/mcp/tools.go` at disjoint sites; they run in order on one branch.
- T4 runs last: its legacy golden is captured on T1–T3's tree, so the guard compares against the answers T1 and T3 intentionally changed.
