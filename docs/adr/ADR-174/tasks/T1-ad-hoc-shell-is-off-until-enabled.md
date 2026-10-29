# Task ADR-174-T1: ad-hoc shell (`--then-sh` and `mrw run`) is refused unless `MRW_SHELL=on`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the `MRW_SHELL` helper and its two refusals (`askedStepsError`, `runAction`); contract §319; the variable in every consumer the record's Context lists; the words on every surface
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the cmd/mrw package runs`, `the off test passed`, `the placeholder test still passes`, `the run depth test still passes`, `the guide test passes`, `the step runner test passes`, `the check-json stress script enables it`, `the write-flags stress script enables it`, `the Windows smoke enables it`, `the contract run`, `section 319 exists`, `the then-sh off row passes`, `the run off row passes`, `the on row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1–6: an ad-hoc shell command — a `--then-sh` step on `write` or `check`, or `mrw run` — runs only when `MRW_SHELL` is exactly `on`; otherwise mrw refuses, exit 2, before anything is written or run, after today's depth, empty-command, placeholder and usage refusals; the declared check and `--then NAME` are unchanged; every consumer in this repository sets the variable; every surface says what the switch does and does not buy.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/shell174_test.go` | new | the red test |
| `cmd/mrw/main.go` | edit | the helper that reads `MRW_SHELL`; `askedStepsError` (`:2605-2621`); the `--then-sh` usage text (`thenShFlag`, `:2554-2560`) |
| `cmd/mrw/run177.go` | edit | `runAction` (`:85-113`): the refusal after the argument count, before `state.Base()`; the `run` help (`:62-80`) |
| `internal/guide/guide.go` | edit | `ThenShCaveat` (`:159-163`), embedded at `:54`, `:66`, `:106` |
| `internal/guide/run177_test.go`, `cmd/mrw/teach092_test.go`, `cmd/mrw/teach095_test.go`, `cmd/mrw/run177_test.go` (`TestRunHelpNamesTheShellTrust`) | edit | the caveat's expected text, where a test pins it |
| `cmd/mrw/*_test.go` that run a `--then-sh` step or `mrw run` — of the 17 `git grep -l -e 'then-sh' -- cmd/mrw` lists and the 6 that invoke `run` (record, Context) | edit | `t.Setenv("MRW_SHELL", "on")` in the tests that run one; a test that only parses or refuses the flag before the switch needs nothing |
| `cmd/mrw/testdata/outcomes113.golden` | regenerate only if its `--then-sh` outcomes change | they should not once `outcomes113_test.go` sets the variable |
| `scripts/contract.sh` | edit | `export MRW_SHELL=on` in the prologue (beside the other exports, before `m()` at `:176`); §319 off and on |
| `docs/break/check-json/stress.py` (`base_env`, `:58-61`), `docs/break/write-flags/stress.py` (`:142`) | edit | set `MRW_SHELL=on` in the environment they pass |
| `.github/workflows/release-smoke-windows.yml` | edit | `MRW_SHELL: "on"` in the `smoke` job's `env` (six spaces deep), which reaches all four shells that run `mrw run`: bash `:79`, pwsh 7 `:113`, Windows PowerShell 5.1 (`run: *ps`, `:124`) and cmd `:151` |
| `README.md` (`:133`, `:157-162`, `:384-392`, `:528`, `:676-702`), `AGENTS.md` (`:105`, `:607-623`, `:691-717`), `docs/agents.md` (`:123`, §7 `:156-170`) | edit | the off rule beside each surface, and Decision 6's sentence in "What allowing mrw allows" and §7 |
| `.claude/skills/mrw/SKILL.md` | regenerate | `mrw skill --dir .claude/skills` after the caveat changes |
| `docs/adr/locks/*.lock` and the Verification Logs that hold them | relock | every record `adr-lint` names after the test edits |

## Ordered Steps

1. [S1] Write `TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt` and see it RED. With `t.Setenv("MRW_SHELL", "")`: `mrw write PLAN --then-sh 'touch marker'` exits 2, the target unchanged, no marker, the message naming `MRW_SHELL=on` and `--then NAME`; `mrw write PLAN --then declared --then-sh 'touch marker'` and `--then-sh 'touch marker' --then declared` the same, the declared step not run either; the same three on `mrw check`; `mrw check --json --then-sh …` prints one `{"error": …}` document; `mrw run 'touch marker'` exits 2, no marker, no `<state home>/mrw-runs/` directory when there was none before, the message naming `MRW_SHELL=on`; `mrw run --json 'touch marker'` prints one `{"error": …}` document; values `yes`, `ON`, ` on` and `on ` refused like unset, each message naming the value; at `MRW_STEP_DEPTH=8` the depth refusal answers, not the switch, on `write`, `check` and `run`; a placeholder command still names its token; `mrw run` with two arguments still answers its usage refusal. With `t.Setenv("MRW_SHELL", "on")` the step runs and `mrw run` runs, each marker exists. With the variable unset, `--then NAME` alone runs and a code write runs the declared check.
2. [S2] The helper, and the refusals in `askedStepsError` and `runAction`. Mutant: the environment check dropped from `askedStepsError` — the write and check off cases go red. Mutant: dropped from `runAction` — the run off cases go red. [proof: mutation]
3. [S3] Mutant: the switch looks only at the first requested step — the mixed-list case with `--then-sh` second must go red. Mutant: the value trimmed or case-folded — the ` on` and `ON` cases must go red. Mutant: in `runAction` the switch moved after `state.Base()` and `MkdirAll` — the no-log-directory assertion must go red. [proof: mutation]
4. [S4] The variable in every consumer the record's Context lists: the cmd/mrw tests, `scripts/contract.sh`, both stress scripts, the Windows smoke; §319; the words; the skill regenerated; relock what `adr-lint` names. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunIsRefusedAtTheStepDepthLimit \(' <<<"$out" \
  && go test ./internal/guide/ -count=1 -run '^TestTheGuideTeachesMrwRun$' -v 2>&1 | grep -qE '^--- PASS: TestTheGuideTeachesMrwRun \(' \
  && go test ./internal/check/ -count=1 -run '^TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass$' \
  && grep -qF 'env["MRW_SHELL"] = "on"' docs/break/check-json/stress.py && grep -qF 'env["MRW_SHELL"] = "on"' docs/break/write-flags/stress.py \
  && grep -qE '^      MRW_SHELL: "?on"?$' .github/workflows/release-smoke-windows.yml \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 319\.' scripts/contract.sh \
  && grep -qF 'PASS  --then-sh without MRW_SHELL=on is refused and runs nothing' <<<"$c" \
  && grep -qF 'PASS  mrw run without MRW_SHELL=on is refused and runs nothing' <<<"$c" \
  && grep -qF 'PASS  the pair: with MRW_SHELL=on, --then-sh and mrw run run their command' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

The `go test ./internal/guide/ … | grep` clause runs under `set -o pipefail`, so a failing test fails the fence. Red at authoring, checked 2026-10-29 at `e8a6a3da`: `git grep -n -e TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt -e MRW_SHELL -e shell174` returns nothing, `grep -c '^# 319\. ' scripts/contract.sh` is 0, and the three `PASS` strings appear nowhere in the tree.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt` | `cmd/mrw/shell174_test.go` | off by default on write, check and run, nothing written, run or logged; only `on` enables; precedence; `--then NAME` and the declared check unaffected | none | S1, S2, S3 |
| `TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten` | `cmd/mrw/then094_test.go` | ADR-094's refusals still fire with the variable on | none | S4 |
| `TestRunIsRefusedAtTheStepDepthLimit` | `cmd/mrw/run177_test.go` | ADR-177 Decision 8's depth refusal still answers first | none | S4 |
| `TestTheGuideTeachesMrwRun` | `internal/guide/run177_test.go` | the guide carries the new caveat | none | S4 |
| `TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass` | `internal/check/steps092_test.go` | the step runner is untouched | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `askedStepsError`, called by `write` and `check`; `runAction` |
| 3 — the caller can discover it | the refusal names the variable; help, `mrw instructions`, README, AGENTS.md, `docs/agents.md` |
| 4 — it is used | MRW-10 in the Rubie report shows `--then-sh` reached through an allowed `mrw`; `mrw run` reaches the same shell (ADR-177 Decision 9); no adoption is measured (ADR-009) |

## Mutation Log
(empty until execute)

## Invariants

- A refused `--then-sh` or `mrw run` writes, runs and logs nothing.
- `--then NAME` and the declared check are unchanged.
- The depth, empty-command, placeholder and usage refusals keep their precedence and their words.
- No engine package changes.

## Risks

- Missing the variable in one consumer shows as a red suite, a red contract row or a red Windows smoke, not a silent pass.
- The Windows release smoke runs only on a release; T1's fence greps the workflow for the variable, and the release that ships this runs it.

## Stop Condition

The fence exits 0 and `adr-lint` passes for every record. Stop if a caller outside this repository is found in the tree that cannot set an environment variable (none is known), or if a consumer not in the record's Context turns up.

## Out of Scope

- MCP (permanent: boundary: `mrw_write` has no `--then-sh` and there is no `mrw_run`, `internal/mcp/mcp.go` says so)

## Verification Log
(empty until execute)
