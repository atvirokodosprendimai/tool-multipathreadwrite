# ADR-187: a write runs no check unless it is asked for

**Status:** Proposed
**Date:** 2026-11-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-054, ADR-113, ADR-115, ADR-061, ADR-094, ADR-152
**Governs:** `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/writer/flow.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/check187_test.go::TestAWriteRunsNoCheckUnlessAskedFor`
**Invalidates:** ADR-054's default: a CLI write to a non-prose path ran the project's check unless `--no-check`; ADR-113's rule that `mrw_write` runs it by the same rule unless `check: false`. Checked and unchanged: `--check` (it still runs the check on any plan and is exit 2 when none can run); `mrw_write`'s `check: true` (it runs the check by ADR-054's old rule: when a written path is not prose and a check exists); exit 3, which still means "applied, and the check that was asked for did not pass" and also "applied, and a step that was asked for failed" (a failing `--then` step after an unchecked write exits 3); `scoped_check` and its tokens (ADR-061, ADR-094), read whenever a check runs; `--then` and `then` (ADR-115), which after a write with no check run their steps alone, as they did after `--no-check`; `mrw check`.
**Served-path change:** `mrw write` and `mrw_write` run no check after a write unless it is asked for (`--check`, or `check: true`); `--no-check` is still accepted and changes nothing; a write that used to exit 3 because its automatic check failed now exits 0 and prints no check line.

## Context

ADR-054 made a write to a non-prose path run the project's check by default, so that an applied write could not leave a broken tree unnoticed. The check is whatever the project declares in `.quality-harness.json`, or `go test ./...` inferred from `go.mod`, narrowed by `scoped_check` when one is declared.

Two observations on 2026-11-01 say that default costs more than it buys where the check is a project's whole gate:

- **A peer session in another repository** (memoryd room `tool-multipathreadwrite`, message #102): its declared check is `make check` (gofmt, vet, the whole suite with a 300-second timeout, and a linter), minutes per write. Three `--create` calls in one command would have run it three times back to back at load ~20. The agent already runs its own focused verification and one final full check, so the per-write run duplicated work, and a result taken under that contention could not be attributed. It passed `--no-check` by hand on every write.
- **This repository, this session:** a `.json` or `.py.txt` write fell back to the whole `go test ./...` (2-4 minutes each) because a `{packages}`-only `scoped_check` cannot place it; the hint line said so each time.

Owner decision, 2026-11-01: *"Default off everywhere"*, answered to the question of whether the default changes.

**Audit of the class** — *a surface that decides whether a write's check runs*. Enumerated from the one type that carries the decision, `writer.CheckMode` (`internal/writer/flow.go`), and its constructors (`grep -rn 'writer.Check' cmd internal --include='*.go'`, excluding tests):

| Surface | Decides | Today | After |
|---|---|---|---|
| `mrw write` (`cmd/mrw/main.go`) | `--check`, `--no-check`, else auto | auto | off unless `--check` |
| `mrw_write` (`internal/mcp/tools.go`) | `check: false`, else auto | auto | off unless `check: true`, which is auto |
| `mrw write --grep … --sub` and `--ast-grep … --rewrite` | the same flags, through `mrw write` | auto | off unless `--check` |
| `mrw check` | always runs | runs | unchanged |

## Existing Primitives Audit

- `writer.CheckMode` and its three values (`CheckAuto`, `CheckOff`, `CheckDemand`): unchanged. Only the value each surface starts from changes.
- `--check`, `--no-check` and their contradiction refusal: unchanged.
- The tests and contract rows that asserted the automatic check (28 Go tests across `cmd/mrw`, `internal/mcp` and `internal/writer` fail with the default flipped, measured 2026-11-01): each now asks for the check, so it keeps testing the check, and the default is tested on its own.

## Decision

1. **`mrw write` runs no check unless `--check` is given.** `--check` keeps its meaning: run the check after a write that landed, on any plan, exit 2 when none can run, exit 3 when it does not pass. `--no-check` stays accepted and does nothing; its help says the default is off.
2. **`mrw_write` runs no check unless `check: true`.** `check: true` runs it by ADR-054's rule (non-prose written, a check declared or inferred), so a host that passed it keeps what it had; `check: false` and an absent `check` run none.
3. **Steps still run.** `--then NAME` and `then` run their steps after a landed write; with no check they run alone, as after `--no-check` today.
4. **Nothing else changes.** `.quality-harness.json`, `scoped_check`, exit 3, the hints, `mrw check`, and every receipt key keep their meaning; a write that runs no check prints no check line and both receipts omit `check` (omitempty: a write that ran none carries no such key, as after `--no-check` today).

**What would make this decision wrong:** adopters who relied on the default to catch broken trees, which nothing here measures (no telemetry, ADR-009). They get it back with `--check`; the release notes call the change out as breaking.

## Alternatives Considered

- **A per-project switch in `.quality-harness.json`.** Rejected by the owner's answer, which chose default-off everywhere; a switch keeps the cost for every project that has not heard of it.
- **A time budget for the automatic check.** Rejected: it still runs the whole gate, only shorter, and a check cut by a budget is a check that did not pass (exit 3), which is noise.
- **Keep ADR-054.** Rejected by the owner.

## Component / Boundary Impact

`cmd/mrw` (the write command's default), `internal/mcp` (`mrw_write`'s default), their help and the MCP tool description. No engine package; `internal/writer` keeps its modes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` | no check unless `--check` | T1 | agents, scripts |
| `mrw_write` | no check unless `check: true` | T1 | MCP hosts |
| `mrw write --help`, the MCP tool description, `mrw instructions` | say the check is opt-in | T1 | agents |
| contract §322 | new rows | T1 | CI |
| README, AGENTS.md, docs/agents.md | the check is opt-in | T1 | readers |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a write costs the write; the project's own verification decides when the gate runs.
- **Negative:** a broken tree after a write is no longer caught unless the caller asks; exit 3 after a plain write becomes exit 0. Breaking for any script that relied on it.
- **Neutral:** `--no-check` becomes a no-op kept for scripts that pass it.

## Out of Scope

- Removing `--no-check` (permanent: boundary: scripts pass it; a no-op flag costs nothing)
- Changing `mrw check` or `scoped_check` (permanent: boundary: they run when asked, which is now the only time a check runs)
- Telling adopters before the release (external: Zy: the release notes)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script read exit 3 after a plain write as "the tree is broken" | Med | Med | breaking change named in the release; `--check` restores it |
| A test that asserted the check stops testing it | Low | Med | each of the 28 asks for the check now; §322 tests the default and `--check` side by side |

## Rollback

Revert T1; the default returns to ADR-054's.

## Follow-ups

- [ ] none at authoring
