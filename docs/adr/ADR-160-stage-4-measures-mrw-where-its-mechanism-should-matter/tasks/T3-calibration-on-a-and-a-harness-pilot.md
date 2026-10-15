# Task ADR-160-T3: difficulty calibrated on configuration A outside the sample, and harness pilots that choose nothing

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (machine time: about 30 attempts and a few image pulls)
**Owner:** Zy
**Produces:** `docs/bench/stage-4-pilot/` receipts, the calibrated corpus (a raised floor, if any, recorded)
**Consumes:** `bench/tasks/<stage-4 ids>/`, `bench/stage-4-tasks.txt`, `bench/swe-bench/selection-adr-160.json` (T2)
**Data dependency:** real agent runs on the subscription and SWE-bench images, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the calibration ran on A alone`, `the calibration tasks are outside the sample`, `the harness pilot ran on B`, `every pilot receipt is stage pilot`

## Goal

Each own category's difficulty measured on configuration A alone on candidates the stage never runs, its floor raised once if A is at ceiling, and B's harness and each new SWE-bench repository checked on runs whose outcomes choose nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/tasks/<calibration ids>/` | add | `wide-rename` candidates 11–14 and `check-coupled` 9–12, built as T2 builds tasks; never in `bench/stage-4-tasks.txt` |
| `docs/bench/stage-4-pilot/calibration/` | add | those 8 × A × 2 on Sonnet 5.5, stage `pilot` |
| `docs/bench/stage-4-pilot/harness/` | add | the first 2 tasks of each own category in registered order × B × 1 on Sonnet 5.5, stage `pilot` |
| `docs/bench/stage-4-pilot/swe/` | add | the first drawn instance of each repository Stage 3 did not use × A/B × 1 on Sonnet 5.5, stage `pilot` |
| `bench/tasks/<redrawn ids>/`, `bench/stage-4-tasks.txt` | edit | only if a floor is raised: the sample redrawn and the earlier draws deleted, none having run in a counted stage |

## Ordered Steps

1. [S1] See the fence red: no pilot receipts. [proof: acceptance]
2. [S2] Zy's word to start, recorded with the price. [proof: human: Zy says go for the Stage 4 pilot]
3. [S3] The calibration on A alone. For each own category count A's solved attempts out of 8; at 7 or more, raise the floor as Decision 2 registers, redraw the sample with T1's code, build it as T2 does, delete the earlier draws, and calibrate once more on the new floor's candidates outside the sample into `calibration-2/`. [proof: human: the executor records each category's count, and any raised floor, before T4]
4. [S4] The harness pilots on B and on the new repositories; every transcript of every pilot read; a harness defect is fixed and its pilot rerun into a new directory, the old one kept. [proof: human: the executor reads every pilot receipt and transcript]

## Acceptance

```bash
set -o pipefail
P=docs/bench/stage-4-pilot
go run ./cmd/bench verify --stage pilot --tasks 8 --repeats 2 $P/calibration \
  && jq -e '.configs == ["A"]' $P/calibration/run.json > /dev/null \
  && test -z "$(jq -r '.tasks[]' $P/calibration/run.json | grep -xFf bench/stage-4-tasks.txt)" \
  && go run ./cmd/bench verify --stage pilot --tasks 4 --repeats 1 $P/harness \
  && jq -e '.configs == ["B"]' $P/harness/run.json > /dev/null \
  && go run ./cmd/bench verify --stage pilot --tasks "$(jq '.tasks | length' $P/swe/run.json)" --repeats 1 $P/swe \
  && jq -s -e 'all(.[]; .stage == "pilot")' $P/*/*.receipt.json > /dev/null
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPilotIsNeverCounted` | `internal/bench/analyze_test.go` | a receipt whose stage begins `pilot` is left out of every verdict (ADR-156, the review of #416) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pilot receipts |
| 2 — something selects it | `bench run --stage pilot --configs A` / `--configs B` (`cmd/bench/main.go:7`) |
| 3 — the caller can discover it | `docs/bench/stage-4-pilot/` |
| 4 — it is used | T4 runs the calibrated corpus |

## Mutation Log
(empty until execute)

## Invariants

- No B outcome is used to choose, keep or drop a task; no measured task is a calibration task.

## Risks

- `bench verify` reads the configurations and models from `run.json` (`internal/bench/runner.go:352-356`), so an A-only directory verifies; checked before S3.

## Stop Condition

A raised floor leaves a category short of candidates; or A is at ceiling in every own category after the raise, which goes back to Zy before T4.

## Out of Scope

- Calibrating the SWE slice: Verified is what it is.

## Verification Log
(empty until execute)
