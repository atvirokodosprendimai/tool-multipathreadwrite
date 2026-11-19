# Task ADR-206-T4: the counting script

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `scripts/bench_stage9.py` with `count`, `report` and `spend` verbs; `scripts/test_bench_stage9.py`
**Consumes:** receipts and `run.json` (T1), the exposed log's format (T2), `classes.json` (T3), `scripts/bench_mechanism.py`
**Data dependency:** none — hermetic (fixture receipts, logs and transcripts)
**Proof map:** v1
**Rests-on:** `exact, wrong_location and other_inexact partition the attempts`, `a null cost is missing`, `a comparison under its coverage prints INSUFFICIENT`, `exposure in N is the log and in M the refusals`, `the runs' hashes must agree`, `the bootstrap is seeded`

## Goal

ADR-206 Decision 7. From fixture run directories: every measure of the backlog entry, each comparison printed as INSUFFICIENT with its reason when under 10 of 12 paired, when a paired cost is missing, when an arm has zero `exact`, or when over 5% of resamples are dropped; `spend` prints the stage's total across directories; and the script refuses run directories whose instruction hashes differ between M and N, whose Claude Code releases differ, or whose task lists are not the class list's.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/bench_stage9.py`, `scripts/test_bench_stage9.py` | new | the script and its tests |
| `.github/workflows/ci.yml` | edit | the new suite beside CI's Python suites |

## Ordered Steps

1. [S1] The tests, RED: no script. [proof: acceptance]
2. [S2] The script. Mutants, each killed: a null cost counted as zero; INSUFFICIENT replaced by a ratio; `wrong_location` and `other_inexact` merged; the hash comparison dropped; the seed taken from the clock. [proof: mutation]

## Acceptance

```bash
set -o pipefail
python3 -B -m unittest discover -s scripts -p 'test_bench_stage9.py' 2> "${TMPDIR:-/tmp}/adr206-t4.out" \
  && grep -q '^OK' "${TMPDIR:-/tmp}/adr206-t4.out" \
  && grep -q 'test_bench_stage9.py' .github/workflows/ci.yml \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_bench_stage9.py` | `scripts/test_bench_stage9.py` | each measure and each INSUFFICIENT rule on fixtures built to hit it; the refusals for mismatched runs | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | T6 runs it |
| 3 — the caller can discover it | its docstring and `docs/bench/README.md` |
| 4 — it is used | T6's fence re-runs it over the receipts |

## Mutation Log

## Verification Log

## Invariants

- The script reads; it writes only to standard output.

## Risks

- Reading refusals out of transcripts depends on ADR-203's fixed words: a test pins them against `internal/apply`'s text.

## Stop Condition

The fence exits 0 and every gate in `AGENTS.md` passes.

## Out of Scope

- Verdict words for ADR-203 or the taught form (permanent: boundary: the pilot decides neither)
