# ADR-024 Tasks

Implementation tasks for ADR-024: A page is known by its served text, not by an error flag. See the
parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A partial answer announces itself in the served text and is not flagged an error | done | — | `set -o pipefail; go test ./internal/mcp/ -run 'TestAPageIsKnownByItsServedText' -count=1 2>&1 \| tee /tmp/adr024-t1.out && ! grep -qE "no tests to run\|^FAIL\|^--- FAIL" /tmp/adr024-t1.out && go test ./internal/mcp/... -count=1 && gofmt -l internal/mcp && go vet ./internal/mcp/...` |
| T2 | The contract drives a paged read through the built server and fails if it is flagged | pending | — | `set -o pipefail; grep -q '^# 62\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `pagedResult()`, `indexResult()` and the served-read path return `isError` absent | T2 | T1 before T2 — §62 asserts against the built binary what T1 changes, so the row cannot go green until the change exists |

## Notes

- T1's fence is scoped to `./internal/mcp/...`, the only package this record touches. The engine
  go/no-go is a separate check and is listed in T1's Invariants rather than folded into the fence.
- T2's fence begins with a `grep` for `^# 62. ` so it is RED before the row exists. Without it,
  `./scripts/contract.sh` passes on the current tree and the gate would go green with nothing done.
- The A/B measurement in the parent ADR's Context is a host observation and is deliberately NOT in
  either fence: no command in this repository can see what a host discards. §62 proves what the
  server sends; the record carries what the host did with it.
