# ADR-098: The MCP surface says what it does, and an ast_grep index pages

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — approved the plan "close the open items after v1.32.0" whose PR B is this record's scope, set the goal "implement this plan end to ned, test properly, test for dead code, gaps", and said "go". The record's own text was drafted and cold-reviewed after that approval and was not shown to Zy before execution: this line is the session's reading of the approval, stated so it can be checked
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-014, ADR-017, ADR-023, ADR-032, ADR-035, ADR-052, ADR-054, ADR-057, ADR-058, ADR-093, ADR-096
**Governs:** `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/instructions.go`, `internal/mcp/schema.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `scripts/contract.sh`, `README.md`, `AGENTS.md`
**Enforced-by:** `internal/mcp/astgrepindex098_test.go::TestAnAstGrepIndexPagesToTheEnd`
**Invalidates:** none — checked. ADR-017's `after` rule ("it resumes a grep's index") is widened to the index an ast_grep read returns, which ADR-058 made the same INDEX; every other clause of ADR-017 holds. The text corrections change what the MCP surface SAYS, never what it does, except T1's one behaviour.
**Served-path change:** `mrw_read` accepts `after` with `ast_grep` and resumes that index, where v1.32.0 refused it ("after without grep"); `after` with plain specs is still refused. The instructions, both tool descriptions and the write outputSchema are corrected in ten places (listed in Decision 2); no other answer, refusal, exit code or field changes.

## Context

**What was observed.** A prompt audit of 2026-09-29 (target Opus 5.5, `main` 25acdc1; its rows are the
palace findings drawer "prompt audit 2026-09-29") found text the MCP server sends that disagrees with
the code. Every row was re-checked against HEAD `db39d42` on 2026-09-29; all still hold:

| Row | What the surface says | What the code does |
|---|---|---|
| H7 | `after`: "Repeat until `next_index` is absent" (`mcp.go:463`); the index prose says the same (`tools.go:1543`) | `next_index` is always present, `""` on the last page (`tools.go:1523-1536`, `:1559`) |
| H8 | an oversized `ast_grep` answer is an INDEX that says "Send the SAME grep again with after=" (`tools.go:1543`, `:1548`) | `after` without `grep` is refused (`tools.go:296-297`) and `astGrepSpecs` takes no cursor (`:1456-1467`): page two cannot be reached |
| M7 | outputSchema `hunks.status`: `ok` "when this hunk's file reached disk" (`schema.go:253`) | a dry run reports `ok` with nothing written (`apply.go:589-592`); the engine comment `apply.go:38-41` says the same wrong thing |
| M8 | "Guards, checked on every op" (`instructions.go:104`, `mcp.go:546`) | `sha=` is checked on every op; create, unlink and rename refuse `anchor=` and `lines=` (`plan.go:794`, `:813`) |
| M9 | `exclude`: "Only meaningful with `grep` or `ast_grep`" (`mcp.go:452`) | refused without them (`tools.go:286-288`) |
| M10 | `specs`: "With `grep` set these are DIRECTORIES OR FILES TO SEARCH" (`mcp.go:432`) | the same holds with `ast_grep`, ranges refused (`tools.go:1462-1463`) |
| M11 | "CLI has --files-from, --check" (`instructions.go:64`); "it also has --check, which runs the project's tests" (`mcp.go:491-492`) | since ADR-054 the CLI's write runs the project's check after a write that touches code, when a check is declared or inferred and `--no-check` is not given (`cmd/mrw/main.go:1289`, `:1346`); `--check` demands it on prose too; this surface runs none (`tools.go:708`) |
| M12 | rename is listed as an op and never shaped | address `-`, one body line naming the destination (`plan.go:801-812`) |
| M13 | ADR-052's rule is served nowhere on this surface's static text | a multi-line replace needs a served line after End, except at the last line (`apply.go:1366`) |
| M14 | `elided` drops "successful and skipped hunk verdicts … and file records" (`schema.go:227`); "drops successes then UNWRITTEN files" (`instructions.go:114`) | ok and skipped verdicts go, then UNWRITTEN file records; failed hunks and written files are always kept (`tools.go:808-834`) |
| M15 | "With a shell prefer the CLI" appears twice in the instructions (`instructions.go:63`, `:68`) | — (costs bytes the corrections need) |

Also stale: the comment at `internal/read/read.go:589` cites `apply.go:748`; the end-pattern block
it means is `THE END IS A DELIMITER` in `Apply` (`apply.go:1169`).

**Budget.** The instructions are 3,987 of 4,096 bytes (`maxInstructionsChars`,
`internal/mcp/instructions_test.go:10`; measured 2026-09-29 through `initialize` on `db39d42`), enforced by
about ten tests and contract §43, §50 and §52. Every addition is paid for inside the same file.

**Audit of the class.** The class is *a sentence the MCP server sends that states a behaviour the code
does not have*. Enumerated 2026-09-29 by reading every served string against its implementation:
`instructionsText()` (`instructions.go:62-117`), `tools()` descriptions and inputSchema (`mcp.go:389-562`),
`writeSchema()` (`schema.go:208-259`), and the index prose in `matchIndex` (`tools.go:1539-1553`). The
rows above are every disagreement found: 11 (H7, H8, M7–M15). **Left out on purpose:** the CLI's
`--help` and `mrw instructions` (`internal/guide`) — a different surface, audited separately by the same
report and already corrected by PR #282 where it applied; the read receipt's field descriptions, which
the audit found accurate.

## Existing Primitives Audit

- **`grepSpecs`' cursor skip** (`tools.go:1440-1450`) — reused, factored into one helper both finders call.
- **`read.AstGrep`** (`internal/read/astgrep.go`) — unchanged; its output is already path-sorted
  (`astgrep.go:200`), so the cursor is a position in the same total order grep's is.
- **`matchIndex`** (`tools.go:1469+`) — unchanged in shape; its prose names the finder that was used.
- **`TestALegacyResultIsUnchangedByTheModernPath`** and `testdata/legacy_golden.jsonl` — the golden is
  regenerated with `MRW_UPDATE_LEGACY_GOLDEN=1` and the regeneration logged in `era_test.go`'s comment,
  as ADR-070, 075, 076, 090, 091 and 093 did.
- **`_adr035_desc_problems`** (`scripts/contract.sh`, §43) — unchanged; the corrected guard sentence must
  pass its wording rules.

## Decision

1. `mrw_read`'s `after` resumes an `ast_grep` index exactly as it resumes a `grep` index: everything at or
   before the cursor is dropped from the finder's path-sorted specs. `after` with neither `grep` nor
   `ast_grep` is refused as before, with the refusal naming both. The index prose names the finder that
   produced it ("Send the SAME ast_grep again …"), and says to repeat until `next_index` is empty.
2. The served text is corrected in exactly these places, each to what the code in the Context table does:
   H7 (`after` description), M7 (`hunks.status`), M8 (guards, both copies), M9 (`exclude`), M10 (`specs`,
   `ast_grep`), M11 (the CLI's check), M12 (rename's shape), M13 (ADR-052's rule), M14 (`elided`, both
   copies), and M15 removes the duplicate sentence. The engine comments `apply.go:38-41` and
   `read.go:589` are corrected to the code beside them; no engine statement changes.
3. The instructions stay within 4,096 bytes; nothing raises the bound.

## Alternatives Considered

- **Refuse `ast_grep` indexes instead of paging them** (serve nothing past page one, say so). Rejected: the
  index already exists and advertises a continuation; ADR-014 and ADR-017 settled that a continuation a
  caller cannot follow is a dead end, and the cursor code already exists for grep.
- **Raise `maxInstructionsChars`.** Rejected: `TestMCPInstructionsContainShared` forbids it on purpose
  (every session pays those bytes); M15 and tightening pay for M12 and M13.
- **Leave the text and document the gaps in README.** Rejected: the MCP-only caller reads the served text,
  not the README (`mcp_test.go:317` forbids the instructions pointing at repository files).

## Component / Boundary Impact

None — internal to `internal/mcp` (one behaviour in `tools.go`, strings elsewhere) plus two engine
comments. `go.mod` keeps its one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` `after` with `ast_grep` | accepted, resumes the index | T1 | MCP hosts paging a structural search |
| `mrw_read` index prose and `after` description | names the finder; "until it is empty" | T1 | MCP hosts |
| instructions, tool descriptions, write outputSchema | ten corrections | T2 | MCP hosts and models reading the handshake |
| `scripts/contract.sh` | §189 (ast_grep index paged through the binary), §190 (corrected text served by the binary) | T1, T2 | CI Linux |
| `README.md`, `AGENTS.md` | the ast_grep index pages with `after` | T3 | readers |

No exit code, flag, CLI behaviour or field is added or removed.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `after` accepted with `ast_grep` (`afterCursor` in `tools.go`) | T1 | T2, T3 | No — T2 describes it, T3 documents it |

## Implementation

See `tasks/README.md`: T1 (the behaviour, red first, contract §189), T2 (the served text, golden
regenerated, contract §190), T3 (README and AGENTS).

## Consequences

- **Positive:** a structural search too large to serve can be read to the end over MCP; a model reading
  the handshake is told what the tools do, including three refusals it was told were optional.
- **Negative:** the golden transcript changes; any host diffing tool descriptions sees ten edits.
- **Neutral:** the CLI is unchanged: it has no index and no `after`.

## Out of Scope

- The CLI `mrw read --ast-grep`, which builds no index (permanent: boundary: the CLI serves or refuses per spec with `--max-lines`; an index exists only where a ceiling does)
- `internal/guide` (`mrw instructions`) and `--help` (permanent: boundary: a different surface, corrected where the same audit found a gap in PR #282)
- Raising the 4,096-byte instructions bound (permanent: boundary: `TestMCPInstructionsContainShared` forbids it; the cost is paid by every session)
- Paging a grep or ast_grep index by anything other than path order (permanent: boundary: ADR-017's cursor is a readable position in `read.Walk`'s and `read.AstGrep`'s shared path order)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a real ast-grep emits hits out of path order | Low | Med | `read.AstGrep` sorts (`astgrep.go:200`); T1's test feeds unsorted hits from a fake and pages to exhaustion |
| the corrections push the instructions over 4,096 bytes | Med | Low | the existing bound tests fail; M15 and tightening pay first; T2 measures |
| a corrected guard sentence trips §43's `_adr035_desc_problems` | Med | Low | T2 runs contract.sh; the rules are known (Context) |

## Rollback

Revert T1–T3 and regenerate the golden. No state, no exit code and no field changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/mcp/astgrepindex098_test.go::TestAnAstGrepIndexPagesToTheEnd` in T1's commit.
