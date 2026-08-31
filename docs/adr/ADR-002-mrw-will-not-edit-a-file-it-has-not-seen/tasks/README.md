# ADR-002 Tasks

Implementation tasks for ADR-002: mrw will not edit a file it has not seen. See
the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

**This ADR is a retrofit.** Both tasks describe behaviour that shipped in
`829aae7` and released as v0.0.2. They are `done` as of 2026-08-31: each carries a tool-written exit-0
Verification Log entry and a killed mutant.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A ledger records what mrw last observed each file to hold | done | — | `go test ./internal/seen/ -v … && go test ./internal/seen/ ./internal/read/ ./cmd/mrw/` |
| T2 | An edit to an unseen or externally-changed file is refused, naming both SHAs | done | — | `go test ./internal/apply/ -run 'TestAFileNeverSeen…' … && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `seen.Ledger`, `seen.Load`, `seen.Record`, `seen.SHA` | T2 | T1 before T2 — T2's check reads a ledger T1 defines |

## Notes

- The mutation for T2 has already been performed by hand once: replacing the
  guard condition with `if false` turned both refusal tests red and made
  `scripts/contract.sh` report 4 failed assertions with exit 1. Executing T2
  means re-recording that through `adr-verify --mutant` so the evidence is
  tool-written rather than described here.
- T1 ships `seen.Forget` with no CLI caller. That is deliberate and recorded as
  deferred in the parent ADR, so it is visible rather than quietly dead.
