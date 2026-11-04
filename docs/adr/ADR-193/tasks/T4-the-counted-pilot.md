# Task ADR-193-T4: the counted pilot and its verdict

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-8/pilot/<model>/` receipts and `run.json`, `counts.json`, `verdict.txt`, `RESULTS.md`
**Consumes:** the frozen paragraph and its sha (T3), `bench_edits.py count` and `verdict` with both scripts' hashes (T2)
**Data dependency:** needs the release asset, Claude Code 2.1.317, both models and the 18 instances ADR-193 Decision 5 lists; about $9 and an hour of attempt time; the fence re-counts the transcripts, which stay on this Mac at the path `RESULTS.md` names
**Proof map:** v1
**Rests-on:** `both scripts are the ones that counted the reference row`, `the run carried the frozen paragraph`, `each model ran the registered matrix`, `every planned receipt and owed replacement exists`, `the counts are the transcripts' and name the final receipts`, `the verdict reproduces from the counts and receipts`, `the verdict line is SHIP or NO SHIP`, `RESULTS.md quotes the verdict`

## Goal

B and S, paired, on the 18 instances and both models, counted and judged exactly as ADR-193 Decisions 4 and 5
registered, with one verdict line: SHIP or NO SHIP.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-8/pilot/<model>/` | add | per model: `run.json`, the receipts and the `.started` files, copied unrenamed from `~/.local/share/mrw-bench/runs/stage8/pilot/<model>/`, where bench ran; the transcripts stay there, outside the tree, as every stage's do, and `counts.json` carries their digests |
| `docs/bench/stage-8/counts.json` | add | `bench_edits.py count` over every final receipt's transcript, with its sha256 |
| `docs/bench/stage-8/verdict.txt` | add | `bench_edits.py verdict` output |
| `docs/bench/stage-8/RESULTS.md` | add | the measures and verdict per model, the descriptive rows, the prediction beside the outcome, a `transcripts: ` line, and `script sha256: ` and `helper sha256: ` lines |
| `docs/bench/README.md` | edit | a row for the pilot and its verdict, stating no more than it |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-8/RESULTS.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] Load the costly-runs skill. Per model, `bench run --stage pilot-8 --profile stage8 --status-line --configs B,S --repeats 1` on the 18 instances in Decision 5's order, into `~/.local/share/mrw-bench/runs/stage8/pilot/<model>/` (`--status-line` because Stage 5's B ran with it and stage8 refuses a run without it); bench owes and runs the replacements Decision 5's walk names. Then copy each `*.receipt.json`, each `*.started` file and `run.json`, unrenamed, into `docs/bench/stage-8/pilot/<model>/`; transcripts stay out of the tree, and `RESULTS.md`'s `transcripts: ` line names `~/.local/share/mrw-bench/runs/stage8/pilot`, which `count` searches recursively by `<attempt>.transcript.jsonl`. `verdict` reads `<receipts>/<model>/run.json` and every receipt below it, so the layout above is the one it and the fence's `$R/pilot/$m/run.json` take. Stop at $15 spent, on an unsupported exclusion, or at a third dropped instance on a model. [proof: human: the receipts, their count and the spend]
3. [S3] `bench_edits.py count` into `counts.json`; `bench_edits.py verdict` into `verdict.txt`. [proof: acceptance]
4. [S4] Write `RESULTS.md`: every measure beside its threshold, the prediction beside the outcome, what the pilot cannot tell (Decision 5), the false-SHIP rate the rule carries, and no claim beyond the verdict. [proof: human: the record against Decision 5]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-8
h1=$(shasum -a 256 scripts/bench_edits.py | cut -d' ' -f1) && h2=$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1) \
  && for f in $R/reference.md $R/RESULTS.md; do test "$(sed -n 's/^script sha256: //p' "$f")" = "$h1" && test "$(sed -n 's/^helper sha256: //p' "$f")" = "$h2" || exit 1; done \
  && s=$(sed -n 's/^paragraph sha256: //p' $R/pilot.md) && test -n "$s" \
  && want=$(jq -c '.selected[2:]' bench/swe-bench/selection.json) && test "$(jq 'length' <<<"$want")" = 18 \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do jq -e --arg m "$m" --arg s "$s" --argjson want "$want" '.stage == "pilot-8" and .models == [$m] and (.configs | sort) == ["B","S"] and .repeats == 1 and .profile == "stage8" and .budget_seconds == 1200 and .cli_version == "2.1.317 (Claude Code)" and (.mrw_version | startswith("v1.83.0 ")) and .edit_paragraph_sha256 == $s and .tasks == $want' "$R/pilot/$m/run.json" > /dev/null && go run ./cmd/bench verify --stage pilot-8 --tasks 18 --repeats 1 "$R/pilot/$m" || exit 1; done \
  && T=$(sed -n 's/^transcripts: //p' $R/RESULTS.md) && test -d "$T" \
  && python3 -B scripts/bench_edits.py count --receipts $R/pilot --transcripts "$T" > /tmp/adr193-counts.json \
  && diff /tmp/adr193-counts.json $R/counts.json \
  && python3 -B scripts/bench_edits.py verdict --receipts $R/pilot --counts $R/counts.json > /tmp/adr193-verdict.txt \
  && diff /tmp/adr193-verdict.txt $R/verdict.txt \
  && grep -qE '^verdict: (SHIP|NO SHIP)$' $R/verdict.txt \
  && grep -qF -- "$(tail -1 $R/verdict.txt)" $R/RESULTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: both scripts are the ones that counted the reference row; each model's `run.json` is the registered matrix (instances in order, one model, B and S, repeat 1, stage8, the budget, CLI, mrw and paragraph); `bench verify` finds every planned receipt and owed replacement; a recount of the transcripts reproduces `counts.json`, digests included; `verdict` (which refuses an omitted, duplicated or foreign instance, T2) reproduces `verdict.txt`, and it is quoted | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts, counts and verdict |
| 2 — something selects it | T5 runs only on its verdict line |
| 3 — the caller can discover it | `docs/bench/README.md`'s row |
| 4 — it is used | T5 on SHIP; the BACKLOG follow-ups either way |

## Mutation Log
- 2026-11-04 · 86bf7511* · mutant killed · exit 1 · `docs/bench/stage-8/verdict.txt` · a verdict file the frozen script does not reproduce · acceptance-sha256:fa599ffd84b1929b96350af8ecef8ad57bb769ae23987717ef870ea85391cae7 · covers:the verdict reproduces from the counts and receipts

## Invariants

- No instance is added, dropped or reordered for an outcome; the only drop is Decision 5's terminal exclusion.
- The scripts, thresholds and paragraph are the ones committed before the first counted call.

## Risks

- Shared-machine load skews wall time; wall time is descriptive only.
- The fence's recount needs this Mac's transcripts; elsewhere it fails at `test -d`, which says so.

## Stop Condition

Spend passes $15, an unsupported exclusion or a third dropped instance on a model appears, or a counted receipt shows S's init tools or prompt sha other than T3's: stop to Zy.

## Out of Scope

- Changing the guide (T5's job, on SHIP only).

## Verification Log
- 2026-11-04 · 86bf7511* · exit 1 · `set -o pipefail …` · acceptance-sha256:fa599ffd84b1929b96350af8ecef8ad57bb769ae23987717ef870ea85391cae7 · ms:59 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-file:docs/adr/locks/deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11.lock
  ```
  --- last 1 line(s) of stderr
  sed: docs/bench/stage-8/RESULTS.md: No such file or directory
  ```
- 2026-11-04 · 86bf7511* · exit 0 · `set -o pipefail …` · acceptance-sha256:fa599ffd84b1929b96350af8ecef8ad57bb769ae23987717ef870ea85391cae7 · ms:411
- 2026-11-04 · 86bf7511* · exit 0 · `set -o pipefail …` · acceptance-sha256:fa599ffd84b1929b96350af8ecef8ad57bb769ae23987717ef870ea85391cae7 · ms:961
