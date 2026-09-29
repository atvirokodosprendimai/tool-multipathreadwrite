# ADR-093 Tasks

Implementation tasks for ADR-093: an MCP tool refuses an argument it does not declare. See the
parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `callTool` refuses an argument its tool does not declare; the decoders and schemas declare one set; contract §175 | done | — | `docs/adr/ADR-093-an-mcp-tool-refuses-an-argument-it-does-not-declare/tasks/T1-the-call-refuses-an-undeclared-argument.md` fence |
| T2 | Both input schemas are closed; the legacy golden changes by that key alone; README and AGENTS.md name the refusal; contract §176 | done | — | `docs/adr/ADR-093-an-mcp-tool-refuses-an-argument-it-does-not-declare/tasks/T2-the-input-schemas-are-closed.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `undeclaredRefusal`, the check in `callTool` | T2 | T1 before T2: a closed schema must not advertise a refusal the server does not make |

## Notes

- There is no `§NN` row in any Tests table (the ADR-052/054 lesson). Contract sections are cited
  in the fences.
- Engine go/no-go: ADR-093 owns `internal/mcp`, `scripts/contract.sh`, `README.md` and `AGENTS.md`.
  `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`,
  `internal/state`, `internal/lines` and `cmd/mrw` stay byte-identical, and both fences assert it.
- Contract sections §175–§177 were allocated to this record on 2026-09-29, while three other
  records were drafted in parallel. It uses §175 and §176, and §177 is left unused.
- Neither fence runs the full `./scripts/contract.sh`. Each greps its section header and claim
  strings, and the row itself is `[proof: human]`, run unpiped before each commit (lifecycle §6), as
  ADR-092 T2 S5 does: the whole contract is minutes long and a fence runs at least three times per
  task. Every listed mutant is killed by a Go test in the fence.
