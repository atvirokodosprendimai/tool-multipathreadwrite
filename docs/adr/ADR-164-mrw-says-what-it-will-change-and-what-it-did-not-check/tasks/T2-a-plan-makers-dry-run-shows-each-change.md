# Task ADR-164-T2: a plan maker's dry run shows each change

**Depends-on:** T1, T4
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `apply.HunkResult.Before`/`After` and the `apply.Options` switch; `internal/preview` (`preview.Lines`); the `~`/`-`/`+` receipt lines; JSON `hunks[].before`/`hunks[].after`
**Consumes:** `internal/apply/apply.go` as T1 leaves it (T1); `internal/mcp/tools.go`, `internal/mcp/schema.go` and `internal/mcp/testdata/legacy_golden.jsonl` as T4 leaves them (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a one-line change is marked in place`, `both ends of a long span stay visible`, `a change past column 200 is shown`, `a multi-line change shows old then new`, `a rewrite shows it too`, `a plain dry run shows nothing`, `a real write shows nothing`, `the MCP text shows it and elides it first`, `the receipt keys are listed`

## Goal

Decision 2 on both surfaces, for both plan makers.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options.Preview`; `HunkResult.Before`/`After` (`json:"before,omitempty"`, `json:"after,omitempty"`) from the replaced lines and body `planFile` already holds |
| `internal/preview/preview.go` | add | `Lines(before, after []string, firstLine int) []string`: the `~` form and the `-`/`+` form with Decision 2's windows |
| `internal/preview/preview_test.go` | add | the unit tests |
| `cmd/mrw/main.go` | edit | sets `Options.Preview` for a plan-maker dry run; `report()` prints the lines under the ok row |
| `cmd/mrw/preview164_test.go` | add | the CLI tests |
| `internal/mcp/tools.go` | edit | sets `Options.Preview` for `grep`/`ast_grep` with `dry_run`; `writeReport` prints the lines; the elision ladder drops them first |
| `internal/mcp/schema.go` | edit | descriptions for `hunks[].before`/`after`; the `elided` description's order |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated for the descriptions, as ADR-152 did |
| `internal/mcp/preview164_test.go` | add | the MCP tests |
| `docs/receipts.txt` | edit | `write hunks[].before`, `write hunks[].after`, `mcp_write hunks[].before`, `mcp_write hunks[].after` (a `[]string` leaf carries no `[]`, as `write hunks[].echo`) |
| `internal/guide/guide.go`, `internal/mcp/instructions.go`, `AGENTS.md` | edit | recipe 2's words say the dry run shows each change; the core is untouched, and the MCP handshake stays within its 4096 bytes (ADR-161 Decision 12c) |
| `scripts/contract.sh` | edit | §274 |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestALineChangeIsMarkedInPlace (the `~` form; a double match on one line and a span over 160 characters keep both ends), TestAMultiLineChangeShowsOldThenNew (the `-`/`+` form; two adjacent 300-character lines each changed at column 10 and at column 290 show both changes on each line), TestASubDryRunShowsEachChange (CLI: a `--sub` dry run over a CSS file prints `~` lines with `[-red-]{+blue+}`; `--json` carries `hunks[].before`/`after`; the same on a CRLF file and a file with no final newline), TestARewriteDryRunShowsEachChange (CLI: an `--ast-grep --rewrite` dry run over a two-line span, with `rewrite161_test.go`'s fake ast-grep), TestAPlainDryRunShowsNoPreview (a hand-written plan's dry run and a real `--sub` write print no `~`, `-` or `+` line and carry no `before`), TestAnMCPSubDryRunShowsEachChange (`grep`+`sub` and `ast_grep`+`rewrite` with `dry_run`; under a ceiling that elides, `before`/`after` go first and `elided` names them). [proof: acceptance]
2. [S2] `internal/preview`. [proof: mutation]
3. [S3] `apply` fills `Before`/`After` under `Options.Preview`. [proof: mutation]
4. [S4] The CLI sets the switch for a plan-maker dry run and prints. [proof: mutation]
5. [S5] The MCP sets it, prints, and elides it first; schema descriptions and the golden. [proof: mutation]
6. [S6] `docs/receipts.txt`; recipe 2's words on the three surfaces. [proof: mutation]
7. [S7] Contract §274: through the built binary, a `--sub --dry-run` prints the `~` line and writes nothing; the pair: a plain plan's dry run prints none. [proof: human: the executor runs ./scripts/contract.sh once and reads the §274 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/preview/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestALineChangeIsMarkedInPlace$|TestASubDryRunShowsEachChange$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestALineChangeIsMarkedInPlace \(' "$out" \
  && grep -qE '^--- PASS: TestASubDryRunShowsEachChange \(' "$out" \
  && go test ./internal/preview/ ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestAMultiLineChangeShowsOldThenNew|TestARewriteDryRunShowsEachChange|TestAPlainDryRunShowsNoPreview|TestAnMCPSubDryRunShowsEachChange|TestDryRunPrintsParsedHunkBodies|TestASubWriteAppliesThePlanInOneCommand|TestAnMCPSubWriteAppliesThePlanInOneCall|TestNoShippedReceiptFieldDisappears|TestNoShippedMCPReceiptFieldDisappears|TestEveryOutputSchemaPropertyIsDescribed|TestALegacyResultIsUnchangedByTheModernPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAMultiLineChangeShowsOldThenNew \(' "$out" \
  && grep -qE '^--- PASS: TestARewriteDryRunShowsEachChange \(' "$out" \
  && grep -qE '^--- PASS: TestAPlainDryRunShowsNoPreview \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPSubDryRunShowsEachChange \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep -x 'write hunks\[\].before' docs/receipts.txt > /tmp/adr164-t2-key.txt \
  && grep '^# 274\. ' scripts/contract.sh > /tmp/adr164-t2-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDryRunPrintsParsedHunkBodies` | `cmd/mrw/dryrun_parsed_test.go` | the `parsed:` lines unchanged | none | — |
| `TestASubWriteAppliesThePlanInOneCommand` | `cmd/mrw/sub161_test.go` | ADR-161's summary line unchanged | none | — |

Planned, named without backticks until they exist: TestALineChangeIsMarkedInPlace and TestAMultiLineChangeShowsOldThenNew in `internal/preview/preview_test.go` (S1, S2), TestASubDryRunShowsEachChange, TestARewriteDryRunShowsEachChange and TestAPlainDryRunShowsNoPreview in `cmd/mrw/preview164_test.go` (S1, S3, S4, S6), TestAnMCPSubDryRunShowsEachChange in `internal/mcp/preview164_test.go` (S1, S5).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `preview.Lines`, `HunkResult.Before`/`After` |
| 2 — something selects it | `Options.Preview`, set by the CLI and `mrw_write` only for a plan maker's dry run; contract §274 drives the binary |
| 3 — the caller can discover it | recipe 2 in `mrw instructions`, the MCP handshake and AGENTS.md; the schema descriptions |
| 4 — it is used | nothing measures this; asked for by one field report (2026-10-20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- A real write's receipt, a plain plan's dry run and every existing receipt key are unchanged (ADR-111).
- The `parsed:` lines and the summary line are unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- A preview for a plain plan's dry run (permanent: boundary: the record's Alternatives; its author wrote every body line)

## Stop Condition

Stop and bring it to Zy if recipe 2's new words do not fit the MCP handshake's 4096 bytes, or if `planFile` does not hold a replace hunk's replaced lines for every address form a plan maker emits.
