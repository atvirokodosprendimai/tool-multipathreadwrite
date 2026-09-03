# ADR-010 Tasks

Implementation tasks for ADR-010: mrw speaks MCP over the same engine, and stays a binary. See the
parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers. This
README is a derived index — when it disagrees with a task file, the task file wins and the README
must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

Forced rather than chosen: T2 routes `tools/call` through the frame T1 builds, and T3 documents the
tools T2 implements.

**T1 carries its own call site**, as ADR-009-T1 did and for the same reason: a package shipped in
one task and wired in the next is how `rooted.Descendable` was built, tested, mutation-logged and
deleted unused on 2026-09-03.

**T2 is the one that can withdraw the ADR.** Its Stop Condition is the decision itself — the tools
are adapters over `read.Run` and `apply.Apply`, and the moment either needs to compute a verdict of
its own there are two answers to "did this apply?", which is the defect class this project exists to
refuse. Its fence enforces that mechanically: exactly one requirement in `go.mod`, and a clean
working tree across the six engine directories including untracked files.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A stdio JSON-RPC transport, hand-rolled, reachable | pending | — | `go test ./internal/mcp/ -v …` + 7 named `--- PASS:` lines, one requirement in `go.mod`, clean engine tree, `mrw --help \| grep mcp` |
| T2 | Two tools over the unchanged engine, one answer | pending | — | `go test ./internal/mcp/ -v …` + 6 named `--- PASS:` lines, `# 38.` in `contract.sh`, one requirement in `go.mod`, clean engine tree, `./scripts/contract.sh` |
| T3 | Make it installable, and say what it does not change | pending | — | `grep -q '### Use it from an MCP host' README.md && … && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `mcp.Serve`, the JSON-RPC frame | T2, T3 | T1 before T2 — T2 routes through a frame T1 defines |
| T2 | MCP tools `mrw_read` / `mrw_write` | T3 | T2 before T3 — T3 documents what T2 implements |

## Notes

- **Run every fence before writing a line of its task.** If it exits 0, it is not a fence. Three in
  the 2026-09-03 session were green on an untouched tree — one named a `contract.sh` section that
  already existed, one filtered out two of its own named tests, one matched README text written by
  unrelated work. Each was written by someone thinking hard about that exact trap, so reasoning
  about the fence does not substitute for running it.
- **Confirm §38 and §39 are unused** before relying on those clauses. The highest section is 37 as
  of `a714ae5`.
- **The go/no-go conditions live in T2's fence, not in prose.** `go.mod` must declare exactly one
  requirement, and `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check` and `internal/state` must carry no change and no new file. If either fails, `mrw mcp` is withdrawn
  and the binary is the whole answer — that is a real outcome, not a formality.
