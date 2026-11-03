# Task ADR-192-T5: the verdicts, the mechanism counts and the README rows

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** Zy
**Produces:** `docs/bench/stage-7/RESULTS.md`, `docs/bench/stage-7/mechanism.json`, Stage 7's rows in `docs/bench/RESULTS.md`
**Consumes:** the looks (T4), receipt `host_reads` (T1)
**Data dependency:** needs T4's transcripts
**Proof map:** v1

## Goal

Stage 7's six verdicts are published with the look each closed at, the mechanism counts beside them, and what each
outcome means as ADR-192 Decision 5 wrote it before the run.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-7/RESULTS.md` | add | the verdicts, the counts, the outcome's meaning |
| `docs/bench/stage-7/mechanism.json` | add | `scripts/bench_mechanism.py` at `3a970837` over every H transcript |
| `docs/bench/RESULTS.md` | edit | Stage 7's rows |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-7/RESULTS.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] Run the classifier over H's transcripts; write the results with each verdict as `bench look` printed it. [proof: acceptance]
3. [S3] Add Stage 7's rows to `docs/bench/RESULTS.md`; a README sentence, if any, states no more than its verdict. [proof: human: Zy reads the README sentence against the verdict]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-7
test -f $R/RESULTS.md && jq -e '.total.host_reads >= 0' $R/mechanism.json > /dev/null \
  && for c in 'H against B' 'B against A' 'H against A'; do grep -q "$c" $R/RESULTS.md || exit 1; done \
  && grep -q 'Stage 7' docs/bench/RESULTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the results exist, name every comparison, and the counts parse | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the results |
| 2 — something selects it | `docs/bench/RESULTS.md` links it |
| 3 — the caller can discover it | the README's benchmark section |
| 4 — it is used | Zy's decision on rule 1 |

## Mutation Log

## Invariants

- The verdicts are copied from `bench look`, never recomputed by hand.

## Risks

- none: no doubles.

## Stop Condition

A verdict in the results differs from the look that closed it: stop and fix before publishing.

## Out of Scope

- The guide change itself (ADR-192 Out of Scope).

## Verification Log
