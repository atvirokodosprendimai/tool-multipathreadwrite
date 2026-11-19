# Task ADR-206-T6: the counted pilot and its report

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-9/pilot/<model>/<arm>/` receipts, kept logs, `.started` files and `run.json`; `counts.json`, `report.txt`, `RESULTS.md`
**Consumes:** `pilot.md` (T5), the script (T4)
**Data dependency:** needs what T5 needs; 240 planned attempts, at most $50 of known spend, several hours of attempt time; the fence re-counts the transcripts, which stay on this Mac at the path `RESULTS.md` names
**Proof map:** v1
**Rests-on:** `the scripts are the ones pinned`, `each run is the registered one with its arm's binary`, `every planned receipt and owed replacement exists`, `the counts are the transcripts'`, `the report is the script's`, `the stage stayed under its stop`

## Goal

ADR-206 Decision 8 and the backlog entry. The six runs in the registered order, then `counts.json`, `report.txt` and `RESULTS.md`: every count beside its class and arm with started, graded and missing; every comparison or its INSUFFICIENT; the N attempts that were exposed and ended `outside`, each with its diff quoted; what the pilot cannot show; and whether either gate of the backlog entry for registering a powered stage holds. No claim beyond the report.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-9/pilot/**`, `counts.json`, `report.txt`, `RESULTS.md` | new | the result |
| `docs/bench/README.md` | edit | Stage 9 and where its results are |

## Ordered Steps

1. [S1] The red state: the fence fails with no receipts. [proof: acceptance]
2. [S2] Load the costly-runs skill; say the price ($50 of known spend at most) before the first run. [proof: human: the price stated to Zy]
3. [S3] With `O=~/.local/share/mrw-bench/runs/stage9/pilot` and `$M`, `$N` the two builds, one at a time and in this order: `bench run --stage stage9 --out $O/claude-opus-5-5/a --models claude-opus-5-5 --configs A --repeats 1 --seed 206 --status-line  --max-spend 14 --task <the 48 ids>`; `bench run --stage stage9 --out $O/claude-opus-5-5/m --models claude-opus-5-5 --configs B --repeats 1 --seed 206 --status-line --mrw $M --mrw-sha256 <m> --mrw-instructions bench/stage-9/instructions.txt --max-spend 14 --task <the 48 ids>`; `bench run --stage stage9 --out $O/claude-opus-5-5/n --models claude-opus-5-5 --configs B --repeats 1 --seed 206 --status-line --mrw $N --mrw-sha256 <n> --mrw-instructions bench/stage-9/instructions.txt --keep-state-file mrw/stage9-exposed.jsonl --max-spend 7 --task <the 24 ids>`; `bench run --stage stage9 --out $O/claude-sonnet-5-5/n --models claude-sonnet-5-5 --configs B --repeats 1 --seed 206 --status-line --mrw $N --mrw-sha256 <n> --mrw-instructions bench/stage-9/instructions.txt --keep-state-file mrw/stage9-exposed.jsonl --max-spend 3 --task <the 24 ids>`; `bench run --stage stage9 --out $O/claude-sonnet-5-5/m --models claude-sonnet-5-5 --configs B --repeats 1 --seed 206 --status-line --mrw $M --mrw-sha256 <m> --mrw-instructions bench/stage-9/instructions.txt --max-spend 6 --task <the 48 ids>`; `bench run --stage stage9 --out $O/claude-sonnet-5-5/a --models claude-sonnet-5-5 --configs A --repeats 1 --seed 206 --status-line  --max-spend 6 --task <the 48 ids>`. A run that stops on spend stops the stage and goes to Zy. `bench verify` on each. [proof: human: the receipts, their count and the spend]
4. [S4] `count`, `report`, `RESULTS.md`, the README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-9
v() { sed -n "s/^$1: //p" $R/pilot.md; }
test "$(v 'script sha256')" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" && test "$(v 'helper sha256')" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)" \
  && test "$(v 'classes sha256')" = "$(shasum -a 256 bench/stage-9/classes.json | cut -d' ' -f1)" \
  && all=$(jq -c '[.[] | .[]] | sort' bench/stage-9/classes.json) && two=$(jq -c '[."follow-on", .control] | add | sort' bench/stage-9/classes.json) \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do for a in a m n; do
       c=B; t=$all; [ $a = a ] && c=A; [ $a = n ] && t=$two
       jq -e --arg m "$m" --arg c "$c" --argjson t "$t" --arg cli "$(v 'claude code')" --arg b "$(v 'bench revision')" '.models == [$m] and .configs == [$c] and .repeats == 1 and .seed == 206 and (.tasks | sort) == $t and .cli_version == $cli and .bench_revision == $b and (.status_line_sha256 | length) == 64 and .max_spend_usd > 0' $R/pilot/$m/$a/run.json > /dev/null || exit 1
       [ $a = a ] || jq -e --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" '.mrw_sha256 == $h and .mrw_instructions_sha256 == $s' $R/pilot/$m/$a/run.json > /dev/null || exit 1
       go run ./cmd/bench verify --out $R/pilot/$m/$a > /dev/null || exit 1
     done; done \
  && [ "$(cat $R/pilot/*/*/run.json | jq -s '[.[].max_spend_usd] | add')" = 50 ] \
  && T=$(sed -n 's/^transcripts: //p' $R/RESULTS.md) && test -d "$T" \
  && python3 -B scripts/bench_stage9.py count --receipts $R/pilot --transcripts "$T" > "${TMPDIR:-/tmp}/adr206-counts.json" \
  && diff "${TMPDIR:-/tmp}/adr206-counts.json" $R/counts.json \
  && python3 -B scripts/bench_stage9.py report --receipts $R/pilot --counts $R/counts.json > "${TMPDIR:-/tmp}/adr206-report.txt" \
  && diff "${TMPDIR:-/tmp}/adr206-report.txt" $R/report.txt \
  && python3 -B scripts/bench_stage9.py spend --receipts $R/pilot | awk 'END { exit !(NR == 1 && $1 == "stage9" && $4 ~ /^[0-9]+(\.[0-9]+)?$/ && $4 + 0 <= 50 && $6 ~ /^[0-9]+$/) }'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the script, helper and class list are the pinned ones; each of the six `run.json` is the registered run (model, configuration, task list, seed, Claude Code, bench revision, status line, its arm's binary and the frozen instructions) and passes `bench verify`; the six stops sum to 50; the committed counts and report are what the script prints; the stage's known spend is one number at most 50 | none | — |

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

- No rule of the backlog entry or of ADR-206 changes after the first counted attempt.
- Transcripts stay out of the tree.

## Risks

- A run interrupted mid-attempt leaves a `.started` with no receipt: it is reported as started and missing, never re-run silently.

## Stop Condition

The fence exits 0, and Zy has read `RESULTS.md`.

## Out of Scope

- A powered stage (deferred: docs/adr/BACKLOG.md)
