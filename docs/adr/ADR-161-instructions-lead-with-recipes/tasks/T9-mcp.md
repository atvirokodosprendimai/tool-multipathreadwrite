# Task ADR-161-T9: the MCP surface takes the same tools — sub, rewrite, quiet, the line after a range, and the recipes

**Depends-on:** T1, T2, T3, T6, T7
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `mrw_read` arguments `sub`, `rewrite`, `quiet`; `mrw_write` arguments `grep`, `sub`, `ast_grep`, `rewrite`; the extra line in served pages; the recipes in the handshake
**Consumes:** the plan builders (T1, T2), `Options.Quiet` (T6), the line after a range (T7), the recipes text (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an MCP sub write applies the plan in one call`, `MCP quiet and next-line license as the CLI does`, `the handshake carries the recipes`

## Goal

A host with no shell — Claude Desktop, the MCP arm — edits many sites without the lines in its context, as the CLI does.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the arguments, their schema and refusals; `mrw_write` building the plan from `grep`+`sub` or `ast_grep`+`rewrite` |
| `internal/mcp/mcp.go` | edit | the tool schemas |
| `internal/mcp/ack.go` | edit | the extra line inside a checkpoint (ADR-108), counted in its `N lines follow` |
| `internal/mcp/instructions.go` | edit | the handshake carries the recipes in their MCP form |
| `internal/mcp/sub161_test.go` | add | `TestAnMCPSubWriteAppliesThePlanInOneCall` |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | only where a golden answer changes by the extra line, each change named |
| `docs/receipts.txt` | edit | any receipt key added (ADR-111) |
| `scripts/contract.sh` | edit | §269, over `$MRW mcp` |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] `mrw_write` with `grep`+`sub` or `ast_grep`+`rewrite` and no `plan` builds the plan through T1's and T2's builders and applies it as one plan, with the CLI's refusals (a refused or unreadable file applies nothing; overlapping rewrites refused); `mrw_read` with `sub` or `rewrite` returns the plan as text and licenses as the CLI does (licensed, not shown). [proof: mutation]
3. [S3] `mrw_read` `quiet` licenses and returns no line; a served page carries the line after each range inside its checkpoint, counted. [proof: mutation]
4. [S4] The handshake text carries the recipes, each in its MCP form (`mrw_write` with `grep` and `sub`); `TestEverySurfaceContainsTheSharedSentences` and the handshake's size tests still hold. [proof: mutation]
5. [S5] Contract §269 over `$MRW mcp`: an `mrw_write` with `grep`+`sub` changes every site and no neighbour in one call; the pair, an unreadable matching file, applies nothing. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/mcp/ ./internal/read/ ./cmd/mrw/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestAnMCPSubWriteAppliesThePlanInOneCall ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  an MCP sub write changes every site in one call and no neighbour' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the arguments in `tools.go` |
| 2 — something selects it | the tool schemas a host reads |
| 3 — the caller can discover it | the handshake's recipes, the schema descriptions |
| 4 — it is used | MCP hosts |

## Mutation Log
(empty until execute)

## Invariants

- An `mrw_write` with a `plan` and an `mrw_read` without the new arguments are unchanged, apart from the extra line T7 serves.
- The result ceiling (ADR-098) still bounds every answer.

## Risks

- A golden test pins a page byte for byte; each change it needs is named in the commit.

## Stop Condition

The checkpoint contract (ADR-108) cannot count the extra line without a change a host would see as a different protocol.

## Out of Scope

- Standard input over MCP (permanent: boundary: `mrw_read` has no standard input to give)

## Verification Log
(empty until execute)
