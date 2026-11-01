# Task ADR-189-T1: a read's gutter is as wide as its planned lines need

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the width rule in `internal/read/read.go`; `TestAReadNumbersLinesAsWideAsTheLargestItPlans`; contract §325; the updated legacy golden
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the gutter is the planned block's width`

## Goal

Decisions 1-3 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/gutter189_test.go` | new | the red test |
| `internal/read/read.go` | edit | the span loop: W from the block's planned spans, before any cut |
| Go tests asserting the five-column gutter (`rg -l ' {2,4}[0-9]+\| ' --glob '*_test.go'`, 13 files on 2026-11-01) | edit | they move with the format |
| `internal/mcp/testdata/legacy_golden.jsonl`, `internal/mcp/era_test.go` | edit | the one read answer, regenerated with `MRW_UPDATE_LEGACY_GOLDEN=1`, and the comment's list of records |
| `scripts/contract.sh` | edit | §325, and the rows asserting five columns |

## Ordered Steps

1. [S1] Write the failing test `TestAReadNumbersLinesAsWideAsTheLargestItPlans` and see it RED. Each case names its expected first served line:
   - a 9-line file read whole serves ` 1| ` (W=1);
   - a 120-line file read at `5-6` serves ` 5| ` (planned up to 7, W=1), and the case a whole-file width would fail;
   - a 12-line file read at `1-9` serves `  1| ` (ADR-161's line 10 makes W=2);
   - the 120-line file read as one spec, `f:5-6,118`, under `--max-lines 2` serves `   5| ` (its spans 5-7 and 118-119 plan up to 119, W=3, though 118 is withheld);
   - two files in one read get each its own width;
   - `-N` serves no gutter.
2. [S2] W from the block's planned spans. Mutants, each killed by the test: back to `%5d`; W from the file's length (the `5-6` case); W from the served lines rather than the planned (the `--max-lines` case); W shared across file blocks (the two-file case). [proof: mutation]
3. [S3] Move every five-column assertion in the Go tests and contract rows to the new form; regenerate the legacy golden and add this record to `TestALegacyResultIsUnchangedByTheModernPath`'s list. [proof: acceptance]
4. [S4] Contract §325: the built binary reading a 9-line file whole serves ` 1| alpha`-shaped lines with no `    1| `; the pair, a 120-line file read at `100-101` serves ` 100| ` (W=3). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ -count=1 -v -run '^TestAReadNumbersLinesAsWideAsTheLargestItPlans$' 2>&1) \
  && grep -qE '^--- PASS: TestAReadNumbersLinesAsWideAsTheLargestItPlans \(' <<<"$out" \
  && go test ./internal/mcp/ -count=1 -run '^TestALegacyResultIsUnchangedByTheModernPath$' >/dev/null \
  && grep -qF 'ADR-189' internal/mcp/era_test.go \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 325\. ' scripts/contract.sh \
  && grep -qF 'PASS  a short file is numbered one digit wide after one space' <<<"$c" \
  && grep -qF 'PASS  the pair: a block planned past line 99 is numbered three wide' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadNumbersLinesAsWideAsTheLargestItPlans` | `internal/read/gutter189_test.go` | one width per file block, from its largest planned line, before any cut; `-N` unchanged | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `mrw read` that serves numbered lines, `--json` and `mrw_read` |
| 3 — the caller can discover it | it is the output itself |
| 4 — it is used | no telemetry (ADR-009); the follow-up run measures it |

## Mutation Log

## Verification Log

## Invariants

- Every served line still matches `^ +[0-9]+\| `.
- Every `==>`, `@@` and checkpoint line keeps its format; under a character budget the ranges and counts it names follow what fits.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The write receipt's echo and the dry-run diff (permanent: boundary: ADR-189 Out of Scope)
