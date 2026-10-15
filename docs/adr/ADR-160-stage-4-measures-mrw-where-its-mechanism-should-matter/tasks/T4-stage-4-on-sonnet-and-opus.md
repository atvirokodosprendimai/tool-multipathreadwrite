# Task ADR-160-T4: Stage 4 on Sonnet 5.5 and Opus 5.5

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** L (machine time: hours per model, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-4-sonnet/`, `docs/bench/stage-4-opus/`, `docs/bench/stage-4-swe/` receipts
**Consumes:** the calibrated corpus (T3), `bench/stage-4-tasks.txt` and `bench/swe-bench/selection-adr-160.json` (T2)
**Data dependency:** real agent runs on the subscription and SWE-bench images, on this Mac; each model's stage starts only on Zy's word
**Proof map:** v1
**Rests-on:** `every registered attempt has a receipt`, `the receipts verify`, `the stages ran the task list on the registered models`, `the slice ran the selection on both models`

## Goal

The registered Stage 4: 24 own tasks × A/B × 2 on each model, and the 20-instance `swe-multi-hunk` slice × A/B × 1 on each model, one image at a time.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-4-sonnet/` | add | own tasks on Sonnet 5.5, stage `stage4` |
| `docs/bench/stage-4-opus/` | add | own tasks on Opus 5.5, stage `stage4` |
| `docs/bench/stage-4-swe/` | add | the slice on both models, stage `stage-4-swe` |

## Ordered Steps

1. [S1] See the fence red: no stage receipts. [proof: acceptance]
2. [S2] Zy's word for each of the three stages — Sonnet's own tasks, Opus's own tasks, the slice on both models — given with its price. [proof: human: Zy says go for that Stage 4 stage]
3. [S3] Each stage with `--task` for every id in `bench/stage-4-tasks.txt` (the slice with `--selection bench/swe-bench/selection-adr-160.json`), sequential, resumable, with a canary per model; every slice transcript scanned for a read of the dataset file, as in Stage 3. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage4 --tasks 24 --repeats 2 docs/bench/stage-4-sonnet \
  && go run ./cmd/bench verify --stage stage4 --tasks 24 --repeats 2 docs/bench/stage-4-opus \
  && jq -e '.models == ["claude-sonnet-5-5"]' docs/bench/stage-4-sonnet/run.json > /dev/null \
  && jq -e '.models == ["claude-opus-5-5"]' docs/bench/stage-4-opus/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-sonnet/run.json | sort) <(jq -r '.tasks[]' docs/bench/stage-4-opus/run.json | sort) \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-sonnet/run.json | sort) \
          <(sort bench/stage-4-tasks.txt) \
  && go run ./cmd/bench verify --stage stage-4-swe --tasks 20 --repeats 1 docs/bench/stage-4-swe \
  && jq -e '(.models | sort) == ["claude-opus-5-5", "claude-sonnet-5-5"]' docs/bench/stage-4-swe/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-swe/run.json | sort) <(jq -r '.selected[]' bench/swe-bench/selection-adr-160.json | sort)
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 4 (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-4-*/` |
| 4 — it is used | T5 |

## Mutation Log
(empty until execute)

## Invariants

- The mrw build, the CLI version and the host are one for each stage, named in every receipt.
- Pilot receipts are never in a stage directory.

## Risks

- Opus may be at ceiling even on the calibrated corpus; that is a result, published as computed.

## Stop Condition

The fence exits 0; or a stage shows the harness cannot measure what ADR-156 Decision 4 names, which goes back to Zy.

## Out of Scope

- Pooling with Stages 1 to 3 (permanent: boundary: one verdict per model, source and stage)

## Verification Log
(empty until execute)
