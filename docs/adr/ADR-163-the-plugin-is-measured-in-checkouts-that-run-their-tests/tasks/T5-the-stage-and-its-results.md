# Task ADR-163-T5: the stage, block by block, and its results

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** L (machine time and usage, by block, up to the surviving pool)
**Owner:** Zy
**Produces:** `docs/bench/stage-6/block-K/` (receipts, `run.json`, `canary.json` per model, `recall.txt`), `docs/bench/stage-6/looks/look-K/` (quality-harness's scorer: its printed output and one `--verdict-out` file per model and pair), a Stage 6 section in `docs/bench/RESULTS.md`
**Consumes:** the pilots' sign-off, the surviving pool and the price (T4), the block plan and the export (T3), W and `--max-turns` (T2), the shims (T1)
**Data dependency:** quality-harness's ADR-112 pre-registration committed there first; real agent runs on the subscription and the surviving pool's SWE-bench images, on this Mac; starts only on Zy's word after T4
**Proof map:** v1
**Rests-on:** `the pre-registration came first`, `every block is a slice of the frozen surviving pool`, `the scorer's look decided each next block`, `the stage ran until every pair stopped or the pool was spent`, `one plugin tag, one CLI, one checks table and one work line ran the stage`, `no transcript read the dataset or started a container`

## Goal

Stage `stage6` runs block 1, has quality-harness's scorer read it, and runs each next block with the arms still open, until every pair has stopped or the surviving pool is spent. Then the verdicts are published (ADR-163 Decisions 4 to 6).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-6/` | add | per block: receipts, `run.json`, canaries, `recall.txt`; `looks/look-K.txt` |
| `docs/bench/RESULTS.md` | edit | Stage 6's verdicts per model and pair, with the recall row and Decision 6's counts beside them |

## Ordered Steps

1. [S1] See the fence red: no stage receipts. [proof: acceptance]
2. [S2] Quality-harness's ADR-112 pre-registration is committed in that repository: the plugin tag, W's line, the scorer blob, D1 read at 1200 s, D3's derivation rule and the surviving pool's digest and count, then the caps' numbers from T4. Before the first red run of this fence, its verdict key and words are set to the ones ADR-112 T1's `--verdict-out` writes (here `.verdict` in `PAYS`, `FUTILE`, `OPEN`). [proof: human: the quality-harness commits and their dates, before the first counted attempt]
3. [S3] Per block:
   - costly-runs and the 25 GiB disk floor;
   - the canary per model;
   - `bench run --stage stage6 --profile stage6 --selection bench/swe-bench/selection-adr-163.json --block K` with the arms the last look left open;
   - the amended Decision 8 scan, where a hit, or any command naming `docker`, a pool image or the container, stops the stage for Zy;
   - `export-paired` per model and pair; quality-harness's scorer at its pre-registered blob with `--verdict-out` for every model and pair into `looks/look-K/<model>-<pair>.json` (a stopped pair's file repeats its verdict), its printed output beside them, committed with the block. A missing or invalid file is not a look, and the next block waits. The next block's `configs_for` is A plus each model's open arms.

   Stop when every pair has stopped or the surviving pool is spent. [proof: acceptance]
4. [S4] Stage 6's section in `RESULTS.md`, one line per model and pair, with the look it closed at, the recall row and Decision 6's counts. [proof: human: the executor records each verdict and where it is published]

## Acceptance

```bash
set -o pipefail
L=docs/bench/stage-6/looks
R=docs/bench/stage-6
S=bench/swe-bench/selection-adr-163.json
n=$(jq '.selected | length' $S); lmax=$(( (n + 39) / 40 ))
blocks=$(ls -d $R/block-*/ | wc -l | tr -d ' ')
[ "$blocks" -ge 1 ] \
  && for k in $(seq 1 "$blocks"); do d=$R/block-$k; t=$(jq '.tasks | length' $d/run.json); go run ./cmd/bench verify --stage stage6 --tasks "$t" --repeats 1 $d || exit 1; for m in claude-sonnet-5-5 claude-opus-5-5; do for p in A-P A-W; do jq -e '.verdict | IN("PAYS","FUTILE","OPEN")' $L/look-$k/$m-$p.json > /dev/null || exit 1; done; done; done \
  && for k in $(seq 2 "$blocks"); do j=$((k - 1)); for m in claude-sonnet-5-5 claude-opus-5-5; do want=A; for p in P W; do jq -e '.verdict == "OPEN"' $L/look-$j/$m-A-$p.json > /dev/null && want="$want,$p"; prev=$(jq -r .verdict $L/look-$j/$m-A-$p.json); [ "$prev" = OPEN ] || [ "$(jq -r .verdict $L/look-$k/$m-A-$p.json)" = "$prev" ] || exit 1; done; got=$(jq -r --arg m $m '(.configs_for[$m] // []) | join(",")' $R/block-$k/run.json); if [ "$want" = A ]; then [ -z "$got" ] || exit 1; else [ "$got" = "$want" ] || exit 1; fi; done; done \
  && open=$(cat $L/look-$blocks/*.json | jq -s '[.[] | select(.verdict == "OPEN")] | length') \
  && { [ "$open" -eq 0 ] || [ "$blocks" -eq "$lmax" ]; } \
  && jq -e -s 'length > 0 and all(.[]; .stage == "stage6" and .profile == "stage6")' $R/block-*/run.json > /dev/null \
  && test "$(jq -r -s '[.[].cli_version] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].plugin | select(. != null) | .commit] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].checks_sha256] | unique | length' $R/block-*/run.json)" = 1 \
  && test "$(jq -r -s '[.[].work_line_sha256] | unique | length' $R/block-*/run.json)" = 1 \
  && grep -q '^## Stage 6' docs/bench/RESULTS.md
```

The fence goes green only when every look has a valid verdict file for each model and pair, every next block ran exactly A plus the arms its previous look left open, a stopped pair kept the same verdict at every later look, and the last look leaves no pair open or every surviving block has run. Each block is verified against its own instance count, so a partial last block passes.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 6 (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts and looks |
| 2 — something selects it | `bench run --profile stage6`, quality-harness's scorer |
| 3 — the caller can discover it | `docs/bench/stage-6/` and `RESULTS.md` |
| 4 — it is used | quality-harness ADR-112 T3 reads the verdicts |

## Mutation Log
(empty until execute)

## Invariants

- No block starts before the previous look is committed.
- One plugin tag, one CLI, one checks table and one work line for the whole stage.

## Risks

- The stage may need the whole surviving pool; the price Zy approves in T4 covers it, and a stop for usage holds at a block boundary.

## Stop Condition

The fence passes when every pair has stopped or the pool is spent. These go back to Zy instead:
- the canary fails;
- a transcript hit, or a command naming `docker`, a pool image or the container;
- the disk floor stops the run;
- the weekly usage limit would pass Zy's hold line.

## Out of Scope

- Pooling with Stage 5 (permanent: boundary: A gains runnable tests, so the rows are not comparable)

## Verification Log
(empty until execute)
