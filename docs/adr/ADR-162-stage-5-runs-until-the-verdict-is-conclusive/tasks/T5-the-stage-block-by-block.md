# Task ADR-162-T5: ADR-106 amended, then Stage 5 block by block until the stop rule says it is complete

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** L (machine time: up to about 38 hours, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-5/block-K/` receipts, `run.json` and `canary.json` per block, and `docs/bench/stage-5/looks/look-K.txt`, `bench look`'s output after block K
**Consumes:** the pilot's sign-off (T4), `selection-adr-162.json`, `--block`, `--configs-for` and `bench look` (T3)
**Data dependency:** quality-harness ADR-106 amended and its pre-registration committed there; real agent runs on the subscription and up to 191 SWE-bench images, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `ADR-106 was amended first`, `every attempt the stop rule scheduled has a receipt and no other does`, `one release, one plugin tag and one CLI ran the stage`, `the stage stopped where the rule says`, `no transcript read the dataset`

## Goal

Quality-harness ADR-106 takes this stage's rows before anything is counted. Then stage `stage5` runs block 1, looks, and runs each next block with the configurations `bench look` names, until `bench look` says the stage is complete (ADR-162 Decisions 5, 7, 8 and 10).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-5/` | add | `block-K/` (receipts, `run.json`, `canary.json`) and `looks/look-K.txt` |

## Ordered Steps

1. [S1] See the fence red: no stage receipts. [proof: acceptance]
2. [S2] Quality-harness ADR-106 amended in that repository, through its own PR and review, to take Stage 5's rows (k = 1, this pool, this stop rule), and its pre-registration committed there, on Zy's word. [proof: human: the quality-harness merge commit of the amendment and its pre-registration]
3. [S3] Zy's word for the stage, given with its price: up to 5 blocks of about 8 hours at both models and three configurations, 1,146 attempts at most; mrw at the current release, the plugin at the pilot's tag, the CLI the pilot ran. [proof: human: Zy says go for Stage 5]
4. [S4] Per block:
   - costly-runs (load under the core count) and the 25 GiB disk floor;
   - the canary per model;
   - `bench run --stage stage5 --profile stage5 --status-line --block K` with the configurations `bench look` printed;
   - every transcript scanned as Decision 8 says, where a hit stops the stage for Zy;
   - `bench look` into `looks/look-K.txt`, committed with the block's receipts.

   Stop when `bench look` exits 0. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-5
go run ./cmd/bench look $R > /tmp/adr162-look.txt \
  && diff /tmp/adr162-look.txt "$(ls $R/looks/look-*.txt | sort -V | tail -1)" \
  && for d in $R/block-*/; do n=40; [ "$(basename "$d")" = block-5 ] && n=31; go run ./cmd/bench verify --stage stage5 --tasks $n --repeats 1 "$d" || exit 1; done \
  && jq -e -s 'length > 0 and all(.[]; .stage == "stage5" and .profile == "stage5")' $R/block-*/run.json > /dev/null \
  && test "$(jq -r -s '[.[].cli_version] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].mrw_version | select(. != null)] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].plugin | select(. != null) | .commit] | unique | length' $R/block-*/*.receipt.json)" = 1
```

`bench look` exits 0 only when the stage is complete, refuses an attempt the rule did not schedule, and finds a missing scheduled one ineligible; `bench verify` checks each block's per-model plan.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 5 (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench look`, `bench verify`, `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-5/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)

## Invariants

- One mrw release, one plugin tag and one CLI executable for the whole stage; the profile in every block.
- No block starts before the previous look is committed.

## Risks

- A multi-day run meets a Claude Code release; the pinned executable is used, and a mismatch is refused rather than disclosed.

## Stop Condition

`bench look` exits 0 and the fence passes. These go back to Zy instead:
- the canary fails;
- a transcript hit;
- the disk floor stops the run and the prune cannot restore it;
- the subscription refuses for a day.

## Out of Scope

- Pooling with Stages 1 to 4b (permanent: boundary: one verdict per model, comparison, source and stage)

## Verification Log
(empty until execute)
