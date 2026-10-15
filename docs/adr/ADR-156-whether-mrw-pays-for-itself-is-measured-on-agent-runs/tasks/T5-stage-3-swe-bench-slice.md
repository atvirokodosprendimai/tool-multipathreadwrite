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
- 2026-10-15 · human-observed · S2 observed 2026-10-15 on this Mac (Docker Desktop 29.8.2, arm64, 8 GB VM): dataset SWE-bench/SWE-bench_Verified test split at revision 78f471bf655a3137b2e8a75af1501690ec009ec3 (500 instances, the same ids as princeton-nlp/SWE-bench_Verified at c104f840, which lacks the image field swebench 5.0.2 reads), harness swebench 5.0.2; the registered selection is in bench/swe-bench/selection.json and equals bench swebench-select's output; the gold patch of the first selected instance, matplotlib__matplotlib-22719, graded resolved: True, all required and regression tests passing, in 25 s, after a 349 s pull of its 10.6 GB amd64 image (pulled with --platform linux/amd64, since no arm64 manifest exists) run under emulation; within the 20-minute bound, so the probe passed. Disk: twenty such images would need about 200 GB against 107 GB free, so each instance's image is removed after its attempts are graded.

## Invariants

- No gold patch or hidden test reaches the agent.

## Risks

- x86 images under emulation on Apple Silicon: S2's probe decides.

## Out of Scope

- Silent mis-edits on this slice (permanent: boundary: a SWE-bench task has no allowed regions)

## Stop Condition

The fence exits 0; or S2's probe fails, which goes back to Zy.
