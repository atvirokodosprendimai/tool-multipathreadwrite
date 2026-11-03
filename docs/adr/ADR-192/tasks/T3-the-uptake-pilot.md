# Task ADR-192-T3: the release, the canary and an uncounted uptake pilot freeze the sentence

**Depends-on:** T1, T2
**Blocked-on:** a tag that contains ADR-191's merge (`git tag --contains 1f01dcdc | grep -q .`)
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** Zy
**Produces:** the frozen sentence and its sha, `docs/bench/stage-7/pilot.md`
**Consumes:** configuration H and profile `stage7` (T1)
**Data dependency:** needs the release binary, the pinned Claude Code, and 2 instances outside the pool
**Proof map:** v1

## Goal

Before any counted attempt, the canary passes for A, B and H on both models and an uncounted pilot shows whether
the sentence moves the agent to the host Read (ADR-192 Decisions 1 and 6).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-7/pilot.md` | add | the canary verdicts, the pilot's host-Read and hook counts, the frozen sentence and its sha |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-7/pilot.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] With the release binary, run the stage7 canary for A, B and H on both models. A failed canary stops here. [proof: human: the canary's verdict per configuration and model]
3. [S3] Run H uncounted on the first 2 instances of `bench/swe-bench/selection.json` (outside the pool), both models, and count host Reads, hook-licensed writes and `hook.stderr` with `scripts/bench_mechanism.py`. With no host Read on either model, change the sentence once, run again, and record both. [proof: human: the pilot's counts and the sentence it froze]
4. [S4] Write `pilot.md` with the counts and the frozen sentence's sha, and commit it before T4. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
f=docs/bench/stage-7/pilot.md
test -f $f && grep -qE '^sentence sha256: [0-9a-f]{64}$' $f && grep -q '^canary: pass$' $f
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the pilot record exists with the sha and a passing canary | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pilot record |
| 2 — something selects it | T4 runs with the sha it froze |
| 3 — the caller can discover it | `run.json` `h_sentence_sha256` |
| 4 — it is used | T4 |

## Mutation Log

## Invariants

- No pilot attempt is counted or kept in the stage's blocks.

## Risks

- Two instances say little about uptake over 191; the gate at look 1 is the counted check.

## Stop Condition

The sentence moves no host Read on either model after one change: stop to Zy before T4.

## Out of Scope

- Any counted attempt (T4's job).

## Verification Log
