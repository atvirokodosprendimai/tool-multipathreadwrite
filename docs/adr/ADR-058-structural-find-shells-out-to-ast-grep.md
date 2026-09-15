# ADR-058: Structural find shells out to ast-grep on the read path

**Status:** Accepted
**Accepted:** 2026-09-15 by M — *"Implement the plan as specified"*, on the four-leftovers spec. Reserved in BACKLOG 2026-09-13.
**Date:** 2026-09-15
**Owner:** M
**Spec:** None — UC-2 of `docs/specs/2026-09-15-find-hint-probe-hook.md` fans across four records; adr-lint coverage is per-record
**Cross-references:** ADR-007 (Walk / `--grep` stays regex), ADR-016 (CLI and MCP must not disagree), ADR-048 (no write-time parser), `docs/adr/BACKLOG.md` (this number reserved), `docs/specs/2026-09-15-find-hint-probe-hook.md`
**Governs:** `internal/read/**`, `cmd/mrw/main.go`, `internal/mcp/**`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/astgrep_test.go::TestMissingAstGrepIsUsageAndNamesTheBinary`
**Invalidates:** none — checked. ADR-007's Walk go/no-go still forbids a fifth matching rule inside `Walk`; this is a separate primitive that produces specs `read.Run` already serves.
**Served-path change:** `mrw read --ast-grep PATTERN` (MCP `ast_grep`) finds by structure through the `ast-grep` CLI on PATH, maps hits to line ranges, and serves them through existing `read`; missing binary is exit 2 and names `ast-grep`.

## Context

BACKLOG reserved ADR-058 for structural find on the READ path only: a flag beside `--grep`, shell out if present, map hits to ranges, serve through `read`. `--grep` stays regex; disagreement is the feature. License is served lines, not AST nodes. Rejected already: tree-sitter in apply; replacing regex `--grep`; bundling a parser.

Class enumerated 2026-09-15: every finder that turns a pattern plus paths into `[]read.Spec` then `read.Run` — today `Walk` (`--grep` / MCP `grep`) and `--files-from`. Members left out: apply, plan, seen, check, state (engine go/no-go). Command: `rg -n 'func Walk|func AstGrep|grepSpecs' internal cmd`.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `read.Walk` | `internal/read/walk.go` | Regex finder. Observes nothing. **Reused as the shape, not extended** — no fifth matching rule. |
| `read.Run` | `internal/read/read.go` | Authoritative serve and ledger. **Reused unchanged**. |
| `grepSpecs` | `internal/mcp/tools.go` | MCP `grep` calls Walk. **Mirrored** as `astGrepSpecs`. |
| `exec.LookPath` | stdlib | Missing binary vs present with zero hits. **Used**. |

## Decision

**1. Flag `--ast-grep` / MCP `ast_grep` beside `--grep`.** Same walk-then-serve shape. `--grep` stays regex. Together: usage, "two sources". `--files-from` with `--ast-grep` is the same sentence.

**2. Shell out to `ast-grep` on PATH.** `LookPath` miss: exit 2, reason names `ast-grep`, not urfave unknown-flag. Present binary, zero hits: exit 1, name the pattern (like `--grep`). Hits: 0-based lines mapped to 1-based `Range`, served through `read.Run`.

**3. License is served lines.** AstGrep records nothing. Run observes what it printed. A file the finder opened with no hit is not served.

**4. Surfaces must not disagree (ADR-016).** MCP calls the same `read.AstGrep`. Missing binary names `ast-grep` on both. Two finders are two sources on both.

## Alternatives Considered

- **A fifth matching rule inside `Walk`** — rejected: ADR-007 go/no-go. Separate primitive.
- **tree-sitter inside apply** — rejected: ADR-048.
- **Replace regex `--grep`** — rejected: disagreement is the feature.
- **Bundle ast-grep** — rejected: missing binary is exit 2.

## Component / Boundary Impact

`internal/read` gains `AstGrep`. CLI and MCP read gain one flag/field. apply/plan/seen/check/state stay byte-identical.

## Wiring & Contract Changes

Inherited from spec §Contracts Touched; delta:

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --ast-grep` / MCP `ast_grep` | new finder | `read.AstGrep` | CLI and MCP readers |
| `scripts/contract.sh` | §107–§109 | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.AstGrep` + CLI `--ast-grep` | T1 | T2 | No |
| MCP `ast_grep` | T1 | T2 | No |

## Implementation

Tasks in `docs/adr/ADR-058-structural-find-shells-out-to-ast-grep/tasks/`.

## Consequences

- **Positive:** structural find without a write-time parser.
- **Negative:** depends on an optional PATH binary.
- **Neutral:** `--grep` unchanged; the two may disagree on the same token.

## Out of Scope

Inherited from spec §Non-Goals; delta: none.

- Write-time / apply parser (permanent: boundary: ADR-048)
- Replacing regex `--grep`; bundling `ast-grep` (permanent: boundary: disagreement is the feature)
- Apply/plan/seen/check/state edits (permanent: boundary: engine go/no-go)

## Risks

Inherited from spec §Risks; delta:

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Real ast-grep JSON shape drifts from the fake | Med | Med | Mapper accepts `file`/`path`; tests pin the fake |
| Windows has no `#!/bin/sh` fake in §109 | Low | Low | `scripts/contract.sh` is Linux-only |

## Rollback

Revert. `--ast-grep` disappears; `--grep` is unchanged.

## Follow-ups

## Stress suite

Added 2026-09-15 after execute. Oracle is Decision 1–2, not the `astSet` switch. `internal/read/leftovers_stress_test.go` maps 0-based JSON independently; a JSON object is not zero hits; a `../` hit is a Problem; Walk still has no ast-grep branch. `internal/adversarial/leftovers_stress_test.go` drives the built binary through a random flag matrix whose pool was grepped from `cmd/mrw/main.go` (`two sources of specs`, `two answers`, `exclude without`). `cmd/mrw/leftovers_stress_test.go` pins `--files-from`+`--ast-grep`, `--exclude` with `--ast-grep`, and exit 1 plus `[]`. A hanging `ast-grep` on PATH is not this record's promise.
