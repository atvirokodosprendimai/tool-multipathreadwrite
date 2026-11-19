# Task ADR-206-T5: the canaries and the uncounted smoke

**Depends-on:** T1, T2, T3, T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench/stage-9/instructions.txt`; `docs/bench/stage-9/pilot.md`; four canaries; `docs/bench/stage-9/smoke/`
**Consumes:** everything T1 to T4 produce
**Data dependency:** needs Claude Code, both models and the two builds; about $5 and half an hour of attempt time (estimates; the smoke's runs carry `--max-spend` figures summing to $4)
**Proof map:** v1
**Rests-on:** `pilot.md names every pinned thing and its hashes are the files'`, `each smoke run carries its arm's binary and the frozen instructions`, `each canary carries its build's hashes`, `a kept file and the kept diffs are beside their receipts`, `a run stopped on spend after one attempt`

## Goal

ADR-206 Decision 9. `pilot.md` has one `key: value` line each for: `tag`, `claude code`, `bench revision`, `m sha256`, `n sha256`, `instructions sha256`, `classes sha256`, `script sha256`, `helper sha256`, `status line sha256`, `budget seconds`, `max turns`, `seed`, and `stop <model> <arm>` for the six figures. A canary per model for each of the two builds, run with the unprofiled invocation these configurations use, each recording its hashes. A smoke, uncounted, per model: two tasks in each of A, M and N, with the flags the counted runs will carry; and one run of two planned attempts with `--max-spend 0.01`, which ends after its first with the spend stop's status. Whether N logs an exposed hunk is T2's test, on these same two binaries; the smoke shows only that a kept file and a kept diff arrive beside a receipt.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/instructions.txt` | new | M's instructions, frozen |
| `docs/bench/stage-9/pilot.md`, `canary-*.json`, `smoke/**` | new | the pinned stage and its smoke |

## Ordered Steps

1. [S1] The red state: the fence fails with no `pilot.md`. [proof: acceptance]
2. [S2] Load the costly-runs skill. Build both binaries (T2), freeze the instructions, run the four canaries and then the smoke into `~/.local/share/mrw-bench/runs/stage9/smoke`; copy receipts, kept logs and `run.json`. [proof: human: the canaries' passes and the smoke's receipts read by Zy]
3. [S3] `pilot.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-9
v() { sed -n "s/^$1: //p" $R/pilot.md; }
for k in tag 'claude code' 'bench revision' 'm sha256' 'n sha256' 'instructions sha256' 'classes sha256' 'script sha256' 'helper sha256' 'status line sha256' 'budget seconds'; do test -n "$(v "$k")" || exit 1; done
test "$(v seed)" = 206 && test "$(v 'max turns')" = 60 \
  && test "$(v 'stop claude-opus-5-5 a')" = 14 && test "$(v 'stop claude-opus-5-5 m')" = 14 && test "$(v 'stop claude-opus-5-5 n')" = 7 \
  && test "$(v 'stop claude-sonnet-5-5 a')" = 6 && test "$(v 'stop claude-sonnet-5-5 m')" = 6 && test "$(v 'stop claude-sonnet-5-5 n')" = 3 \
  && test "$(v 'instructions sha256')" = "$(shasum -a 256 bench/stage-9/instructions.txt | cut -d' ' -f1)" \
  && test "$(v 'classes sha256')" = "$(shasum -a 256 bench/stage-9/classes.json | cut -d' ' -f1)" \
  && test "$(v 'script sha256')" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" \
  && test "$(v 'helper sha256')" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)" \
  && test "$(v 'm sha256')" = "$(sed -n 's/^m sha256: //p' bench/stage-9/builds.md)" && test "$(v 'n sha256')" = "$(sed -n 's/^n sha256: //p' bench/stage-9/builds.md)" \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do
       for a in m n; do
         jq -e --arg m "$m" --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" --arg c "$(v 'claude code')" '.pass == true and .model == $m and .mrw_sha256 == $h and .mrw_instructions_sha256 == $s and .cli_version == $c' $R/canary-$m-$a.json > /dev/null || exit 1
         jq -e --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" --arg c "$(v 'claude code')" --arg sl "$(v 'status line sha256')" '.mrw_sha256 == $h and .mrw_instructions_sha256 == $s and .cli_version == $c and .status_line_sha256 == $sl and .keep_diff == true and (.tasks | length) == 2' $R/smoke/$m/$a/run.json > /dev/null || exit 1
       done
       jq -e --arg c "$(v 'claude code')" '.configs == ["A"] and .cli_version == $c and (.tasks | length) == 2' $R/smoke/$m/a/run.json > /dev/null || exit 1
       for a in a m n; do [ "$(ls $R/smoke/$m/$a/*.diff | wc -l | tr -d ' ')" -ge 2 ] || exit 1; done
       k=0; for r in $R/smoke/$m/n/*.json; do h=$(jq -r '.kept_state_file_sha256 // empty' "$r"); [ -n "$h" ] || continue; test "$h" = "$(shasum -a 256 "${r%.json}.kept" | cut -d' ' -f1)" || exit 1; k=$((k+1)); done; [ $k -ge 1 ] || exit 1
     done \
  && jq -e '.max_spend_usd == 0.01 and (.tasks | length) == 2' $R/smoke/stop/run.json > /dev/null \
  && [ "$(ls $R/smoke/stop/*.started | wc -l | tr -d ' ')" -eq 1 ] \
  && [ "$(cat $R/smoke/stop/*.json | jq -s '[.[] | select(.attempt != null and .total_cost_usd > 0.01)] | length')" -eq 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: `pilot.md` names every pinned thing, its figures are the registered ones and its hashes are the files'; each canary and each smoke `run.json` of M and N carries its arm's binary hash, the frozen instruction hash and the pinned Claude Code; every smoke run kept its diffs; an N receipt's kept-file hash is its kept file's; the stopped run planned two attempts, started one, and that one cost more than its figure | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the smoke ran |
| 2 — something selects it | T6 |
| 3 — the caller can discover it | `docs/bench/stage-9/pilot.md` |
| 4 — it is used | T6 |

## Mutation Log

## Verification Log

## Invariants

- Nothing in the smoke is counted.

## Risks

- The unprofiled invocation may not work with the current Claude Code: the canary is where that shows, before anything is pinned.

## Stop Condition

The fence exits 0 and Zy has read `pilot.md`.

## Out of Scope

- The counted attempts (permanent: boundary: ADR-206 T6)
