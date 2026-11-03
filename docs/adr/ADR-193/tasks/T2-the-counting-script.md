# Task ADR-193-T2: the frozen counting script and Stage 5's reference row

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `scripts/bench_edits.py` (`count`, `verdict`), `docs/bench/stage-8/reference.md`
**Consumes:** none
**Data dependency:** hermetic for the tests; the reference row needs Stage 5's B transcripts on this Mac (`~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5/block-*/`)
**Proof map:** v1
**Rests-on:** `a write call is classified by its result`, `a bench hook block is not a write`, `uptake counts the format flag`, `a SEARCH/REPLACE heredoc is one write call`, `the verdict follows Decision 5`, `the reference row names the script that counted it`

## Goal

ADR-193 Decision 4's counting rule and Decision 5's decision rule exist as one script, tested on synthetic
transcripts and receipts, committed before any counted call; and Stage 5's B is counted by it as a reference row.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/bench_edits.py` | add | `count TRANSCRIPT...` prints per-transcript landed, refused (by marker), undecided, blocked and SEARCH/REPLACE write calls as JSON; `verdict --receipts DIR --counts FILE` prints the per-model measures (i), (ii), (iii) with its interval, (U), (a), (b), (c), the per-model result and a last line `verdict: SHIP` or `verdict: NO SHIP`. Imports `mrw_calls` and `result_text` from `bench_mechanism.py` |
| `scripts/test_bench_edits.py` | add | the tests below |
| `docs/bench/stage-8/reference.md` | add | Stage 5's B per model under the frozen rule, the transcripts' path and count, and a `script sha256: ` line |

## Ordered Steps

1. [S1] Write the four failing tests and see them red (the script does not exist). [proof: acceptance]
2. [S2] `count`: a write call is one Bash tool call with an `mrw write` not `--dry-run`; blocked, landed, refused and undecided exactly as ADR-193 Decision 4 lists the markers; SEARCH/REPLACE by the format flag. [proof: mutation]
3. [S3] `verdict`: the pairing by instance and model, the exclusion rule, (i), (ii), (iii) as ratios of sums, the bootstrap (4,000 resamples, seed 193), the thresholds of Decision 5 as constants, and the verdict line. [proof: mutation]
4. [S4] Run `count` over Stage 5's 382 B transcripts and write `reference.md`, with the script's sha256. Its figures should match ADR-193's Context (counted 2026-11-03 with the rule's draft); a difference is written down beside them, never smoothed. [proof: human: the counts against the Context's]
5. [S5] Commit before T4's first counted call. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(python3 -B -m unittest discover -s scripts -p test_bench_edits.py -v 2>&1) \
  && for t in test_a_write_call_is_refused_only_on_an_mrw_refusal test_a_bench_hook_block_is_not_a_write test_uptake_counts_search_replace_write_calls test_a_search_replace_heredoc_is_one_write_call test_the_verdict_follows_the_registered_rule; do grep -qE "^$t \(.*\) \.\.\. ok$" <<<"$out" || exit 1; done \
  && test "$(sed -n 's/^script sha256: //p' docs/bench/stage-8/reference.md)" = "$(shasum -a 256 scripts/bench_edits.py | cut -d' ' -f1)"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_a_write_call_is_refused_only_on_an_mrw_refusal` | `scripts/test_bench_edits.py` | `— applied` lands; each refusal marker refuses; a result with neither is undecided; two writes chained in one call count once; a `--dry-run` write is no write call | none | S2 |
| `test_a_bench_hook_block_is_not_a_write` | `scripts/test_bench_edits.py` | a result holding `PreToolUse:Bash hook error` is blocked and out of the rate, even when it also holds a refusal marker | none | S2 |
| `test_uptake_counts_search_replace_write_calls` | `scripts/test_bench_edits.py` | `--format=search_replace` and `--format search_replace` count; a native plan and `--format=apply_patch` do not | none | S2 |
| `test_a_search_replace_heredoc_is_one_write_call` | `scripts/test_bench_edits.py` | a quoted heredoc to `$TMPDIR` then `mrw write --format=search_replace FILE`, and a document piped to `-`, each count as one SEARCH/REPLACE write call, not undecided | none | S2 |
| `test_the_verdict_follows_the_registered_rule` | `scripts/test_bench_edits.py` | synthetic receipts and counts: 8 of 18 using SEARCH/REPLACE is UNTESTED and 9 is not, 7 of 16 is and 8 is not; solved minus 3 FAILs and minus 2 does not; (i) at two thirds passes and just above it does not unless (ii) is at most 0.85; (iii) at +10% passes and +11% FAILs; one model short of PASS gives `verdict: NO SHIP` | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the script and its four tests |
| 2 — something selects it | T4's fence runs `verdict` and diffs its output |
| 3 — the caller can discover it | the script's usage line; ADR-193 Decision 4 names it |
| 4 — it is used | `reference.md`, T4 |

## Mutation Log

## Invariants

- The script's thresholds, markers, seed and resample count are ADR-193 Decisions 4 and 5's, and do not change after T4's first counted call.

## Risks

- A receipt the agent hid (`> /dev/null`, `--quiet`) is undecided; the count reports how many.

## Stop Condition

The reference row differs from the Context's by more than a few calls and the difference cannot be explained by a named rule: stop to Zy before T4.

## Out of Scope

- The bench configuration (T1's job).

## Verification Log
