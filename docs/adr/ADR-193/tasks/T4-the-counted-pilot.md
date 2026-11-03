# Task ADR-193-T4: the counted pilot and its verdict

**Depends-on:** T3
**Awaiting-decision:** run the counted pilot before, after, or between the blocks of ADR-192's Stage 7 (ADR-192 T4), which is pending and paid on the same machine — Zy's call
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-8/pilot/` receipts, `counts.json`, `verdict.txt`, `RESULTS.md`
**Consumes:** the frozen paragraph and its sha (T3), `bench_edits.py count` and `verdict` (T2)
**Data dependency:** needs the release asset, Claude Code 2.1.317, both models and the 18 instances ADR-193 Decision 5 lists; about $9 and an hour of attempt time
**Proof map:** v1
**Rests-on:** `the script that judged is unchanged`, `the reference row was counted by the same script`, `every counted attempt has a receipt`, `every counted receipt ran the pinned CLI and mrw`, `every counted run carried the frozen paragraph`, `the verdict reproduces from the counts and receipts`, `the verdict line is SHIP or NO SHIP`, `RESULTS.md quotes the verdict`

## Goal

B and S, paired, on the 18 instances and both models, counted and judged exactly as ADR-193 Decisions 4 and 5
registered, with one verdict line: SHIP or NO SHIP.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-8/pilot/` | add | the receipts and `run.json` per model; transcripts stay on this Mac, as every stage's do |
| `docs/bench/stage-8/counts.json` | add | `bench_edits.py count` over every counted transcript |
| `docs/bench/stage-8/verdict.txt` | add | `bench_edits.py verdict` output |
| `docs/bench/stage-8/RESULTS.md` | add | the measures and verdict per model, the secondaries, the transcripts' path, and a `script sha256: ` line |
| `docs/bench/README.md` | edit | a row for the pilot and its verdict, stating no more than it |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-8/RESULTS.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] Load the costly-runs skill. `bench run --profile stage8 --configs B,S` on the 18 instances, one model at a time, one attempt each; re-run each setup exclusion once, as Decision 5 says. Stop at $15 spent. [proof: human: the receipts, their count and the spend]
3. [S3] `bench_edits.py count` over every counted transcript into `counts.json`; `bench_edits.py verdict` into `verdict.txt`. [proof: acceptance]
4. [S4] Write `RESULTS.md`: every measure beside its threshold, the prediction beside the outcome, what the pilot cannot tell (Decision 5), and no claim beyond the verdict. [proof: human: the record against Decision 5]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-8
test "$(sed -n 's/^script sha256: //p' $R/RESULTS.md)" = "$(shasum -a 256 scripts/bench_edits.py | cut -d' ' -f1)" \
  && test "$(sed -n 's/^script sha256: //p' docs/bench/stage-8/reference.md)" = "$(sed -n 's/^script sha256: //p' $R/RESULTS.md)" \
  && test "$(ls $R/pilot/*.receipt.json | wc -l | tr -d ' ')" -ge 72 \
  && jq -s -e 'all(.[]; .cli_version == "2.1.317 (Claude Code)" and (.mrw_version | startswith("v1.83.0 ")) and (.config == "B" or .config == "S"))' $R/pilot/*.receipt.json > /dev/null \
  && s=$(sed -n 's/^paragraph sha256: //p' docs/bench/stage-8/pilot.md) && test -n "$s" \
  && jq -s -e --arg s "$s" 'length >= 2 and all(.[]; .edit_paragraph_sha256 == $s)' $R/pilot/run-*.json > /dev/null \
  && python3 -B scripts/bench_edits.py verdict --receipts $R/pilot --counts $R/counts.json > /tmp/adr193-verdict.txt \
  && diff /tmp/adr193-verdict.txt $R/verdict.txt \
  && grep -qE '^verdict: (SHIP|NO SHIP)$' $R/verdict.txt \
  && grep -qF -- "$(tail -1 $R/verdict.txt)" $R/RESULTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the script that judged is the one that counted the reference row, unchanged; every receipt ran the pinned CLI and mrw; the verdict reproduces from the committed counts and receipts and is quoted | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts, counts and verdict |
| 2 — something selects it | T5 is blocked on its verdict line |
| 3 — the caller can discover it | `docs/bench/README.md`'s row |
| 4 — it is used | T5 on SHIP; the BACKLOG follow-ups either way |

## Mutation Log

## Invariants

- No instance is added, dropped or reordered for an outcome; the only drop is Decision 5's double setup exclusion.
- The script, thresholds and paragraph are the ones committed before the first counted call.

## Risks

- Shared-machine load skews wall time; wall time is descriptive only.

## Stop Condition

Spend passes $15, a canary-shaped failure (S's init tools, a prompt sha that differs) appears in a counted receipt, or more than 2 instances drop on a model: stop to Zy.

## Out of Scope

- Changing the guide (T5's job, on SHIP only).

## Verification Log
