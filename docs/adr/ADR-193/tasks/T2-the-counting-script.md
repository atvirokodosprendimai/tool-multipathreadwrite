# Task ADR-193-T2: the frozen counting script and Stage 5's reference row

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `scripts/bench_edits.py` (`count`, `verdict`, `null`), `docs/bench/stage-8/reference.md` with both scripts' hashes
**Consumes:** none
**Data dependency:** hermetic for the tests; the reference row and the null rate need Stage 5's B transcripts on this Mac (`~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5/block-*/`)
**Proof map:** v1
**Rests-on:** `a write call is classified by its result`, `a bench hook block is not a write`, `uptake needs one attributed write`, `a SEARCH/REPLACE heredoc is one write call`, `an untokenized write is undecided`, `a mixed call is refused`, `global flags before write count`, `the verdict follows Decision 5`, `undefined measures fail`, `the exclusion walk takes the final receipt`, `another instance list is refused`, `counts match receipts and digests`, `the reference row names both scripts`

## Goal

ADR-193 Decision 4's counting rule and Decision 5's decision rule exist as one script, tested on synthetic
transcripts and receipts, committed with the hash of the helper it imports before any counted call; Stage 5's B is
counted by it as a reference row, and its `null` mode reproduces Decision 5's false-SHIP rate.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/bench_edits.py` | add | `count --receipts DIR --transcripts DIR` walks each model's final receipts (Decision 5's walk) and prints, per counted attempt, its id, its transcript's SHA-256, landed, refused (by marker), undecided, untokenized, blocked, decided and attributed SEARCH/REPLACE write calls, landed attributed calls, and whether the transcript has a final result, as JSON; `verdict --receipts DIR --counts FILE` refuses any instance list, order, model, arm or repeat but Decision 5's, any count set that is not exactly the final receipts, and any unsupported exclusion, then prints the measures, (U), (O), (a), (b), (g), (c), the descriptive rows, the per-model result and a last line `verdict: SHIP` or `verdict: NO SHIP`; `null --transcripts DIR` reproduces Decision 5's resampling (20,000, seed 193). Imports `mrw_calls` and `result_text` from `bench_mechanism.py`, which this task does not edit |
| `scripts/test_bench_edits.py` | add | the tests below |
| `docs/bench/stage-8/reference.md` | add | Stage 5's B per model under the frozen rule, the null rate, the transcripts' path and count, a `script sha256: ` line for `bench_edits.py` and a `helper sha256: ` line for `bench_mechanism.py` |

## Ordered Steps

1. [S1] Write the failing tests and see them red (the script does not exist). [proof: acceptance]
2. [S2] `count`: write calls after `-C DIR`, `--root DIR` and `--root=DIR`; untokenized calls; blocked, refused, landed, refused-by-`mrw:` and undecided in Decision 4's order; attribution of a SEARCH/REPLACE call to mrw's own verdict for its one write; missing measures when no final result; each transcript's sha256. [proof: mutation]
3. [S3] `verdict`: the registered instance list and matrix, the final-receipt walk and its stops, (i), (ii), (iii), (g), (O) as ratios of sums, the imputation against S, the thresholds of Decision 5 as constants, the descriptive bootstrap (4,000 resamples, seed 193), and the verdict line. [proof: mutation]
4. [S4] Run `count` and `null` over Stage 5's 382 B transcripts and write `reference.md` with both scripts' sha256. Its figures should match ADR-193's Context (555/94 and 359/106 decided/refused; 5.6% false SHIP, the frozen script's figure, which replaced the draft simulation's 6.1% on 2026-11-03); a difference is written down beside them, never smoothed. [proof: human: the counts and the rate against the Context's]
5. [S5] Commit before T4's first counted call. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-8
out=$(python3 -B -m unittest discover -s scripts -p test_bench_edits.py -v 2>&1) \
  && for t in test_a_write_call_is_refused_only_on_an_mrw_refusal test_a_bench_hook_block_is_not_a_write test_uptake_needs_one_attributed_search_replace_write test_a_search_replace_heredoc_is_one_write_call test_an_untokenized_command_holding_mrw_write_is_undecided test_a_mixed_call_counts_as_refused test_global_flags_before_write_are_counted test_the_verdict_follows_the_registered_rule test_undefined_measures_fail_conservatively test_the_exclusion_walk_takes_the_final_receipt test_the_verdict_refuses_another_instance_list test_counts_must_match_the_final_receipts_and_digests; do grep -qE "^$t \(.*\) \.\.\. ok$" <<<"$out" || exit 1; done \
  && test "$(sed -n 's/^script sha256: //p' $R/reference.md)" = "$(shasum -a 256 scripts/bench_edits.py | cut -d' ' -f1)" \
  && test "$(sed -n 's/^helper sha256: //p' $R/reference.md)" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_a_write_call_is_refused_only_on_an_mrw_refusal` | `scripts/test_bench_edits.py` | `— applied` lands; each refusal marker of Decision 4 refuses; a result with an `mrw: ` line and no landed marker is refused; a result with neither is undecided; a `--dry-run` write is no write call | none | S2 |
| `test_a_bench_hook_block_is_not_a_write` | `scripts/test_bench_edits.py` | a result holding `PreToolUse:Bash hook error` is blocked and out of every measure, even when it also holds a refusal marker | none | S2 |
| `test_uptake_needs_one_attributed_search_replace_write` | `scripts/test_bench_edits.py` | `--format=search_replace` and `--format search_replace` with a landed or refused verdict are attributed; `false && mrw write --format=search_replace F` with no verdict is not; `false && mrw write --format=search_replace F; mrw write P` is not (two writes); a native plan and `--format=apply_patch` are not | none | S2 |
| `test_a_search_replace_heredoc_is_one_write_call` | `scripts/test_bench_edits.py` | a quoted heredoc to `$TMPDIR` then `mrw write --format=search_replace FILE`, and a document piped to `-`, each count as one attributed SEARCH/REPLACE write call | none | S2 |
| `test_an_untokenized_command_holding_mrw_write_is_undecided` | `scripts/test_bench_edits.py` | a `printf` holding `<<<<<<< SEARCH` piped to `mrw write --format=search_replace -`, which `mrw_calls` cannot tokenize, is an untokenized, undecided write call that counts against coverage | none | S2 |
| `test_a_mixed_call_counts_as_refused` | `scripts/test_bench_edits.py` | the Opus `sphinx-doc__sphinx-7748` shape: one call whose first write's plan does not parse (`plan has 1 error(s)`) and whose second lands is refused, not landed | none | S2 |
| `test_global_flags_before_write_are_counted` | `scripts/test_bench_edits.py` | `mrw -C DIR write P`, `mrw --root DIR write P` and `mrw --root=DIR write P` are write calls; `mrw -C DIR read F` is not | none | S2 |
| `test_the_verdict_follows_the_registered_rule` | `scripts/test_bench_edits.py` | synthetic receipts and counts: 8 of 18 attributed is UNTESTED and 9 is not, 7 of 16 is and 8 is not; coverage 89% is UNTESTED and 90% is not; solved minus 3 FAILs and minus 2 does not; (i) at two thirds passes with B at 3 refusals and not at 2; (ii) at 0.85 passes and above it does not; (g) at 1.2 passes and above it FAILs; (c) at 1.10 passes and above it FAILs; one model short of PASS gives `verdict: NO SHIP` | none | S3 |
| `test_undefined_measures_fail_conservatively` | `scripts/test_bench_edits.py` | zero decided calls on an arm is UNTESTED; zero landed calls FAILs (g); B and S at zero refusals leave (b) its second branch only; zero solved on either arm FAILs (c); an empty both-solved subset prints no interval and changes nothing; an S attempt with no final result takes the model's largest output and cost, a B attempt the smallest, never zero | none | S3 |
| `test_the_exclusion_walk_takes_the_final_receipt` | `scripts/test_bench_edits.py` | a setup failure replaced by a good `.r1` counts `.r1`; a start error replaced twice counts `.r2`; a setup failure whose `.r1` is excluded, and a start error whose `.r2` is, drop the instance from both arms for that model; an infrastructure exclusion, a missing replacement and three dropped instances are refused | none | S3 |
| `test_the_verdict_refuses_another_instance_list` | `scripts/test_bench_edits.py` | an omitted instance, a duplicated one, a foreign one, the list out of order, a third arm or a second repeat are refused; retries are not counted toward the 72 | none | S3 |
| `test_counts_must_match_the_final_receipts_and_digests` | `scripts/test_bench_edits.py` | counts naming an attempt that is not a final receipt, missing one, or recording another transcript sha256 than `count` computes are refused | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the script and its tests |
| 2 — something selects it | T4's fence runs `count` and `verdict` and diffs their output |
| 3 — the caller can discover it | the script's usage line; ADR-193 Decision 4 names it |
| 4 — it is used | `reference.md`, T4 |

## Mutation Log
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: --root DIR before write is not seen · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:global flags before write count
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: bench's own hook block is counted · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:a bench hook block is not a write
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: a call holding two writes is attributed · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:uptake needs one attributed write
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: a result with an mrw: line and no landed marker is undecided · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:a write call is classified by its result
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: an untokenized call no longer counts against coverage · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:an untokenized write is undecided
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: refusal markers other than nothing was written are ignored, so a mixed call lands · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:a mixed call is refused
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S2: a SEARCH/REPLACE document written to a file is not attributed · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:a SEARCH/REPLACE heredoc is one write call
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S3: coverage of exactly 90% is UNTESTED · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:the verdict follows Decision 5
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S3: an S attempt without a result takes the smallest value, not the largest · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:undefined measures fail
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S3: a start error is replaced once, not twice · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:the exclusion walk takes the final receipt
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S3: the instance order is not checked · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:another instance list is refused
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `scripts/bench_edits.py` · S3: a recomputed digest is not compared · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · covers:counts match receipts and digests

## Invariants

- The script's markers, order, thresholds, imputation, seed and resample counts are ADR-193 Decisions 4 and 5's, and neither script changes after T4's first counted call.
- `scripts/bench_mechanism.py` is not edited (ADR-192 froze it at `3a970837`).

## Risks

- A receipt the agent hid (`> /dev/null`, `--quiet`) is undecided and counts against coverage.

## Stop Condition

The reference row or the null rate differs from the Context's and the difference cannot be explained by a named rule: stop to Zy before T4.

## Out of Scope

- The bench configuration (T1's job).

## Verification Log
- 2026-11-03 · 5c2a308b* · exit 1 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:189 · test-lock-sha256:c00359992f6ba8ddc9257362b4b36ee04d325e2b11139bde3784c36ed7f1e522 · test-lock-file:docs/adr/locks/c00359992f6ba8ddc9257362b4b36ee04d325e2b11139bde3784c36ed7f1e522.lock
  ```
  ```
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:918
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:798
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:772
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:792
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:725
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:778
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:756
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:723
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:790
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:832
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:695
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:761
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:0a55a766b9af6e47fb037f30ced4d4ac0c6513a2cd955e535b60828fc90a2164 · ms:729
