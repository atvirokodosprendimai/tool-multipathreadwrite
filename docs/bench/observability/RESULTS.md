# Observable costs and warnings — verification results

**Implementation:** accepted and locally verified; fresh code review approved. Owner accepted implementation on
2026-10-30. [PREREGISTRATION.md](PREREGISTRATION.md) was written before these observations.
The tool is [scripts/mrw_observe.py](../../../scripts/mrw_observe.py), with focused checks in
[scripts/test_mrw_observe.py](../../../scripts/test_mrw_observe.py).

## What ran and what passed

- Python 3.15.2 on macOS arm64, 10 logical CPUs. Installed MRW v1.79.0; executable and script hashes,
  source revision/dirty state, environment and load are recorded in the JSON metadata.
- Offline native reporting exited 0: 36 groups; all 60 unsuccessful original samples retained, not
  dropped from the distributions. Legacy stderr bytes were not recorded and are shown as null, not zero.
  These are descriptive additions to the original data, not new trials.
- Cost collection exited 0: complete, 63 timed samples and 21 discarded warm-ups. Every child passed,
  written contents matched, and scoped/full checker coverage matched the registered sizes.
  Load before/after was about 2.30/2.43; free memory before was 27%. No build, suite, service or container ran.
- A corrupt-file control passed: scoped check exited 0 for the clean edited file; full check exited 3
  with checker exit 1 for the corrupt untouched file. This proves the final verifier can reject bad data.
- Warning collection completed and exited 1 for its explicit quality failure: two false positives,
  two true positives, four true negatives, zero false negatives. All edits and receipt evidence verified.
- Thirteen focused unit tests pass. The initial test run failed with the tool absent. Mutants replacing p95 with p50
  and the actual nonblank distance with a constant both exited 1; the restored tests exited 0.
  Two additional boundary checks cover cancellation/timeout output and expired budgets. A real CLI
  exhausted-budget probe exits 1 with incomplete JSON and no timed samples.

The observed collector's exact source is retained as
[collector-555a36b8.py.txt](collector-555a36b8.py.txt), matching the script hash in the cost/warning
metadata. Cancellation handling was subsequently hardened and covered by two unit tests. A separate regression
corrected unknown legacy stderr reporting; successful live command timing and fixtures were unchanged,
and timings were not re-taken to select a better result.

Workflow structural validation passes (`actionlint -shellcheck=`). Full actionlint with ShellCheck
reports four SC2086 findings in the pre-existing shard script, identical in HEAD; no whole-workflow
lint pass is claimed. Scoped ruff and the existing prose gate pass.

## Cost observations (milliseconds, nearest-rank p50)

Three samples per cell verify the harness; they do **not** establish a performance verdict. The synthetic
checker is deliberately cheap and is not a substitute for a real repository suite. Output and command
CPU, p25/p50/p75/p90/p95/max, sample/failure counts and scaling ratios are retained in the raw JSON.
p95 at n=3 is simply the maximum. No number below is an internal lock, validation or commit timer.

| timed mode | 1 file | 10 files | 100 files |
|---|---:|---:|---:|
| read | 9.939 | 10.174 | 14.330 |
| write_no_check | 12.211 | 14.902 | 43.121 |
| write_scoped | 74.701 | 85.657 | 118.423 |
| write_full | 76.498 | 84.114 | 104.855 |
| check_scoped | 67.196 | 68.450 | 81.919 |
| check_full | 70.182 | 68.612 | 70.183 |
| queue_full | 83.324 | 191.182 | 1178.656 |

A no-check write's first table cell excludes its later verification. The comparable workflows below
include the full check that covers all edits; queue_full also includes each one-file write. Licensing
and restoration are untimed fixture preparation. The extra final full check on the scoped arm makes
coverage equivalent here; it is not a claim that every real project requires both checks.

| workflow including final full verification | 1 file | 10 files | 100 files |
|---|---:|---:|---:|
| write_no_check | 80.837 | 83.184 | 116.491 |
| write_scoped | 142.027 | 155.259 | 185.147 |
| write_full | 76.498 | 84.114 | 104.855 |
| queue_full | 83.324 | 191.182 | 1178.656 |

The 100-write queue costs more than one 100-file batch in this synthetic fixture. This is descriptive
native process evidence, not evidence of model-token savings, a real project's test cost, or better task
outcomes. The separate paired agent assessment remains necessary. No optimization was made.

## Warning evidence

| fixture | expected hint | actual hint | matched line | nonblank distance |
|---|---|---|---:|---:|
| sibling_spec.rb | False | True | 10 | 3 |
| duplicate.md | True | True | 5 | 1 |
| statement.src | False | False | None | None |
| nested.go | False | True | 5 | 1 |
| blanks.md | True | True | 7 | 1 |
| no_repeat.src | False | False | None | None |
| own_closer.src | False | False | None | None |
| outside_window.src | False | False | None | None |

The sibling expectation's `)` and nested function's `}` belong to valid structures, yet the hint reports
them. Both genuine duplicate-fence cases still warn. The report validates the match actually emitted by
MRW; it does not run a competing heuristic. An absent warning says `not_reported`, without inventing a
reason. This tiny synthetic set does not estimate a real-world false-positive rate or meet ADR-119's bar.
The [existing obligation](../../adr/BACKLOG.md#from-adr-119-sibling-closer-false-positive) stays open.

## Reproduce

From the repository root, with Python 3 and the pinned binary available:

```sh
python3 -m unittest discover -s scripts -p test_mrw_observe.py -v
python3 scripts/mrw_observe.py native docs/bench/native-baseline/raw/run2/samples.jsonl
python3 scripts/mrw_observe.py warnings --mrw /path/to/pinned/mrw
python3 scripts/mrw_observe.py costs --mrw /path/to/pinned/mrw
```

`native` only reads existing samples. Live collection currently requires POSIX. Before collecting costs,
follow costly-runs, honor the registered load/memory/peer-build gates and keep the binary pinned.
The collector caps the budget at 120 seconds. Interrupted/incomplete collections preserve completed samples
in their JSON output and exit nonzero. A warning-quality mismatch also exits nonzero, distinct from the
`status` field reporting complete collection. Fixture files and their private ledger are removed.
The tool writes JSON to stdout; keep that output rather than piping it to a truncating consumer.

## Evidence and limits

- [native.json](native.json): descriptive re-analysis with the original input SHA256.
- [costs.json](costs.json): every timed command, full verification, warm-up and aggregate, including failures.
- [warnings.json](warnings.json): all eight receipts' evidence and the confusion counts.
- [mutations.json](mutations.json): observed mutant and restored-test exit codes/output; personal repository
  paths normalized to `<repo>`.
- [faults.json](faults.json): real CLI exhausted-budget refusal and scoped/full corrupt-file control.

These are experimental report shapes, not additions to MRW's public receipt contract. No Windows/Linux
live collection, real-repository performance campaign, history replay, model task run or shipped warning
fix is claimed. No private project names or source fixtures are included.

For the subsequent final-check evidence pilot, see [VERIFICATION.md](VERIFICATION.md).
The [Claude smoke pilot](CLAUDE-RESULTS.md) exercises that evidence path on two paired synthetic tasks.
The [grouped Sonnet pilot](GROUPED-RESULTS.md) compares fifty edit sites per arm and retains its contrary result.
The [trace diagnosis](TRACE-DIAGNOSIS.md) reconciles both pilots' token costs and distinguishes turns from messages.
