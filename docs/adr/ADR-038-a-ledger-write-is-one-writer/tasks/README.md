# ADR-038 Tasks

Implementation tasks for ADR-038: A ledger write is one writer, even across processes. See the
parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `Record` holds the lock for load-merge-save | done | — | `go test ./internal/seen/ -run TestConcurrentRecordsKeepEveryPath …` |
| T2 | Contract §76 drives 40 processes through `$MRW` | done | — | `grep -q '^# 76\. ' scripts/contract.sh && …` |
| T3 | README and the handshake stop teaching the race | done | — | `go test ./internal/mcp/ ./cmd/mrw/ -run 'TestTheHandshakeDoesNotTeachALedgerRace\|TestReadmeDoesNotTeachALedgerRace'` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `Record` holds the lock | T2, T3 | T1 before T2 — the row is red for the right reason only after the lock exists. T3 may not teach a lock that is not there |
