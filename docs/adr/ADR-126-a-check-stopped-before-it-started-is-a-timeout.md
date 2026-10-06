# ADR-126: a check stopped before it started is a timeout, exit 3

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "plan these fixes then", approving the plan whose item 2 is this record (BACKLOG "From ADR-120"); the Codex review of v1.42.0..v1.47.0 found its Windows case the same day. The record's text was drafted after that approval.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-080, ADR-113, ADR-120
**Invalidates:** None — ADR-080's "interrupted before it started" keeps its meaning; a deadline joins it
**Governs:** `internal/check/check.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/timeout126_test.go::TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3`
**Served-path change:** a check whose deadline passed before it started — a deadline already gone when the write reached its check, or one that landed while Windows was putting the child in its job (ADR-120) — is reported "timed out before it started", exit 3, the same exit a check that times out while running gives, where it said "no check could run: timed out", exit 2.

## Context

`check.run` (`internal/check/check.go:292`) reads a check whose process never started as "could not start", `Ran == false`, which the write path (`cmd/mrw/main.go:1419`) and `mrw check` (`:1913`) report as "no check could run", exit 2: the configuration problem ADR-003 files a missing check under. ADR-080 already took out the interrupt: a cancel before the start is `Interrupted`, exit 3. A deadline before the start was left: `check.go:396` overwrites the reason with "timed out after …", `Ran` stays false, and the caller is told exit 2 for what exit 3 means — the write applied, nothing verified it.

Two routes reach it: a deadline already past when the check starts (BACKLOG "From ADR-120", the in-process review of #325), and on Windows a deadline that lands while the child is being contained, since a child killed before it was resumed carries no process state (`internal/subproc/job.go:123`; the Codex review of v1.42.0..v1.47.0, finding 1).

**Audit of the class** — *an exit mapping that reads a check that did not run*: `mrw read --grep '!receipt.Check.Ran|!res.Ran|!v.Check.Ran' cmd/mrw/main.go internal/mcp/tools.go` — the write path (1413-1423), `mrw check` (1910-1918), and `mrw_write`'s isError (`tools.go:891`).

## Existing Primitives Audit

- **`check.Interrupted`** (ADR-080) — the before-start reason that already maps to exit 3; the timeout joins it through one predicate both CLI sites ask.
- **`TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted`** — the shape of the CLI test: a context done before the run, driven through `rootCommand().Run`.

## Decision

1. **A deadline that passed before the check started is `TimedOutBeforeStart`**: `Skipped` reads "timed out before it started (the N limit had passed)", `Ran` stays false, `exit_code` stays -1. No receipt key changes.
2. **`check.StoppedBeforeStart(r)`** answers whether a check that did not run was stopped — interrupted or timed out — rather than unable to start; the write path and `mrw check` both ask it and exit 3, naming which.
3. **`mrw_write` is unchanged**: a check that did not run is isError with the receipt (ADR-113 Decision 4), interrupted before it started included; the timeout joins it there too.

## Alternatives Considered

- **Map the timeout to exit 3 by reading `Skipped` text in main.go** — rejected: two sites matching a free-form prefix is the drift ADR-080's constant exists to prevent.
- **Report a pre-start timeout as `Ran: true`** — rejected: the receipt would say a process ran that never did (ADR-101).

## Component / Boundary Impact

Owns `internal/check` (engine): one constant, one predicate, one branch. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a write whose check's deadline passed before it started | exit 3, "timed out before it started" | T1 | CLI callers |
| `mrw check`, the same | exit 3 | T1 | CLI callers |
| `scripts/contract.sh` | none — a deadline before a start is not reachable through the built binary without a race; `cmd/mrw/timeout126_test.go` drives the command itself | T1 | — |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** an applied write whose check never got to run for lack of time exits 3, as every other unverified write does.
- **Negative:** a caller that keyed on exit 2 for this case sees 3; that caller was reading "fix the configuration" for a timeout.
- **Neutral:** `mrw_write` is unchanged.

## Out of Scope

- MCP isError for a check that did not run (permanent: boundary: ADR-113 Decision 4 keeps it, with the receipt)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a "could not start" whose deadline also expired is reported as a timeout | Low | Low | the deadline is the reason the start failed or did not matter; either way the write is unverified, exit 3 |

## Rollback

Revert the task: the pre-start timeout is exit 2 again.

## Follow-ups

- None — the record carries no open follow-up.
