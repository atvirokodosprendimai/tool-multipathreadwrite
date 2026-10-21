# Task ADR-165-T2: a read fits a call-wide character budget and names what it withheld

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `read.Options.MaxChars`; `--max-chars N` on `mrw read`; the `WITHHELD … --max-chars reached` lines; the `incomplete:` ending for a reached budget
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the answer never exceeds N characters`, `every unserved range or file is named`, `only printed lines are licensed`, `the WITHHELD lines are not charged`, `the refused combinations`, `the exit is 1 incomplete`

## Goal

Decision 1: `mrw read --max-chars N …` writes at most N characters to standard output, stops at the last whole line that fits, names every range and file it did not serve, licenses only what it printed, and exits 1 `mrw: incomplete: …` naming the budget.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.MaxChars *int`; the budget check beside `MaxLines`'s in the range loop, against the shared counter; the WITHHELD list written to the error writer, not to `w` |
| `cmd/mrw/main.go` (the read action) | edit | one counting writer wrapping standard output, passed to `read.Run` and used by every note and walk report the read action prints; the WITHHELD list and the `incomplete:` ending (ADR-146's text plus `; --max-chars N reached`) on standard error |
| `cmd/mrw/main.go` | edit | the flag, its negative refusal, the refused combinations (`--sub`, `--rewrite`, `--count`, `--stat`), the option wiring, the usage text |
| `internal/read/maxchars165_test.go` | add | the engine test |
| `cmd/mrw/maxchars165_test.go` | add | the flag test through the built binary |
| `scripts/contract.sh` | edit | §282 |
| `README.md`, `AGENTS.md`, `internal/guide/guide.go` | edit | the flag beside `--max-lines`; `mrw instructions` recipe 8's sibling sentence |
| `docs/receipts.txt` | none | the CLI read has no JSON receipt until T3 |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestAReadStopsAtItsCharacterBudgetAndNamesWhatItWithheld (`internal/read`: three files, a budget that fits one and a half; the standard-output writer receives at most N runes; file 2's unserved range and file 3 are listed WITHHELD on the error writer by name; `observed` holds only the printed lines; `problems` counts each WITHHELD) and TestMaxCharsBoundsTheWholeAnswer (`cmd/mrw`, the binary: standard output's rune count is at most N for a budgeted `--grep` walk that also prints a skipped summary and a note; standard error lists the withheld files and ends `mrw: incomplete: … ; --max-chars N reached`, exit 1; `--max-chars -1` refused exit 2; `--max-chars 0` prints nothing on standard output and lists every spec on standard error; with `--sub`, `--count`, `--stat` refused exit 2 naming the pair; with `--quiet` the headers are bounded; a later write to a withheld line is refused). [proof: acceptance]
2. [S2] `Options.MaxChars` and the counting writer; the per-range and per-file checks; the ending. [proof: mutation]
3. [S3] The flag, its refusals and the usage text; the three documents. [proof: mutation]
4. [S4] Contract §282: a two-file read under a budget that fits one exits 1, names the other WITHHELD, and a plan to a withheld line is refused; the pair: the same read with no budget serves both and the plan applies. [proof: human: the executor runs ./scripts/contract.sh once and reads the §282 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAReadStopsAtItsCharacterBudgetAndNamesWhatItWithheld$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAReadStopsAtItsCharacterBudgetAndNamesWhatItWithheld \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestMaxCharsBoundsTheWholeAnswer$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestMaxCharsBoundsTheWholeAnswer \(' "$out" \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestACapOfZeroServesNothing$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACapOfZeroServesNothing \(' "$out" \
  && go test ./internal/adversarial/ -count=1 -timeout 300s -run 'TestARangedReadLicensesOnlyTheLinesItServed$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestARangedReadLicensesOnlyTheLinesItServed \(' "$out" \
  && grep -q '^# 282\. ' scripts/contract.sh \
  && grep -q 'max-chars' README.md && grep -q 'max-chars' AGENTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadStopsAtItsCharacterBudgetAndNamesWhatItWithheld` | `internal/read/maxchars165_test.go` | the bound, the naming, the licence, the uncharged report | none | S1, S2 |
| `TestMaxCharsBoundsTheWholeAnswer` | `cmd/mrw/maxchars165_test.go` | the flag, its refusals, the exit, the later write refused | none | S1, S3 |
| `TestACapOfZeroServesNothing` | `internal/read/read_test.go` | `--max-lines` unchanged beside the new budget | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Options.MaxChars`, the flag |
| 2 — something selects it | `cmd/mrw` wires the flag into `read.Options`; contract §282 drives the binary |
| 3 — the caller can discover it | `mrw read --help`, README, AGENTS.md, `mrw instructions` |
| 4 — it is used | nothing measures it; the Codex report's host limit is the case it exists for |

## Invariants

- A read with no `--max-chars` is byte-identical to today's.
- `--max-lines` keeps its per-spec meaning and its WITHHELD lines; the two budgets compose (a range withheld by either is named once).
- A WITHHELD range is not observed (ADR-117): `mrw seen` shows nothing for it.
- The count is of characters (runes) written to standard output, not bytes, and the test measures it in runes (testing.md: a bound is measured in the unit it is declared in). Standard error is not counted; the WITHHELD list goes there so standard output stays at or under N with no exception.

## Risks

- A header alone can exceed a tiny budget: then the file is WITHHELD at the header and nothing of it is served; the test covers `--max-chars 0` and a budget smaller than one header.
- `--grep` walks: a file never opened because the budget was reached before it is still named WITHHELD, which needs the walk to continue counting without serving; the engine test has a three-file walk.

## Out of Scope

- A budget on `mrw_read`: ADR-032's ceiling (permanent: fact: the MCP ceiling bounds the encoded result; citation: file `docs/adr/ADR-032-the-ceiling-is-the-callers.md:1`)
- An environment variable for the budget (permanent: boundary: a shell caller passes a flag; `MRW_MAX_RESULT_CHARS` exists because an MCP host's config cannot pass one per call)

## Stop Condition

Stop if the WITHHELD-by-name rule cannot hold for a `--grep` walk without opening every file the budget excluded (that is, if naming them costs the read the walk's whole time): then report the cost and ask whether a count of withheld files is enough.

## Mutation Log
(empty until execute)
- 2026-10-21 · f3824d00* · mutant killed · exit 1 · `internal/read/read.go` · the budget never drops a write · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66
- 2026-10-21 · f3824d00* · mutant killed · exit 1 · `cmd/mrw/main.go` · the withheld list never reaches standard error · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66
- 2026-10-21 · f3824d00* · mutant killed · exit 1 · `cmd/mrw/main.go` · a dropped line outside any range leaves the read complete · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66

## Verification Log
(empty until execute)
- 2026-10-21 · f3824d00* · exit 0 · `set -o pipefail …` · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66 · ms:3895
- 2026-10-21 · f3824d00* · exit 0 · `set -o pipefail …` · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66 · ms:4236
- 2026-10-21 · f3824d00* · exit 0 · `set -o pipefail …` · acceptance-sha256:d91180fd5a9324639c3e6bf934b8b1e695bd4b15648b4347b0d5beb3799efa66 · ms:4250
