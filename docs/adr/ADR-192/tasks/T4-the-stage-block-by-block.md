# Task ADR-192-T4: Stage 7, block by block, until bench look says it is complete

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** L (cross-boundary)
**Owner:** Zy
**Produces:** `docs/bench/stage-7/block-K/` receipts, `run.json` and `looks/look-K.txt`
**Consumes:** the frozen sentence and its sha (T3), comparisons `H:B` and UNTESTED (T2)
**Data dependency:** needs the release binary, the pinned Claude Code, Docker images for the 191 instances, the subscription
**Proof map:** v1

## Goal

The stage runs ADR-162's blocks with A, B and H on both models, a look after each, until every comparison is
closed or five looks have run (ADR-192 Decisions 2 to 4).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-7/` | add | each block's receipts, `run.json`, the look after it, the contamination scan |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-7/looks/` holds no look, so the fence fails. [proof: acceptance]
2. [S2] For each block: the load below the core count and the disk above the floor (ADR-162 Decision 9); `bench run --profile stage7` with the configurations `bench look` printed; the contamination scan; `bench look`, its output committed as `looks/look-K.txt`. A scan hit stops the stage to Zy. [proof: human: each look's eligibility, verdicts and the scan's result]
3. [S3] Stop when `bench look` says the stage is complete. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-7
go run ./cmd/bench look $R > /tmp/adr192-look.txt \
  && diff /tmp/adr192-look.txt "$(ls $R/looks/look-*.txt | sort -V | tail -1)" \
  && grep -q 'the stage is complete' /tmp/adr192-look.txt \
  && jq -e -s 'length > 0 and all(.[]; .stage == "stage7" and .profile == "stage7")' $R/block-*/run.json > /dev/null \
  && test "$(jq -r -s '[.[].h_sentence_sha256] | unique | length' $R/block-*/run.json)" = 1 \
  && test "$(jq -r -s '[.[].mrw_version | select(. != null)] | unique | length' $R/block-*/*.receipt.json)" = 1
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the committed looks reproduce, the stage is complete, one profile, one sentence, one release | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the blocks |
| 2 — something selects it | `bench look` reads every block |
| 3 — the caller can discover it | `looks/look-K.txt` |
| 4 — it is used | T5 |

## Mutation Log

## Invariants

- Nothing is chosen after a look except what ADR-162's stop rule and ADR-192's gate decide.

## Risks

- Shared-machine load: wall time stays descriptive; a block waits for the load to fall.

## Stop Condition

A failed canary, a contamination hit, three start errors in a row, or a disk below the floor: stop to Zy.

## Out of Scope

- Interpreting the verdicts (T5's job).

## Verification Log
