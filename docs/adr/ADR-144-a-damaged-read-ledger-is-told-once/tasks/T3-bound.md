# Task ADR-144-T3: The ledger scan reads a bounded amount

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `maxScanBytes`
**Consumes:** `seen.DamageNotice` (T1, T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a ledger past the scan bound is not read whole and says nothing`

## Goal

The second Codex review of PR #390: T2 closed the file before parsing by reading it whole with an unbounded `io.ReadAll`, so a multi-gigabyte hostile ledger could exhaust memory where `Load` streams it. The read is bounded.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `maxScanBytes`, and the read is `io.LimitReader`-bounded |
| `internal/seen/damage144_test.go` | edit | the test |

## Ordered Steps

1. [S1] Write `TestAHugeLedgerIsNotReadWhole`. Confirm RED.
2. [S2] `maxScanBytes` (64 MiB) and the bounded read; a ledger past it says nothing. Mutant: the read is unbounded. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/seen/ -count=1 -timeout 600s -run 'TestAHugeLedgerIsNotReadWhole' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAHugeLedgerIsNotReadWhole \(' "$out" \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAHugeLedgerIsNotReadWhole` | `internal/seen/damage144_test.go` | with the bound lowered, a ledger past it says nothing; within it the bad lines are counted | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `maxScanBytes` |
| 2 — something selects it | `DamageNotice`, which `Before` calls for `read`, `write` and `seen` |
| 3 — the caller can discover it | a ledger past 64 MiB says nothing, as it did before ADR-144; the boundary is in the record |
| 4 — it is used | the test lowers the bound; no telemetry (ADR-009) |

## Invariants

- A ledger within the bound is counted as in T1.
- `Load` is unchanged.

## Risks

- A legitimate ledger past 64 MiB is not scanned: it would hold roughly half a million observations; a notice for it is the price of a bounded read.

## Stop Condition

Stop and ask if a real ledger past the bound is met.

## Out of Scope

- A damaged ledger past the scan bound (permanent: boundary: the scan reads whole to close the file first, and a bound is what keeps that read from exhausting memory; `Load` still serves what parses)

## Mutation Log

## Verification Log
- 2026-10-10 · e7dc360* · exit 1 · `set -o pipefail …` · acceptance-sha256:cb7cfb97a9aee105de45e8e5fe5088d1aa321b49ea3cc24ca0e8a5f2e3bb6ff7 · ms:237 · test-lock-sha256:e3fef53aee13a6ce7e935416434a2f78a1dde79f8331c4d844c8b674175f6658 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvc2Vlbi9kYW1hZ2UxNDRfdGVzdC5nbwlUZXN0QURhbWFnZWRMZWRnZXJJc0NvdW50ZWRBbmRUb2xkCTkwZTYyZWNlNjY1MjMwMDcxMmIwYTYwN2FmYjRkZWYwMTRjMDBjZDBiZWY4YWM3YTAxMWQ1OGNhMmE3NjNmMTIKYm9keQlpbnRlcm5hbC9zZWVuL2RhbWFnZTE0NF90ZXN0LmdvCVRlc3RBSHVnZUxlZGdlcklzTm90UmVhZFdob2xlCTUxY2M2MGVkM2U5MWNkZmZlYTAxODFiMzFmYzAyMmU4MzljYzg3YzdhNThhODYzNmVlY2RlMmUxMTg5NGUxZjIKYm9keQlpbnRlcm5hbC9zZWVuL2RhbWFnZTE0NF90ZXN0LmdvCWEgbGVkZ2VyIG1ydyB3cm90ZSBzYXlzIG5vdGhpbmcJZDFlNmZkZjQ5MzMzOWNhYjU1MzU2ODE1YjgyN2UwMDg0YWFjZjM5MDM2MmY0OTAyNTk4MGMyZDU1MDI3ZTBjNApib2R5CWludGVybmFsL3NlZW4vZGFtYWdlMTQ0X3Rlc3QuZ28JYSBsaW5lIHBhc3QgdGhlIHJlY29yZCBib3VuZCBpcyB0b2xkIGFzIGRpc2NhcmRlZAk0YmY3MWQ0NGQ0MjUzOThhNmI3ZmE3MDkzYjA5YjdmNjgxMGEzZWVlMzgzNjZlMWIyMGJlYTBjZTVhNjAwZjg0CmJvZHkJaW50ZXJuYWwvc2Vlbi9kYW1hZ2UxNDRfdGVzdC5nbwlhIHN0YWxlLCBhbiBlbXB0eSBhbmQgYSBtaXNzaW5nIGxlZGdlciBzYXkgbm90aGluZwkwN2I2YWFkYWMwOWY4YmNlY2M1OGQxODY0MGY3MDczMGM3YTU2M2NjY2ZmYmNiMzE2ZDQwMjcyMTQxNjdlYTZiCmJvZHkJaW50ZXJuYWwvc2Vlbi9kYW1hZ2UxNDRfdGVzdC5nbwlnYXJiYWdlLCBhbiBlbXB0eSBsaW5lIGFuZCBOVUxzIGFyZSBjb3VudGVkCTU4N2FjZjIxOWEyZGQzNGYwM2NiNzEyMTU1OGViZDJkNDJlMDM3ZGJmNjMwZmI5ODk1NTc3ZGY0YmNlMjJhMjM
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen.test]
  internal/seen/damage144_test.go:109:9: undefined: maxScanBytes
  internal/seen/damage144_test.go:110:2: undefined: maxScanBytes
  internal/seen/damage144_test.go:112:2: undefined: maxScanBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [build failed]
  FAIL
  ```
