# Task ADR-174-T1: `--then-sh` is refused unless `MRW_THEN_SH=on`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the refusal in `askedStepsError`; `MRW_THEN_SH`; contract §299; the variable in every test, row and stress script that drives `--then-sh`; the words on every surface
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the cmd/mrw package runs`, `the off test passed`, `the placeholder test still passes`, `the step runner test passes`, `the check-json stress script enables it`, `the write-flags stress script enables it`, `the contract run`, `section 299 exists`, `the off row passes`, `the on row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1–5: an ad-hoc step runs only when `MRW_THEN_SH` is exactly `on`; otherwise `write` and `check` refuse, exit 2, before anything is written or run, after today's depth, empty-command and placeholder refusals.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/thensh174_test.go` | new | the red test, driving the built binary |
| `cmd/mrw/main.go` | edit | `askedStepsError` (`:2239-2255`); the `--then-sh` usage text (`:2230`) |
| `internal/guide/guide.go` | edit | `ThenShCaveat` (`:139-141`) |
| `cmd/mrw/*_test.go` that run a `--then-sh` step (of the 14 files `grep -ln 'then-sh' cmd/mrw/*_test.go` lists) | edit | set `MRW_THEN_SH=on` in that command's environment only, never process-wide, so the off cases stay isolated |
| `scripts/contract.sh` | edit | export the variable for the existing `--then-sh` rows; §299 off and on |
| `docs/break/check-json/stress.py`, `docs/break/write-flags/stress.py` | edit | set the variable |
| `README.md`, `AGENTS.md`, `docs/agents.md` | edit | the off rule beside the flag |

## Ordered Steps

1. [S1] Write `TestThenShRunsOnlyWhenTheEnvironmentEnablesIt` and see it RED. With `MRW_THEN_SH` removed from the command's environment: `mrw write PLAN --then-sh 'touch marker'` exits 2, the target unchanged, no marker; `mrw write PLAN --then declared --then-sh 'touch marker'` and `--then-sh 'touch marker' --then declared` the same, the declared step not run either; the same three on `mrw check`; `mrw check --json --then-sh …` prints one `{"error": …}` document; values `yes`, `ON`, ` on` and `on ` refused like unset; at `MRW_STEP_DEPTH=8` the depth refusal answers, not the switch; a placeholder command still names its token. With `MRW_THEN_SH=on` the step runs and the marker exists; `--then NAME` alone runs with the variable unset.
2. [S2] The refusal in `askedStepsError`. Mutant: the environment check dropped. [proof: mutation]
3. [S3] Mutant: the switch looks only at the first requested step — the mixed-list case with `--then-sh` second must go red; and a mutant that trims or folds the value — the ` on`/`ON` cases must go red. [proof: mutation]
4. [S4] The variable in every test, row and stress script; §299; the words. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestThenShRunsOnlyWhenTheEnvironmentEnablesIt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten \(' <<<"$out" \
  && go test ./internal/check/ -count=1 -run '^TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass$' \
  && grep -qF 'MRW_THEN_SH' docs/break/check-json/stress.py && grep -qF 'MRW_THEN_SH' docs/break/write-flags/stress.py \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 299\.' scripts/contract.sh \
  && grep -qF 'PASS  --then-sh without MRW_THEN_SH=on is refused and runs nothing' <<<"$c" \
  && grep -qF 'PASS  --then-sh with MRW_THEN_SH=on runs its step' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestThenShRunsOnlyWhenTheEnvironmentEnablesIt` | `cmd/mrw/thensh174_test.go` | off by default on write and check, nothing written or run; only `on` enables; `--then NAME` unaffected | none | S1, S2, S3 |
| `TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten` | `cmd/mrw/then094_test.go` | ADR-094's refusals still fire with the variable on | none | S4 |
| `TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass` | `internal/check/steps092_test.go` | the step runner is untouched | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `askedStepsError`, called by `write` and `check` |
| 3 — the caller can discover it | the refusal names the variable; help, README, AGENTS.md |
| 4 — it is used | MRW-10 in the Rubie report shows `--then-sh` reached through an allowed `mrw` |

## Mutation Log
(empty until execute)

## Invariants

- A refused `--then-sh` reads, writes and runs nothing.
- `--then NAME` and the check are unchanged.

## Risks

- Missing the variable in one of the 15 test files shows as a red suite, not a silent pass.

## Stop Condition

The fence exits 0. Stop if a caller outside this repository is found in the tree that cannot set an environment variable (none is known).

## Out of Scope

- MCP (permanent: boundary: `mrw_write` has no `--then-sh`, `internal/mcp/mcp.go` says so)

## Verification Log
(empty until execute)
