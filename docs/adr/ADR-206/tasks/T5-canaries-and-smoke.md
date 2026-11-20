# Task ADR-206-T5: the canaries and the uncounted smoke

**Depends-on:** T1, T2, T3, T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench/stage-9/instructions.txt`; `docs/bench/stage-9/pilot.md`; four canaries; `docs/bench/stage-9/smoke/`
**Consumes:** everything T1 to T4 produce, and what ADR-211 T1, T2 and T3 produce (adr-lint does not resolve a dependency into a record in the `ADR-NNN/tasks` layout, so it is named here): the committed `bench/stage-9/instructions.txt`, `plans.json`, `smoke-plans.json`, `docs/bench/stage-9/rehearsal.md`, and `bench_stage9.py gate`, `count` and `project`
**Data dependency:** needs Claude Code, both models and the two builds, and Zy's word to spend (ADR-211 Decision 12); about $5 and half an hour of attempt time (estimates; the canaries' `--max-spend` figures sum to $2 and the smoke's to $4.40 with its probe)
**Proof map:** v1
**Rests-on:** `pilot.md names every pinned thing and its hashes are the files'`, `each smoke run carries its arm's binary and the frozen instructions`, `each canary carries its build's hashes`, `a kept diff and N's kept scratch file are beside their receipts`, `a run stopped on spend after one attempt`, `each step is gated before the next`

## Goal

ADR-206 Decision 9 as ADR-211 Decisions 8, 9 and 10 amend it. `pilot.md` has one `key: value` line each for: `tag`, `claude code`, `bench revision`, `m sha256`, `n sha256`, `instructions sha256`, `classes sha256`, `script sha256`, `helper sha256`, `status line sha256`, `plans sha256`, `smoke plans sha256`, `tasks sha256`, `budget seconds`, `max turns`, `seed`, `stop <model> <arm>` for the six figures, `init model <model>` for each model, `smoke transcripts` (a directory on this Mac) and `rehearsal` (the path of `rehearsal.md`). `init model <model>` is the one model name both of that model's canaries reported in `init_models`; two canaries of a model that disagree stop this task for Zy.

The four canaries run one at a time with the unprofiled invocation these configurations use and the instructions committed for M and N: Opus N, Opus M, Sonnet N and Sonnet M, the Opus ones with `--max-spend 0.75` and the Sonnet ones with `--max-spend 0.25`. Each one's exit status is recorded in `canary-<model>-<arm>.exit` and `bench_stage9.py gate canary` clears it, its line appended to `smoke/gates.txt`, before the next starts.

The smoke, uncounted, is the seven runs of `bench/stage-9/smoke-plans.json`, one at a time in the order of ADR-211 Decision 9 (Opus N, M, A, then Sonnet N, M, A) with the flags the counted runs will carry, N with `--keep-scratch-file stage9-exposed.jsonl`; each run's exit status is recorded in `<directory>.exit` and `gate run` clears it before the next starts. The probe, Sonnet A on the same two tasks with `--max-spend 0.01`, ends after its first attempt with the spend stop's status 3, and its exit and stop line (`<directory>.exit`, `<directory>.stderr`) are kept. Then `count` runs with the smoke plans on the smoke's own output, and `project` exits 0 only when each arm's smoke cost per graded attempt times its planned attempts is under its figure; otherwise the counted pilot waits for Zy. The smoke shows that a kept diff and a receipt arrive and that the runner, the gates and `count` work on a real run. Whether N logs an exposed hunk is shown by ADR-211 T2's stub rehearsal, which counts the stub's event as `exposed`: a real N attempt with no exposed write leaves no kept file, and the fence does not require one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/instructions.txt` | exists | M's instructions, frozen by ADR-211 T2 |
| `docs/bench/stage-9/pilot.md`, `canary-*.json`, `canary-*.exit`, `smoke/**` | new | the pinned stage and its smoke |

## Ordered Steps

1. [S1] The red state: the fence fails with no `pilot.md`. [proof: acceptance]
2. [S2] Zy's word to spend is in hand (ADR-211 Decision 12). Load the costly-runs skill. Build both binaries (T2), run the four canaries one at a time and then the smoke into `~/.local/share/mrw-bench/runs/stage9/smoke`, recording each `bench` exit status in `<step>.exit` and gating each step before the next; copy each canary's `canary.json` as `canary-<model>-<m|n>.json`, and each smoke directory's receipts, `.diff`, `.kept` and `.started` files and `run.json`, the probe's `.stderr` and the gates. [proof: human: the canaries' passes, the gates and the smoke's receipts read by Zy]
3. [S3] `pilot.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=${STAGE9_DIR:-docs/bench/stage-9}
B="python3 -B scripts/bench_stage9.py"
SP=bench/stage-9/smoke-plans.json
v() { sed -n "s/^$1: //p" $R/pilot.md; }
r() { sed -n "s/^$1: //p" $R/rehearsal.md; }
for k in tag 'claude code' 'bench revision' 'm sha256' 'n sha256' 'instructions sha256' 'classes sha256' 'script sha256' 'helper sha256' 'status line sha256' 'plans sha256' 'smoke plans sha256' 'tasks sha256' 'budget seconds' 'smoke transcripts' rehearsal 'init model claude-opus-5-5' 'init model claude-sonnet-5-5'; do test -n "$(v "$k")" || exit 1; done
test "$(v seed)" = 206 && test "$(v 'max turns')" = 60 \
  && test "$(v 'stop claude-opus-5-5 a')" = 14 && test "$(v 'stop claude-opus-5-5 m')" = 14 && test "$(v 'stop claude-opus-5-5 n')" = 7 \
  && test "$(v 'stop claude-sonnet-5-5 a')" = 6 && test "$(v 'stop claude-sonnet-5-5 m')" = 6 && test "$(v 'stop claude-sonnet-5-5 n')" = 3 \
  && test "$(v 'instructions sha256')" = "$(shasum -a 256 bench/stage-9/instructions.txt | cut -d' ' -f1)" \
  && test "$(v 'classes sha256')" = "$(shasum -a 256 bench/stage-9/classes.json | cut -d' ' -f1)" \
  && test "$(v 'plans sha256')" = "$(shasum -a 256 bench/stage-9/plans.json | cut -d' ' -f1)" \
  && test "$(v 'smoke plans sha256')" = "$(shasum -a 256 $SP | cut -d' ' -f1)" \
  && test "$(v 'script sha256')" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" \
  && test "$(v 'helper sha256')" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)" \
  && test "$(v 'm sha256')" = "$(sed -n 's/^m sha256: //p' bench/stage-9/builds.md)" && test "$(v 'n sha256')" = "$(sed -n 's/^n sha256: //p' bench/stage-9/builds.md)" \
  && test "$(basename "$(v rehearsal)")" = rehearsal.md && test "$(r revision)" = "$(v 'bench revision')" \
  && for k in 'plans sha256' 'smoke plans sha256' 'tasks sha256' 'instructions sha256'; do test "$(r "$k")" = "$(v "$k")" || exit 1; done \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do
       test "$(r date)" = "$(jq -r '.started[:10]' $R/canary-$m-m.json)" && test "$(r date)" = "$(jq -r '.started[:10]' $R/canary-$m-n.json)" || exit 1
       for a in m n; do
         jq -e --arg m "$m" --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" --arg c "$(v 'claude code')" --arg i "$(v "init model $m")" '.pass == true and .model == $m and .mrw_sha256 == $h and .mrw_instructions_sha256 == $s and .cli_version == $c and (.init_models | length) > 0 and ([.init_models[]] | all(. == $i))' $R/canary-$m-$a.json > /dev/null || exit 1
         jq -e --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" --arg c "$(v 'claude code')" --arg sl "$(v 'status line sha256')" '.mrw_sha256 == $h and .mrw_instructions_sha256 == $s and .cli_version == $c and .status_line_sha256 == $sl and .keep_diff == true and (.tasks | length) == 2' $R/smoke/$m/$a/run.json > /dev/null || exit 1
       done
       jq -e '.keep_scratch_file == "stage9-exposed.jsonl" and (.keep_state_file // "") == ""' $R/smoke/$m/n/run.json > /dev/null || exit 1
       jq -e --arg c "$(v 'claude code')" '.configs == ["A"] and .cli_version == $c and (.tasks | length) == 2' $R/smoke/$m/a/run.json > /dev/null || exit 1
       for a in a m n; do d=0; for r in $R/smoke/$m/$a/*.receipt.json; do h=$(jq -r '.kept_diff_sha256 // empty' "$r"); [ -n "$h" ] && test "$h" = "$(shasum -a 256 "${r%.receipt.json}.diff" | cut -d' ' -f1)" || exit 1; d=$((d+1)); done; [ $d -ge 2 ] || exit 1; done
     done \
  && prev="" \
  && for s in canary:claude-opus-5-5:n canary:claude-opus-5-5:m canary:claude-sonnet-5-5:n canary:claude-sonnet-5-5:m run:claude-opus-5-5:n run:claude-opus-5-5:m run:claude-opus-5-5:a run:claude-sonnet-5-5:n run:claude-sonnet-5-5:m run:claude-sonnet-5-5:a; do
       IFS=: read -r k m a <<<"$s"
       if [ "$k" = canary ]; then t=$R/canary-$m-$a.json; l=canary-$a; else t=$R/smoke/$m/$a; l=$a; fi
       out=$($B gate $k $m $a --target "$t" --plans $SP --pilot $R/pilot.md --gates $R/smoke/gates.txt ${prev:+--after "$prev"}) || exit 1
       grep -qxF "$out" $R/smoke/gates.txt || exit 1
       prev="$m $l"
     done \
  && p=$R/smoke/claude-sonnet-5-5/a-probe && [ "$(cat $p.exit)" = 3 ] && grep -q 'reached --max-spend' $p.stderr \
  && jq -e '.max_spend_usd == 0.01 and (.tasks | length) == 2' $p/run.json > /dev/null \
  && [ "$(ls $p/*.started | wc -l | tr -d ' ')" -eq 1 ] \
  && s=$(ls $p/*.started) && i=$(basename "${s%.started}") && jq -e --arg i "$i" '.attempt == $i and (.total_cost_usd | type) == "number" and .total_cost_usd >= 0.01' "${s%.started}.receipt.json" > /dev/null \
  && T=$(v 'smoke transcripts') && test -d "$T" \
  && $B count --receipts $R/smoke --transcripts "$T" --pilot $R/pilot.md --classes bench/stage-9/classes.json --plans $SP > /dev/null \
  && $B project --receipts $R/smoke --plans $SP
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: `pilot.md` names every pinned thing, its figures are the registered ones and its hashes are the files'; `rehearsal.md` is the canaries' day and `pilot.md`'s revision and inputs; each canary and each smoke `run.json` of M and N carries its arm's binary hash, the frozen instruction hash and the pinned Claude Code, and each canary reported the model `pilot.md` observed; every smoke run kept its diffs, and N's `run.json` names `keep_scratch_file`; every step of the ten is gated in order against the times recorded in `smoke/gates.txt`; the probe exited 3 with the stop's reason, planned two attempts, started one, and that one cost more than its figure; `count` runs on the smoke and no arm's projection reaches its figure. `ADR-211 T3`'s `test_the_paid_fences_pass_a_fixture_and_fail_each_broken_one` runs this fence over a generated fixture | none | — |

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
- No step starts before its predecessor's gate has cleared.

## Risks

- The unprofiled invocation may not work with the current Claude Code: the canary is where that shows, before anything is pinned.

## Stop Condition

The fence exits 0 and Zy has read `pilot.md`.

## Out of Scope

- The counted attempts (permanent: boundary: ADR-206 T6)
