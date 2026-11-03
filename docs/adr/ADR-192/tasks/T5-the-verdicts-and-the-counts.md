# Task ADR-192-T5: the verdicts, the mechanism counts and the README rows

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** Zy
**Produces:** `docs/bench/stage-7/RESULTS.md`, `docs/bench/stage-7/mechanism.json`, Stage 7's rows in `docs/bench/RESULTS.md`
**Consumes:** the looks and H's transcripts (T4), receipt `host_reads` and `hook_runs` (T1)
**Data dependency:** needs T4's transcripts
**Proof map:** v1
**Rests-on:** `every verdict line of the last look is quoted`, `the counts cover every H transcript`

## Goal

Stage 7's six verdicts are published as the last look printed them, the mechanism counts beside them, and what each
outcome means as ADR-192 Decision 5 wrote it before the run.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-7/RESULTS.md` | add | the verdict lines quoted from the last look, the counts, the outcome's meaning |
| `docs/bench/stage-7/mechanism.json` | add | `scripts/bench_mechanism.py` at `3a970837` over every H transcript |
| `docs/bench/RESULTS.md` | edit | Stage 7's rows |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-7/RESULTS.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] Run the classifier over every H transcript; quote each verdict line of the last look in the results; add the receipts' `host_reads`, `hook_runs` and `hook_stderr_bytes`. [proof: acceptance]
3. [S3] Add Stage 7's rows to `docs/bench/RESULTS.md`; a README sentence, if any, states no more than its verdict. [proof: human: Zy reads the README sentence against the verdict]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-7
last=$(ls $R/looks/look-*.txt | sort -V | tail -1) \
  && test "$(grep -cE ' · (B against A|H against A|H against B) · ' "$last")" = 6 \
  && grep -E ' · (B against A|H against A|H against B) · ' "$last" | while IFS= read -r l; do grep -qF -- "$l" $R/RESULTS.md || exit 1; done \
  && test "$(jq '.cells | length' $R/mechanism.json)" = "$(ls $R/block-*/*.H.*.transcript.jsonl | wc -l | tr -d ' ')"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: all six verdict lines of the last look are quoted in the results, and the counts cover every H transcript | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the results |
| 2 — something selects it | `docs/bench/RESULTS.md` links it |
| 3 — the caller can discover it | the README's benchmark section |
| 4 — it is used | Zy's decision on rule 1 |

## Mutation Log

## Invariants

- The verdicts are quoted from `bench look`, never recomputed by hand.

## Risks

- none: no doubles.

## Stop Condition

A verdict line in the results differs from the look that closed it: stop and fix before publishing.

## Out of Scope

- The guide change itself (ADR-192 Out of Scope).

## Verification Log
