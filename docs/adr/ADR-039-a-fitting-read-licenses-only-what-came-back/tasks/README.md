# ADR-039 Tasks

Implementation tasks for ADR-039: A fitting read licenses only what came back. See the parent ADR
for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A fitting serve is held pending, per file | done | — | `go test ./internal/mcp/ -run 'TestAFittingReadLicensesOnlyWhatCameBack\|TestAFittingReadHoldsPendingPerFile\|TestAReadUnderTheLimitIsUnchanged'` |
| T2 | The CLI still licenses a fitting read without ack | done | — | `go test ./internal/mcp/ -run 'TestACLIReadStillLicensesWithoutAck'` |
| T3 | The contract drives a fitting serve, and the wire says served not only paged | done | — | `grep -q '^# 77\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | fitting serves held pending per file | T2, T3 | T1 first — there is nothing to refuse until Record-on-serve is gone |
| T2 | CLI fitting reads still license without ack | T3 | T2 before T3 — CLI rows of `contract.sh` must stay green; §77 is the MCP half |

## Notes

- ⚠ **The two-file fixture is load-bearing.** `hold` today picks one path. A single-file test is
  green against that code. T1 must ack file A and write file B.
- ⚠ **`TestAReadUnderTheLimitIsUnchanged` currently asserts the hole.** T1 rewrites it. Leaving it
  is how Record-on-serve keeps a green suite.
- Engine go/no-go: this record owns `internal/mcp` only. `internal/read`, `internal/apply`,
  `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical
  against the merge-base.
- ⚠ Breaking for MCP callers, deliberately and safely: a fitting read licenses nothing until
  acked. The CLI is unchanged.
- Do not raise `maxInstructionsChars`. Do not generate `AGENTS.md` from `guide.Shared()`.
