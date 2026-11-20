# Task ADR-206-T6: the counted pilot and its report

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-9/pilot/<model>/<arm>/` receipts, kept logs, `.started` files, `<arm>.exit` and `run.json`; `pilot/gates.txt`, `counts.json`, `report.txt`, `RESULTS.md`
**Consumes:** `pilot.md` (T5), the script (T4 and ADR-211 T1, T3), `bench/stage-9/plans.json`
**Data dependency:** needs what T5 needs and Zy's word to spend (ADR-211 Decision 12); 240 planned attempts, six stops that sum to $50 of known spend (each can be passed by one attempt), several hours of attempt time; the fence re-counts the transcripts, which stay on this Mac at the path `RESULTS.md` names
**Proof map:** v1
**Rests-on:** `the scripts are the ones pinned`, `each run is the registered one with its arm's binary`, `every planned receipt and owed replacement exists`, `the counts are the transcripts'`, `the report is the script's`, `the stage stayed under its stop`, `each step is gated before the next`, `every exposed attempt has its review row`

## Goal

ADR-206 Decision 8 as ADR-211 Decisions 1, 2, 5, 6 and 9 amend it, and the Stage 9 backlog entry. The six runs in the order of ADR-211 Decision 9, Opus N, M, A, then Sonnet N, M, A, each one's `bench` exit status recorded in `<arm>.exit` beside its directory and `bench_stage9.py gate run` run on it before the next starts, its clearance line appended to `pilot/gates.txt`; the per-arm figure is what `gate` judges, not a total of $53. Then `counts.json`, `report.txt` and `RESULTS.md`: every count beside its class and arm with started, graded and missing; the cost statistic under its name, with each arm's spend reconciled; every comparison or its INSUFFICIENT; one review row for each exposed attempt of M and N, whatever its outcome, each quoting its kept diff and carrying one adjudication (`bench_stage9.py review` checks them); what the pilot cannot show; and whether the gate of the backlog entry holds: M against A, single-site or batch, the interval excluding 1 on the same side of 1 on both models. No claim beyond the report.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-9/pilot/**`, `counts.json`, `report.txt`, `RESULTS.md` | new | the result |
| `docs/bench/README.md` | edit | Stage 9 and where its results are |

## Ordered Steps

1. [S1] The red state: the fence fails with no receipts. [proof: acceptance]
2. [S2] Zy's word to spend is in hand (ADR-211 Decision 12). Load the costly-runs skill; say the price (six stops summing to $50 of known spend) before the first run. [proof: human: the price stated to Zy]
3. [S3] With `O=~/.local/share/mrw-bench/runs/stage9/pilot`, `$M` and `$N` the two builds, and the `--claude`, `--config-dir`, `--containment` and `--canary` values T5's smoke used: six runs, one at a time, in the order Opus N, M, A, then Sonnet N, M, A. Each is `bench run --stage stage9 --out $O/<model>/<arm> --model <model> --configs <A or B> --repeats 1 --seed 206 --status-line --keep-diff --max-turns 60 --max-spend <its figure>` with one `--task ID` per id of its list in `bench/stage-9/plans.json` (all 48 for A and M, the 24 follow-on and control ids for N); M and N add `--mrw <build> --mrw-sha256 <its hash> --mrw-instructions bench/stage-9/instructions.txt`, and N adds `--keep-scratch-file stage9-exposed.jsonl`. After each, record its exit status in `$O/<model>/<arm>.exit` and run `bench_stage9.py gate run <model> <arm> --target $O/<model>/<arm> --plans bench/stage-9/plans.json --pilot docs/bench/stage-9/pilot.md --gates docs/bench/stage-9/pilot/gates.txt --sequence pilot [--after "<model> <predecessor>"]`, appending its output to `pilot/gates.txt`. A run whose gate refuses, that stops on spend, or leaves an attempt started with no receipt stops the stage and goes to Zy. Then `bench verify --stage stage9 --tasks <48 or 24> --repeats 1 <dir>` on each. [proof: human: the receipts, their count and the spend]
4. [S4] `count`, `report`, `RESULTS.md` with its review rows, the README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=${STAGE9_DIR:-docs/bench/stage-9}
B="python3 -B scripts/bench_stage9.py"
v() { sed -n "s/^$1: //p" $R/pilot.md; }
test "$(v 'script sha256')" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" && test "$(v 'helper sha256')" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)" \
  && test "$(v 'classes sha256')" = "$(shasum -a 256 bench/stage-9/classes.json | cut -d' ' -f1)" \
  && test "$(v 'plans sha256')" = "$(shasum -a 256 bench/stage-9/plans.json | cut -d' ' -f1)" \
  && all=$(jq -c '[.[] | .[]] | sort' bench/stage-9/classes.json) && two=$(jq -c '[."follow-on", .control] | add | sort' bench/stage-9/classes.json) \
  && prev="" \
  && for s in claude-opus-5-5:n claude-opus-5-5:m claude-opus-5-5:a claude-sonnet-5-5:n claude-sonnet-5-5:m claude-sonnet-5-5:a; do
       IFS=: read -r m a <<<"$s"
       c=B; t=$all; [ $a = a ] && c=A; [ $a = n ] && t=$two
       jq -e --arg m "$m" --arg c "$c" --argjson t "$t" --arg cli "$(v 'claude code')" --arg b "$(v 'bench revision')" --arg sl "$(v 'status line sha256')" --argjson bs "$(v 'budget seconds')" --argjson stop "$(v "stop $m $a")" '.models == [$m] and .configs == [$c] and .repeats == 1 and .seed == 206 and (.tasks | sort) == $t and .cli_version == $cli and .bench_revision == $b and .status_line_sha256 == $sl and .budget_seconds == $bs and .max_turns == 60 and .keep_diff == true and .max_spend_usd == $stop' $R/pilot/$m/$a/run.json > /dev/null || exit 1
       [ $a = a ] || jq -e --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" '.mrw_sha256 == $h and .mrw_instructions_sha256 == $s' $R/pilot/$m/$a/run.json > /dev/null || exit 1
       [ $a != n ] || jq -e '.keep_scratch_file == "stage9-exposed.jsonl" and (.keep_state_file // "") == ""' $R/pilot/$m/$a/run.json > /dev/null || exit 1
       out=$($B gate run $m $a --target $R/pilot/$m/$a --plans bench/stage-9/plans.json --pilot $R/pilot.md --gates $R/pilot/gates.txt --sequence pilot ${prev:+--after "$prev"}) || exit 1
       grep -qxF "$out" $R/pilot/gates.txt || exit 1
       prev="$m $a"
       n=48; [ $a = n ] && n=24
       go run ./cmd/bench verify --stage stage9 --tasks $n --repeats 1 $R/pilot/$m/$a > /dev/null || exit 1
     done \
  && T=$(sed -n 's/^transcripts: //p' $R/RESULTS.md) && test -d "$T" \
  && $B count --receipts $R/pilot --transcripts "$T" --pilot $R/pilot.md --classes bench/stage-9/classes.json > "${TMPDIR:-/tmp}/adr206-counts.json" \
  && diff "${TMPDIR:-/tmp}/adr206-counts.json" $R/counts.json \
  && $B report --receipts $R/pilot --counts $R/counts.json > "${TMPDIR:-/tmp}/adr206-report.txt" \
  && diff "${TMPDIR:-/tmp}/adr206-report.txt" $R/report.txt \
  && $B review --results $R/RESULTS.md --counts $R/counts.json --receipts $R/pilot \
  && $B spend --receipts $R/pilot | awk 'END { exit !(NR == 1 && NF == 6 && $1 == "stage9" && $2 == "spend" && $3 == "usd" && $4 ~ /^[0-9]+(\.[0-9]+)?$/ && $4 + 0 <= 53 && $5 == "missing" && $6 ~ /^[0-9]+$/) }'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the script, helper, class list and plans are the pinned ones; each of the six `run.json` is the registered run (model, configuration, task list, seed, Claude Code, bench revision, status line, its arm's binary and the frozen instructions, N's `keep_scratch_file`) and each is gated in order against the clearances recorded in `pilot/gates.txt`, and passes `bench verify`; each run's stop is its registered figure; the committed counts and report are what the script prints; every exposed attempt of M and N has its review row; the stage's known spend is one number at most 53. `ADR-211 T3`'s `test_the_paid_fences_pass_a_fixture_and_fail_each_broken_one` runs this fence over a generated fixture | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | the backlog entry's gate |
| 3 — the caller can discover it | `docs/bench/stage-9/RESULTS.md`, linked from `docs/bench/README.md` |
| 4 — it is used | a powered stage's pre-registration, or the note that none is registered |

## Mutation Log

## Verification Log

## Invariants

- No rule of the backlog entry or of ADR-206 and ADR-211 changes after the first counted attempt.
- Transcripts stay out of the tree.
- No step starts before its predecessor's gate has cleared.

## Risks

- A run interrupted mid-attempt leaves a `.started` with no receipt and fails `gate` and `bench verify`: the stage stops and goes to Zy; the attempt is never re-run silently.

## Stop Condition

The fence exits 0, and Zy has read `RESULTS.md`.

## Out of Scope

- A powered stage (deferred: docs/adr/BACKLOG.md)
- The hazard question (deferred: docs/adr/BACKLOG.md)
