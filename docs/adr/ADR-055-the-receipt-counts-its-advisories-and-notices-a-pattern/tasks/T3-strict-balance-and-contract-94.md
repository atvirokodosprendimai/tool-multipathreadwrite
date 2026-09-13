# Task ADR-055-T3: `--strict-balance` opt-in refusal of the wrap-tail signature; contract §94

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `Options.StrictBalance` / `--strict-balance` (T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `signature refused`, `balanced replace applies`, `multi-line untouched`, `prose exempt`, `off by default`, `mcp flag`

## Goal

With `--strict-balance` (MCP `strict_balance`), a hunk is refused when `SrcOp == "replace"`, `Start == End`, the path is not prose, and some family among `{}` `()` `[]` has a non-zero net in the consumed line that the body does not match. The reason names the family and both nets. Refused means failed: siblings skip, nothing written, exit 1 (ADR-001). Off by default; the same plan without the flag applies with the balance row. Multi-line addresses are ADR-052's and untouched. Default stays off — T4 pre-registers the campaign that could change that.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options.StrictBalance`; the check beside the ADR-052 licence at the same resolve site. |
| `internal/apply/apply_test.go` | edit | Red: signature refused with both nets named; balanced single-line applies; multi-line address not touched by this check; `.md` exempt; off by default applies with Balance. |
| `cmd/mrw/main.go` | edit | `--strict-balance` flag into `Options`. The CLI selector. |
| `internal/mcp/tools.go` | edit | `strict_balance` argument into `Options`. The MCP selector (a flag on the existing tool, ADR-044). |
| `internal/mcp/mcp.go` | edit | Declare `strict_balance` on `mrw_write`'s input schema, beside `echo_pad`. |
| `internal/mcp/strictbalance_test.go` | create | Red: the declaration and the refusal through the MCP surface. |
| `scripts/contract.sh` | edit | **§94** — pair: flag + signature → exit 1, file unchanged, reason names `{ +1` / same plan no flag → exit 0 with `balance` row / flag + balanced body → exit 0 / flag + `.md` → exit 0. |

## Ordered Steps

1. [S1] Write `TestStrictBalanceRefusesTheWrapTailSignature` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestStrictBalanceLeavesBalancedMultiLineAndProseAlone` and `TestStrictBalanceIsOffByDefault` — RED until the option exists. [proof: mutation]
3. [S3] Implement the check and both selectors. Confirm GREEN. Deleting the `Start == End` guard must fail the multi-line half; deleting the net compare must fail S1. [proof: mutation]
4. [S4] Write §94 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 94\. ' scripts/contract.sh \
  && go test ./internal/apply/ ./internal/mcp/ -count=1 -v \
    -run 'TestStrictBalanceRefusesTheWrapTailSignature|TestStrictBalanceLeavesBalancedMultiLineAndProseAlone|TestStrictBalanceIsOffByDefault|TestWriteToolDeclaresStrictBalance|TestAnMCPWriteWithStrictBalanceRefusesTheSignature' 2>&1 | tee /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceRefusesTheWrapTailSignature' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceLeavesBalancedMultiLineAndProseAlone' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceIsOffByDefault' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestWriteToolDeclaresStrictBalance' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestAnMCPWriteWithStrictBalanceRefusesTheSignature' /tmp/adr055-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStrictBalanceRefusesTheWrapTailSignature` | `internal/apply/apply_test.go` | refused; reason names family, nets, flag; sibling skips; nothing written; no advisories | — | S1, S3 |
| `TestStrictBalanceLeavesBalancedMultiLineAndProseAlone` | `internal/apply/apply_test.go` | balanced single-line, multi-line address and `.md` all apply under the flag | — | S2, S3 |
| `TestStrictBalanceIsOffByDefault` | `internal/apply/apply_test.go` | without the flag the signature applies with a row and one advisory | — | S2, S3 |
| `TestWriteToolDeclaresStrictBalance` | `internal/mcp/strictbalance_test.go` | `strict_balance` is a boolean on `mrw_write`, described as an opt-in refusal | — | S3 |
| `TestAnMCPWriteWithStrictBalanceRefusesTheSignature` | `internal/mcp/strictbalance_test.go` | the MCP surface refuses the signature the same way | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §94 |
| 2 — something selects it | `--strict-balance` and `strict_balance` set `Options.StrictBalance`; deleting either selector fails its half of §94 |
| 3 — the caller can discover it | T4 on `write --help`; the refusal reason itself |
| 4 — it is used | opt-in; the pre-registered campaign (T4) is how use is measured |

## Mutation Log
(empty until execute)
- 2026-09-13 · 54beb7b* · mutant killed · exit 1 · `internal/apply/apply.go` · the single-line guard is deleted: a multi-line replace with differing nets is refused under the flag, which is the licence territory this check must not touch; the multi-line half of TestStrictBalanceLeavesBalancedMultiLineAndProseAlone must go red · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd
- 2026-09-13 · 54beb7b* · mutant killed · exit 1 · `internal/apply/apply.go` · the net compare is inverted: the signature applies and a balanced line is refused, so TestStrictBalanceRefusesTheWrapTailSignature and the balanced half must both go red · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd
- 2026-09-13 · 54beb7b* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP selector is cut: strict_balance is declared and ignored, and TestAnMCPWriteWithStrictBalanceRefusesTheSignature must go red · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd

## Invariants

- ADR-001: a refusal writes nothing; siblings skip; exit 1.
- ADR-048: arithmetic on one hunk; no parser.
- ADR-052: multi-line addresses are the licence's, not this check's.
- ADR-054: prose exempt; the balance row on an unflagged run is unchanged.
- Default off.

## Risks

- Braces in string literals refuse a correct single-line replace when opted in. The reason names both nets; the caller drops the flag for that plan. This is the bargain the flag states.

## Stop Condition

If the only way to go green is to refuse without the flag, or to touch multi-line addresses, stop.

## Out of Scope

- Advisory count (T1), pattern line (T2), teaching (T4)
- A default (BACKLOG pre-registration, T4)

## Verification Log
(empty until execute)
- 2026-09-13 · 54beb7b* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4851b3b7ff711061099f68e4d34bf9fcd4e669f39974273f4220aa9039ede65 · ms:80 · test-lock-sha256:32b282bb83d8e9caa34dbf2104446c9dedbea85db1837e684a8f3cd079ce195a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFCb2R5bGVzc0RlbGV0ZUlzVW5jaGFuZ2VkCWMzNzIyMjcyMTVjODkyMzcxYTRmYWJlZTE3N2U4NmU2YzdkYjcxZWRiMjdlYzJlYzZlMWUwODVjZDRiNTgxNjEKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBQ3JlYXRlQ2Fycmllc05vQmFsYW5jZQk4Yjk4ZDIwODEwOTI3M2RmYzFkODkwMjU3YjM0MWJhYmI2MTVhNTliYTQ0M2IxMmVjZmYxNTJjMmMzMDVhZmVkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZU9mQmxhbmtMaW5lc1N0aWxsUmVjb3Jkc0JvdW5kcwk4ZGUzYTQwN2I4ODg2MDg0ZjI2MDQ1ZjhjODA2NmY1NzI1ZDQ0NzUzZjY4YzBmNjRiZjBhMjk2MmZjYjQ1MGFkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZVdob3NlRXhwZWN0ZWRSZW1vdmFsRGlmZmVyc05hbWVzVGhlTGluZQllZjdmYTg4ZTFmYzRjYTJhMDUxYmE3YjYxYTE4MTJkZjc0NWI4NzMxYzY3Mzc5ZDZkYjliZTEzOTk4YmViMWFiCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZVdob3NlRXhwZWN0ZWRSZW1vdmFsTWF0Y2hlc0FwcGxpZXMJZGEzMWQ1NjhjZjI0NmZkZTIwNjFkZWIwOGEyNGNkYTJkZTcxMjNhNjQ3ZTFhNDNlY2UxMzhmMzU5NDU5YjMxOApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFEZWxldGVXaXRoQW5FeHBlY3RlZEJvZHlTdGlsbFJlcG9ydHNJdHNOZXQJNWNhOTIxZGY2NGY4ZmIyYTU0ZGRkOTNmYWY1YjgyMzgyMDZkNTkxYmJkYmQyZjVjNmE2NmQwNDNhYTNkNjI3MApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFEZWxpbWl0ZXJCYWxhbmNlRGVsdGFEb2VzTm90RmFpbFRoZUh1bmsJMzljYjVmYWYwNTQ2OTljZDRlZTcxYzRhMzVhNzRjOWQ1NTBmNmYyMTZmYzYyOWMyNTNjNGMwNTQ3ZjMxZWE5ZQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFGYWlsZWRGaWxlSXNSZXBvcnRlZE9uY2VOb3RQZXJIdW5rCTIwODQ0YTQ2YTg5ODg5YTk3ZmM3NDY5NTk2MGU1NDYzODQzZDFmM2VjOTA2ZmU4OTdjZjYxNzY0ZTE5Yzg1Y2UKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmFpbGVkT3JTa2lwcGVkRGVsZXRlUmVjb3Jkc05vQm91bmRzCWJjYzBmYzEyYzY2NjFmNDMyZDA1N2QwZmQxNGM2MDk0MmVhMTI0MGFmMDc5MDliNDNmMDMwNTRiNGZlZTUzM2MKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmFpbGVkU3RhZ2VMZWF2ZXNUaGVUcmVlVW50b3VjaGVkCWVhZTI2Njc3YzRlYmVlNGNhY2NiZTgwMzgzYTczZDE5NmNmNmNiNTI5NDIxYWQ0MDQ0Y2Q2ZmQyNzAwZDNhYjAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmlsZUNoYW5nZWRCZWhpbmRNcndzQmFja0Nhbm5vdEJlRWRpdGVkCTgwYjY2N2NiMDA1Mjk4ZjIwZGRlOTVmNjkzNzNiNmRmMDYxNWM2ZWVkY2U2NjYxNWVmZWYzMjgwMDY5NmQzYmIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmlsZU5ldmVyU2VlbkNhbm5vdEJlRWRpdGVkCTkzOGNlY2UwNzIzOWRiNDFmOWJjODQ3ZDFiMTZkNTM5NjI5YzE0M2EyZDIxYmEwZGRiODU1YjFlMTRjYzg3NjcKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBTXVsdGlMaW5lUmVwbGFjZVdpdGhBU2VydmVkTGluZUFmdGVyRW5kQXBwbGllcwk3ZmI2ODE0ZGVjNzVlNzE1NDg3YzA5MjA1NzUwOGFjMjU2NDM0OGJiNjk4ZmJmYzZiOTEzODE4MWUyYzgyYjc0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QU11bHRpTGluZVJlcGxhY2VXaXRoQW5BbmNob3JTdGlsbEFwcGxpZXMJYTRiNGI1OGJlNWVlODE2ZWM5NDc5ZmNmZWQ3MDdiMmM5MjJiMzkwYWM0OWExYTQzODFiNTFmN2ZlNmRmZDFkMQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFNdWx0aUxpbmVSZXBsYWNlV2l0aG91dEFTZXJ2ZWRMaW5lQWZ0ZXJFbmRXcml0ZXNOb3RoaW5nCTRlYjJmY2JjNzFiNmQ5MjhjODUyZjFiNzEyZDhmNzBlYWUwM2QwY2MzNGU3YmUyNTg2YzRjZTFhNzg1Mzk3MzIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBTXVsdGlMaW5lUmVwbGFjZVdpdGhvdXRBbkFuY2hvcklzUmVmdXNlZAk5MGM1ODkxNzkxZDMxZTdkYTY0NzRmN2U1NTVlYTAwMTNiNDdmMmY4MzE4MjY5NDdhNzE1NGRhYmFjZjU0Nzk5CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QU9uZUxpbmVEZWxldGVSZWNvcmRzVGhlU2FtZUxpbmVUd2ljZQk1NDFiY2Y5YWExZmNjZGFhNTE4OTNhMmRkYTQzNTdlNjA2NWJhNzJjYzYxMjUwNDE1MWYzNzE0NjhlNjhhNWZkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhZGRlZFdyaXRlRWNob1Nob3dzVGhlTGluZUFmdGVyVGhlQm9keQllNWQzYjY4MWMwYTFhNjZlMjExZGNiNmRlZjVmNjZmNjBlYTk1NzRhOTQ4YzQwYTg0ZjhlMjJkMmMwNGVhYTI2CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhZGRlZFdyaXRlRWNob1Nob3dzVGhlV3JpdHRlbkZpbGVOb3RUaGVPcmlnaW5hbFRhaWwJZGExNTcwOTgxZWM3OTY5Y2MzODkyYTg5NjAyMDU2MDlhOTEzOTc0ZTBmZjg2NGI0N2QwOGQzMjNiMDQwYjg2Zgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQYXJ0aWFsV3JpdGVOYW1lc1doYXRXYXNBbHJlYWR5V3JpdHRlbgljMzk3OTBkNGI0NTFmYWVjMDYzZjg2YzI4MjBhMjFiYzhjYjE4MzZhYjYzY2ExMzY0NGVlMmE4NmYwZjg1OTZhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhdHRlcm5NYXRjaGluZ05vdGhpbmdJc1JlZnVzZWQJY2E4MTgxYjkxZDJhOGMzNWFkMmE4MGZhZjZiNzhmMzVmZmNiMmQ3YWRjYjE3YjkzNDAxZmRmMmE3NDZhZDM3OQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQYXR0ZXJuUmFuZ2VTcGFubmluZ01hbnlMaW5lc05lZWRzQW5BbmNob3IJNDMyYzYzMzdiMjY1ODFmNDQzZTg1ZDg5ZWM4ZjNiYmExZWVlYzE1YjQyOGQ5MGRlZWNiOGEzZWRkOWNkNjIxMApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQbGFuVGhhdE5hbWVzT25lRmlsZVR3aWNlSXNSZWZ1c2VkV2hpY2hldmVyVGhlU3BlbGxpbmcJZTlhNGE0ZWExZDBmMjUxM2QzNmMxMjZmZDVjMzIwOGIxMTliMjYxMzEzNjM2YzA0NDE5YTRmMmQ4NGY5OTVlOQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWFkVW5kZXJPbmVTcGVsbGluZ0xpY2Vuc2VzVGhlU2FtZUZpbGVVbmRlckFub3RoZXIJZjYyMmE0NjRmY2MzNjIzNGQ5NTUxMzEyZjkzOWVjZDdlYzU0YjQyNzE4ZjQ3Mzk1ZTU1NDc1YzkyNTRkOGU4MQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWNlaXB0RWNob2VzVGhlUmVsYXRpdmVFbmRUaGVDYWxsZXJXcm90ZQk1NjBhYmFjYTFhOTEyMjZlOGRkYWQ4YWMxNGNlZDgyYTI5NTBkOGE4NzAzNTQ5MmViNDNmMzdjYWFkYzFkOTJhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlZ2V4QWRkcmVzc0lzU3RpbGxTdWJqZWN0VG9UaGVMZWRnZXIJZjc0Y2RkZmM1NzcyMzlhODYwMzlkMDUyZDdmMGRjODNjYzE5MDYyMDNkODFhOWM5OGIxZmNiMjJiODBmZmY0Ywpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWdleEFkZHJlc3NSZXNvbHZlc0FnYWluc3RUaGVPcmlnaW5hbEZpbGUJMzQwY2QxYmIzYzgyOWNmYjViMjU1MjE0NmM0ZjFiYzUwNzI4MjYxMWM5N2E4NWZhZDhjZDAxMzRmMTc2YjU1Mgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWxhdGl2ZUVuZEFkZHJlc3Nlc1RoZUxpbmVzSXRSZXBsYWNlcwljNjg3ZGJhN2UyNTcwNzcyODQ0ZjAzNmRkYTE0OTRlM2I3Y2ZhNTAxOGVjYjRhMzM5ZWYwZGNiZGJhYzhlZWJhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kQXRUaGVJbnRlZ2VyQm91bmRhcnlEb2VzTm90V3JhcAk5NjkwMmRiMzdkZWNkOWEwNWZiM2Y4MGRhMTlhMTkzMWYyYzI4NjNmOTYzMmFhMTY1MTIyZWNjOTE4OTE1NGVkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kUGFzdFRoZUxhc3RMaW5lSXNSZWZ1c2VkT25UaGVQbGFuUGF0aAkzMWE0NTkyOWUzYTdjMTE0N2FjZWI1MjdmMjQwZjU0MjA4YTc5NzgyMWNhMDg5NDkyNjRjNzUxNmZlYjU2YTFhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJldmVyc2VkUmFuZ2VGcm9tRU9GU2F5c1NvCTE1YjMyM2M0NTM4ZjMzOGRmNzhhMjI0NTQ5MzE0NzI0YTY4YmMyZjE0NDE4YWJmNDY2ZWM4NDdmMTRkZDQ3NTIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBU2VlbkZpbGVJc0VkaXRhYmxlCTViY2JkMDgyNGU5NjMxYTMyZWJjODMwYjY1OTMzMzIzMjk5NTcwNjZlNmNjYTQ5YWRiZTNjMjkzODczOGYzNGMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBU2luZ2xlTGluZVJlcGxhY2VEb2VzTm90TmVlZEFOZWlnaGJvdXIJNTk1NGJlOWJjNWM4MjczZmU4YzBhZmJmNjRmNjcyZTliMTU1Y2ZmNjc3YmY3NGM0MmNmOWMyMTMzZWM2YjlmMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFTaW5nbGVMaW5lUmVwbGFjZU5lZWRzTm9BbmNob3IJMTZhOGM0NzJmODJjOWZhNGQzZjQ5ZGUyNGNmNGNiYWNjMWFlZWI4ZDdjZDdkNzEyOTIyNzU4YjFjNTU4ZGM0Ywpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFTa2lwcGVkSHVua09taXRzRWNobwkzNmViZjA1N2I2MzI0NzRjMGQ1MTE0MzZmNzkwMGM3NWRiYTRkZGNlMWVlMTNiYjUzNWE3MzdiM2NlYTU5ZDU1CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVN0YWdlVGhhdEZhaWxzQWZ0ZXJNYWtpbmdEaXJlY3Rvcmllc0dpdmVzVGhlbUJhY2sJMmFmNzBmMTBjM2Q4ZDFjZWNjYzc2NDY5MDRjMTBjZTVlNGQxZWU3NjUwNTMwZjlkYWMzN2RiZmQxZDkzNzJjOApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFkZHJlc3Nlc1Jlc29sdmVBZ2FpbnN0VGhlT3JpZ2luYWxGaWxlCWZiN2I3MDMxMzdlODFhNWZkYmQzZDU0MjEyZmUzOGNiZWFhMzA0ZGU1YjRhNWI2NGE0MWQwYzFlZGRhMzU4YmQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBZHZpc29yaWVzQ291bnRzT25seU9rSHVua3NXaXRoQUJhbGFuY2UJNzgzYWQ3ZWQ0OThiMzQyYjY1ZGNjM2RjOTdhNjA1MzAzN2UwMDQxYjhiNGRiOTAzYTQxNWEwMjYxMDE3YjU0MQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuQWJvcnRlZFN0YWdlVGFrZXNCYWNrT25seVRoZURpcmVjdG9yaWVzSXRNYWRlCWUyMmIzYzkwY2FiNDcyNTc3YjhlYzgxM2ZlNzdlNTc0NTg5MzM1ZjZkMzRlM2JhZThhOGNjYjM2Zjk0MzMzYTMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkFic29sdXRlUGF0aFNheXNJdFdhc1Jlc29sdmVkQWdhaW5zdFRoZVJvb3QJM2JhYzIyN2UzNGExNDllNGY5MTAxOTczNWI3ZDAwMTYwZTRkYTczN2U0MTcxNmIxOTY1NGMxOTg3N2Y0OWVmNwpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuQW1iaWd1b3VzUmVnZXhBZGRyZXNzSXNSZWZ1c2VkCTlmZDczODAxMWUzOTFiMzhiYmRjMWJmNDQ0ZmEwZTA1ZjEzMmYwOTA5MDllOTI1ZDFlY2FjMTZiNGE2Y2Y4YjQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkVPRk11bHRpTGluZVJlcGxhY2VEb2VzTm90TmVlZEFMaW5lQWZ0ZXJFbmQJNzE2NmU2YjMzMTM0MTZjMjFhMWE5NjRkN2NiZWI0ZDcwMTNlZGE0ZWRlYWI4M2YyMTkyNjRmYmFjOTc4ZDYxMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuRW5kUGF0dGVybk9ubHlBYm92ZVRoZVN0YXJ0SXNSZWZ1c2VkCTZjYmJhZmJlNmQ1Y2MzNzgxYjk3ZmQ2MDEwMjEzN2M5OGQyMTQ4MjdlNGJlYmYzMDMyMzdhZmQyOGIyY2Y2ZWYKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkV4cGVjdGVkUmVtb3ZhbERpZmZlcnNPbmx5SW5XaGl0ZXNwYWNlCWE5YjZmZDA3ZTRkYTczMDgxMWNhZTQ5ZDRkZTQ5NjkzMWU3OGQ4ZDgyOWEwNDZkN2QxY2JmN2ZkOTAyOWI2NTMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkV4cGVjdGVkUmVtb3ZhbElzTm90Q2hlY2tlZEFnYWluc3RBblVuc2VlbkZpbGUJZjlhM2I4YmY5MDU0ZTc0MzM3NGYwMmZjZDcyZjBmZWIwMGFjMTA0ZTJjZjhiNWU3NzUxN2Q3NDcxZWE0Mjc5YQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuRXhwZWN0ZWRSZW1vdmFsT2ZUaGVXcm9uZ0xlbmd0aElzUmVqZWN0ZWQJNGNhNzA4Y2FmZDM4N2U2NDY3ZmQ4YjQ1MTVmZmVmNDBkZjI3YTNlMzNmOTA3OGI3N2Y4NTcyMjI2YzMwODFiYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdENyZWF0ZU5lZWRzTm9Qcmlvck9ic2VydmF0aW9uCWVhODEyYzZlNTljY2JmZWY1NDRjMjgxOWZhOGFhNTJlZDJmZTI0Y2E0Y2EwODZhNDE3NjdlNTliYjkyY2U5YjAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVCb3VuZHNBcmVUcmltbWVkCTEyNDM4NmRhMmQ2NGQ1ZjU5MzNhZDQ1M2RhNzRkNjNiZjA0ZGRiMjk3YTVkZDA5YTg0ODM3ODA5MmEwNWQyNjkKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVCb3VuZHNBcmVUcmltbWVkT25BUnVuZUJvdW5kYXJ5CTUzNmFmMjg1NDQ5ZmZlMWFjYjk1YWZmOThiZTEyMjVlYzE1ZjM2NjEzN2JhZWJkOTkyYTM1MGU5YTBjODQxY2MKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVSZWNvcmRzSXRzQm91bmRzCTA1M2UwMzcxZDY0NmVhMWE2YjJlYjVlOWJjZTE2MzdlMTI4YTQ5NmY5MjVjMzAwMTg4ODI5MGNjMzhiMjZiMWUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REcnlSdW5Xcml0ZXNOb3RoaW5nQnV0UmVwb3J0c1RoZU91dGNvbWUJMDEyZjc4OTdkZjc4YTRkOGMwODA2YjIzN2ExNDYxMTg1MjRlZTZlZDNlYzI3MmFjYjRkZmQ0NGRiMTNkYjYzYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVPRkFkZHJlc3NpbmcJNWQxMjk3MDU3MzgzYWM3MTNlMzAwMWNiZDU3Y2YyZGY3ZWE3NjUxM2U2MDRmNDg2NzgzZGFlNjRlZTAzNzQ1Nwpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVhcmxpZXJIdW5rc05ldmVyU2hpZnRMYXRlckFkZHJlc3NlcwlmZjgxY2JmODVmODJiYjJiY2I0MTBhY2MzYmQyNjE0YTFiMzJmYTFlMmQ3OTFlZjcyMWY0YzkwNGYxNzRhYzZhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0RWNob1BhZENsYW1wc0F0RU9GCThjYzQ0ODkwNzRhYmIwNjA5YzdkMzgzZGVlYjQ0NTY5MGE1ZjY4Y2NiYjdkMjRjNmE2ZmQzYTc3MGMwZDBjOTAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RFY2hvUGFkRGVmYXVsdFByaW50c05vTGluZXMJYWEwZDdjZWIzMDk3NWRlMDQ4MDQ2ZjhhYTA5ZDM4NzcxMjFlYjVlOTdlNTRmMGQ5MTg2ZDVmNzI0OTBlYzFhNgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVjaG9QYWROUHJpbnRzTk51bWJlcmVkTGluZXMJNzE5NjgxM2RhNjhhODNiOWEwNDkyZjFiNzYzNGU0ZjhiZTRiMzM0MmE3Mjc1OGQwZTkzZmVjOTBhMmEzZWNiMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZhaWxlZEZpbGVzU3RpbGxBcHBlYXJJblRoZVJlY2VpcHQJOTBhNTIxMmE5MTgwMDEyZDI0Y2Y5N2QxNzE5N2Y2YjM4OGE4YzZlYWI4NDRhMDRmOGJlMDFlNzIxNmZhMWVmNgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZpbGVQZXJtaXNzaW9uc1N1cnZpdmUJMmUyM2RlYzEyNzAwODRkOWE3YzVhMDVjNGVmNzBkYjdiOWFhYjA3NzQyNjc1Y2U5MTIyYTk5YTAxZWM0MjViMQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZvcmNlQnlwYXNzZXNUaGVHdWFyZAlhODJmNjliYjg5ZmVlNzlkYjI4NTEwNWI4MDY3Y2ZjZWRlMzZlZjcyOTM2MWIzZjUxZGIzNTY2NWE4ZDJlNGUyCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0R29vZFNoYUFuZENvdW50c1Bhc3MJMGM2NWM3Mzc2ODEzOWVjYTBjNzlmMzVjYTA5YjA4ZGM0MjAyZjEwYzliM2Q2OTlhNTQ0ZDE5YjdhNmY4YTZkYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE1hdGNoaW5nTmV0c09taXRCYWxhbmNlCTJhNGJmZmQ2NmM1OGIzNGFlMDRiZjljYmQwOTQzZjM4ZjhiYjAzNDg3NmU0YTlhNjE4NDcyOGUzNThjZmViNGEKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RNdWx0aXBsZUZpbGVzSW5PbmVSdW4JZmVjNWRkOGU1MWM0ZjczZTg0MDgwZjQyNTQ3NTc3YTY5MWMxZGUxYzI4MDdlMzlhZGUyODdlMjUyNGYxZGExMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE5pbExlZGdlckRpc2FibGVzVGhlR3VhcmQJZGMwNjU5NWYwOWJmMjdmNmE3MTMxYTRmOTQzMzJmNmFkMTczNWNkMmE4M2Q4MmZlNGZlMTdlZDhmNzZkOWI5YQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE9uZUJhZEh1bmtBYm9ydHNFdmVyeXRoaW5nQW5kU2F5c1doaWNoCWNmMTQzOWEyOTQxMWY5MTkzYThlM2VkZjM4Mjg1YmMxM2UyNjgzMWNkOGIxYjlmY2YxYjVmYzkwZTlmYjdmNmUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RPbmx5RGVsZXRlUmVjb3Jkc0JvdW5kcwliMDc0N2Q2YWQyYjA1YzM3MGM1NjcxYTkwOWRkNTM3ZDYxNzZiNWJhNDVlODYxNmUyZjg2Y2Q4Yjk4YTk2MzU0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0T3ZlcmxhcHBpbmdIdW5rc0FyZVJlamVjdGVkCTU5NzA2MWI0YzdkYWZiNzgyY2ZiMjRmYzk4ZmZmOGQ5NDczMGM4OTM3ODIxNzlhNGY3NGZlMWFlYTk2Mzk2Y2EKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RQcmVjb25kaXRpb25zCWE0YjU0YzNhYjA0MGU4OTMxYTQxNWM1YjFhNmM4MzkzYmZmNWM4YjhiNGI1MGY3MDZjMzIxZDVhNGNiNGE1MjYKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RQcm9zZU9taXRzQmFsYW5jZQlkMDFhNTdlNDkyOGE1M2EyOWQyMGVmNWVhZmUwZWI5NWFiYmI4YzU2NjIzMzI5MmU0OWE2OWViMmYxYmZmZDk0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0U3RyaWN0QmFsYW5jZUlzT2ZmQnlEZWZhdWx0CTAwN2EyNGQ0MTc1Nzc5OWU4ODZkYTQ5NTJhN2IzMjcwMWRmOTZhNGFkYzg0ZTFkZTZmMzNmMWU4ODgzNWY2YjMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RTdHJpY3RCYWxhbmNlTGVhdmVzQmFsYW5jZWRNdWx0aUxpbmVBbmRQcm9zZUFsb25lCWM0YWE2ZmIyMTYyMTVjZjRkNjFlMzhkMTEyMTM0OGM4NjAxYzcyZTE3YWMwZTNjYmEzNTFkM2JjYmUwZDcyZmMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RTdHJpY3RCYWxhbmNlUmVmdXNlc1RoZVdyYXBUYWlsU2lnbmF0dXJlCWY1MmI0NjlmODcyYmI2YzQxYWE2MGZiOGNlMDgzOWRjNTg1NmQ5ZDE5MjAwYmU2MmNlY2M3YWQzMTE2YTExNTUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RUaGVFbmRQYXR0ZXJuSXNUaGVGaXJzdE1hdGNoQXRPckFmdGVyVGhlU3RhcnQJOTY3M2I4MGMwNTVjNTk4ZTk2YzYwNTBhNzYxMzJiNzdhODFiNWNiMDYxZDQ3MWJmOTVkMDk3MTAyZTE2M2E4Ngpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdFRoZUVuZ2luZVJlZnVzZXNBQm9keUxlc3NDcmVhdGUJY2NjNWY2NjA0MzRkMzc0ZTk1YzBjZDMxMmZjOGU1NzkwOTU0ZWM2NDA5NjE4NzI0NDI3Yzc0ZjE2NWEzMGE2Mgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdFRoZUVuZ2luZVJlZnVzZXNBUmVsYXRpdmVFbmRUaGVPcENhbm5vdEhvbm91cgkyMDNmMWYxOTJmMjI3MzJiYWYxZTc1MzE4ZmY0NzQ5OGE4NzYwYmQ4ZGEzNjlkNGM3NDY1YjE3ZTUyYjBmNmQzCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0VGhlRW5naW5lUmVmdXNlc0V2ZXJ5U2hhcGVUaGVQYXJzZXJSZWZ1c2VzCTM3NjcxOWE4ZWIwNWM4Njc3ZWFlYWRiYjFmZGM5NzUyNWE5YmI2NDNhYzIwYmYzYmVlNGE3Yjk4OTM2ZjNlZGQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RUcmFpbGluZ05ld2xpbmVJc1ByZXNlcnZlZAkxZTY3NmNmY2IzOTA2OTFmYTZiMmQyYTJjNjg1MWZhNjUwMjk5M2UxZWMxNjc1MWFmY2UxYTQ0OGQ0ZjMwMWEwCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlzeW1saW5rIGFsaWFzCWVmOTgwYmIwYTc0ZmFmN2VhNDU3MjlkYTk2ODU0NDQ4NWNlZjQxZjFkNzc3YzFhNGJiYTE2MzcxM2U1NzFmMzAKYm9keQlpbnRlcm5hbC9tY3Avc3RyaWN0YmFsYW5jZV90ZXN0LmdvCVRlc3RBbk1DUFdyaXRlV2l0aFN0cmljdEJhbGFuY2VSZWZ1c2VzVGhlU2lnbmF0dXJlCTU4YjNjOGU3ZmI3NDNmYTA2NzNiYWRjZjVjY2U5MTUwNjlmNGEwMWZlNzNlOGZlYjA2NDcxNzMzZjM0YmU4ZWEKYm9keQlpbnRlcm5hbC9tY3Avc3RyaWN0YmFsYW5jZV90ZXN0LmdvCVRlc3RXcml0ZVRvb2xEZWNsYXJlc1N0cmljdEJhbGFuY2UJNjlmZjI1M2M2Mjk2NjI5MmVhNzRlYTI3MDQyYzU4NmRlYjE2ZjFmYzQwZmJlYWUxYjY1OTk0NzVmYjkwNWJmZA
  ```
  ```
- 2026-09-13 · 54beb7b* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4851b3b7ff711061099f68e4d34bf9fcd4e669f39974273f4220aa9039ede65 · ms:5165
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.326s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.552s [no tests to run]
  === RUN   TestWriteToolDeclaresStrictBalance
  --- PASS: TestWriteToolDeclaresStrictBalance (0.00s)
  === RUN   TestAnMCPWriteWithStrictBalanceRefusesTheSignature
  --- PASS: TestAnMCPWriteWithStrictBalanceRefusesTheSignature (0.02s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.480s
  ```
- 2026-09-13 · 54beb7b* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4851b3b7ff711061099f68e4d34bf9fcd4e669f39974273f4220aa9039ede65 · ms:1260
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.313s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.451s [no tests to run]
  === RUN   TestWriteToolDeclaresStrictBalance
  --- PASS: TestWriteToolDeclaresStrictBalance (0.00s)
  === RUN   TestAnMCPWriteWithStrictBalanceRefusesTheSignature
  --- PASS: TestAnMCPWriteWithStrictBalanceRefusesTheSignature (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.664s
  ```
- 2026-09-13 · 54beb7b* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4851b3b7ff711061099f68e4d34bf9fcd4e669f39974273f4220aa9039ede65 · ms:1061
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.212s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.327s [no tests to run]
  === RUN   TestWriteToolDeclaresStrictBalance
  --- PASS: TestWriteToolDeclaresStrictBalance (0.00s)
  === RUN   TestAnMCPWriteWithStrictBalanceRefusesTheSignature
  --- PASS: TestAnMCPWriteWithStrictBalanceRefusesTheSignature (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.485s
  ```
- 2026-09-13 · 54beb7b* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4851b3b7ff711061099f68e4d34bf9fcd4e669f39974273f4220aa9039ede65 · ms:1696
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.369s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.176s [no tests to run]
  === RUN   TestWriteToolDeclaresStrictBalance
  --- PASS: TestWriteToolDeclaresStrictBalance (0.00s)
  === RUN   TestAnMCPWriteWithStrictBalanceRefusesTheSignature
  --- PASS: TestAnMCPWriteWithStrictBalanceRefusesTheSignature (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.529s
  ```
- 2026-09-13 · 54beb7b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd · ms:2620
- 2026-09-13 · 54beb7b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd · ms:1980
- 2026-09-13 · 54beb7b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1257a624d50294106a56580a5babbad4e55ebef7cfa820470a8d76310abaecbd · ms:3218
