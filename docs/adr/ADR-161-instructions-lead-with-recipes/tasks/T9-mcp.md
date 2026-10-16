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
| `TestAnMCPSubWriteAppliesThePlanInOneCall` | `internal/mcp/sub161_test.go` | `mrw_write` with `grep`+`sub` and with `ast_grep`+`rewrite` applies one plan: every site, no neighbour, nothing on a dry run, nothing for an unreadable file or overlapping matches | none | S1, S2 |
| `TestAnMCPSubRefusesWhatTheCLIRefuses` | `internal/mcp/sub161_test.go` | `plan` beside a walk, `sub` without `grep`, `rewrite` without `ast_grep`, a line break in `sub`, no match, no change, and `quiet`/`count`/`stat` beside `sub` are refused by name; an undeclared argument still is | none | S2, S3 |
| `TestAnMCPSubReadReturnsAPlanAndLicensesItsLines` | `internal/mcp/sub161_test.go` | `mrw_read` with `sub` or `rewrite` returns exactly the plan, licenses the planned lines with no ack and no other; an oversized plan is refused and licenses nothing | none | S2 |
| `TestAnMCPQuietReadLicensesAndReturnsNoLine` | `internal/mcp/sub161_test.go` | `quiet` returns headers and a count, licenses the range and its next line at once and nothing more; the unquiet read licenses nothing until acked | none | S3 |
| `TestAnMCPRangeServesTheLineAfterInsideItsCheckpoint` | `internal/mcp/sub161_test.go` | a range, a grep match and a JSON path serve the line after inside a checkpoint that counts it; a range ending on the last line and a whole file gain none | none | S3 |
| `TestAnMCPPageContinuationNeverRepeatsOrDropsALine` | `internal/mcp/sub161_test.go` | following `next_read` from a range's start reassembles the rest of the file exactly | none | S3 |
| `TestTheHandshakeCarriesTheRecipesInTheirMCPForm` | `internal/mcp/sub161_test.go` | the handshake names the recipes as tool arguments and stays within its bound | none | S4 |
| `TestAnMCPSubWriteWalksOnlyThePathsItIsGiven` | `internal/mcp/sub161_test.go` | `paths` narrows an `mrw_write` walk so a matching file outside it is untouched; a range, a path outside the root and `paths` without a walk are refused | none | S2 |
| `TestAnMCPSubReadRefusesAPlanJSONWouldCorrupt` | `internal/mcp/sub161_test.go` | a plan holding a byte that is not valid UTF-8 is refused by `mrw_read` and licenses nothing; `mrw_write` applies the same plan in memory byte for byte | none | S2 |
| `TestAnMCPSubTakesAnEmptyPatternAndAnEmptyReplacement` | `internal/mcp/sub161_test.go` | an empty `grep` is a pattern and an empty `sub` a replacement on both tools, by presence as the CLI's IsSet; a missing half is still refused | none | S2 |
| `TestAnMCPPlanCallTakesTheLastValueOfADuplicateKey` | `internal/mcp/sub161_test.go` | `{"grep":"Old","grep":null,"sub":"New"}` is a missing grep on both tools (the raw keys decide alone; the struct keeps the earlier value): refused, nothing written or licensed | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the arguments in `tools.go` |
| 2 — something selects it | the tool schemas a host reads |
| 3 — the caller can discover it | the handshake's recipes, the schema descriptions |
| 4 — it is used | MCP hosts |

## Mutation Log
(empty until execute)
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/sub161.go` · mrw_write applies a partial plan when the walk could not read a matching file · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:an MCP sub write applies the plan in one call
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/tools.go` · a quiet mrw_read licenses nothing · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:MCP quiet and next-line license as the CLI does
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/tools.go` · a served range does not carry the line after it inside its checkpoint · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:MCP quiet and next-line license as the CLI does
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the handshake no longer names the grep and sub recipe · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:the handshake carries the recipes
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write ignores grep and sub, so the contract row sees no plan applied · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:an MCP sub write applies the plan in one call
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/sub161.go` · paths is ignored, so the walk covers the whole root · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:an MCP sub write applies the plan in one call
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/sub161.go` · mrw_read returns a plan JSON corrupts and licenses the original line · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:an MCP sub write applies the plan in one call
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/sub161.go` · an empty grep is taken for a missing argument · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · covers:an MCP sub write applies the plan in one call
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `internal/mcp/sub161.go` · S2: a duplicate grep key whose last value is null still plans from the earlier value (the Codex review of T9) · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15

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
- 2026-10-16 · 81668383* · exit 1 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:60385 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:149502
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:141328
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:143576
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:137507
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:137198
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:136657
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:137229
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:140174
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:137214
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ea1e62d6bb37e22af8f06e06f9a527320286dc4c394dce0abb32dc722f7eb15 · ms:137858
