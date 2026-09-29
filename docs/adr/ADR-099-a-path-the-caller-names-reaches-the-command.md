# ADR-099: A path the caller names reaches the command

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — chose "Refuse, exit 2 (Recommended)" for `mrw check --full PATH` when asked, approved the plan "close the open items after v1.32.0" whose PR C is this record's scope, set the goal "implement this plan end to ned, test properly, test for dead code, gaps", and said "go". The record's own text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of the approval, stated so it can be checked
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-054, ADR-078, ADR-096, ADR-097
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/path099_test.go::TestACheckWithFullAndAPathIsRefused`
**Invalidates:** none — checked. ADR-096 ("a path the caller names is never dropped by a finder") is extended to two more places a named path was dropped; ADR-097's named refusal is unchanged because `--help` stays a defined flag on every command.
**Served-path change:** `mrw check --full PATH…` exits 2 with "--full runs the whole project; it takes no PATH" and runs nothing, where v1.32.0 ran the whole-project check and dropped PATH in silence; `mrw read help`, `mrw read h`, `mrw write help`, `mrw check help` (and `h`) treat `help`/`h` as the path they name, where v1.32.0 printed the command's help. `--help` and `-h` still print help; `mrw help` still exists; every other answer is unchanged.

## Context

**What was observed** (2026-09-29, v1.32.0, read from `cmd/mrw/main.go` at `7ebdf51` and probed):

1. `mrw check --full x.go` runs the whole-project check and exits by its verdict: `checkCmd` reads the
   paths (`:2000`) and then sets `paths = nil` when `--full` is set (`:2008-2010`). Nothing reports that
   `x.go` was dropped. The flag's usage says "ignoring any scope" (`:1991`), which a caller reads as the
   working set, not the paths they typed. Zy chose on 2026-09-29: refuse, exit 2.
2. `mrw read help`, `mrw read h` and `mrw read -- help` print `read`'s help instead of serving a file named
   `help` (observed 2026-09-29 against v1.31.0 and v1.32.0; filed in BACKLOG from ADR-097). urfave/cli v3.11.0
   adds a `help` subcommand (alias `h`) to every command (`command_setup.go` `ensureHelp`), and the parser
   dispatches a first positional to a subcommand even after `--`. `cli.Command.HideHelpCommand`
   (`command.go:48-53`) suppresses that subcommand and keeps the `--help` flag.

**Audit of the class.** The class is *a path the caller named that the command does not act on*. Enumerated
2026-09-29 over `rootCommand().Commands` (`cmd/mrw/main.go:173`): the commands whose first positional is a
path are `read` (PATH…), `write` ([PLAN|-]) and `check` ([PATH…]) — **3**. `iter`'s first positional is a verb
(`add|rm|clear|note`), `mcp`, `seen`, `stats`, `instructions` and `version` take none. Within `check`, `--full`
is the one flag that discards the positionals — **1**. **Left out on purpose:** `iter` (a spec follows a verb,
so `mrw iter add help` already reaches the spec), and the root (`mrw help` is the documented way to list
commands, pinned by `agentsdoc_test.go:38` and `teach094_test.go:55`).

## Existing Primitives Audit

- **`cli.Exit(..., exitUsage)`** — reused for the refusal, as every usage error is (ADR-078).
- **`HideHelpCommand`** (urfave v3.11.0) — reused; no parser change.
- **`runIn`, `runSplit`** (cmd/mrw tests) — reused to drive the real command.

## Decision

1. `mrw check --full` with one or more PATHs is refused before anything runs: exit 2,
   `--full runs the whole project; it takes no PATH`, nothing on stdout. `--full` alone is unchanged.
   The flag's usage says it takes no PATH.
2. `read`, `write` and `check` carry `HideHelpCommand: true`: `help` and `h` are the paths they name;
   `--help` and `-h` still print help. The root keeps its `help` command.

## Alternatives Considered

- **Run the whole project and name the dropped paths** — offered to Zy and not chosen: a check whose scope is
  not what was asked should not run at all, the rule ADR-096 made for finders.
- **Scope `--full PATH` to the paths** — rejected: `--full` exists to mean the opposite.
- **Hide the help command on every command, root included** — rejected: `mrw help` is taught and tested.

## Component / Boundary Impact

None — `cmd/mrw` only. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw check --full PATH` | refused, exit 2, nothing run | T1 | CLI callers |
| `mrw read/write/check help` | the path `help` reaches the command | T2 | CLI callers |
| `scripts/contract.sh` | §191 (T1), §192 (T2) | T1, T2 | CI Linux |
| `docs/adr/BACKLOG.md` | the ADR-097 `mrw read help` entry closed | T2 | readers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | T1 and T2 are independent |

## Implementation

See `tasks/README.md`: T1 (`--full PATH`), T2 (the help subcommand).

## Consequences

- **Positive:** a named path is acted on or refused by name, never dropped.
- **Negative:** a script that ran `mrw check --full x.go` now exits 2; `mrw read help` no longer prints help.
- **Neutral:** `--help`, `-h` and `mrw help` are unchanged.

## Out of Scope

- `iter`'s verbs (permanent: boundary: a spec follows a verb, so a path named `help` already reaches `iter add`)
- The root `help` command (permanent: boundary: `mrw help` is taught and tested)
- MCP (permanent: boundary: the MCP tools take JSON arguments; no subcommand dispatch reaches them)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a script relies on `mrw check --full PATH` running | Low | Low | the refusal names the fix; release notes say so |
| `HideHelpCommand` also hides the `--help` flag | Low | Med | T2's test asserts `--help` and `-h` still print help on all three |

## Rollback

Revert T1 and T2. No state, format or exit code for any other input changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `cmd/mrw/path099_test.go::TestACheckWithFullAndAPathIsRefused` in T1's commit.
