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
**Served-path change:** `mrw version`, `instructions` and every spelling of the version flag the parser accepts as the first argument (`-v`, `--v`, `-version`, `--version`, and either with `=true`) no longer copy a legacy `./.mrw/` directory into the state directory, nor print the "moved" line. Any other command still does, once. Exit codes and output of these commands are unchanged.

## Context

`main` runs `state.Migrate(".")` before it parses anything: a pre-ADR-004 `.mrw/` directory in the working directory is copied into the state directory and announced on stderr. That is a write, and `mrw version` is the command a caller runs to check what is installed, in whatever directory the shell happens to be in. ADR-136 stopped these commands reading the ledger and named this migration as what was left. The quality-harness peer reproduced it on macOS.

**Audit of the class** — *a command that touches state before it needs to*: `mrw read --grep 'state\.(Migrate|Dir|Hold)' --exclude '*_test.go' cmd/mrw` names one call in `main`. `stats` opens the tally (`authoring.Load`) and reads the migrated state on purpose, so it keeps the migration.

## Existing Primitives Audit

- **The `Before` switch (ADR-136)** — already lists `version`, `instructions` and `stats` as commands that read no ledger. It runs after the parse; the migration runs before it, so the predicate here works on `os.Args` and cannot reuse it.
- **`state.Migrate`** — unchanged; this record decides only who calls it.

## Decision

1. **`startsWithoutState(args)` is true for `version` and `instructions`, and for the version flag in the spellings the parser accepts (`-v`, `--v`, `-version`, `--version`, and either with `=true`), as the first argument.** `main` skips `state.Migrate` for them. Found by the Codex review of #392: the first draft knew only `-v` and `--version`, and `--v`, `-version` and `--version=true` printed the version and migrated.
2. **Everything else migrates as before**, `stats` included, since it reads what the migration moves.
3. **Not extended to a flag before the verb.** `mrw -C dir version` migrates; the first argument decides, because the parse has not run.

## Alternatives Considered

- **Move the migration into `Before`** — rejected: it would run after the parse for the right root, but it changes when and where the migration happens for every command, and the legacy directory is keyed by the working directory today.
- **Drop the migration (ADR-004 is old)** — rejected: a checkout with a pre-ADR-004 `.mrw/` would silently lose its ledger; that is the owner's call, not a cleanup.

## Component / Boundary Impact

`cmd/mrw` only (one predicate, one condition in `main`). No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `startsWithoutState` | new function | T1 | `main` |
| `mrw version`, `-v`, `--version`, `instructions` | no migration | T1 | CLI callers |
| `scripts/contract.sh` | §249 | T1 | CI Linux |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the install check writes nothing, wherever it is run.
- **Negative:** a legacy `.mrw/` is migrated by the first command that is not one of the four, not by `mrw version`.
- **Neutral:** `stats` and every other command migrate as before.

## Out of Scope

- `mrw -C dir version` and `mrw --root dir version` (permanent: boundary: the migration runs before the parse, so only the first argument is known)
- `--help` and `-h` (permanent: boundary: they print usage, and still migrate a legacy `./.mrw/` first and announce it before the usage; the usage text does not need state, but the flag is not one of the install-check spellings this record covers)
- `stats` opening the tally (permanent: boundary: it reads the state the migration moves)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a legacy `.mrw/` is no longer migrated by `version` | Low | none: the next ledger-using command migrates it | the contract pair shows `read` does |

## Rollback

Revert T1: `version` and `instructions` migrate again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the boundaries above.
