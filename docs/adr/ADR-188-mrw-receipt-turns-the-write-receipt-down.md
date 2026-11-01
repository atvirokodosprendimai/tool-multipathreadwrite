# ADR-188: `MRW_RECEIPT` turns the write receipt down for a whole session

**Status:** Proposed
**Date:** 2026-11-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-179, ADR-152, docs/bench/observability/NATIVE-RESULTS.md, docs/bench/observability/QUIET-RESULTS.md
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`, `internal/guide/guide.go`
**Enforced-by:** `cmd/mrw/receipt188_test.go::TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes`
**Invalidates:** None — `mrw write` prints what it printed when `MRW_RECEIPT` is unset; `--quiet` and `--terse` keep their meaning and win over the variable; `--json` and `mrw_write` are unchanged.
**Served-path change:** `MRW_RECEIPT=quiet` in mrw's environment makes every `mrw write` print as `--quiet` does, `MRW_RECEIPT=terse` as `--terse` does, and `full` or an unset variable prints the receipt as now; a flag on the call wins, and any other value is ignored with one note on standard error.

## Context

Native editing tools print little or nothing on success; `mrw write` prints a row per hunk, a line per file and a summary. The native-versus-MRW runs (`docs/bench/observability/NATIVE-RESULTS.md`) put MRW's write results at 23,396 characters over 20 cells, about a quarter of what its agent received, and native cheaper in all four rounds. With every write quieted by a shim (`QUIET-RESULTS.md`, run 8), the write results fell to 3,672 characters over 10 cells and native's lead per round from $0.031 to $0.016; across those different runs, an estimate from the characters removed (about 4 characters a token, cache creation at $4 a million) puts the receipts' share near $0.004 a round, not an isolated effect.

`--quiet` and `--terse` already exist (ADR-179), but an agent picks its own flags per call, and nothing in these runs ever passed one. Owner direction, 2026-11-01: *"I mean not to remove these but add switch to turn them off"*, then *"Env var (Recommended)"* for where the switch lives.

**Audit of the class** — *a per-call output choice a host may want to set once*. Enumerated from the environment variables mrw reads (`grep -rn 'os.Getenv("MRW_' cmd internal --include='*.go'`, excluding tests): `MRW_MAX_COLS` sets `--max-cols` for a read when the flag is absent, and a bad value is ignored with a note (ADR-179); `MRW_SHELL` and `MRW_MCP` enable features; `MRW_STEP_DEPTH` and `MRW_WRITE_LOCK_TIMEOUT` are limits. The write receipt's form has no such variable. This record adds the one, on `MRW_MAX_COLS`'s pattern.

## Existing Primitives Audit

- `reportAs` (`cmd/mrw/main.go`): picks the receipt form from `--terse` and `--quiet`; the three write paths that print a receipt call it. The variable is read there, so every one of them follows it.
- `effectiveMaxCols`'s handling of `MRW_MAX_COLS` (`cmd/mrw/longline179.go`): a flag wins, an unparseable value is a one-line note and is ignored. Followed.
- `--quiet`'s promise (failed hunks, the summary line, the check's report and every step verdict still print) and `--terse`'s (every failed hunk, balance and closer row): unchanged; the variable only selects one of them.

## Decision

1. **`MRW_RECEIPT` selects the receipt `mrw write` prints when neither `--quiet` nor `--terse` is given:** `quiet` prints as `--quiet`, `terse` as `--terse`, `full` (or unset, or empty) as today.
2. **A flag on the call wins.** `--quiet` or `--terse` sets the form whatever the variable says.
3. **Any other value is ignored with one note on standard error**, `mrw: MRW_RECEIPT=X is not quiet, terse or full; ignored`, and the receipt is the full one.
4. **Nothing else follows it:** `--json`, `mrw_write`, `mrw read`, `mrw check` and every refusal are unchanged; a failed hunk prints under every value.

**What would make this decision wrong:** an agent that needs the per-hunk rows to understand a successful write it cannot reconstruct, under a host that set `quiet`. The summary line still names hunks and files; `MRW_RECEIPT=full` or a re-read restores the detail.

## Alternatives Considered

- **Make `--quiet` the default.** Rejected by the owner: the receipts stay by default.
- **A `.quality-harness.json` key.** Rejected by the owner's choice of the environment, which a host sets once for every project and which mixes no mrw output policy into quality-harness's file.
- **Refuse a bad value, exit 2.** Rejected: a typo in a host's environment would refuse every write; `MRW_MAX_COLS` set the precedent of a note.

## Component / Boundary Impact

`cmd/mrw` only: `reportAs`, the write help, the `--quiet` and `--terse` usage strings. No engine package.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` receipt | follows `MRW_RECEIPT` when no flag says | T1 | agents, hosts |
| `mrw write --help`, `mrw instructions` | name the variable | T1 | agents |
| contract §323 | new rows | T1 | CI |
| README, AGENTS.md | name the variable | T1 | readers |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a host can make every write cost one line on success, without each agent learning a flag.
- **Negative:** one more variable to know; under `quiet` a successful write's per-hunk detail is not printed.
- **Neutral:** unset, nothing changes.

## Out of Scope

- `mrw_write` and `--json` (permanent: boundary: their answers are structured receipts a host parses, not text an agent rereads)
- Read output size, the other measured cost (permanent: boundary: one change at a time; a later record)
- Measuring the switch again (permanent: boundary: run 8 measured what it does; this record makes it settable)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host sets `quiet` and an agent misses an advisory row | Med | Low | `terse` keeps balance and closer rows; the record names the choice |

## Rollback

Revert T1; the variable is then ignored.

## Follow-ups

- [ ] none at authoring
