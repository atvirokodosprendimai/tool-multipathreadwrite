# Task ADR-206-T4: the counting script

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `scripts/bench_stage9.py` with `count`, `report` and `spend` verbs; `scripts/test_bench_stage9.py`; the suite in CI
**Consumes:** receipts and `run.json` (T1), the kept log (T2), `classes.json` (T3), `scripts/bench_mechanism.py`
**Data dependency:** none — hermetic (fixture run directories, logs and transcripts)
**Proof map:** v1
**Rests-on:** `correct, outside, other and missing partition the planned attempts`, `a null cost is missing`, `a comparison under its coverage prints INSUFFICIENT`, `exposure is per attempt in both arms`, `mismatched runs are refused`, `the bootstrap is seeded`, `spend prints one number and the count without a cost`

## Goal

ADR-206 Decision 7, on fixture run directories built to hit each rule: every measure; each comparison printed as INSUFFICIENT with its reason (under 10 of 12 paired, a paired cost missing, zero `correct` in an arm, over 5% of resamples dropped); exposure in M read only from the output of a command that ran `mrw write`, with the count of write results it could read; the list of N attempts that were exposed and ended `outside`; refusals for run directories that differ where they must not, whose mrw hash is not the arm's, or whose M and N hashes are equal. The statistic of a comparison is the first arm's spend per `correct` over the second's; a resample in which either is undefined or zero is dropped and counted. `spend` prints exactly one line of six fields, `stage9 spend usd <number> missing <count>`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/bench_stage9.py`, `scripts/test_bench_stage9.py` | new | the script and its tests |
| `.github/workflows/ci.yml` | edit | the suite beside CI's Python suites |

## Ordered Steps

1. [S1] The tests, RED: no script. [proof: acceptance]
2. [S2] The script. Mutants, each killed: a null cost counted as zero; INSUFFICIENT replaced by a ratio; `outside` and `other` merged; an excluded attempt counted as not correct; the instructions' own text counted as a refusal; the hash checks dropped; the seed taken from the clock. [proof: mutation]

## Acceptance

```bash
set -o pipefail
python3 -B -m unittest discover -s scripts -p 'test_bench_stage9.py' 2> "${TMPDIR:-/tmp}/adr206-t4.out" \
  && grep -qE '^Ran [1-9][0-9]* tests?' "${TMPDIR:-/tmp}/adr206-t4.out" && grep -qx 'OK' "${TMPDIR:-/tmp}/adr206-t4.out" \
  && grep -v '^ *#' .github/workflows/ci.yml | grep -q 'test_bench_stage9.py' \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_each_outcome_and_the_partition` | `scripts/test_bench_stage9.py` | correct, outside, other and missing partition the planned attempts | none | S1, S2 |
| `test_a_null_cost_is_missing_never_zero` | `scripts/test_bench_stage9.py` | a null cost is missing, and the spend line counts it as missing | none | S1, S2 |
| `test_under_ten_paired` | `scripts/test_bench_stage9.py` | INSUFFICIENT under 10 of 12 paired | none | S1, S2 |
| `test_a_paired_cost_missing` | `scripts/test_bench_stage9.py` | INSUFFICIENT when a paired cost is missing | none | S1, S2 |
| `test_zero_correct_in_an_arm` | `scripts/test_bench_stage9.py` | INSUFFICIENT when either arm has no correct attempt | none | S1, S2 |
| `test_over_five_percent_of_resamples_dropped` | `scripts/test_bench_stage9.py` | INSUFFICIENT over 5% of resamples dropped, and no ratio beside it | none | S1, S2 |
| `test_in_m_it_is_a_refusal_in_the_output_of_a_write` | `scripts/test_bench_stage9.py` | exposure in M, per attempt, from the output of a write | none | S1, S2 |
| `test_in_n_it_is_a_line_in_the_kept_log` | `scripts/test_bench_stage9.py` | exposure in N, per attempt, from the kept log | none | S1, S2 |
| `test_the_text_of_mrw_instructions_alone_does_not_count` | `scripts/test_bench_stage9.py` | the instructions' own text is not a refusal | none | S1, S2 |
| `test_the_phrase_is_the_one_internal_apply_writes` | `scripts/test_bench_stage9.py` | the refusal's words pinned against `internal/apply`'s text | none | S1, S2 |
| `test_a_run_of_the_wrong_model_or_configuration_is_refused` | `scripts/test_bench_stage9.py` | mismatched runs are refused | none | S1, S2 |
| `test_the_seed_is_206_and_the_interval_is_pinned` | `scripts/test_bench_stage9.py` | the bootstrap is seeded | none | S1, S2 |
| `test_spend_is_exactly_one_line_of_six_fields_over_every_receipt` | `scripts/test_bench_stage9.py` | spend prints one line of six fields over every receipt | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | T5 and T6 run it |
| 3 — the caller can discover it | its docstring, `docs/bench/README.md` |
| 4 — it is used | T6's fence re-runs it over the receipts |

## Mutation Log
- 2026-11-20 · d5150a2c* · mutant killed · exit 1 · `scripts/bench_stage9.py` · an excluded attempt is graded · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · covers:correct, outside, other and missing partition the planned attempts
- 2026-11-20 · d5150a2c* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a missing paired cost does not stop the comparison · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · covers:a comparison under its coverage prints INSUFFICIENT
- 2026-11-20 · d5150a2c* · mutant killed · exit 1 · `scripts/bench_stage9.py` · the seed is taken from the clock · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · covers:the bootstrap is seeded
- 2026-11-20 · d5150a2c* · mutant killed · exit 1 · `scripts/bench_stage9.py` · runs of another Claude Code or seed are accepted · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · covers:mismatched runs are refused
- 2026-11-20 · 89ae41a5* · mutant killed · exit 1 · `scripts/bench_stage9.py` · an excluded attempt is graded · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · covers:correct, outside, other and missing partition the planned attempts

## Verification Log
- 2026-11-20 · d5150a2c* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:98 · test-lock-sha256:74225a09fe25d5784c67f5416524e061a3b8fac30bfea9afbd8d714c211d1643 · test-lock-file:docs/adr/locks/74225a09fe25d5784c67f5416524e061a3b8fac30bfea9afbd8d714c211d1643.lock
  ```
  ```
- 2026-11-20 · d5150a2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:517
- 2026-11-20 · d5150a2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:591
- 2026-11-20 · d5150a2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:614
- 2026-11-20 · d5150a2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:550
- 2026-11-20 · d5150a2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:564
- 2026-11-20 · b2b20d48* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:3 · test-lock-sha256:aceedd6dfd27db87792696ea24d01a60ad2fa42a76e059dce8ba6dd636971879 · test-lock-file:docs/adr/locks/aceedd6dfd27db87792696ea24d01a60ad2fa42a76e059dce8ba6dd636971879.lock · test-lock-kind:replace
- 2026-11-20 · b2b20d48* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:554
- 2026-11-20 · 89ae41a5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:0 · test-lock-sha256:ae614c9174bf4665f30170d92aac1514b00796e865f4c8e348b2a40cedc9d2a6 · test-lock-file:docs/adr/locks/ae614c9174bf4665f30170d92aac1514b00796e865f4c8e348b2a40cedc9d2a6.lock · test-lock-kind:replace
- 2026-11-20 · 89ae41a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:588
- 2026-11-20 · 89ae41a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:736
- 2026-11-25 · 1e55658d* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:35 · test-lock-sha256:37093683b8ec0c14ac28623e306b9f4f148f7c5da2f8db1fcbecba297e89f414 · test-lock-file:docs/adr/locks/37093683b8ec0c14ac28623e306b9f4f148f7c5da2f8db1fcbecba297e89f414.lock · test-lock-kind:replace
- 2026-11-26 · 418388db* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4165a7078b709ce8ecf35441edbc3b8fd71bec6cea7fd605430a27c6f375351 · ms:34 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace

## Invariants

- The script reads; it writes only to standard output.

## Risks

- A host can cut or redirect a tool result: the script reports how many write results it read, and does not call the rest zero.

## Stop Condition

The fence exits 0 and every gate in `AGENTS.md` passes.

## Out of Scope

- Verdict words for ADR-203 or the taught form (permanent: boundary: the pilot decides neither)
