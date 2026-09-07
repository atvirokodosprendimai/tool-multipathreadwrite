# ADR-028 Tasks

Implementation tasks for ADR-028: A guard does not read back what was not served. See the parent ADR
for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The anchor check moves below the ledger | done | — | `go test ./internal/adversarial/ -run 'TestAFailedAnchorDoesNotReadBackAnUnservedLine' …` |
| T2 | The contract drives both halves through the built binary | done | — | `grep -q '^# 65\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the anchor check evaluated only for lines the ledger licensed | T2 | T1 before T2 — the row drives the built binary, so the ordering must exist before the row can be red for the right reason |

## Notes

- ⚠ The fixture must serve a NARROW range and address a line just outside it. `docs/adr/BACKLOG.md:226` pre-registers the trap: a fixture that serves nothing is refused by the whole-file gate before either guard runs, so it passes with the ordering reversed and proves nothing.
- The engine go/no-go applies: `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base. This record owns `internal/apply` only.
