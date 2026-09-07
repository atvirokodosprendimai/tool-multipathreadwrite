# ADR-032 Tasks

Implementation tasks for ADR-032: The ceiling is the caller's, and it bounds the whole answer.

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The budget is configured, and it bounds the encoded result | pending | — | `go test ./internal/mcp/ -run 'TestTheAdvertisedCeilingBoundsEveryAnswer' …` |
| T2 | A write receipt that fits | pending | — | `go test ./internal/mcp/ -run 'TestAWriteReceiptElidesSuccessesNotFailures' …` |
| T3 | The contract drives the ceiling | pending | — | `grep -q '^# 70\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the configured budget | T2, T3 | T1 first — there is no budget to bound a receipt with until it exists |
| T2 | the bounded write receipt | T3 | T2 before T3 — §70 drives the built server |

## Notes

- ⚠ **Measured, not assumed:** a 4,000-hunk dry-run through `mrw_write` returned 453,632 characters against an advertised 200,000 (2026-09-07, `267c453`).
- ⚠ **The bound is on the ENCODED result**, not the report text. ADR-031 shipped a size check on a buffer that was not what got sent, twice; this record's tests assert against the JSON the server writes.
- ⚠ A FAILED hunk is never elided. The counts and the failures are the floor.
- `0` means zero and absence means the default, per ADR-033. Not re-argued here.
