# Task ADR-206-T6: the counted pilot and its report

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-9/pilot/<model>/<arm>/` receipts and `run.json`, `counts.json`, `report.txt`, `RESULTS.md`
**Consumes:** `pilot.md` (T5), the script (T4)
**Data dependency:** needs what T5 needs; 240 attempts, at most $40, several hours of attempt time; the fence re-counts the transcripts, which stay on this Mac
**Proof map:** v1
**Rests-on:** `each run is the registered one`, `every planned receipt exists`, `the counts are the transcripts'`, `the report is the script's`, `the stage stayed under its stop`

## Goal

ADR-206 Decision 8 and the backlog entry. Per model: arm A and arm M over the 48 tasks, arm N over the 24 follow-on and control tasks, each run with its `--max-spend`. Then `counts.json`, `report.txt` and `RESULTS.md`: every count beside its class, every comparison or its INSUFFICIENT, what the pilot cannot show, and whether either gate of the backlog entry for registering a powered stage holds. No claim beyond the report.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-9/pilot/**`, `counts.json`, `report.txt`, `RESULTS.md` | new | the result |
| `docs/adr/BACKLOG.md` | edit | the pilot marked closed, with where its RESULTS are |

## Ordered Steps

1. [S1] Load the costly-runs skill; say the price ($40 at most) before the first run. [proof: human: the price stated to Zy]
2. [S2] The six runs per ADR-206 Decision 8, one at a time; stop and go to Zy if the script's `spend` reaches $40. [proof: human: the receipts, their count and the spend]
3. [S3] `count`, `report`, `RESULTS.md`, the backlog mark. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-9
s=$(sed -n 's/^instructions sha256: //p' $R/pilot.md) && test -n "$s" \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do for a in m n; do jq -e --arg m "$m" --arg s "$s" '.models == [$m] and .configs == ["B"] and .repeats == 1 and .mrw_instructions_sha256 == $s' $R/pilot/$m/$a/run.json > /dev/null || exit 1; done; jq -e --arg m "$m" '.models == [$m] and .configs == ["A"] and .repeats == 1' $R/pilot/$m/a/run.json > /dev/null || exit 1; done \
  && T=$(sed -n 's/^transcripts: //p' $R/RESULTS.md) && test -d "$T" \
  && python3 -B scripts/bench_stage9.py count --receipts $R/pilot --transcripts "$T" > "${TMPDIR:-/tmp}/adr206-counts.json" \
  && diff "${TMPDIR:-/tmp}/adr206-counts.json" $R/counts.json \
  && python3 -B scripts/bench_stage9.py report --receipts $R/pilot --counts $R/counts.json > "${TMPDIR:-/tmp}/adr206-report.txt" \
  && diff "${TMPDIR:-/tmp}/adr206-report.txt" $R/report.txt \
  && python3 -B scripts/bench_stage9.py spend --receipts $R/pilot | awk '{ exit !($NF + 0 <= 40) }'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: each `run.json` is the registered run with the frozen instruction hash; the committed counts and report are what the frozen script prints from the receipts and transcripts; the stage's spend is at most $40 | none | — |

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
