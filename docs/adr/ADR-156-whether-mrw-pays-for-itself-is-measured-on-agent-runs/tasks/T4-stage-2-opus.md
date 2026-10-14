# Task ADR-156-T4: Stage 2, our own tasks on Opus 5.5

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** M (machine time: days, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-2-opus/` receipts
**Consumes:** `bench run` (T2), `bench/tasks/` (T1)
**Data dependency:** real agent runs on the subscription; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `every registered attempt has a receipt`, `the receipts verify`

## Goal

The same 30 tasks × 2 configurations × 2 repeats on Opus 5.5.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-2-opus/` | add | the receipts |

## Ordered Steps

1. [S1] Write the failing check and see it red: `bench verify docs/bench/stage-2-opus` exits non-zero while the directory is missing or incomplete. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 2]
3. [S3] The full matrix, sequential, resumable. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage-2-opus --tasks 30 --repeats 2 docs/bench/stage-2-opus
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | Stage 1's receipts survive Stage 2 (T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; T6 |
| 3 — the caller can discover it | `docs/bench/stage-2-opus/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- Same tasks, same mrw build as Stage 1 unless a change is named in the receipts.

## Risks

- Opus costs more of the subscription's limits per run: the stage is paused, never cut, when a limit hits.

## Out of Scope

- Pooling with Stage 1 (permanent: boundary: the bar gives one verdict per model)

## Stop Condition

The fence exits 0.
