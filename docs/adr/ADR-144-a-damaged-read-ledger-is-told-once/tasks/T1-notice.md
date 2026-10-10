# Task ADR-144-T1: The CLI tells a ledger it had to ignore lines of

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `seen.DamageNotice`, the branch in `Before`
**Consumes:** `parseLine`, `scanLF`, `maxRecordBytes`, `seen.IsStale` (existing)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a ledger with lines mrw cannot parse, or a line past the record bound, is named once and the next run is silent`

## Goal

Decisions 1 to 5 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `DamageNotice` |
| `cmd/mrw/main.go` | edit | `Before` prints it after the stale check |
| `internal/seen/damage144_test.go` | add | the counts, the silences, the bound |
| `cmd/mrw/damage144_test.go` | add | the sentence on stderr, once, and not on `version` |
| `scripts/contract.sh` | edit | §248 |

## Ordered Steps

1. [S1] Write `TestADamagedLedgerIsCountedAndTold` and `TestADamagedLedgerIsToldOnceByTheCLI`. Confirm RED.
2. [S2] `DamageNotice`: header check, `parseLine` count, the too-long branch. Mutant: the count not incremented. [proof: mutation]
3. [S3] The branch in `Before`, skipped for `version`, `instructions` and `stats`. Mutant: the branch removed. [proof: mutation]
4. [S4] Contract §248 drives the built binary with a garbage line and a clean ledger. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestADamagedLedgerIsCountedAndTold|TestADamagedLedgerIsToldOnceByTheCLI' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestADamagedLedgerIsCountedAndTold \(' "$out" \
  && grep -qE '^--- PASS: TestADamagedLedgerIsToldOnceByTheCLI \(' "$out" \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && grep -q '^# 248\. ' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestADamagedLedgerIsCountedAndTold` | `internal/seen/damage144_test.go` | garbage, empty and NUL lines are counted; a clean, missing, empty or stale ledger says nothing; a line past the bound is told as discarded | none | S1, S2 |
| `TestADamagedLedgerIsToldOnceByTheCLI` | `cmd/mrw/damage144_test.go` | `read` prints the sentence once, the ledger it writes is clean so the next run is silent, and `version` prints nothing | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `seen.DamageNotice` |
| 2 — something selects it | `Before` in `cmd/mrw/main.go`, which every command passes |
| 3 — the caller can discover it | the sentence names the cause and the remedy |
| 4 — it is used | contract §248 drives the built binary; no telemetry (ADR-009) |

## Invariants

- `Load` is unchanged: a damaged ledger still loads as far as it parses, and a too-long line still discards it.
- A ledger `Record` wrote reports nothing.
- Exit codes and receipts are unchanged.

## Risks

- A second scan of the ledger per command; bounded by the ledger, which `Load` scans anyway.

## Stop Condition

Stop and ask if a ledger mrw itself wrote ever reports damage (the notice would then be a false alarm).

## Out of Scope

- The MCP server telling the caller (permanent: boundary: it has no stderr)

## Mutation Log

## Verification Log
- 2026-10-10 · 00a4331* · exit 1 · `set -o pipefail …` · acceptance-sha256:2518aca2d7b370f94fd540c5281cc6f2dac90b8a9476d459428cb819d122eaea · ms:545 · test-lock-sha256:3e0fc6e50906e99d6c0bce456a5884c86f37efc99e17eaa774017df8bba404f5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kYW1hZ2UxNDRfdGVzdC5nbwlUZXN0QURhbWFnZWRMZWRnZXJJc1RvbGRPbmNlQnlUaGVDTEkJYTQ1YjcxMGM1NTBlNGVlNGQ1MjU3ZWFlNzRjZjdiNzRhNDBmODY4N2I3Yjk2ZDI3N2RkYTU2OGJiOTY5OWQ5Ngpib2R5CWludGVybmFsL3NlZW4vZGFtYWdlMTQ0X3Rlc3QuZ28JVGVzdEFEYW1hZ2VkTGVkZ2VySXNDb3VudGVkQW5kVG9sZAk5MGU2MmVjZTY2NTIzMDA3MTJiMGE2MDdhZmI0ZGVmMDE0YzAwY2QwYmVmOGFjN2EwMTFkNThjYTJhNzYzZjEyCmJvZHkJaW50ZXJuYWwvc2Vlbi9kYW1hZ2UxNDRfdGVzdC5nbwlhIGxlZGdlciBtcncgd3JvdGUgc2F5cyBub3RoaW5nCWQxZTZmZGY0OTMzMzljYWI1NTM1NjgxNWI4MjdlMDA4NGFhY2YzOTAzNjJmNDkwMjU5ODBjMmQ1NTAyN2UwYzQKYm9keQlpbnRlcm5hbC9zZWVuL2RhbWFnZTE0NF90ZXN0LmdvCWEgbGluZSBwYXN0IHRoZSByZWNvcmQgYm91bmQgaXMgdG9sZCBhcyBkaXNjYXJkZWQJNGJmNzFkNDRkNDI1Mzk4YTZiN2ZhNzA5M2IwOWI3ZjY4MTBhM2VlZTM4MzY2ZTFiMjBiZWEwY2U1YTYwMGY4NApib2R5CWludGVybmFsL3NlZW4vZGFtYWdlMTQ0X3Rlc3QuZ28JYSBzdGFsZSwgYW4gZW1wdHkgYW5kIGEgbWlzc2luZyBsZWRnZXIgc2F5IG5vdGhpbmcJMDdiNmFhZGFjMDlmOGJjZWNjNThkMTg2NDBmNzA3MzBjN2E1NjNjY2NmZmJjYjMxNmQ0MDI3MjE0MTY3ZWE2Ygpib2R5CWludGVybmFsL3NlZW4vZGFtYWdlMTQ0X3Rlc3QuZ28JZ2FyYmFnZSwgYW4gZW1wdHkgbGluZSBhbmQgTlVMcyBhcmUgY291bnRlZAk1ODdhY2YyMTlhMmRkMzRmMDNjYjcxMjE1NThlYmQyZDQyZTAzN2RiZjYzMGZiOTg5NTU3N2RmNGJjZTIyYTIz
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/seen/damage144_test.go:70:18: undefined: DamageNotice
  internal/seen/damage144_test.go:76:18: undefined: DamageNotice
  internal/seen/damage144_test.go:79:18: undefined: DamageNotice
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [build failed]
  === RUN   TestADamagedLedgerIsToldOnceByTheCLI
      damage144_test.go:45: read printed "", want one line counting the bad line
  --- FAIL: TestADamagedLedgerIsToldOnceByTheCLI (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.131s
  FAIL
  ```
