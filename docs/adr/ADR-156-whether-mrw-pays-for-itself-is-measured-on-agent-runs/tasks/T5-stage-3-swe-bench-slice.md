# Task ADR-156-T5: Stage 3, the SWE-bench Verified slice

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** L (machine time: heavy — Docker, one container at a time)
**Owner:** Zy
**Produces:** `docs/bench/stage-3-swe-bench/` receipts, `bench/swe-bench/selection.json`
**Consumes:** `bench run` (T2)
**Data dependency:** the SWE-bench Verified dataset and the official harness, version recorded in S2; Docker; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the selection is the registered rule`, `the harness grades a gold patch here`, `every registered attempt has a receipt`

## Goal

20 instances, chosen by the registered rule, on both models × 2 configurations × 1, graded by the official harness.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/swebench.go` | add | selection by `sha256("adr-156:" + instance_id)`, grading through the official harness |
| `internal/bench/swebench_test.go` | add | the selection rule |
| `bench/swe-bench/selection.json` | add | the 20 instance ids |
| `docs/bench/stage-3-swe-bench/` | add | the receipts |

## Ordered Steps

1. [S1] Write the failing test and see it red: `TestTheSWEBenchSliceIsTheLowestTwentyHashes` (given a fixed list of ids, the selection is the 20 lowest hashes, in that order).
2. [S2] Record the dataset revision and the harness version; grade ONE gold patch here as a feasibility probe, bounded at 20 minutes. If it fails or runs over, stop and report. [proof: human: the executor records the probe's time and verdict]
3. [S3] Zy's word to start, recorded. [proof: human: Zy says go for Stage 3]
4. [S4] The runs, one container at a time. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -run 'TestTheSWEBenchSliceIsTheLowestTwentyHashes' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheSWEBenchSliceIsTheLowestTwentyHashes \(' "$out" \
  && go run ./cmd/bench verify --stage stage-3-swe-bench --tasks 20 --repeats 1 docs/bench/stage-3-swe-bench
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSWEBenchSliceIsTheLowestTwentyHashes` | `internal/bench/swebench_test.go` | the registered selection rule | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the selection and the receipts |
| 2 — something selects it | `bench run --source swe-bench`; T6 |
| 3 — the caller can discover it | `docs/bench/stage-3-swe-bench/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- No gold patch or hidden test reaches the agent.

## Risks

- x86 images under emulation on Apple Silicon: S2's probe decides.

## Out of Scope

- Silent mis-edits on this slice (permanent: boundary: a SWE-bench task has no allowed regions)

## Stop Condition

The fence exits 0; or S2's probe fails, which goes back to Zy.
