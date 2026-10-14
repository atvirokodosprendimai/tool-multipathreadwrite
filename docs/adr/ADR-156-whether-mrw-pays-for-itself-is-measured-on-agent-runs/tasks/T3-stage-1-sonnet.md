# Task ADR-156-T3: Stage 1, our own tasks on Sonnet 5.5

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (machine time: days, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-1-sonnet/` receipts
**Consumes:** `bench run` (T2), `bench/tasks/` (T1)
**Data dependency:** real agent runs on the subscription; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot ran and found no harness defect`, `every registered attempt has a receipt`, `the receipts verify`

## Goal

The pilot, then the full Sonnet matrix on our own tasks, with every attempt's receipt kept.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-1-sonnet/pilot/` | add | 5 tasks × 2 configurations × 1, not counted |
| `docs/bench/stage-1-sonnet/` | add | 30 tasks × 2 configurations × 2 repeats |

## Ordered Steps

1. [S1] Write the failing check and see it red: `bench verify docs/bench/stage-1-sonnet` exits non-zero while the directory is missing or incomplete. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 1]
3. [S3] The pilot; any harness defect it shows is fixed in T2 and the pilot rerun into a new directory, the old one kept. [proof: human: the executor reads every pilot receipt and transcript]
4. [S4] The full matrix, sequential, resumable. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage-1-sonnet --tasks 30 --repeats 2 docs/bench/stage-1-sonnet
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerResumesWithoutRepeatingAStartedAttempt` | `internal/bench/runner_test.go` | the stage can stop and resume (T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify` and T6's analysis read them |
| 3 — the caller can discover it | `docs/bench/stage-1-sonnet/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The mrw build and the CLI version are pinned for the whole stage and named in every receipt.

## Risks

- Rate limits stretch the stage over days: resumable by design.

## Out of Scope

- Changing a task after the pilot (permanent: boundary: a task changed after it was run is a new task, with a new id)

## Stop Condition

The fence exits 0; or the pilot shows the harness cannot measure what Decision 4 names, which goes back to Zy.
