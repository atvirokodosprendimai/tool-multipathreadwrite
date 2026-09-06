# ADR-025 Tasks

Implementation tasks for ADR-025: A read that served nothing is an error, on the path that serves.
See the parent ADR for the decision.

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
| T1 | The flag returns when a read served nothing | done | — | `go test ./internal/mcp/ -run 'TestAReadThatServedNothingIsAnError' …` |
| T2 | The contract drives a served-nothing read through the built server | done | — | `grep -q '^# 63\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the served-read return flags when `len(observed) == 0` | T2 | T1 before T2 — the row drives the built binary, so the condition must exist before the row can be red for the right reason |

## Notes

- The engine go/no-go applies: `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.
- ADR-024's `§62` and `TestAPageIsKnownByItsServedText` are invariants of both tasks, not targets. If either goes red, the scope is wrong.
