# Task ADR-162-T2: the claim line, the harness's own result, and ADR-106's paired rows

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench run --status-line`, the receipt's `status`, `oracle.broke_tests` and `oracle.harness_resolved`, `bench export-paired`
**Consumes:** configuration P and the receipt's `plugin` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the claim is one structured line`, `the harness's verdict survives the budget`, `a broken PASS_TO_PASS test is recorded`, `the rows are ADR-106's`

## Goal

Every Stage 5 attempt ends with a claim bench reads without classifying prose. Every graded SWE attempt keeps the harness's own verdict and says whether it broke a passing test. `bench export-paired` writes the rows quality-harness ADR-106's scorer reads, with every configuration field it requires (ADR-162 Decision 3).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `--status-line` appends the registered sentence; its sha in `run.json`; `status` read from the result text; `harness_resolved` kept before `runner.go:580` adjusts `solved` |
| `internal/bench/swebench.go` | edit | `grade` reads `tests_status.PASS_TO_PASS.failure` from the report it already opens |
| `internal/bench/oracle.go` | edit | `Verdict.BrokeTests`, `Verdict.HarnessResolved` |
| `internal/bench/export.go` | add | `ExportPaired`: ADR-106 Decision 2's config line, then one row per instance, arm and repeat |
| `internal/bench/export_test.go` | add | the rows |
| `internal/bench/swebench_test.go` | edit | broke-tests from a synthetic report |
| `internal/bench/runner_test.go` | edit | the status line; the harness's verdict over budget |
| `cmd/bench/main.go` | edit | `--status-line`, `bench export-paired`, the usage text |
| `docs/bench/README.md` | edit | the claim line, the fields, the subcommand |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] The sentence, fixed now: "When you have finished, end your reply with one line, exactly `STATUS: done` if the task is complete, or `STATUS: not done: <reason>` if it is not." `status` is `done` or `not done` when exactly one line of the result text matches `^STATUS: (done|not done: .+)$`, else `unreadable`. [proof: mutation]
3. [S3] `harness_resolved` is the harness's `resolved`, written before the budget sets `solved` false; ADR-156's analysis keeps reading `solved`. [proof: mutation]
4. [S4] `grade` reads `tests_status.PASS_TO_PASS.failure` (swebench 5.0.2's report; the source file and line named in the code comment); any entry sets `broke_tests`. [proof: mutation]
5. [S5] `bench export-paired --pair X,Y --model M DIR...`. Line 1 is `{"config": {...}}` with every field ADR-106 Decision 2 names:
   - dataset, split and revision;
   - the selection rule and salt, the instance count and the instance list's SHA-256;
   - k, n as run, and the model id;
   - the CLI version, and the plugin's tag and commit;
   - the scaffold and its flags (the profile);
   - the prompt's and the status sentence's SHA-256;
   - the turn and time limits;
   - the git blob sha of ADR-106's scorer, `scripts/eval-paired.mjs`, from its pre-registration (`--scorer-sha`, required), with the exporter's own sha beside it;
   - the canary's result.

   Then one row per instance, arm and repeat of model M: `{instance, arm, repeat, status, resolved, brokeTests, limit, unrun}`. Each row is read from the last try. `resolved` is `harness_resolved`, `limit` is `over_budget` or `timed_out`, and `unrun` only when every try was excluded. [proof: mutation]
6. [S6] Wire it in `cmd/bench/main.go`'s usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestTheStatusLineIsReadFromTheFinalMessage|TestTheHarnessVerdictSurvivesTheBudget|TestABrokenPassToPassTestIsRecorded|TestExportPairedWritesADR106Rows)$' 2>&1) \
  && for t in TestTheStatusLineIsReadFromTheFinalMessage TestTheHarnessVerdictSurvivesTheBudget TestABrokenPassToPassTestIsRecorded TestExportPairedWritesADR106Rows; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'bench export-paired'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2) | none | — |

Added by this task, and rowed here when they exist:
- `TestTheStatusLineIsReadFromTheFinalMessage` (`internal/bench/runner_test.go`; S2): `done`, `not done: x`, no line, two lines, another value; the sentence appended only with `--status-line`, and its sha in `run.json`.
- `TestTheHarnessVerdictSurvivesTheBudget` (`internal/bench/runner_test.go`; S3): an over-budget attempt the harness resolved keeps `harness_resolved` true with `solved` false.
- `TestABrokenPassToPassTestIsRecorded` (`internal/bench/swebench_test.go`; S4): a synthetic report with a failing PASS_TO_PASS test sets `broke_tests`; one with none leaves it false.
- `TestExportPairedWritesADR106Rows` (`internal/bench/export_test.go`; S5): every Decision 2 field, with a missing one refused, and a missing `--scorer-sha` refused. One row per instance, arm and repeat, the retry chain collapsed. No row for the third configuration or the other model. A resolved over-budget row is `resolved: true, limit: true`.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--status-line` and `export-paired` in `cmd/bench/main.go`; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4, T5, and T6 hands the rows to quality-harness |

## Mutation Log
(empty until execute)

## Invariants

- Without `--status-line` the instruction is byte-identical to today's.
- `harness_resolved` never changes `solved`; ADR-156's verdicts read `solved` as before.

## Risks

- The sentence may change what an agent does; it is the same in all three configurations, and its sha is in `run.json` (ADR-106's own assumption).

## Stop Condition

The fence exits 0; or swebench 5.0.2's report does not carry PASS_TO_PASS failures, which goes back to Zy.

## Out of Scope

- ADR-106's false-completion test and guards (permanent: boundary: quality-harness's scorer computes them from these rows)

## Verification Log
(empty until execute)
