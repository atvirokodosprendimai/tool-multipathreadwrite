# Task ADR-032-T2: A write receipt that fits

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (bounding one receipt, and choosing what to drop)
**Owner:** Zy
**Produces:** the bounded write receipt
**Consumes:** the configured budget replacing the constant (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write receipt within the advertised budget`, `every failed hunk surviving the elision`

## Goal

Keep `mrw_write` inside the number it advertises, without losing the verdicts a caller must act on.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | The write receipt is bounded like a read's. Over budget, SUCCESSFUL hunks are elided and the counts and every FAILED hunk are kept — a failure is what a caller acts on, and ADR-001's "nothing was written" verdict is carried by the counts. |
| `internal/mcp/limit_test.go` | edit | `TestAWriteReceiptElidesSuccessesNotFailures`. |

## Ordered Steps

1. [S1] Write `TestAWriteReceiptElidesSuccessesNotFailures` and confirm it is RED: a plan with thousands of successes and a handful of failures returns every failure, the counts, and a notice naming what was elided. [proof: mutation]
2. [S2] Bound the receipt with the configured budget. [proof: mutation]
3. [S3] Elide successes first and never a failure; state the elision in the text rather than silently shortening, which is ADR-014's rule for a partial answer. [proof: mutation]
4. [S4] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestAWriteReceiptElidesSuccessesNotFailures' 2>&1 | tee /tmp/adr032-t2.out \
  && grep -q '^--- PASS: TestAWriteReceiptElidesSuccessesNotFailures' /tmp/adr032-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr032-t2.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteReceiptElidesSuccessesNotFailures` | `internal/mcp/limit_test.go` | An oversized write receipt fits the budget, carries every FAILED hunk and the counts, and says what it elided | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAWriteReceiptElidesSuccessesNotFailures` |
| 2 — something selects it | Every `mrw_write` call composes its receipt through it; the S3 mutation elides failures too and the fence goes red |
| 3 — the caller can discover it | The elision notice in the receipt, and `_meta` |
| 4 — it is used | Contract §70 (T3) drives the built server |

## Mutation Log

## Invariants
- A FAILED hunk is never elided. ADR-001's all-or-nothing means a failure explains why nothing was written, and dropping it would leave the caller the one fact they cannot act without.
- The elision is stated, not silent — ADR-014's rule for any partial answer.
- The counts survive: `applied`, `failed` and `skipped` are what a caller checks first.

## Risks

- Eliding the wrong end. S3's mutant elides failures and must go red.
- A receipt small enough to fit but useless. The counts plus every failure is the floor, and the test asserts it rather than a size alone.

## Stop Condition

Stop and ask if the budget cannot fit the counts and every failure for some plan — that would mean a
receipt this record cannot honestly bound, and the answer is a refusal rather than a lie.

## Out of Scope

- The configured budget itself — T1
- The contract row — T3

## Verification Log
