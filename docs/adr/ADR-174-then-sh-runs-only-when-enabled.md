# ADR-174: `--then-sh` runs only when the environment enables it

**Status:** Proposed
**Date:** 2026-10-24
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (5 findings, each checked against the code): the refusal comes before anything is written or run, not before anything is read (`check` loads its working set and harness first, `cmd/mrw/main.go:2537-2547`); the existing refusals keep their precedence (Decision 1); mixed step lists and raw values are tested; the fence runs the whole `cmd/mrw` package; 14 test files, not 15.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-092, ADR-094, ADR-095, ADR-100, ADR-170, README.md, AGENTS.md, docs/agents.md, internal/guide/guide.go
**Governs:** `cmd/mrw/main.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `docs/break/check-json/stress.py`, `docs/break/write-flags/stress.py`
**Enforced-by:** None — until T1 lands, which adds `cmd/mrw/thensh174_test.go::TestThenShRunsOnlyWhenTheEnvironmentEnablesIt`
**Invalidates:** ADR-092 Decision 8 in part — that `--then-sh` runs any shell its caller writes. From this record it runs only when `MRW_THEN_SH=on` is in mrw's environment; `--then NAME` is unchanged.
**Served-path change:** `mrw write … --then-sh CMD` and `mrw check --then-sh CMD` without `MRW_THEN_SH=on` are refused, exit 2, before anything is written or any step or check is run, the message naming the variable and `--then NAME`; with it, they run as before.

## Context

ADR-092 Decision 8 accepted that "a harness rule that allows `mrw` without reading its arguments allows arbitrary shell through `--then-sh`", and every surface says so beside the flag (`guide.ThenShCaveat`, `internal/guide/guide.go:139-141`). An outside A/B showed the cost in the field (Rubie tech summary v1, 2026-10-24, mrw v1.74.0, issue MRW-10): their repository guard blocks `git push --force origin master` typed directly, and `mrw … --then-sh 'git push --force origin master'` passed it, so allowing `mrw` in a host's permission rules allowed any shell. Reproduced on v1.75.0, 2026-10-24, with `--then-sh 'echo then-sh-ran > then.txt'`: the step ran, exit 0. The report's ask: running any shell command is opt-in.

Zy chose this record on 2026-10-24 — *"A. Guide cost (Recommended), B. Bounded fixes, C. Contract ADRs"*.

Who uses `--then-sh` (counted 2026-10-24 with `grep -c 'then-sh'` and `grep -l`): `scripts/contract.sh` 40 lines, 14 Go test files in `cmd/mrw` (some only teach or parse the flag; T1 sets the variable only where a step runs), `docs/break/check-json/stress.py` 6, `docs/break/write-flags/stress.py` 1, README 5, AGENTS.md 9, `docs/agents.md` 2, `internal/mcp/mcp.go` and `tools.go` 1 each (both say it is CLI-only), `internal/check/check.go` 2 (comments). The opencode plugin does not use it (`grep -rn 'then-sh' cmd/opencode/mrw-plugin/src cmd/opencode/mrw-plugin/test`: none). This repository's own drill uses `--then static` (a declared step), not `--then-sh`.

## Existing Primitives Audit

- `askedStepsError` (`cmd/mrw/main.go:2239-2255`), called by `write` (`:1718`) before the plan is applied and by `check` (`:2547`) after it loads its working set and harness (`:2537`, `:2543`) and before the check runs: the place every `--then-sh` refusal already lives (empty command, placeholders, depth). The new refusal joins it. Reuse.
- ADR-170's switch (`MRW_MCP=on`, only the exact raw `on` enables): the shape this record copies.
- `--then NAME` (`.quality-harness.json` `"steps"`): unchanged — the project declared it, as it declares the check.

## Decision

**1. Off unless the environment says `on`.** `askedStepsError` refuses any ad-hoc step (`--then-sh`) — wherever it sits in a list mixed with `--then NAME` — unless `MRW_THEN_SH` is exactly the raw string `on` (not `ON`, not ` on` or `on `): exit 2, before anything is written or any step or check is run (as ADR-094's refusals; `check` has already read its configuration by then, which runs nothing), the message `--then-sh runs any shell command, so it is off unless MRW_THEN_SH=on is in mrw's environment; a step the project declares in .quality-harness.json runs with --then NAME`. Any other non-empty value is refused the same way and the message names the value. **Precedence is today's, with the switch last:** the depth refusal answers first (ADR-095; for `check`, ADR-100 Decision 2 — `main.go:2522`), then an empty command and a placeholder (ADR-094), each naming what it names today; only a list that passes those meets the switch. Under `mrw check --json` the refusal is the usual `{"error": …}` document (ADR-100).

**2. Why the environment, not a flag.** The problem is that a host guard reads `mrw`'s arguments as data: a flag such as `--allow-then-sh` sits in the same argument list as `--then-sh` and would pass the same guard. The environment is set by whoever starts the session or writes the command: a guard that inspects commands sees `MRW_THEN_SH=on mrw …` as an assignment in front of the command, and a session-wide export is a person's choice outside the agent's command. Neither is proof against an agent that can run `export` — a guard that allows `export` already allows any shell.

**3. Nested runs inherit it.** A step or check that runs mrw again inherits the variable as it inherits `MRW_STEP_DEPTH` (ADR-095); nothing new is needed.

**4. This repository and its harnesses enable it where they test it.** `scripts/contract.sh` sets `MRW_THEN_SH=on` for the rows that exercise `--then-sh` and adds §299, which runs the same step with the variable unset and asserts the refusal and that the marker file was not written; the Go tests that drive `--then-sh` set it in their command's environment; the two `docs/break` stress scripts set it.

**5. Told where callers look, and released with a loud note.** `guide.ThenShCaveat` becomes the off rule; `mrw write --help`, `mrw check --help`, `mrw instructions`, README, AGENTS.md and `docs/agents.md` say `--then-sh` is off until `MRW_THEN_SH=on`. The release notes lead with it beside ADR-173 if both are accepted.

**What would make this decision wrong:** a caller who needs an ad-hoc step and cannot set an environment variable. Such a caller declares the step in `.quality-harness.json` and runs `--then NAME`.

## Alternatives Considered

- **Keep it, say it louder.** Rejected: ADR-092 Decision 8 already says it on every surface, and the field case went through anyway.
- **Remove `--then-sh`.** Rejected: ad-hoc verification in the same call is useful to a person at a shell, and the opt-in keeps it.
- **A flag (`--allow-then-sh`).** Rejected (Decision 2): visible to nothing that does not already read the arguments.
- **An allow-list of commands in `.quality-harness.json`.** That is `--then NAME`, which exists.
- **Version: v1.76.0 with the change leading the notes, or v2.0.0.** Zy's call; the record recommends the minor with the loud note, as ADR-170 shipped in v1.75.0.

## Component / Boundary Impact

- `cmd/mrw`: one more refusal in `askedStepsError`.
- `internal/guide`: `ThenShCaveat`'s sentence.
No package changes ownership; no engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `--then-sh` on `write` and `check` | refused, exit 2, unless `MRW_THEN_SH=on` | `cmd/mrw` | shell callers, host guards |
| `MRW_THEN_SH` | new variable, only `on` enables | `cmd/mrw` | sessions that want ad-hoc steps |
| help, `mrw instructions`, README, AGENTS.md, `docs/agents.md` | the off rule | `internal/guide`, docs | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `MRW_THEN_SH=on` | T1 | T1 | Yes for a caller that relied on `--then-sh`: it sets the variable or declares the step |

## Implementation

`tasks/README.md`: T1 (the refusal, the harnesses, the words). One pull request.

## Consequences

- **Positive:** allowing `mrw` in a host's rules no longer allows arbitrary shell by default.
- **Negative:** a caller of `--then-sh` must set the variable; the repository's own tests and stress scripts carry it.
- **Neutral:** `--then NAME`, the check, and MCP (which has no `--then-sh`) are unchanged.

## Out of Scope

- The project's declared check and steps (permanent: boundary: a repository's `.quality-harness.json` is the project's own command, as a Makefile is; running it is what `--then NAME` and the default check mean)
- Host guards that parse plans as shell (permanent: boundary: the host's code; ADR-171 moves the plan out of the shell)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A user's script with `--then-sh` breaks on upgrade | Med | Low | exit 2 with the variable named; the release notes lead with it |
| A test that drives `--then-sh` without the variable goes red after the change | High | Low | T1 sets it in every such test and row; the full suite is the check |

## Rollback

Accept any environment again (one condition); nothing persistent changes.

## Follow-ups

- [ ] none at authoring
