# ADR-189: a read numbers its lines no wider than it must

**Status:** Proposed
**Date:** 2026-11-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-020, ADR-161, ADR-165, ADR-188, docs/bench/observability/QUIET-RESULTS.md, docs/bench/observability/receipt-breakdown.json
**Governs:** `internal/read/read.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `internal/mcp/era_test.go`, `scripts/contract.sh`
**Enforced-by:** `internal/read/gutter189_test.go::TestAReadNumbersLinesAsWideAsTheLargestItPlans`
**Invalidates:** None — every served line still opens with a space, its number and `| `, so a parser of `^ *[0-9]+\|` or `^ +[0-9]+\|`, or one that splits on `|`, reads it as before; the `==>` header, every `@@` range line, every refusal, `--quiet` and `-N` are unchanged.
**Served-path change:** a served line's number is right-aligned, after one space, to the width of the largest line number its file block's spans plan to serve (` 7| ` in a block planned up to line 9, `  7| ` up to 99), instead of to five columns (`    7| `). `mrw read --json` content and the `mrw_read` answer change with it.

## Context

The quiet-receipts run (`docs/bench/observability/QUIET-RESULTS.md`, run 8, 2026-11-01) left native lower by $0.016 a round, and named MRW's numbered reads as part of what remains. Measured from that run's private transcripts on 2026-11-01, by a script that summed each part of every `mrw read` result: MRW's `mrw read` results were 30,435 characters over 10 cells (`receipt-breakdown.json`, which counts every read-shaped call, puts MRW's reads at 33,298 and native's at 26,471). Of the 30,435:

| Part | Characters |
|---|---:|
| the five-column gutter, `    N| `, 7 characters on each of 720 served lines | 5,040 |
| the `==>` file headers | 3,754 |
| the `@@ A-B` range lines | 1,286 |

The task files are short: the 720 served line numbers in run 8 averaged under two digits. A gutter as wide as the block's largest number, after one space, is 3,509 characters there, a saving of 1,531, about 5% of MRW's read text. The private transcripts are not published; the figures are this record's, re-derivable by whoever holds the run.

Owner direction, 2026-11-01: *"Draft ADR, both changes (Recommended)"*, then, when the challenge below showed the second change saves 246 characters and breaks two consumers, *"Drop it, gutter only (Recommended)"*.

**Audit of the class** — *every renderer that prints a line number beside a served or written line*. Enumerated with `rg -n 'd\| %s' internal cmd --glob '*.go' --glob '!*_test.go'` on 2026-11-01: five sites. `internal/read/read.go:1114` is the served read; `internal/apply/apply.go:2028` is the write receipt's `--echo-pad` echo; `internal/preview/preview.go:32,40,47` is the dry-run diff. This record changes the first only: the other four are write receipts, which ADR-188 lets a host quiet, and their gutter is not what a read costs.

**Consumers of the served line's shape**, enumerated on 2026-11-01 by `rg -n '\[0-9\]\+\\?\||\\d\+\\\|'` over the tree outside tests and docs, by `rg -n '"\| "|NNNNN'` over `internal`, and by the Codex challenge of this draft:
- `internal/mcp/ack.go:363` finds a served line's number by `|` and `TrimSpace`: any width reads.
- `internal/mcp/tools.go:2068-2085`, the overflow classifier, counts completed lines and lines holding `| `, never the gutter's width.
- `scripts/measure.sh:165-175` reads the `==>` and `@@` lines only.
- `scripts/contract.sh` matches `^ +[0-9]+\|` at three rows, `^ *[0-9]+\|` at four and Python's `^ *\d+\|` at two, which is why one leading space stays; AGENTS.md's plan recipe matches `^ *[0-9]+\|`.
- `.claude/hooks/rules-on-read.py` reads the `==>` header only.
- `TestALegacyResultIsUnchangedByTheModernPath` (`internal/mcp/era_test.go`) freezes one read answer and lists every record that changed it; this record is added to that list.
- 27 lines in 13 Go test files and 15 lines of `scripts/contract.sh` assert the five-column gutter by its spaces (`rg -c ' {2,4}[0-9]+\| '`); they move with the format. Tests that budget `--max-chars` from the rendered text (`internal/read/maxchars165_test.go`, `cmd/mrw/maxchars165_test.go`) measure the output and follow it.

## Existing Primitives Audit

- The span loop in `internal/read/read.go` computes a file block's spans — ADR-161's line after a range included — before it prints one, so the width is the digits of the largest span end, known before the first line.
- The `--max-lines` budget and ADR-165's `--max-chars` budget cut inside that loop; computing the width before either is what keeps the width from depending on the cut.

## Decision

1. **A served line is `" "`, its number right-aligned to W, `"| "`, then the line**, where W is the number of digits of the largest line number the file block's spans plan to serve — ADR-161's line after a range included — fixed before `--max-lines`, `--max-chars`, `--fold` or `--no-comments` cut or elide anything. One file block, one width; the next file block computes its own.
2. **Everything that renders the read follows it:** `mrw read`, `--json`'s `content`, `mrw_read`. Every `==>`, `@@`, `WITHHELD`, `!!`, `-- …` and checkpoint line keeps its format; where a character budget (`--max-chars`, an `mrw_read` page) decides what fits, the ranges and counts those lines name follow what now fits, which can be more lines than before (the Codex re-challenge: nine one-character lines under `--max-chars 63` serve 1-3 today, 1-4 with this gutter).
3. **Nothing else changes:** `-N`, `--quiet`, `--stat`, `--count`, `--outline`, `--json-path`, the write receipt's echo and the dry-run diff.

**What would make this decision wrong:** a client that reads line numbers less accurately from a narrower gutter. ADR-020's reading 8 found that a client does not count rows when the only gutter it sees is mrw's; nothing has measured a narrower one. A run against native after T1 (Follow-ups) answers it; a rise in refused or misaddressed writes there reverses this record.

## Alternatives Considered

- **No `@@ 1-N` line on a whole-file span.** Drafted as a second decision and withdrawn: 32 of run 8's 162 range lines were whole-file, 246 characters, and the line is read by `scripts/measure.sh:173` for its baseline and assumed by `internal/mcp/tools.go:2074`'s overflow advice (the Codex challenge of this draft).
- **No leading space (`17| `).** Saves 720 more characters in run 8, but breaks `^ +[0-9]+\|`, which this repository's own contract wrote three times: evidence that callers write it.
- **The width of the lines actually served.** Under `--max-chars` the width would change which lines fit, and which lines fit would change the width (the Codex challenge); planned spans fix it first.
- **A width per span, or from the file's length.** Lines of one file would shift between spans; a 9,000-line file read at line 5 would pay four digits for one.
- **Drop the gutter (`-N` by default).** Rejected: the numbers are what a plan addresses.

## Component / Boundary Impact

`internal/read` (owned by this record), the MCP legacy golden and its comment, contract rows. No other engine package.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` served lines | the gutter's width | T1 | agents, parsers of `^ *[0-9]+\|` |
| `mrw read --json` `content`, `mrw_read` text | the same | T1 | adapters, MCP hosts |
| `internal/mcp/testdata/legacy_golden.jsonl` | the one read answer | T1 | `TestALegacyResultIsUnchangedByTheModernPath` |
| contract §325 | new rows; five-column assertions moved | T1 | CI |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** in run 8's shape, 1,531 characters fewer over ten cells, about 5% of MRW's read text.
- **Negative:** a file block planned past line 9,999 gets a gutter one character wider than today, the leading space added to five digits; saved outputs differ from new ones.
- **Neutral:** a parser that splits on `|` sees the same numbers.

## Out of Scope

- The write receipt's `--echo-pad` echo and the dry-run diff's gutter (permanent: boundary: write receipts, which ADR-188 lets a host quiet; not what a read costs)
- The `==>` header and the `@@` range line (permanent: boundary: their sha, length and range are what `--if-changed`, `scripts/measure.sh` and the MCP overflow advice read; see Alternatives)
- The measurement against native (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A narrower gutter costs address accuracy | Low | Med | the follow-up run; revert T1 |
| A caller's parser assumed five columns | Low | Low | the consumer sweep above found none outside tests; one space kept |

## Rollback

Revert T1; the five-column gutter returns.

## Follow-ups

- [ ] A grouped run against native on the release that carries T1, its predictions registered first (BACKLOG.md)
