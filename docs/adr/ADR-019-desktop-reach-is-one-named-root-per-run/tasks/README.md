# ADR-019 Tasks

Implementation tasks for ADR-019: Desktop reach is one named root per run. See the parent ADR
for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|--------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Encode M's naming pick into the Decision | done | — | `grep -qE '^Naming pick: (A — launch --root only\|B — launch allow-list\|C — MCP roots/list)$' docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md` |
| T2 | The pick's first slice, through the built binary | done | — | `grep -q '^# 78\. ' scripts/contract.sh && ./scripts/contract.sh` |
| T3 | Teaching matches the pick | done | — | `go test ./internal/mcp/ -run 'TestTheSurfaceSaysTheCLIIsRicher\|TestTheSurfaceNamesTheRootThePickChose'` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | Naming pick encoded in this Decision as `Naming pick:` | T2, T3 | T1 first — T2 must not guess a Desktop |
| T2 | Binary behaves as the pick | T3 | T2 before T3 — teaching that precedes the binary is ADR-012's defect |

## Notes

- M named *"A"* on 2026-09-10. That is `Naming pick: A — launch --root only`.
  `mrw mcp` stays single-root. Do not implement B or C.
- Engine go/no-go: this record owns `internal/mcp` (and `mcpCmd` wiring). `internal/read`,
  `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay
  byte-identical against the merge-base.
- Do not raise `maxInstructionsChars`. Do not generate `AGENTS.md` from `guide.Shared()`.
- Cargo (`check` / `iter` / `seen` / `stats` over MCP) is not a slice of this record.
- `# 78.` is the next free contract section; `# 77.` is ADR-039 on `origin/main` `c444fc5`.
