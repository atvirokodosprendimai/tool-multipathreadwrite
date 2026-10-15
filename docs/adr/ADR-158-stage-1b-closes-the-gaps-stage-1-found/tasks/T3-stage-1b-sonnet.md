# Task ADR-158-T3: Stage 1b on Sonnet 5.5, a pilot first

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (machine time: hours)
**Owner:** Zy
**Produces:** `docs/bench/stage-1b-sonnet/` receipts
**Consumes:** the attempt environment (T1), the revised corpus and per-stage verdicts (T2)
**Data dependency:** real agent runs on the subscription, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot ran and its transcripts were read`, `every registered attempt has a receipt`, `the receipts verify`, `the stage ran exactly the unsuperseded corpus`

## Goal

The registered Stage 1b: a pilot, then 40 tasks × 2 configurations × 2 repeats on Sonnet 5.5, on this Mac.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-1b-sonnet/pilot/` | add | 5 tasks (three `varied-multi-site`, at least one agentsmemory) × 2 × 1, stage `pilot`, not counted |
| `docs/bench/stage-1b-sonnet/` | add | the stage's receipts, `run.json` and canary |

## Ordered Steps

1. [S1] Write the failing check and see it red: the fence exits non-zero while the directory is missing. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 1b]
3. [S3] The canary on the pinned CLI, then the pilot; every pilot transcript is read. Any one-turn attempt with no tool call, or a harness defect, is fixed in T1 or T2 and the pilot rerun into a new directory, the old one kept; a task is superseded after the pilot only for an oracle defect shown on its base or its reference fix. [proof: human: the executor reads every pilot receipt and transcript]
4. [S4] The full matrix, `--stage stage1b`, sequential, on this Mac. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage1b --tasks 40 --repeats 2 docs/bench/stage-1b-sonnet \
  && diff <(jq -r '.tasks[]' docs/bench/stage-1b-sonnet/run.json | sort) \
          <(for f in bench/tasks/*/task.json; do jq -r 'select(.shape != "fixture" and (has("superseded_by") | not)) | .id' "$f"; done | sort)
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerResumesWithoutRepeatingAStartedAttempt` | `internal/bench/runner_test.go` | the stage can stop and resume (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-1b-sonnet/` |
| 4 — it is used | ADR-156 T6's `RESULTS.md` |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The mrw build, the CLI version and the host are one for the whole stage, and the first two are named in every receipt.

## Risks

- Rate limits stretch the stage: resumable by design.

## Out of Scope

- Changing a task after the pilot (permanent: boundary: a task changed after it was run is a new task, with a new id)

## Stop Condition

The fence exits 0; or the pilot shows the harness cannot measure what ADR-156 Decision 4 names, which goes back to Zy.
