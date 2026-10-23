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
| `TestConfigurationWIsPPlusTheWorkLine` | `internal/bench/checks_test.go` | W's arguments are P's; its instruction is P's plus the line, whose sha is in `run.json`; A and P carry no line; W without `--work-line` or without `stage6` is refused; the canary refuses the line in P | none | S3 |
| `TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm` | `internal/bench/checks_test.go` | under `stage6`, A, P and W trees hold the same `.quality-harness.json` in the base commit and not in the diff; without the profile no file is written | none | S3 |
| `TestTheChecksTableNamesARepositorySourceForEveryPoolRepository` | `internal/bench/checks_test.go` | every repository of `selection-adr-162.json` has a command and a source; a missing repository, a missing source, or a command with a path or `-k` argument is refused | none | S2, S3 |
| `TestMaxTurnsIsTheStagesAndTheRowsCarryIt` | `internal/bench/checks_test.go` | `--max-turns 90` makes 61 to 90 turns within budget and is in `run.json`, the rows and `scaffold`; without the flag it is 60; blocks with different `checks_sha256`, `work_line_sha256` or `max_turns` are refused by the export | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--configs …,W`, `--work-line`, `--checks`, `--max-turns` in `cmd/bench/main.go` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4 and T5 |

## Mutation Log
(empty until execute)
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/materialize.go` · the declared check is never written into the tree before the base commit · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/runner.go` · --max-turns does not reach the over-budget rule · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/export.go` · the export accepts blocks whose checks_sha256 differ · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/checks.go` · configuration W carries no work line: W is P · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/export.go` · the export accepts blocks whose work_line_sha256 differ · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/checks.go` · a command naming a test path, node id or -k subset is accepted as the whole runner · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/checks.go` · a pool repository the table does not name is accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/runner.go` · run.json carries no work_line_sha256 · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/checks.go` · every configuration carries the work line: A and P become W · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · mutant killed · exit 1 · `internal/bench/checks.go` · a -k subset is accepted as the whole runner · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 6c1fdbd7* · mutant killed · exit 1 · `internal/bench/checks.go` · a quoted -k is not seen as -k · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 6c1fdbd7* · mutant killed · exit 1 · `internal/bench/checks.go` · options set before the runner (PYTEST_ADDOPTS) are accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 6c1fdbd7* · mutant killed · exit 1 · `internal/bench/checks.go` · a test file named after the runner is accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 6c1fdbd7* · mutant killed · exit 1 · `internal/bench/materialize.go` · a check file is written without stage6 · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · fe76d09d* · mutant killed · exit 1 · `internal/bench/checks.go` · a backslashed -k is not seen as -k · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · fe76d09d* · mutant killed · exit 1 · `internal/bench/checks.go` · a -m marker selection is accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · fe76d09d* · mutant killed · exit 1 · `internal/bench/checks.go` · pytest selection options (--keyword, --lf, --deselect) are accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · fe76d09d* · mutant killed · exit 1 · `internal/bench/checks.go` · a variable set through env is accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 772448e9* · mutant killed · exit 1 · `internal/bench/checks.go` · --stepwise-skip, which enables --stepwise, is accepted · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93

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
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6057
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6026
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6050
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6399
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6070
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6011
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:5908
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6043
- 2026-10-23 · human-observed · S2 recorded 2026-10-23: bench/swe-bench/checks.json holds one entry per pool repository (11), each read from the repository's own files fetched from GitHub at the newest pool base commit (by instance created_at, base_commit from the dataset row only) and checked at the oldest; sources astropy setup.cfg:120, django tox.ini:35, matplotlib pytest.ini:10, seaborn Makefile:4, requests Makefile:9, xarray setup.cfg:139, pylint tox.ini:37, pytest tox.ini:28, scikit-learn Makefile:27, sphinx tox.ini:27, sympy bin/test:4, each @ its 40-hex newest base. Runners that named their suite root (sklearn, matplotlib, pylint, seaborn, requests) are recorded bare per Zy 2026-10-23 ('Bare runner'); differences at the oldest base are noted per entry ('Newest, note the gap'). No entry was taken from FAIL_TO_PASS, PASS_TO_PASS or a patch. SHA-256 7f169124fbdcdc8e5fa118614aeb275122fc7947fdd7034570141c3c93fb71a7, registered in BACKLOG 'Pre-registered for ADR-163' · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6128
- 2026-10-23 · 1ded736c* · exit 1 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:4055 · test-lock-sha256:703aeab0d90d0cbb5d6a264d98ef8b95ff837d3f3f9b955860eb2169d1893f7b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25XSXNQUGx1c1RoZVdvcmtMaW5lCTI3M2Y0MjFiYmU0NDI4YTBkZGFmYzAxNzMzZDIxODA3YjIyY2ZlYmY3NTNmZTFiYzY3NzFlOWM3MWQyYzYxNDAKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0TWF4VHVybnNJc1RoZVN0YWdlc0FuZFRoZVJvd3NDYXJyeUl0CTYyYzc4YzAxMDhhZDc3Yzc3ODBiODNjN2QyMzIwMWIzMmM2MDU3Y2Y2NmYxNDQwN2M0ZDEyODJjOTcyYzkzZDEKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0VGhlQ2hlY2tzVGFibGVOYW1lc0FSZXBvc2l0b3J5U291cmNlRm9yRXZlcnlQb29sUmVwb3NpdG9yeQk0NWYwYzRiMmYwODgwNWRmM2FmYzI5ZDA1YzQ1NGZhOWFkZTY3Y2U3NWU5OGQzMjUzMjA1MTBjZGRlZDlmNWM4CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdFRoZURlY2xhcmVkQ2hlY2tJc1dyaXR0ZW5CZWZvcmVUaGVCYXNlQ29tbWl0SW5FdmVyeUFybQk4ZmUwNjNlZjQ0YzA2NTRmMGQ5MTFmNDEyZTlmZDcwNjJkNTJiMWIxMjU4MjY2ZTdjZDM2Y2MxOGRmMDY4NTY5
  ```
  ```
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:5888
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6387
- 2026-10-23 · 1ded736c* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6129
- 2026-10-23 · 6c1fdbd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6986
- 2026-10-23 · 6c1fdbd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6387
- 2026-10-23 · 6c1fdbd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6546
- 2026-10-23 · 6c1fdbd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6352
- 2026-10-23 · 6c1fdbd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:5661
- 2026-10-23 · 6c1fdbd7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:0 · test-lock-sha256:0a990e45e62df648fdbb9dbff40cea1537dd083e2a38b551487089f5023d1888 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25XSXNQUGx1c1RoZVdvcmtMaW5lCTI3M2Y0MjFiYmU0NDI4YTBkZGFmYzAxNzMzZDIxODA3YjIyY2ZlYmY3NTNmZTFiYzY3NzFlOWM3MWQyYzYxNDAKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0TWF4VHVybnNJc1RoZVN0YWdlc0FuZFRoZVJvd3NDYXJyeUl0CTYyYzc4YzAxMDhhZDc3Yzc3ODBiODNjN2QyMzIwMWIzMmM2MDU3Y2Y2NmYxNDQwN2M0ZDEyODJjOTcyYzkzZDEKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0VGhlQ2hlY2tzVGFibGVOYW1lc0FSZXBvc2l0b3J5U291cmNlRm9yRXZlcnlQb29sUmVwb3NpdG9yeQkyN2IyMzNiZWU5MmI0ZjY4OWEwODljZTAzOWVmY2FiOTNjNjQ3MTE2NzRiNGE2OTYzYzk0MWEzYmQwMTUwN2UxCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdFRoZURlY2xhcmVkQ2hlY2tJc1dyaXR0ZW5CZWZvcmVUaGVCYXNlQ29tbWl0SW5FdmVyeUFybQk0ZjQ0MWY0YzM4Y2MxYWI0MDVlOTFmNDE3NTcxMzZkMGQ3NzMzZWI4OGI5Y2QyZmIxMTE4ODAzNjcyZThjMWRm · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed, after Codex on #513: TestTheChecksTableNamesARepositorySourceForEveryPoolRepository gains three refusals (a quoted -k, PYTEST_ADDOPTS set before the runner, a test file named after pytest) for wholeRunner, which now reads words with their quotes removed, refuses a variable set before the runner and a .py operand other than python's script; TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm's no-profile case now asserts the file is absent from the tree (test -e) rather than excusing any git fatal. No assertion removed or weakened; approved · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · fe76d09d* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:7065
- 2026-10-23 · fe76d09d* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:5927
- 2026-10-23 · fe76d09d* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6063
- 2026-10-23 · fe76d09d* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:5921
- 2026-10-23 · fe76d09d* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:0 · test-lock-sha256:d5aa304c753b3735e83d2ff443f3743d76fdae988fb2123c41a96aa35cca8435 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25XSXNQUGx1c1RoZVdvcmtMaW5lCTI3M2Y0MjFiYmU0NDI4YTBkZGFmYzAxNzMzZDIxODA3YjIyY2ZlYmY3NTNmZTFiYzY3NzFlOWM3MWQyYzYxNDAKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0TWF4VHVybnNJc1RoZVN0YWdlc0FuZFRoZVJvd3NDYXJyeUl0CTYyYzc4YzAxMDhhZDc3Yzc3ODBiODNjN2QyMzIwMWIzMmM2MDU3Y2Y2NmYxNDQwN2M0ZDEyODJjOTcyYzkzZDEKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0VGhlQ2hlY2tzVGFibGVOYW1lc0FSZXBvc2l0b3J5U291cmNlRm9yRXZlcnlQb29sUmVwb3NpdG9yeQkyZjYyMWRjYTkzMzZiOGExYzRlMzBkMGRmZmYyYjk1Yzg5MzYyYTE2NjQyOWIxN2U3ZmI3ZjE2MDhhNDdkNjA1CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdFRoZURlY2xhcmVkQ2hlY2tJc1dyaXR0ZW5CZWZvcmVUaGVCYXNlQ29tbWl0SW5FdmVyeUFybQk0ZjQ0MWY0YzM4Y2MxYWI0MDVlOTFmNDE3NTcxMzZkMGQ3NzMzZWI4OGI5Y2QyZmIxMTE4ODAzNjcyZThjMWRm · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed, after Codex round 2 on #513: TestTheChecksTableNamesARepositorySourceForEveryPoolRepository gains six refusals (a backslashed -k, --keyword, -m slow, env PYTEST_ADDOPTS=, --lf, --deselect=); wholeRunner strips backslashes, refuses a variable anywhere, -m other than -m pytest, and pytest's selection options by name, and says it is a guard beside a person's review of a registered table. No assertion removed or weakened; approved · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · 772448e9* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6393
- 2026-10-23 · 772448e9* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:0 · test-lock-sha256:1ff356b6cf662c469f682e30e7afb9b213e1b9321fa729e9d23bc3fd53493817 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25XSXNQUGx1c1RoZVdvcmtMaW5lCTI3M2Y0MjFiYmU0NDI4YTBkZGFmYzAxNzMzZDIxODA3YjIyY2ZlYmY3NTNmZTFiYzY3NzFlOWM3MWQyYzYxNDAKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0TWF4VHVybnNJc1RoZVN0YWdlc0FuZFRoZVJvd3NDYXJyeUl0CTYyYzc4YzAxMDhhZDc3Yzc3ODBiODNjN2QyMzIwMWIzMmM2MDU3Y2Y2NmYxNDQwN2M0ZDEyODJjOTcyYzkzZDEKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0VGhlQ2hlY2tzVGFibGVOYW1lc0FSZXBvc2l0b3J5U291cmNlRm9yRXZlcnlQb29sUmVwb3NpdG9yeQk4YmRhMjg1MTI1MTQ1MjI5Y2NhNDRmNzQ1ZTllNWRiNGZjMzNjNjBlZDc4ZTAyYjE4ODIyN2JiMGFhN2MxYmQ4CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdFRoZURlY2xhcmVkQ2hlY2tJc1dyaXR0ZW5CZWZvcmVUaGVCYXNlQ29tbWl0SW5FdmVyeUFybQk0ZjQ0MWY0YzM4Y2MxYWI0MDVlOTFmNDE3NTcxMzZkMGQ3NzMzZWI4OGI5Y2QyZmIxMTE4ODAzNjcyZThjMWRm · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed, after Codex round 3 on #513: TestTheChecksTableNamesARepositorySourceForEveryPoolRepository gains the --stepwise-skip refusal (pytest documents it enables --stepwise); --sw-skip and --stepwise-skip join the selectors. No assertion removed or weakened; approved · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93
- 2026-10-23 · e1ef1b00* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:0 · test-lock-sha256:9cf114cc8fa933d6e8acd5aa924a31bf6c8a3916098a3111515fa9f06976cae5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25XSXNQUGx1c1RoZVdvcmtMaW5lCTk5NGFlODI4NmU2YjVjYmY5NWZhNWMzZWEyZTc3OTEzNTkwODI3ZTkxYjA2MjQzZjU4YWNjYTg4MTdmZGFhYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0TWF4VHVybnNJc1RoZVN0YWdlc0FuZFRoZVJvd3NDYXJyeUl0CWJjNzllOWY3NjNmN2RmMDg3ZjhhY2Y2NmVjYTE0NjQ2Y2YyNDQ0NjE2NTA0YzFjODg0YWI3MjE0MTFhNjcwZmEKYm9keQlpbnRlcm5hbC9iZW5jaC9jaGVja3NfdGVzdC5nbwlUZXN0VGhlQ2hlY2tzVGFibGVOYW1lc0FSZXBvc2l0b3J5U291cmNlRm9yRXZlcnlQb29sUmVwb3NpdG9yeQk4YmRhMjg1MTI1MTQ1MjI5Y2NhNDRmNzQ1ZTllNWRiNGZjMzNjNjBlZDc4ZTAyYjE4ODIyN2JiMGFhN2MxYmQ4CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2hlY2tzX3Rlc3QuZ28JVGVzdFRoZURlY2xhcmVkQ2hlY2tJc1dyaXR0ZW5CZWZvcmVUaGVCYXNlQ29tbWl0SW5FdmVyeUFybQk0ZjQ0MWY0YzM4Y2MxYWI0MDVlOTFmNDE3NTcxMzZkMGQ3NzMzZWI4OGI5Y2QyZmIxMTE4ODAzNjcyZThjMWRm · test-lock-kind:replace
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:e562e6b3b419db3340ab47782fd30600dae327da78c4af144dfed0874f210a93 · ms:6048
