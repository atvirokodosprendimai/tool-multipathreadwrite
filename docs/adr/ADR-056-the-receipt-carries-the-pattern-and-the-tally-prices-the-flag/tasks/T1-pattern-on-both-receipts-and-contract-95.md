# Task ADR-056-T1: `pattern` on both JSON receipts; contract §95

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `authoring.PatternInfo`; `pattern` on `mrw write --json` and `mrw_write`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `pattern always present`, `both transports`, `fires on the third`, `schema describes it`

## Goal

The CLI prints a `pattern:` line; neither JSON receipt carries the fact. Put a `pattern` object `{advisory_writes, window, fires}` on both, on every write, read from the ring after this write joined it. The transport-equality contract row must keep passing, which is why it is both and not one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/authoring/authoring.go` | edit | `PatternInfo` + `PatternOf(root)`. |
| `cmd/mrw/main.go` | edit | `receipt.Pattern`, set after `RecordRecent`. |
| `internal/mcp/tools.go` | edit | `writeReceipt.Pattern`, set after `RecordRecent`. |
| `internal/mcp/schema.go` | edit | Describe `pattern` and its three members; the conformance test refuses an undescribed key. |
| `internal/mcp/pattern_test.go` | create | Red: three delta writes over MCP; first receipt `fires:false`, third `fires:true, advisory_writes:3, window:3`; a refused plan still carries the object. |
| `cmd/mrw/advisory_test.go` | edit | Red: `--json` receipt has `pattern`. |
| `scripts/contract.sh` | edit | **§95** — next free after §94. |

## Ordered Steps

1. [S1] Write `TestTheMCPReceiptCarriesThePattern` and `TestTheCLIJSONReceiptCarriesThePattern` — RED. [proof: mutation]
2. [S2] Implement `PatternInfo`, both fields, four schema rows. S1 GREEN. [proof: mutation]
3. [S3] Write §95 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 95\. ' scripts/contract.sh \
  && go test ./internal/mcp/ ./cmd/mrw/ -count=1 -v \
    -run 'TestTheMCPReceiptCarriesThePattern|TestTheCLIJSONReceiptCarriesThePattern|TestEveryOutputSchemaPropertyIsDescribed' 2>&1 | tee /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestTheMCPReceiptCarriesThePattern' /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestTheCLIJSONReceiptCarriesThePattern' /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestEveryOutputSchemaPropertyIsDescribed' /tmp/adr056-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr056-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/mcp/ ./cmd/mrw/ ./internal/authoring/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheMCPReceiptCarriesThePattern` | `internal/mcp/pattern_test.go` | present on write 1 with `fires:false`; write 3 `fires:true, 3, 3`; present on a refused plan | — | S1, S2 |
| `TestTheCLIJSONReceiptCarriesThePattern` | `cmd/mrw/advisory_test.go` | `--json` has `pattern.window == 1` after one write and `fires` false | — | S1, S2 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | the four new keys are described | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §95 |
| 2 — something selects it | every write sets it; deleting the assignment fails S1 |
| 3 — the caller can discover it | the output schema describes it; T3 names it |
| 4 — it is used | ADR-055's motivating caller read the JSON and not the row |

## Mutation Log
_(tool-written)_
- 2026-09-13 · 9a1ba2a* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP receipt carries an empty pattern: the key is present but never the ring, and the third-write assertion in TestTheMCPReceiptCarriesThePattern must go red · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e
- 2026-09-13 · 9a1ba2a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI receipt carries a constant pattern: advisory_writes is never read from the ring, and TestTheCLIJSONReceiptCarriesThePattern must go red · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e
- 2026-09-13 · 9a1ba2a* · mutant inconclusive · exit 1 · `internal/authoring/authoring.go` · fires is never set: the third write reports fires:false and the MCP test must go red · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-13 · 9a1ba2a* · mutant killed · exit 1 · `internal/authoring/authoring.go` · fires is inverted: the first write reports fires:true and the third fires:false, and both halves of TestTheMCPReceiptCarriesThePattern must go red (the earlier deletion left fires unused and was inconclusive) · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e

## Invariants

- `pattern` is present on every receipt, both transports, whether or not it fires.
- The human `pattern:` line is unchanged.
- The existing transport-equality contract row is not edited.

## Risks

| Risk | Mitigation |
|------|------------|
| Ring state differs between the CLI and MCP fixtures in the equality row | both are fresh roots with one landed write: `{1 or 0, 1, false}` on both |

## Stop Condition

Stop if the equality row cannot pass with the field on both sides: that means the two transports record the ring differently, which is a T2-of-ADR-055 bug, not this task's.

## Out of Scope

- Anything about the pricing (T2).

## Verification Log
_(tool-written)_
- 2026-09-13 · 9a1ba2a* · exit 1 · `set -o pipefail …` · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e · ms:60 · test-lock-sha256:814b0b4b691dd32d149d41cac12e9078368461ea67e38a4e8c72d5cb69a948bc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYWR2aXNvcnlfdGVzdC5nbwlUZXN0UXVpZXRBbmRKU09OQ2FycnlUaGVBZHZpc29yeUNvdW50CTVjODliNjNjYWMzM2E4YWYwNzczZGViMGUyNDIxNTY4NmNjOGI1MDI3MGNmZmEwMDcyNDhmYjI3NGI3MGI3MTAKYm9keQljbWQvbXJ3L2Fkdmlzb3J5X3Rlc3QuZ28JVGVzdFRoZUNMSUpTT05SZWNlaXB0Q2Fycmllc1RoZVBhdHRlcm4JNDI5YzRjNDI1OTFhODkyZDI3NzI0NGMwZjliNWMxYTViZjMyMjIxMDU1ZWIyZDFlNDQ0YjA2MTA5NzFmNzU1MQpib2R5CWNtZC9tcncvYWR2aXNvcnlfdGVzdC5nbwlUZXN0VGhlUmVjZWlwdFByaW50c1RoZVBhdHRlcm5MaW5lT25UaGVUaGlyZEFkdmlzb3J5CWNhZTdmODc3Y2E2NTQyZTk3NGVhOWE5NmI5Yzc1ZDU2NTY4ZGMzOTk4NmQ2OGY5MzMzNjFjN2JjMWNiNzFjYWIKYm9keQljbWQvbXJ3L2Fkdmlzb3J5X3Rlc3QuZ28JVGVzdFRoZVN1bW1hcnlMaW5lQ291bnRzQWR2aXNvcmllcwllMTA4OTYwMGY0NWY4YjlmODY0MWM5ZjVmNmFjMWU4NmQyMDZhMTVkMTA0NWE4OGIxYTI5MGE4ZjZkOTUyMWEzCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEFSZWFkUmVzdWx0Q2Fycmllc05vU3RydWN0dXJlZENvbnRlbnQJNWI0MTUwZTVjMmUwNTlkYzcxMTM0NDg2ZjJhOWQ2MGJiYTRhZGI1M2FjMTBhOWMxYmY0MWM5OTQ2ZGUzNTliOApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeURlY2xhcmVkT3V0cHV0U2NoZW1hVmFsaWRhdGVzQVJlYWxSZXNwb25zZQljNjBlYTkyNGU2MWIwM2JhYzUxMDUyNjA2ZjhiODJkNmMyMmJmN2Y2OTE0MmQyMmQ1ZDE0NzM3YTA0ODJlZWY5CmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEV2ZXJ5RW1iZWRkZWRFeGFtcGxlUGxhblJlYWxseUFwcGxpZXMJOWFhZmE3MjE3MmI2Y2E1YmM1MjkwNzg2YTJhMzZlOWE5NDlkNGVjYjIzNGIwMTZhYWM5MWI0YTRjNTM0OGFlNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeU91dHB1dFNjaGVtYVByb3BlcnR5SXNEZXNjcmliZWQJNjlmYjBlOGQ4MjNiOWEyZTFkMWIzOTEyMDYxMDcwMDBmNjNhY2YxNDM1ODI4OTMyYzE5YzQyNzZmYmM3N2MzNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RUaGVBbm5vdGF0aW9uc01hdGNoV2hhdFRoZVRvb2xEb2VzCTU1ZWNlMTZjMDhhNjEwMGM1MTlmZjFjZDY1MWY1ZmQxYzk3NjAzM2Y4M2U4OTJhOGM3ZDU0M2JhNWY4Njg4MGEKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0VGhlRmlyc3RDb250ZW50QmxvY2tJc1RoZVNlcmlhbGl6ZWRTdHJ1Y3R1cmVkQ29udGVudAlhNTliOGVmMDUxMWJkMTI1MjRlM2QwMzE2YjYxZWUwMjc1N2U4Mzg3YTgyYzU4MjI0OWQzMzE1N2QxNjY4ZThjCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdFRoZVN0YXR1c0Rlc2NyaXB0aW9uTmFtZXNUaGVWYWx1ZXNUaGVFbmdpbmVTZW5kcwllNTZiZDIzZWExZTc4ZjFhZTFlZTI3MjUxM2FiODU5NTE3ZTU3Y2MxMzkxZmY4NWY0MzFmZjUwNmZjNWE5Zjg4CmJvZHkJaW50ZXJuYWwvbWNwL3BhdHRlcm5fdGVzdC5nbwlUZXN0VGhlTUNQUmVjZWlwdENhcnJpZXNUaGVQYXR0ZXJuCTBjNzliOTI2MDM3NTNiZWM0MzY0ZTY2NzlhNTE5YzVjNDVkNWEyNzJmZGI4N2JiZmJkZGU3NTZlYjRjYzQ5NDk
  ```
  ```
- 2026-09-13 · 9a1ba2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e · ms:761
- 2026-09-13 · 9a1ba2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e · ms:577
- 2026-09-13 · 9a1ba2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e · ms:567
- 2026-09-13 · 9a1ba2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:7136eddc0641d396d3f3bc1b1c7e2fd0f02e494f51978b8288273e9649dddb7e · ms:578
