# Stage 8: the counted pilot (ADR-193 T4)

Run 2026-11-04 on this Mac. Bench built from `1973d056`, Claude Code 2.1.317, mrw v1.83.0 (the release asset), the paragraph
frozen in `pilot.md` (sha `44bb3579`). 18 instances (`bench/swe-bench/selection.json` from its third entry), B and S, one
repeat, Opus then Sonnet. Spend $8.61 (Opus $6.88, Sonnet $1.73), under the $15 stop. One Opus attempt
(`django__django-14404`) was excluded at setup and bench ran its replacement, as Decision 5's walk admits; no instance was
dropped on either model.

script sha256: 16e62d90339b7159e1cf74fa7cbc3197b6afabbd1075edd264b8c45754f31487
helper sha256: 49001558d004aec635965ec4a53c7628ff5aaffcd80a91aec1f44255f1fcc1ac
transcripts: /Users/zy/.local/share/mrw-bench/runs/stage8/pilot

Receipts, `*.started` and `run.json` per model in `pilot/<model>/`; `counts.json` is `bench_edits.py count` over them and
the transcripts, `verdict.txt` is `bench_edits.py verdict`, both unedited.

## Every measure beside its threshold

| Clause | Opus B | Opus S | Opus | Sonnet B | Sonnet S | Sonnet |
|---|---|---|---|---|---|---|
| (U) S attempts with an attributed SEARCH/REPLACE write, need 9 | — | 18 | ok | — | 18 | ok |
| (O) coverage ≥ 90%, a decided call each arm | 100.0% | 100.0% | ok | 100.0% | 97.2% | ok |
| (a) S solved ≥ B solved − 2 | 17 | 17 | ok | 14 | 12 | ok |
| (b) (i) refused rate, S ≤ ⅔ of B's | 7.3% | 7.3% | fails | 15.0% | 48.6% | fails |
| (b) or (ii) output per attempt, S ≤ 0.85 of B's | 2928.7 | 3696.7 | fails | 1186.0 | 1481.2 | fails |
| (g) landed per attempt, S ≤ 1.2 × B's | 2.11 | 2.11 | ok | 0.94 | 1.00 | ok |
| (c) spend per success, S ≤ 1.10 × B's | 0.1990 | 0.2014 | ok | 0.0561 | 0.0788 | fails |
| model | | | **FAIL** | | | **FAIL** |

Descriptive, judged by nothing: on the both-solved subset S's spend over B's minus one is [−18.3%, +22.3%] on Opus
(17 instances) and [−14.6%, +34.1%] on Sonnet (12), 4,000 resamples, seed 193. Output per landed write call: Opus B 1387,
S 1751; Sonnet B 1256, S 1481.

## The prediction beside the outcome

- *Uptake holds on both models:* held, 18 of 18 on each.
- *(i) likely unavailable or small, the route to PASS is (ii):* B refused 3 on each model, so (i) was available; it did
  not fall on Opus (7.3% both arms) and rose on Sonnet.
- *(ii) more likely holds or rises than falls by 15%:* it rose, +26% on Opus and +25% on Sonnet.
- *S's refused count no higher than B's:* held on Opus (3 and 3), failed on Sonnet (3 and 17).
- *NO SHIP at least as likely as SHIP:* NO SHIP.

## What the Sonnet refusals were (descriptive)

15 of Sonnet S's 17 refused calls ended `mrw: search_replace: no hunks`, in 8 of its 18 attempts; none on Opus, none in
either B arm. In each the agent passed the file it meant to edit as `FILE` and the document on standard input, as
`mrw write --format=search_replace lib/mpl_toolkits/mplot3d/axes3d.py <<'EOF'` in `matplotlib__matplotlib-23314`, so
mrw read the Python source as the SEARCH/REPLACE document and found no block in it. The paragraph says "apply it with
mrw write --format=search_replace FILE", and FILE can be read as the target. The T3 smoke did not show it (its 2 Sonnet S
attempts named the document). This is a finding about the paragraph's wording and about mrw's refusal, which does not
say the file it read holds no SEARCH marker; it changes nothing here, because the rule was registered before the run
and a reworded paragraph is a new pilot (Zy's call, ADR-193 Out of Scope).

## What the pilot cannot tell

Decision 5's limits stand: a refusal rate over about 2 write calls per attempt is close to noise at 18 attempts; a
solving margin of 2 in 18 cannot see a loss under about 11 points; spend per success moves by one expensive failure.
Under the null this rule says SHIP 5.6% of the time (the frozen script's figure); a NO SHIP says the paragraph did not
clear the bar in this run, not that SEARCH/REPLACE costs more.

## The verdict

Nothing in mrw's guide changes; T5 does not run.

verdict: NO SHIP
