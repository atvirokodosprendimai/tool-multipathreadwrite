# Task ADR-211-T1: the instrument says what it cannot see, reconciles cost, and holds its identity

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L (multi-file)
**Owner:** Zy
**Produces:** `bench canary --max-spend` and `canary.json` `init_models` and `max_spend_usd`; `bench run --keep-scratch-file`, `run.json` `keep_scratch_file`, receipt `kept_scratch_file_sha256`; receipt `transcript_sha256`; every run's canary model check; the amended patch, `builds.md` and ADR-206 T2's table; `bench/stage-9/plans.json` and `smoke-plans.json`; `count`'s exposure states, reconciliation and identity checks, and M against A on every class
**Consumes:** the runner, the two builds, the tasks and the counting script ADR-206 built
**Data dependency:** `TestArmNDiffersFromMInOneRefusal` needs the pinned tag in the clone (it skips in CI's shallow checkout); the fence runs it where the tag is and forbids the skip
**Proof map:** v1
**Rests-on:** `a run takes only a canary of its own model`, `a canary records each configuration's init model`, `a canary stops on its spend at either boundary`, `bench canary takes --max-spend`, `a receipt's transcript hash is taken at close`, `a stopped run is not resumed`, `a scratch file is kept beside its receipt`, `N logs only an event, beside the state home`, `a check-coupled task grades the base's tests`, `the grader's files are not the agent's`, `exposure has the states the evidence can tell apart`, `cost reconciles to the stop's receipts`, `count requires the registered plans`, `count refuses a swapped transcript and an unobserved model`, `M against A is read on every class`, `diff statistics leave the oracle's index alone`

## Goal

ADR-211 Decision 2's report and Decisions 3 to 9 and 11, each with a test that fails before its change, except the stopped-run refusal, which exists and gets a test and a killed mutant:

- the runner: every run takes only a canary whose `model` it runs and whose profile is its own; `bench canary --max-spend`, checked before each configuration's attempt and after the last, exit 3 on reaching the figure or on a null cost, `canary.json` written `pass: false` with the costs on a spend stop and on an interrupt; `init_models` per configuration, a missing one failing the canary; `max_spend_usd` recorded in `canary.json`; `transcript_sha256` in every receipt whose attempt captured a transcript, and none in one whose setup failed before it opened; `--keep-scratch-file` and its key, note and `bench verify` check, refused beside `--keep-state-file`;
- the patch: the log written only by a landed write with an exposed hunk, in the parent of `$XDG_STATE_HOME`; one `mrw: stage9 log:` stderr line on any failure, the write's result unchanged; `build.sh` rebuilt, `builds.md`'s `n sha256` updated, and ADR-206 T2's table and `TestArmNDiffersFromMInOneRefusal` changed to match: the anchored write leaves no log; the log beside the state home and none inside it; a log path made a directory gives the same stdout and the one line on stderr; an event logged by one write survives a later write whose append fails, which prints the line;
- the oracle and the runner's order (#709, #710), the diff statistics taken in an index of their own under the scratch, as `keepDiff` takes the kept diff;
- the script: Decision 2's comparisons, M against A on all four classes beside M against N on follow-on and control; Decision 5's states and fields, a clipped write result unattributable; Decision 6's buckets and labels over the runner's own receipt population (direct entries of the run directory and of `side/`, compared in rounded millionths), a deeper receipt refused, and paired and unpaired spend printed under INSUFFICIENT too; Decision 7's checks against the plans files and the transcript hash required on every graded attempt, Decision 8's `init_model` check, `MENTION` with flags, `MAY_DIFFER` without the instruction hash, a changed `bin/mrw` as `missing`; and the two plans files, `plans.json` holding the six registered runs in Decision 9's order with their figures and task lists from `classes.json`, `smoke-plans.json` the smoke's runs and probe.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go`, `canary.go`, `pins.go`, `oracle.go` | edit | Decisions 4, 7, 8, 9, 11 |
| `cmd/bench/main.go`, `cmd/bench/stage9_test.go` | edit | `--max-spend` on canary, `--keep-scratch-file` on run; the wiring test no longer refuses `canary --max-spend` |
| `internal/bench/stage9_test.go`, `oracle_test.go` | edit | the Go tests below; the "a run that pins nothing checks no model" case is reversed |
| `bench/stage-9/no-refusal.patch`, `builds.md`, `internal/bench/stage9_build_test.go`, `docs/adr/ADR-206/tasks/T2-the-two-builds.md` | edit | Decision 3 |
| `bench/stage-9/plans.json`, `bench/stage-9/smoke-plans.json` | new | Decisions 7 and 9 |
| `scripts/bench_stage9.py`, `scripts/test_bench_stage9.py` | edit | Decisions 2, 5 to 8; `test_m_against_a_is_read_on_single_site_and_batch_only`, which requires M against A on follow-on to be absent, is replaced, and the lock rows that name it move |
| `docs/bench/README.md` | edit | the new flags and keys |

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARunTakesOnlyACanaryOfItsOwnModel` | `internal/bench/stage9_test.go` | Decision 8: pinned or not, another model's canary and another profile's are refused | none | S1, S2 |
| `TestACanaryRecordsTheModelEachConfigurationReported` | `internal/bench/stage9_test.go` | Decision 8: `init_models` from the init lines; a configuration with none fails | none | S1, S2 |
| `TestACanaryStopsOnItsSpendAtEitherBoundary` | `internal/bench/stage9_test.go` | Decision 9: A's cost reaching the figure ends it before B starts, exit 3; B crossing it on the last attempt, exit 3; a null cost, exit 3; `canary.json` records the applied `max_spend_usd`; each writes `canary.json` `pass: false` with the costs | none | S1, S2 |
| `TestBenchCanaryTakesMaxSpend` | `cmd/bench/stage9_test.go` | Decision 9: the flag reaches the canary's options, and a spend stop ends `bench canary` with status 3 | none | S1, S2 |
| `TestAReceiptRecordsItsTranscriptHashAtClose` | `internal/bench/stage9_test.go` | Decision 7: the hash of the bytes read back after close; a setup failure before the transcript opens leaves the key absent | none | S1, S2 |
| `TestASpendStoppedRunIsNotResumedWithItsFigure` | `internal/bench/stage9_test.go` | Decision 7: exit 3 and no new `.started` | none | S3 |
| `TestAScratchFileIsKeptBesideTheReceipt` | `internal/bench/stage9_test.go` | Decision 4: kept and hashed before grading; a link, a directory or a name with a separator is a `scratch:` note; refused beside `--keep-state-file`; `bench verify` refuses a changed kept file | none | S1, S2 |
| `TestArmNDiffersFromMInOneRefusal` | `internal/bench/stage9_build_test.go` | Decision 3: ADR-206 T2's table as amended | none | S4 |
| `TestACheckCoupledTaskGradesOnlyTheBasesTests` | `internal/bench/oracle_test.go` | Decision 11 (#709): a weakened original assertion is restored and fails the grade; an added `_test.go` file is removed before the run | none | S1, S2 |
| `TestAGraderWrittenFileIsNeitherInTheDiffNorASlippedWrite` | `internal/bench/stage9_test.go` | Decision 11 (#710): the grader ran (its mark exists), and the receipt's `diff` and `slipped_writes` omit the file it created and hold the one the agent created; and after the diff statistics, the tree's index is the one the agent left, so the oracle still sees the agent's untracked file as untracked | none | S1, S2 |
| `test_n_exposure_states` | `scripts/test_bench_stage9.py` | Decision 5, N: exposed, not seen, unknown; each malformed field and span; an event before a failed append is exposed, flagged, a lower bound | none | S1, S5 |
| `test_m_exposure_states` | `scripts/test_bench_stage9.py` | Decision 5, M: a real text refusal, a JSON one, a quoted imitation (not counted), a truncation-only result (unknown), a clipped result holding a refusal (unattributable, never a lone failure), a complete result with no refusal (not seen), the three kinds | none | S1, S5 |
| `test_cost_reconciles_to_the_stops_receipts` | `scripts/test_bench_stage9.py` | Decision 6: a priced replacement chain and a priced unmatched terminal attempt, each receipt in one bucket, the sums the stop's; a receipt nested below the run directory refused, not summed; costs a millionth apart around the figure compared as the runner rounds them; paired and unpaired spend printed beside an INSUFFICIENT comparison; missing counts printed | none | S1, S5 |
| `test_count_requires_the_registered_plans` | `scripts/test_bench_stage9.py` | Decision 7: a seventh directory, a missing one, a changed figure or task list are refused; the smoke plans pass the smoke's directories | none | S1, S5 |
| `test_count_refuses_swapped_transcripts` | `scripts/test_bench_stage9.py` | Decision 7: M's and N's transcripts swapped, stems equal, are refused; so is a graded receipt with no `transcript_sha256` | none | S1, S5 |
| `test_count_refuses_an_unobserved_init_model` | `scripts/test_bench_stage9.py` | Decision 8: absent, or not `pilot.md`'s | none | S1, S5 |
| `test_instruction_hash_is_absent_in_a_and_shared_by_m_and_n` | `scripts/test_bench_stage9.py` | Decision 7: a valid six-arm set passes; M and N differing, or A carrying one, is refused | none | S1, S5 |
| `test_mention_sees_flags_before_write` | `scripts/test_bench_stage9.py` | Decision 7 | none | S1, S5 |
| `test_a_changed_mrw_is_missing_and_listed` | `scripts/test_bench_stage9.py` | Decision 7 | none | S1, S5 |
| `test_m_against_a_is_read_on_every_class` | `scripts/test_bench_stage9.py` | Decision 2: M against A on single-site, batch, follow-on and control, M against N on follow-on and control, and no M against N on single-site or batch | none | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | `bench run`, `bench canary`, `count` |
| 3 — the caller can discover it | `docs/bench/README.md`, `bench --help` |
| 4 — it is used | ADR-211 T2's rehearsal, ADR-206 T5 and T6 |

## Ordered Steps

1. [S1] The red state: each named test except `TestASpendStoppedRunIsNotResumedWithItsFigure` and `TestArmNDiffersFromMInOneRefusal` fails before its change. [proof: acceptance]
2. [S2] The runner, canary, CLI and oracle changes, each with a killed mutant logged: the model check skipped for an unpinned run; a canary checked only before an attempt; a canary's `init_models` from the requested model; the transcript hashed before close; the scratch file kept after grading; base tests not restored; diff stats after grading. [proof: mutation]
3. [S3] The stopped-run test, green on today's runner, and its mutant: `spendStop` skipped on resume. [proof: mutation]
4. [S4] The patch rebuilt with `bench/stage-9/build.sh`, `builds.md` and ADR-206 T2's table and test amended in the same commit; mutants: the log written on a write with no event; a failure printing nothing; the log written inside the state home. [proof: mutation]
5. [S5] The plans files and the script; mutants: an unknown counted as not exposed; a span left unchecked; a quoted refusal counted; a superseded receipt also counted as terminal; the transcript hash unchecked; M against A dropped on follow-on; a nested receipt summed; a clipped refusal counted as a lone failure; the instruction hash allowed to differ between M and N. [proof: mutation]
6. [S6] `adr-lint` passes on ADR-206 again, whose done tasks fence on the tests and the script this task edits, and `scripts/fence-prose.py` is green. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
a=$(go test ./internal/bench/ ./cmd/bench/ -count=1 -v -run '^(TestARunTakesOnlyACanaryOfItsOwnModel|TestACanaryRecordsTheModelEachConfigurationReported|TestACanaryStopsOnItsSpendAtEitherBoundary|TestBenchCanaryTakesMaxSpend|TestAReceiptRecordsItsTranscriptHashAtClose|TestASpendStoppedRunIsNotResumedWithItsFigure|TestAScratchFileIsKeptBesideTheReceipt|TestArmNDiffersFromMInOneRefusal|TestACheckCoupledTaskGradesOnlyTheBasesTests|TestAGraderWrittenFileIsNeitherInTheDiffNorASlippedWrite)$' 2>&1) \
  && for t in TestARunTakesOnlyACanaryOfItsOwnModel TestACanaryRecordsTheModelEachConfigurationReported TestACanaryStopsOnItsSpendAtEitherBoundary TestBenchCanaryTakesMaxSpend TestAReceiptRecordsItsTranscriptHashAtClose TestASpendStoppedRunIsNotResumedWithItsFigure TestAScratchFileIsKeptBesideTheReceipt TestArmNDiffersFromMInOneRefusal TestACheckCoupledTaskGradesOnlyTheBasesTests TestAGraderWrittenFileIsNeitherInTheDiffNorASlippedWrite; do grep -qE "^--- PASS: $t\b" <<<"$a" || exit 1; done \
  && ! grep -q -- '--- SKIP' <<<"$a" \
  && p=$(python3 -m unittest -v scripts.test_bench_stage9 -k test_n_exposure_states -k test_m_exposure_states -k test_cost_reconciles_to_the_stops_receipts -k test_count_requires_the_registered_plans -k test_count_refuses_swapped_transcripts -k test_count_refuses_an_unobserved_init_model -k test_instruction_hash_is_absent_in_a_and_shared_by_m_and_n -k test_mention_sees_flags_before_write -k test_a_changed_mrw_is_missing_and_listed -k test_m_against_a_is_read_on_every_class 2>&1) \
  && [ "$(grep -cE '^test_.* \.\.\. ok$' <<<"$p")" -eq 10 ] \
  && for t in test_n_exposure_states test_m_exposure_states test_cost_reconciles_to_the_stops_receipts test_count_requires_the_registered_plans test_count_refuses_swapped_transcripts test_count_refuses_an_unobserved_init_model test_instruction_hash_is_absent_in_a_and_shared_by_m_and_n test_mention_sees_flags_before_write test_a_changed_mrw_is_missing_and_listed test_m_against_a_is_read_on_every_class; do grep -qE "^$t .* ok$" <<<"$p" || exit 1; done \
  && jq -e 'length == 6 and ([.[].model] | unique | length) == 2' bench/stage-9/plans.json > /dev/null \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Mutation Log
- 2026-11-20 · 79054ce2 · mutant killed · exit 1 · `internal/bench/runner.go` · the model check skipped for an unpinned run · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a run takes only a canary of its own model
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/canary.go` · a canary checked only before an attempt · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a canary stops on its spend at either boundary
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/canary.go` · a canary's init_models from the requested model · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a canary records each configuration's init model
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/runner.go` · the transcript hashed before it was complete · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a receipt's transcript hash is taken at close
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/oracle.go` · base tests not restored · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a check-coupled task grades the base's tests
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/runner.go` · diff stats after grading · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:the grader's files are not the agent's
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/runner.go` · spendStop skipped on resume · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a stopped run is not resumed
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · the log written on a write with no event · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:N logs only an event, beside the state home
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · a failure printing nothing · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:N logs only an event, beside the state home
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · the log written inside the state home · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:N logs only an event, beside the state home
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · an unknown counted as not exposed · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:exposure has the states the evidence can tell apart
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a span left unchecked · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:exposure has the states the evidence can tell apart
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a quoted refusal counted · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:exposure has the states the evidence can tell apart
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a superseded receipt also counted as terminal · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:cost reconciles to the stop's receipts
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · the transcript hash unchecked · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:count refuses a swapped transcript and an unobserved model
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · M against A dropped on follow-on · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:M against A is read on every class
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a nested receipt summed · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:cost reconciles to the stop's receipts
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a clipped refusal counted as a lone failure · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:exposure has the states the evidence can tell apart
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `scripts/bench_stage9.py` · the instruction hash allowed to differ between M and N · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:count requires the registered plans
- 2026-11-20 · 79054ce2* · mutant killed · exit 1 · `internal/bench/runner.go` · the scratch file kept after grading · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · covers:a scratch file is kept beside its receipt
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/runner.go` · the model check skipped for an unpinned run · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a run takes only a canary of its own model
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/canary.go` · a canary's init_models from the requested model · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a canary records each configuration's init model
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/canary.go` · a canary checked only before an attempt · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a canary stops on its spend at either boundary
- 2026-11-20 · df911068* · mutant killed · exit 1 · `cmd/bench/main.go` · the canary flag misnamed · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:bench canary takes --max-spend
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/runner.go` · the transcript hash not taken at close · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a receipt's transcript hash is taken at close
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/pins.go` · spendStop skipped on resume · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a stopped run is not resumed
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/pins.go` · the kept scratch file not hashed · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a scratch file is kept beside its receipt
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/oracle.go` · base tests not restored · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:a check-coupled task grades the base's tests
- 2026-11-20 · df911068* · mutant survived · exit 0 · `internal/bench/runner.go` · diff stats not taken from the private index · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:the grader's files are not the agent's
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · df911068* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · a failure message reworded · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:N logs only an event, beside the state home
- 2026-11-20 · df911068* · mutant killed · exit 1 · `scripts/bench_stage9.py` · an unknown counted as not exposed · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:exposure has the states the evidence can tell apart
- 2026-11-20 · df911068* · mutant killed · exit 1 · `scripts/bench_stage9.py` · a superseded receipt also counted as terminal · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:cost reconciles to the stop's receipts
- 2026-11-20 · df911068* · mutant killed · exit 1 · `scripts/bench_stage9.py` · the instruction hash allowed to differ between M and N · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:count requires the registered plans
- 2026-11-20 · df911068* · mutant killed · exit 1 · `scripts/bench_stage9.py` · the transcript hash unchecked · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:count refuses a swapped transcript and an unobserved model
- 2026-11-20 · df911068* · mutant killed · exit 1 · `scripts/bench_stage9.py` · M against A dropped on follow-on · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:M against A is read on every class
- 2026-11-20 · df911068* · mutant survived · exit 0 · `internal/bench/runner.go` · diff stats counted in the tree index · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:diff statistics leave the oracle's index alone
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/runner.go` · diff stats after grading · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:the grader's files are not the agent's
- 2026-11-20 · df911068* · mutant killed · exit 1 · `internal/bench/runner.go` · the diff statistics staged in the real index · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · covers:diff statistics leave the oracle's index alone

## Verification Log
- 2026-11-20 · 39400ec0* · exit 1 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:21984 · test-lock-sha256:e7faf557d44702f619f7bd74243f2c76d663ecb592ff4036252e2ca86fe356e0 · test-lock-file:docs/adr/locks/e7faf557d44702f619f7bd74243f2c76d663ecb592ff4036252e2ca86fe356e0.lock
  ```
  ```
- 2026-11-20 · b6caed28 · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19328
- 2026-11-20 · 79054ce2 · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:20697
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19785
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19861
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19397
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:18884
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:18717
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:18913
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19310
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:20910
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:21591
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19941
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:23059
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:23719
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:29501
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19681
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:21000
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:20650
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:42499
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:23242
- 2026-11-20 · 79054ce2* · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:22756
- 2026-11-20 · 79054ce2* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:2 · test-lock-sha256:a7c603cc0a29e3ac8cb11fd17f649136179d06af757a163bb9f12ae873fb4f7c · test-lock-file:docs/adr/locks/a7c603cc0a29e3ac8cb11fd17f649136179d06af757a163bb9f12ae873fb4f7c.lock · test-lock-kind:replace
- 2026-11-20 · 500e67e6 · exit 0 · `set -o pipefail …` · acceptance-sha256:479986de06324cb5198e9d4909d5690fd4611d12a12cf6b078c3d81f9f2d52fa · ms:19385
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:17829
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:24303
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:19107
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:19384
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:20831
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:19244
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:20981
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:21920
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:23716
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:19375
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:19196
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:20840
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:21142
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:23453
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:21860
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:22705
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:22473
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:22538
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:21935
- 2026-11-20 · df911068* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:28243
- 2026-11-20 · 79ed6aac* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:0 · test-lock-sha256:0b72b067b1d34970178b315c4c24f276f742070f978fa74cbf581ef4d5cf3689 · test-lock-file:docs/adr/locks/0b72b067b1d34970178b315c4c24f276f742070f978fa74cbf581ef4d5cf3689.lock · test-lock-kind:replace
- 2026-11-20 · 79ed6aac* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2e3efc6818e4c3fbf8b1f5b42c43582493ca034909b4e8bd236a74a8a3a0698 · ms:17651

## Invariants

- No package of `mrw` changes; the patch lives under `bench/`.
- A run that passes none of the new flags behaves as before, apart from the canary's model check.

## Risks

- The canary's model check now applies to every run: a stage before ADR-206, resumed on another model's canary, is refused and needs a canary of its own model.

## Stop Condition

The fence exits 0 on a clone that holds the pinned tag.

## Out of Scope

- The paid runs (permanent: boundary: ADR-206 T5 and T6)
- The gate and review validators, and the amended paid tasks (permanent: boundary: ADR-211 T3)
