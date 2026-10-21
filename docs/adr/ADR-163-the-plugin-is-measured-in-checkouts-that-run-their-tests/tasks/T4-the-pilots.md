# Task ADR-163-T4: the baseline pilot, the whole-pool baseline, then the agent pilot

**Depends-on:** T3, T6
**Covers:** none — no spec
**Estimated scope:** M (two pilots, one baseline over the pool, a sign-off)
**Owner:** Zy
**Produces:** `docs/bench/stage-6-pilot/` (`pilot-instances.json`, `baseline.json`, `d1.txt`; if D1 passes, the agent pilot's receipts, `run.json` and one canary per model); `docs/bench/stage-6/baseline.json` and `bench/swe-bench/selection-adr-163.json` (the surviving pool); D3's caps; the price
**Consumes:** the shims and `--profile stage6` (T1), W, `checks.json` and `--max-turns` (T2), `bench baseline` and the surviving pool (T3), configurations G and C, the core export projection and `bench side` (T6)
**Data dependency:** real SWE-bench images on this Mac, after Stage 5 closes (ADR-162 T5's `bench look` exits 0, or Zy ends it); the agent pilot also uses the subscription and needs quality-harness's pre-registration committed first; each step on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot instances follow the rule`, `D1 was read before any usage`, `the surviving pool was frozen before the pre-registration`, `the caps come from W's pilot`, `the container was clean in every pilot attempt`

## Goal

Before any counted attempt: settle ADR-112 D1 on a baseline pilot that uses no model; if it passes, freeze the surviving pool with a baseline over the whole pool; then, after quality-harness pre-registers, run a small agent pilot that sets D3's caps; then bring Zy the price for the whole surviving pool (ADR-163 Decision 7).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-6-pilot/` | add | `pilot-instances.json` (derived by rule, committed before the pilot), `baseline.json`, `d1.txt` (the rule's inputs and answer); the agent pilot's receipts, `run.json` and `canary*.json`, stage `pilot-6`, never counted |
| `docs/bench/stage-6/baseline.json` | add | the whole pool's baselines |
| `bench/swe-bench/selection-adr-163.json` | add | the surviving pool |

## Ordered Steps

1. [S1] See the fence red: no pilot. [proof: acceptance]
2. [S2] After Stage 5 closes, and on Zy's word: derive `pilot-instances.json` by ADR-163 Decision 7's rule and commit it; then `bench baseline` over it. Write `d1.txt`: the green count, the median wall time, 1200 s, and ADR-112 D1's answer (at least 8 of 10 green and a median of at most 300 s), its last line `D1: passes` or `D1: fails`. If D1 fails, stop here and send `d1.txt` to quality-harness and Zy. [proof: human: the executor records the green count, the median and D1's answer]
3. [S3] If D1 passes, and on Zy's word with its machine time: `bench baseline` over all 191 pool instances, `--surviving bench/swe-bench/selection-adr-163.json`. Send quality-harness the surviving pool's digest by the scorer's convention (its `.selected` ids joined by a newline, no trailing newline, then sha256), its count, and the red count. [proof: human: the executor records the surviving count, the red count and the digest as sent]
4. [S4] Only after quality-harness's pre-registration is committed with D1's 1200 s reading, D3's derivation rule and the surviving pool's digest (its commit recorded here), and on Zy's word with its price (24 attempts): the canary under `stage6` with A, P and W on each model; then the first 4 green pilot instances × A/P/W × both models at 1200 s and 60 turns, stage `pilot-6`. Read one P and one W transcript whole. Record each arm's median wall time, turns and cost; D3's caps from W's medians over both models' attempts together, noting any censored at the caps, sent to quality-harness to register before block 1; ADR-163 Decision 6's counts; `sync_failures`, `write_backs`, `write_back_conflicts` and `shim_wait_ms` per arm. If the host-interpreter count is not zero, decide with Zy how the host interpreters are hidden before block 1. [proof: human: the executor records the quality-harness commit, the canary, the two transcripts, the medians per arm, the caps as sent, the counts and any decision on host interpreters]
5. [S5] The price for the whole surviving pool, from the pilots, to Zy. [proof: human: Zy's word on the price, or the reason the stage waits]

## Acceptance

```bash
set -o pipefail
B=docs/bench/stage-6-pilot
jq -e --slurpfile want $B/pilot-instances.json '[.[].instance] == $want[0] and all(.[]; (.green | type) == "boolean" and (.wall_ms | type) == "number")' $B/baseline.json > /dev/null \
  && jq -e 'length == 10' $B/pilot-instances.json > /dev/null \
  && if tail -1 $B/d1.txt | grep -qx 'D1: passes'; then
       jq -e '(map(select(.green)) | length) >= 8 and ((map(.wall_ms) | sort | .[4] + .[5]) / 2) <= 300000' $B/baseline.json > /dev/null \
       && jq -e '(.selected | length) > 0' bench/swe-bench/selection-adr-163.json > /dev/null \
       && jq -e 'length == 191' docs/bench/stage-6/baseline.json > /dev/null \
       && go run ./cmd/bench verify --stage pilot-6 --tasks 4 --repeats 1 $B \
       && jq -e '(.configs | sort) == ["A","P","W"] and ((.side_admitted // []) - ["G","C"] | length) == 0 and .profile == "stage6" and (.work_line_sha256 | test("^[0-9a-f]{64}$")) and (.checks_sha256 | test("^[0-9a-f]{64}$")) and .max_turns == 60 and .budget_seconds == 1200' $B/run.json > /dev/null \
       && jq -e -s 'length == 2 and all(.[]; .pass == true)' $B/canary*.json > /dev/null;
     else tail -1 $B/d1.txt | grep -qx 'D1: fails' \
       && jq -e '(map(select(.green)) | length) < 8 or ((map(.wall_ms) | sort | .[4] + .[5]) / 2) > 300000' $B/baseline.json > /dev/null; fi
```

The fence passes on either outcome of D1, because each is a result: a baseline pilot that fails D1 ends the stage before any usage, as ADR-112 decided. Either way it checks that `d1.txt`'s answer is the one the baseline gives.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive the pilot (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the baselines, the surviving pool and the pilot receipts |
| 2 — something selects it | `bench baseline`, `bench run --profile stage6 --configs A,P,W` |
| 3 — the caller can discover it | `docs/bench/stage-6-pilot/`, `docs/bench/stage-6/baseline.json` |
| 4 — it is used | T5 starts on its sign-off |

## Mutation Log
(empty until execute)

## Invariants

- Nothing here runs while Stage 5 runs.
- The baseline pilot and the whole-pool baseline invoke no model; the agent pilot runs only if D1 passes and after quality-harness's pre-registration.
- Pilot instances are outside ADR-162's pool.

## Risks

- Whole suites under emulation are too slow; then D1 fails, which is the cheapest place to learn it.
- The whole-pool baseline is 191 image pulls; the 25 GiB disk floor applies to each.

## Stop Condition

The fence exits 0 and S5 is recorded. D1 failing, a dirty container, or a canary failure goes back to Zy and quality-harness.

## Out of Scope

- Choosing the pilot instances by outcome (permanent: boundary: file order and the pool's repository mix decide them)

## Verification Log
(empty until execute)
