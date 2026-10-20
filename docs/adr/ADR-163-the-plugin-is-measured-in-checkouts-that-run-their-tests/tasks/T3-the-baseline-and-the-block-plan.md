# Task ADR-163-T3: the baseline, the surviving pool and the block plan

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench baseline`, `baseline.json`, the surviving pool as a selection-shaped file (`bench baseline --surviving FILE`), blocks drawn from it, `export-paired --pair A,W` against it
**Consumes:** the container and the shims (T1), the declared check, `checks.json` and the bound export fields (T2)
**Data dependency:** hermetic tests against a fake `docker`
**Proof map:** v1
**Rests-on:** `a baseline runs the declared check on the unmodified tree, with no model`, `the surviving pool is the green instances in pool order, frozen`, `a block is a slice of the surviving pool`, `the rows export for W against the surviving pool`, `a capped run exports unresolved with no done claim`

## Goal

`bench baseline` records, per instance, whether its declared check is green on the unmodified tree, and writes the green ones in ADR-162's pool order as the surviving pool. Blocks are 40-instance slices of it, and `export-paired` exports against it (ADR-163 Decisions 4 and 5).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/baseline.go` | add | per instance: pull the image if absent, start the container (T1), run the declared check once, record green or red, exit code and wall time; drop the image as the runner does; `--surviving FILE` writes the green ids in pool order as a selection |
| `internal/bench/runner.go` | edit | `--block K` over a selection whose ids are the surviving pool; the last block may be partial |
| `internal/bench/export.go` | edit | `instance_list_sha256` and `instance_count` from the selection given, its `.selected` ids joined by a newline with no trailing newline, so the surviving pool's digest is what the rows carry; under `stage6` a capped run is written `not done`, unresolved, `limit` true |
| `internal/bench/baseline_test.go` | add | the tests below |
| `cmd/bench/main.go` | edit | `bench baseline`; the usage text |
| `docs/bench/README.md` | edit | the baseline, the surviving pool and its blocks |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] `bench baseline --stage S --out DIR --selection FILE --checks FILE [--surviving FILE]`: one container per instance, the check on the unmodified tree once, `baseline.json` with `{instance, green, exit_code, wall_ms}` per instance; a check that cannot start is red, with its reason; `--surviving` writes the green ids in the selection's order, with the rule, the source selection's sha and the red count. [proof: mutation]
3. [S3] `bench run --selection selection-adr-163.json --block K` runs instances 40(K−1)+1 to 40K, the last block partial; `export-paired` takes the surviving pool's digest (the scorer's convention) and count from the selection, and under `stage6` writes a capped run as unresolved with no done claim. [proof: mutation]
4. [S4] `export-paired --pair A,W` is tested; the usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestTheBaselineRecordsGreenRedAndTime|TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder|TestExportPairedTakesAWAgainstTheSurvivingPool|TestACappedRunExportsUnresolvedWithNoDoneClaim)$' 2>&1) \
  && for t in TestTheBaselineRecordsGreenRedAndTime TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder TestExportPairedTakesAWAgainstTheSurvivingPool TestACappedRunExportsUnresolvedWithNoDoneClaim; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'bench baseline'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2); the rows below replace it, with their Steps, in the commit that adds the tests | none | — |

Planned tests, which this table names in the commit that adds them:

- TestTheBaselineRecordsGreenRedAndTime in internal/bench/baseline_test.go: a check exiting 0 is green, non-zero red, one that cannot start red with its reason; the wall time is recorded; no model is invoked; each instance runs once (steps S2)
- TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder in internal/bench/baseline_test.go: with reds interleaved, the surviving file holds exactly the greens in pool order with the red count; blocks 1 and 2 are its first and second 40, a last block of fewer is accepted (steps S2, S3)
- TestExportPairedTakesAWAgainstTheSurvivingPool in internal/bench/baseline_test.go: `--pair A,W` writes arms A and W; the config line's `instance_list_sha256` equals the sha256 of the surviving `.selected` ids joined by a newline with no trailing newline, and its `instance_count` their number, not ADR-162's 191 (steps S3, S4)
- TestACappedRunExportsUnresolvedWithNoDoneClaim in internal/bench/baseline_test.go: under `stage6`, a receipt over its cap with `STATUS: done` and `harness_resolved` true exports `not done`, unresolved, `limit` true; under `stage5` the row is today's (steps S3)

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `bench baseline` in `cmd/bench/main.go`; `run --selection` with the surviving file |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's baselines and T5's blocks |

## Mutation Log
(empty until execute)

## Invariants

- A baseline invokes no model.
- No instance enters the surviving pool, or a block, by any rule but ADR-163 Decision 4's.

## Risks

- A flaky check is green on one run and red on another; one run decides, and T4 reports instances whose check took over a quarter of the budget.

## Stop Condition

The fence exits 0.

## Out of Scope

- Re-running a red baseline to make it green (permanent: boundary: one run decides, so no instance is chosen for an outcome)

## Verification Log
(empty until execute)
