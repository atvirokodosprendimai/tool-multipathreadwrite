# ADR-145: `version` and `instructions` touch no state

**Status:** Accepted
**Accepted:** 2026-10-10 on Zy's standing instruction "/loop continue delivering, end to end, no dead code", taking the open BACKLOG entry "`mrw version` and the other state-free commands still touch state" (quality-harness peer, reproduced on macOS). This is not a per-record yes: the pull request is where Zy can refuse it.
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-136, docs/adr/BACKLOG.md
**Invalidates:** None — ADR-136 made these commands skip the ledger check and left the migration it names out of scope
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/startup.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/startup145_test.go::TestVersionAndInstructionsMigrateNothing`
**Served-path change:** `mrw version`, `instructions`, the version flag in every spelling the parser takes, `-h` and `--help` (a subcommand's included, `mrw read -h`) no longer copy a legacy `./.mrw/` directory into the state directory, nor print the "moved" line; neither does a usage error. Any command that reaches the root command's `Before` — every other one, `stats` included — still does, once. Exit codes and output are unchanged.

## Context

`main` runs `state.Migrate(".")` before it parses anything: a pre-ADR-004 `.mrw/` directory in the working directory is copied into the state directory and announced on stderr. That is a write, and `mrw version` is the command a caller runs to check what is installed, in whatever directory the shell happens to be in. ADR-136 stopped these commands reading the ledger and named this migration as what was left. The quality-harness peer reproduced it on macOS.

**Audit of the class** — *a command that touches state before it needs to*: `mrw read --grep 'state\.(Migrate|Dir|Hold)' --exclude '*_test.go' cmd/mrw` names one call in `main`. `stats` opens the tally (`authoring.Load`) and reads the migrated state on purpose, so it keeps the migration.

## Existing Primitives Audit

- **The `Before` switch (ADR-136)** — already runs after the parse, and already lists `version`, `instructions` and `stats` as commands that read no ledger. The migration joins it, so the parser's own dispatch decides what reaches it.
- **`state.Migrate`** — unchanged; this record decides only who calls it.

## Decision

1. **The migration runs from the root command's `Before`, not from `main`.** The version flag, `--help` and a usage error are answered by the parser before `Before` runs, in every spelling it takes (`--v`, `-version`, `--version=false`, `--version=`, `-v=T` and the rest of what `strconv.ParseBool` reads), so none of them can migrate. An enumeration of those spellings in `main` was the first design and the Codex review of #392 found it incomplete twice: `--v`, `-version` and `--version=true`, then `--version=false` and the other values, which urfave/cli v3 treats as set whatever they say.
2. **`version` and `instructions` skip it by verb**, since they do reach `Before`. `-C dir version` is therefore covered too. Every other verb migrates, `stats` included.
3. **The migration still keys on the working directory** (`state.Migrate(".")`), as before: only when it runs moved.

## Alternatives Considered

- **A predicate on `os.Args` in `main`, before the parse** — rejected after two review rounds: it has to name every spelling the parser takes, and the parser, not this record, owns that set.
- **Drop the migration (ADR-004 is old)** — rejected: a checkout with a pre-ADR-004 `.mrw/` would silently lose its ledger; that is the owner's call, not a cleanup.

## Component / Boundary Impact

`cmd/mrw` only (one function, one condition in `Before`; `main` loses its migration block). No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `migrateLegacyState` | moved from `main` into `Before` | T1 | every command that reaches `Before` |
| `mrw version`, `instructions`, the version flag, `-h`, `--help` | no migration | T1 | CLI callers |
| `scripts/contract.sh` | §249 | T1 | CI Linux |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the install check writes nothing, wherever it is run.
- **Negative:** a legacy `.mrw/` is migrated by the first command that reaches `Before` with another verb, not by `mrw version`; a command that fails to parse no longer migrates either.
- **Neutral:** `stats` and every other command migrate as before.

## Out of Scope

- `stats` opening the tally (permanent: boundary: `stats` still migrates, as every command but two does, and `authoring.Load` opens the tally)
- Where the migration looks (permanent: boundary: the working directory, as ADR-004 chose, not `--root`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a legacy `.mrw/` is no longer migrated by `version` | Low | none: the next ledger-using command migrates it | the contract pair shows `read` does |

## Rollback

Revert T1: `version` and `instructions` migrate again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the boundaries above.
