# Task ADR-163-T2: configuration W, the declared check and the stage's caps

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration W, `bench/swe-bench/checks.json`, the declared check written before the base commit, `bench run --max-turns`, `run.json`'s `work_line_sha256`, `checks_sha256` and `max_turns`, the same three inside the rows' `scaffold`
**Consumes:** `--profile stage6` and the shims (T1)
**Data dependency:** hermetic; the checks table is read from the pool's repository clones at the commits it names, never from the dataset
**Proof map:** v1
**Rests-on:** `W is P plus the pre-registered line`, `every arm gets the same check file, outside the graded diff`, `the check is the repository's whole runner and nothing from the dataset`, `the turn cap is the stage's and the scorer sees it`

## Goal

Under `stage6`: configuration W is P plus one pre-registered prompt line; every arm's tree carries the same `.quality-harness.json` whose `check` is the whole runner the repository's own files document; and the stage sets its own caps, which the rows carry where quality-harness's scorer compares them (ADR-163 Decisions 2 and 3).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["W"]`; W's line read from `--work-line FILE`, its SHA-256 in `run.json`; `MaxTurns` becomes `RunOptions.MaxTurns`, default 60, in `run.json` |
| `internal/bench/export.go` | edit | `max_turns` from the run; `checks_sha256`, `work_line_sha256` and `max_turns` in the per-block equality check and inside `scaffold` |
| `internal/bench/materialize.go` | edit | under `stage6`, write `.quality-harness.json` with the repository's check before the base commit |
| `internal/bench/checks.go` | add | load `checks.json`; refuse a pool repository it does not name, an entry without a source, or a command with path or subset arguments |
| `bench/swe-bench/checks.json` | add | one whole runner per pool repository, with the file, line and commit it came from |
| `internal/bench/canary.go` | edit | W engaged as P; the line in W's prompt and in no other configuration's |
| `internal/bench/checks_test.go` | add | the tests below |
| `cmd/bench/main.go` | edit | `W` in `--configs`, `--work-line`, `--checks`, `--max-turns`; the usage text |
| `docs/bench/README.md` | edit | W, the declared check and the caps |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] `checks.json`: for each of the pool's 11 repositories, read its own files at the commit named (`tox.ini`, `setup.cfg`, `pytest.ini`, `pyproject.toml`, its contributing docs, `tests/runtests.py`, `bin/test`) and record its whole runner, with no path or subset arguments, and `source: <file>:<line>@<commit>`. Nothing is read from the dataset. Write its SHA-256 into BACKLOG's "Pre-registered for ADR-163" before T4's baseline pilot. [proof: human: the executor records each entry's source line, the SHA-256 as registered, and that no entry was taken from FAIL_TO_PASS, PASS_TO_PASS or a patch]
3. [S3] `Materialize` writes the file before the base commit in every arm under `stage6`; W is P plus the line from `--work-line`; `--max-turns` reaches the over-budget rule, `run.json`, the rows and their `scaffold`; the export refuses blocks whose `checks_sha256`, `work_line_sha256` or `max_turns` differ. [proof: mutation]
4. [S4] Wire the flags, W in `--configs` and the canary's W checks; the usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationWIsPPlusTheWorkLine|TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm|TestTheChecksTableNamesARepositorySourceForEveryPoolRepository|TestMaxTurnsIsTheStagesAndTheRowsCarryIt)$' 2>&1) \
  && for t in TestConfigurationWIsPPlusTheWorkLine TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm TestTheChecksTableNamesARepositorySourceForEveryPoolRepository TestMaxTurnsIsTheStagesAndTheRowsCarryIt; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--max-turns' \
  && jq -e 'length == 11 and ([.[] | select((.command | type) != "string" or (.source | test("^[^:]+:[0-9]+@[0-9a-f]{40}$") | not))] | length == 0)' bench/swe-bench/checks.json > /dev/null
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2); the rows below replace it, with their Steps, in the commit that adds the tests | none | — |

Planned tests, which this table names in the commit that adds them:

- TestConfigurationWIsPPlusTheWorkLine in internal/bench/checks_test.go: W's arguments are P's; its instruction is P's plus the line, whose sha is in `run.json`; A and P carry no line; W without `--work-line` or without `stage6` is refused; the canary refuses the line in P (steps S3)
- TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm in internal/bench/checks_test.go: under `stage6`, A, P and W trees hold the same `.quality-harness.json` in the base commit and not in the diff; without the profile no file is written (steps S3)
- TestTheChecksTableNamesARepositorySourceForEveryPoolRepository in internal/bench/checks_test.go: every repository of `selection-adr-162.json` has a command and a source; a missing repository, a missing source, or a command with a path or `-k` argument is refused (steps S2, S3)
- TestMaxTurnsIsTheStagesAndTheRowsCarryIt in internal/bench/checks_test.go: `--max-turns 90` makes 61 to 90 turns within budget and is in `run.json`, the rows and `scaffold`; without the flag it is 60; two blocks with different `checks_sha256`, `work_line_sha256` or `max_turns` are refused by the export (steps S3)

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--configs …,W`, `--work-line`, `--checks`, `--max-turns` in `cmd/bench/main.go` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4 and T5 |

## Mutation Log
(empty until execute)

## Invariants

- Without `stage6`, no check file is written and `MaxTurns` is 60, so Stage 5's rows and verdicts are unchanged.
- `checks.json` never reads the dataset, and is frozen before any baseline.

## Risks

- A repository may document no single whole runner; S2 records the closest its own files name, with its source, and Zy sees it in the pilot's sign-off.

## Stop Condition

The fence exits 0 and S2 is recorded; or a pool repository's files document no runnable test command, which goes back to Zy.

## Out of Scope

- The scoped check form (external: quality-harness: the scoped check ADR-111 defers; ADR-163 is amended to use it)

## Verification Log
(empty until execute)
