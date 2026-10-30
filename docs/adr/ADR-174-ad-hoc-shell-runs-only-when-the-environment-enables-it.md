# ADR-174: Ad-hoc shell runs only when the environment enables it

**Status:** Accepted
**Date:** 2026-10-24
**Accepted:** 2026-10-29 by Zy — "accept, add, merge all, push" (the three records #566, #567, #568 together; "add" is the `capped` receipt key ADR-173 left to Zy)
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (5 findings, each checked against the code): the refusal comes before anything is written or run, not before anything is read (`check` loads its working set and harness first); the existing refusals keep their precedence (Decision 1); mixed step lists and raw values are tested; the fence runs the whole `cmd/mrw` package. 2026-10-29, before acceptance, on Zy's direction of that day: rewritten from "`--then-sh` runs only when `MRW_THEN_SH=on`" to one switch, `MRW_SHELL=on`, for both ad-hoc shell surfaces, `--then-sh` and `mrw run` (ADR-177, released since the first draft, which made the draft's safety claim false); ADR-177 Decision 9 is named in Invalidates; the safety claim is made exact and its limit stated (Decision 6); the consumers are enumerated by a recorded command (Context); the line numbers are re-taken against main `e8a6a3da`; renamed from `ADR-174-then-sh-runs-only-when-enabled.md`. Then from a Codex xhigh challenge of the rewrite (5 findings, each checked against the code): "before anything is written" is defined, since mrw's own state bookkeeping precedes every refusal (Decision 1); the Windows smoke enables the switch for the job, because a fourth step, Windows PowerShell 5.1 (`run: *ps`, `:124`), runs `mrw run` too; the log directory's absence is asserted; the consumer fence clauses check the assignment, not the spelling. The challenge ran once, with a prompt narrowed to four questions, after the broad first prompt for ADR-173 was killed at host-review's 600 s budget.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-092, ADR-094, ADR-095, ADR-100, ADR-170, ADR-177, ADR-181, README.md, AGENTS.md, docs/agents.md, internal/guide/guide.go
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/run177.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `docs/break/check-json/stress.py`, `docs/break/write-flags/stress.py`, `.github/workflows/release-smoke-windows.yml`
**Enforced-by:** None — until T1 lands, which adds `cmd/mrw/shell174_test.go::TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt`
**Invalidates:** ADR-092 Decision 8 in part — that `--then-sh` runs any shell its caller writes — and ADR-177 Decision 9's "`mrw run` runs any shell command it is given". From this record both run only when `MRW_SHELL=on` is in mrw's environment. ADR-092 Decision 8's `--then NAME` half, the declared check (ADR-054) and ADR-177's other decisions stand; ADR-181 is a consumer (its Windows smoke runs `mrw run`), not invalidated.
**Served-path change:** `mrw write … --then-sh CMD`, `mrw check --then-sh CMD` and `mrw run 'CMD'` without `MRW_SHELL=on` in mrw's environment are refused, exit 2, before anything is written or any step, check or command is run, the message naming the variable (and, for `--then-sh`, `--then NAME`); with it, they run as before.

## Context

ADR-092 Decision 8 accepted that "a harness rule that allows `mrw` without reading its arguments allows arbitrary shell through `--then-sh`", and ADR-177 Decision 9 extended the same sentence to `mrw run`; every surface says it beside the flag (`guide.ThenShCaveat`, `internal/guide/guide.go:159-163`, embedded at `:54`, `:66` and `:106`). An outside A/B showed the cost in the field (external tech summary v1, 2026-10-24, mrw v1.74.0, issue MRW-10): their repository guard blocks `git push --force origin master` typed directly, and `mrw … --then-sh 'git push --force origin master'` passed it, so allowing `mrw` in a host's permission rules allowed any shell. Reproduced on v1.75.0, 2026-10-24, with `--then-sh 'echo then-sh-ran > then.txt'`: the step ran, exit 0. The report's ask: running any shell command is opt-in.

Zy chose this record on 2026-10-24 — *"A. Guide cost (Recommended), B. Bounded fixes, C. Contract ADRs"*, and the acceptance line of ADR-171, ADR-175 and ADR-176 quotes *"ADR-173 + ADR-174"* among the records chosen that day. The draft was never committed: it stayed in the stash `4f2e05af`, and before it could be implemented `mrw run` shipped (ADR-177, v1.77.0), so gating `--then-sh` alone no longer supported the draft's claim that allowing mrw excludes arbitrary shell — `docs/adr/BACKLOG.md`'s "ADR-174 (draft needs revision)" entry. On 2026-10-29 Zy decided the direction: **one switch for both.** `MRW_SHELL=on` in mrw's environment (the exact raw value `on`) enables `--then-sh` (on `write` and `check`) and `mrw run`; without it both are refused, exit 2, before anything is written or run, the message naming the variable and, for `--then-sh`, `--then NAME`; the declared check and `--then NAME` steps are unchanged; nested runs inherit the variable, as with `MRW_STEP_DEPTH`; this repository's contract, Go tests and `docs/break` stress scripts set it where they exercise either feature. This record stays Proposed until Zy accepts the amended text.

**The class, and who consumes it.** The class is *every way the mrw binary runs a shell command its caller wrote on the command line*: `--then-sh` on `write` and `check` (`thenShFlag`, `cmd/mrw/main.go:2554-2557`) and `mrw run` (`runAction`, `cmd/mrw/run177.go:85`). The declared check and `--then NAME` are outside it: the project declared them. Enumerated 2026-10-29 at `e8a6a3da`:

- `git grep -l -e 'then-sh' -- . ':!docs/adr' ':!docs/break/campaign-*'` — **35 files**: the binary and its help (`cmd/mrw/main.go`), `internal/guide/guide.go`, `internal/check/check.go` (comments), `internal/mcp/mcp.go`, `tools.go`, `undeclared093_test.go`, `testdata/legacy_golden.jsonl` (MCP says `--then-sh` is CLI-only), `scripts/contract.sh`, `docs/break/check-json/stress.py`, `docs/break/write-flags/stress.py` and its README, README.md, AGENTS.md, `docs/agents.md`, `docs/architecture.md`, `docs/measure.md`, `.claude/skills/mrw/SKILL.md`, `cmd/mrw/testdata/outcomes113.golden`, and 17 `cmd/mrw/*_test.go`.
- `git grep -l -E -e 'mrw run' -e '"run"' -e "'run'" -e '\$MRW" run' -e 'm run ' -- . ':!docs/adr' ':!docs/break/campaign-*'` — **40 files**, which the pattern over-matches; read by hand, the ones that invoke the subcommand are `scripts/contract.sh`, `.github/workflows/release-smoke-windows.yml` (`:79` bash; `:113`, a PowerShell script anchored as `ps` and run twice, by pwsh 7 and by Windows PowerShell 5.1 at `:124`; `:151` cmd — ADR-181's smoke), and `cmd/mrw/run177_test.go`, `runnotice_test.go`, `runread181_windows_test.go`, `paddedstress_test.go`, `receipts111_test.go`, `refused176_test.go`. The rest are text about it (README, AGENTS.md, `docs/agents.md`, the skill, `internal/guide`, `docs/receipts.txt`, `docs/bench/FAQ.md`, `docs/bench/output-study/*`, whose `measure_output.py` classifies transcripts and runs nothing), the subcommand's own code (`run177.go`, `refused176.go`, `main.go`), or a different `run`: `npm run` (`.github/workflows/ci.yml`, `.claude/skills/release/SKILL.md`, the opencode plugin's `smoke.test.mjs`), `docker run` and the bench's `run` stage (`internal/bench/container.go`, `container_test.go`, `swebench_test.go`, `cmd/bench/main.go`), `deno run` (`scripts/brew_tap.py`, `test_brew_tap.py`), and prose "mrw runs" (`internal/apply/pathop_commit_test.go`, `internal/curve/score.go`, `internal/mcp/schema.go`, `bench/tasks/…/reference.patch`, `docs/curve/reading-02-plan.md`). `internal/check/check.go` and `runlog177_test.go` call `check.Exec` directly, below the switch, and are unaffected.
- **Named negatives** (each checked): the compaction host hook `docs/agents.md` §8 documents runs `mrw seen --forget`, not `mrw run`; the bench B arm (`internal/bench`, `cmd/bench`) uses neither `--then-sh` nor `mrw run` (`grep -rn "mrw run\|then-sh" internal/bench cmd/bench`: none); `.claude/settings.json`'s two hooks (`rules-on-read.py`, `static-after-commit.py`) call `git` and `scripts/static.sh`, never mrw's shell; the opencode plugin (`cmd/opencode/mrw-plugin/src`, `test`) has `npm run` only. This repository's own drill uses `--then static` (a declared step).

So the consumers that must set `MRW_SHELL=on`: `scripts/contract.sh`, the cmd/mrw tests that run a `--then-sh` step or `mrw run`, the two `docs/break` stress scripts, and the Windows release smoke workflow in all three of its shells.

## Existing Primitives Audit

- `askedStepsError` (`cmd/mrw/main.go:2605-2621`), called by `write` (`:2028`) before the plan is applied and by `check` (`:2913`) after it loads its working set (`iter.Load`) and harness (`check.Load`) and before the check runs: the place every `--then-sh` refusal already lives (depth, empty command, placeholder). The `--then-sh` half of the switch joins it. Reuse.
- `runAction`'s refusals (`cmd/mrw/run177.go:85-104`): depth (`:95`), padded arguments, the argument count, each before `state.Base()` and the log directory (`:106-113`). The `mrw run` half of the switch joins them, and `refuse` already answers `--json` with the `checkRefusal` document (ADR-177 Decision 7). Reuse.
- One helper in `cmd/mrw` reads the variable for both, so the rule and its wording live once; it stays out of `internal/check`, which is an engine package this record does not own.
- ADR-170's switch (`mcpEnabled`, `cmd/mrw/main.go:551-565`: only the exact raw `on` enables, a value that is neither `on` nor `off` is warned about): the shape this record copies, with a refusal in place of the warning, since here the call is refused anyway.
- `check.DepthRefusal` and `MRW_STEP_DEPTH` (ADR-095): the inheritance model for Decision 3.
- `--then NAME` (`.quality-harness.json` `"steps"`) and the declared check: unchanged — the project declared them.

## Decision

**1. Off unless the environment says `on`.** `--then-sh` — wherever it sits in a list mixed with `--then NAME` — on `mrw write` and `mrw check`, and `mrw run`, are refused unless `MRW_SHELL` is exactly the raw string `on` (not `ON`, not ` on` or `on `): exit 2, before anything is written or any step, check or command is run (as ADR-094's refusals; `check` has already read its working set and configuration by then, which runs nothing), and before `mrw run` makes its log directory. "Written" here means the working tree, a `mrw-runs/` directory or log, and an ADR-176 refused-plan copy (none is kept: the refusal comes before a plan is read); mrw's own state bookkeeping that precedes every refusal these commands make today is unchanged and is not counted — the legacy-state migration every command but `version`, `instructions` and `skill` runs in `Before` (`cmd/mrw/main.go:185-193`), the state directory and its root marker, `iteration.lock` taken by `iter.Load` on `check` (`:2903`), and `tally.lock` taken by `authoring.PatternOf` on a `write --json` refusal (`:2016`), as with ADR-094's and ADR-100's refusals (Codex challenge, 2026-10-29). The `--then-sh` message: `--then-sh runs any shell command, so it is off unless MRW_SHELL=on is in mrw's environment; a step the project declares in .quality-harness.json runs with --then NAME`. The `mrw run` message: `mrw run runs any shell command, so it is off unless MRW_SHELL=on is in mrw's environment`. When `MRW_SHELL` is set to a non-empty value other than `on` (an empty value names nothing and is treated as unset), the message adds `(MRW_SHELL is "VALUE"; only the exact value on enables it)`. **Precedence is today's, with the switch last among the refusals:** for `--then-sh`, the depth refusal answers first (ADR-095; for `check`, ADR-100 Decision 2), then an empty command and a placeholder (ADR-094), each naming what it names today; for `mrw run`, depth, then padded arguments and the argument count (ADR-177 Decision 7). Only a call that passes those meets the switch. Under `--json`, `mrw check` and `mrw run` answer the refusal with their usual `{"error": …}` document (ADR-100, ADR-177 Decision 7), and `mrw write --json` with its refusal receipt.

**2. Why the environment, not a flag.** A host guard reads `mrw`'s arguments as data: a flag such as `--allow-shell` sits in the same argument list as `--then-sh` and would pass the same guard. The environment is set by whoever starts the session or writes the command: a guard that inspects commands sees `MRW_SHELL=on mrw …` as an assignment in front of the command, and a session-wide export is a person's choice outside the agent's command. Neither is proof against an agent that can run `export` — a guard that allows `export` already allows any shell.

**3. Nested runs inherit it.** A step, check or `mrw run` command that runs mrw again inherits the variable as it inherits `MRW_STEP_DEPTH` (ADR-095); nothing new is needed. A command that clears the environment (`env -i`, `sudo`) drops it, and the nested call is refused — the safe side.

**4. This repository and its harnesses enable it where they exercise it.** `scripts/contract.sh` exports `MRW_SHELL=on` once in its prologue, so every row that drives `--then-sh` or `mrw run` keeps working, and adds §319, whose rows drive the binary with the variable removed (`env -u MRW_SHELL`) and set to `ON`, assert the refusal, exit 2, and that the marker file was not written, beside the same calls with `on`, which run. The cmd/mrw tests run the command in-process (`runIn`, `cmd/mrw/paddedarg_test.go:14-25`), so "only that command's environment" is not available: a test that runs a `--then-sh` step or `mrw run` sets `t.Setenv("MRW_SHELL", "on")`, scoped to that test (Go refuses `t.Setenv` beside `t.Parallel`, so a conflict panics rather than leaks), and the off cases set `t.Setenv("MRW_SHELL", "")` so a developer's exported `on` cannot make them pass; a test that builds and execs the binary sets it in `exec.Cmd.Env`. The two `docs/break` stress scripts set it in the environment they pass mrw. `.github/workflows/release-smoke-windows.yml` sets it once in the `smoke` job's `env`, which reaches all four shells that run `mrw run` — bash, pwsh 7, Windows PowerShell 5.1 and cmd — where per-step settings would have to be repeated on the `*ps` alias step too.

**5. Told where callers look, and released with a loud note.** `guide.ThenShCaveat` becomes the off rule for both; `mrw write --help`, `mrw check --help`, `mrw run --help`, `mrw instructions`, `mrw skill`, README (the `--then-sh` and `mrw run` passages and "What allowing mrw allows"), AGENTS.md and `docs/agents.md` (§7 and `:123`) say that `--then-sh` and `mrw run` are off until `MRW_SHELL=on`. The release notes lead with it, beside ADR-173 if both are accepted.

**6. What the switch buys, exactly.** With the switch off, allowing mrw still runs the project's declared check (after a code write, and `mrw check`) and its declared `--then NAME` steps, and runs no ad-hoc shell: no command the caller writes on mrw's command line reaches a shell. It is not containment. mrw writes any file its user can (`docs/agents.md` §7), `.quality-harness.json` and the test files a check runs included, so an agent allowed mrw can still change what the declared check runs and then trigger it with a code write. What the switch removes is the one-call path a guard cannot see by reading mrw's arguments — the MRW-10 case; containment stays the host's sandbox or the operating system's permissions, as §7 says. README and `docs/agents.md` §7 carry this sentence, not a claim that allowing mrw excludes shell.

**What would make this decision wrong:** a caller who needs an ad-hoc step and cannot set an environment variable. Such a caller declares the step in `.quality-harness.json` and runs `--then NAME`, or runs the command outside mrw.

## Alternatives Considered

- **Gate `--then-sh` alone (`MRW_THEN_SH=on`), the 2026-10-24 draft.** Rejected by Zy on 2026-10-29: `mrw run` runs the same shell, so the gate would leave the field case open through another subcommand and its claim false.
- **Two switches, one per surface.** Not taken: both run a caller-written command through `sh -c`, and a host that wants one wants the other; one variable is one thing to set and to explain.
- **Keep both, say it louder.** Rejected: ADR-092 Decision 8 and ADR-177 Decision 9 already say it on every surface, and the field case went through anyway.
- **Remove `--then-sh` and `mrw run`.** Rejected: ad-hoc verification in the same call, and a long command's output kept out of context, are useful to a person at a shell, and the opt-in keeps them.
- **A flag (`--allow-shell`).** Rejected (Decision 2): visible to nothing that does not already read the arguments.
- **An allow-list of commands in `.quality-harness.json`.** That is `--then NAME`, which exists; `mrw run` of a declared step is not asked for.
- **Version: v1.79.0 with the change leading the notes, or v2.0.0.** Zy's call; the record recommends the minor with the loud note, as ADR-170 shipped in v1.75.0.

## Component / Boundary Impact

- `cmd/mrw`: one helper that reads `MRW_SHELL`; one more refusal in `askedStepsError` and one in `runAction`.
- `internal/guide`: `ThenShCaveat`'s sentence.
No package changes ownership; no engine package changes; `docs/architecture.md`'s map is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `--then-sh` on `write` and `check` | refused, exit 2, unless `MRW_SHELL=on` | `cmd/mrw` | shell callers, host guards |
| `mrw run` | refused, exit 2, unless `MRW_SHELL=on`; under `--json` the `{"error": …}` document | `cmd/mrw` | shell callers, host guards, the Windows release smoke |
| `MRW_SHELL` | new variable, only the exact `on` enables, inherited by nested runs | `cmd/mrw` | sessions that want ad-hoc shell |
| help, `mrw instructions`, `mrw skill`, README, AGENTS.md, `docs/agents.md` | the off rule and Decision 6's sentence | `internal/guide`, docs | agents, people configuring a host |
| `.github/workflows/release-smoke-windows.yml` | sets `MRW_SHELL: "on"` in the `smoke` job's `env`, for its four `mrw run` shells | CI | release |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `MRW_SHELL=on` | T1 | T1 | Yes for a caller that relied on `--then-sh` or `mrw run`: it sets the variable or declares the step |

## Implementation

`tasks/README.md`: T1 (the switch, the consumers, the words). One pull request.

## Consequences

- **Positive:** allowing `mrw` in a host's rules no longer allows ad-hoc shell by default; a guard that reads the command sees the switch.
- **Negative:** a caller of `--then-sh` or `mrw run` must set the variable; the repository's own tests, rows, stress scripts and release smoke carry it. The declared check stays reachable through a write (Decision 6).
- **Neutral:** `--then NAME`, the check, and MCP (which has neither `--then-sh` nor `run`) are unchanged.

## Out of Scope

- The project's declared check and steps (permanent: boundary: a repository's `.quality-harness.json` is the project's own command, as a Makefile is; running it is what `--then NAME` and the default check mean — Decision 6 states the limit)
- Containing an agent that may write the files the declared check runs (permanent: boundary: containment is the host's sandbox or the operating system's permissions, `docs/agents.md` §7, ADR-168)
- Host guards that parse plans as shell (permanent: boundary: the host's code; ADR-171 moves the plan out of the shell)
- MCP (permanent: boundary: `mrw_write` has no `--then-sh` and there is no `mrw_run`, `internal/mcp/mcp.go` says so)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A user's script with `--then-sh` or `mrw run` breaks on upgrade | Med | Low | exit 2 with the variable named; the release notes lead with it |
| A test, row or workflow step that drives either without the variable goes red after the change | High | Low | T1 sets it in every consumer the Context lists; the full suite, the contract and the Windows smoke are the check |
| A reader takes the switch for containment | Med | Med | Decision 6's sentence on every surface that says what allowing mrw allows |

## Rollback

Accept any environment again (one condition in the helper); nothing persistent changes.

## Follow-ups

- [ ] none at authoring
