# Task ADR-060-T3: unquoted `anchor=` containing `"` is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** unquoted-quote refuse; §102
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `unquoted embedded quote refused`, `quoted escaped form still parses`, `ADR-040 spaced unquoted still parses`

## Goal

An unquoted `anchor=` whose raw span contains `"` is a parse error naming `anchor="…"`. Quoted anchors with embedded quotes still parse.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | Raw-header scan before or after split. |
| `internal/plan/plan_test.go` | edit | Red: refuse / quoted ok / ADR-040 still ok. |
| `scripts/contract.sh` | edit | **§102**. |

## Ordered Steps

1. [S1] Write the failing tests. [proof: mutation]
2. [S2] Refuse unquoted `"` in `anchor=`. S1 GREEN. [proof: mutation]
3. [S3] §102 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 102\. ' scripts/contract.sh \
  && go test ./internal/plan/ -count=1 -v \
    -run 'TestUnquotedAnchorWithEmbeddedQuotesIsRefused|TestAnUnquotedSpacedAnchorParsesUntilTheNextKey' 2>&1 | tee /tmp/adr060-t3.out \
  && grep -q '^--- PASS: TestUnquotedAnchorWithEmbeddedQuotesIsRefused' /tmp/adr060-t3.out \
  && grep -q '^--- PASS: TestAnUnquotedSpacedAnchorParsesUntilTheNextKey' /tmp/adr060-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestUnquotedAnchorWithEmbeddedQuotesIsRefused` | `internal/plan/plan_test.go` | `anchor=import { inject, vi } from "vitest"; body=1` errors and names `anchor="`; the same value double-quoted with `\"` parses with `"vitest"` in Anchor | — | S1, S2 |

Regression names in the fence are existing tests (`TestAnUnquotedSpacedAnchorParsesUntilTheNextKey`, `TestAnAnchorCanContainAQuote`).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `parseHeader` / Parse. Deleting the scan leaves S1 red |
| 3 — the caller can discover it | refusal text; §102; T5 teach |
| 4 — it is used | quality-blueprints vitest import; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · unquoted embedded quote is accepted so TestUnquotedAnchorWithEmbeddedQuotesIsRefused must go red · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · covers:unquoted embedded quote refused
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · quoted form is scanned as unquoted so TestUnquotedAnchorWithEmbeddedQuotesIsRefused/quoted must go red · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · covers:quoted escaped form still parses
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · every unquoted anchor is refused so TestAnUnquotedSpacedAnchorParsesUntilTheNextKey must go red · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · covers:ADR-040 spaced unquoted still parses

## Invariants

- ADR-040 unquoted spaced `anchor=` still parses.
- Quoted `anchor="case \"anchor\":"` still parses (`TestAnAnchorCanContainAQuote`).

## Risks

- Scanning after `splitHeader` misses the case because quotes were stripped. Scan the raw header.

## Stop Condition

Stop if the refuse needs a splitter rewrite that breaks pattern addresses.

## Out of Scope

- Treating mid-token `"` as literal so the original plan applies. (record Alternatives)

## Verification Log
_(tool-written)_
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:a3d8b1d7770600f184669906ffa0f6cf64a073fa4a821e5749a297622404e1b6 · ms:341 · test-lock-sha256:aa1451839fb98b48466a82013303699edeb8e4fa80fc096c08eff5b1c84d62ed · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBQk9NSW5BQm9keUlzV3JpdHRlbkJhY2tWZXJiYXRpbQk1NDJmZWYxNGI2MDE1NmNkZWRhNjYxZTgzOWQ2MzQwMTgwMWE3YjY1OTMyZjMzYTZlMzcwN2ZmY2IyMDQ0YWNlCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFCT01JbnNpZGVBUGxhbklzQ29udGVudE5vdFN5bnRheAkzMmJlNGFjNTVkYjA3MGNiY2M5YzljZGE2NzFkNzc1M2UwZTQwODZmMmE3M2QwMzQ5ODJiYmQ2ZTQzOWVlNTk4CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFCb2R5TGluZVRoYXRMb29rc0xpa2VBSGVhZGVyU2F5c1NvCWVkMTEzZDQ5ZWYxODYwZGUxNTk0YzE4Njc0ZWQ3OTA2NTk2MWM2MmQ3NmQ4ZmFlMjdmZTk5ZDFiZThmMDk0ZWUKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QUNyZWF0ZVdpdGhOb0JvZHlJc1JlZnVzZWRVbmxlc3NJdFNheXNCb2R5WmVybwljOWQzY2E4Y2NkMjM3MjNkY2UzNGQ5NTdhZTI1NjJiYjZhZDFkM2M5NTY5NDRjYzU4NjU0Y2Q4Njc1MWEzN2Q4CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFNYWxmb3JtZWRQYXR0ZXJuSXNSZWZ1c2VkQXRQYXJzZVRpbWUJMWU0NTM1YmU5YjYzYzQ5MmYzMzhlYjQxYTQzYTdkNDVjNDE3MDEzNDkyZjFlOGRlYmFkNmNmNjc4MWMxZTE4Ywpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBUGF0dGVybkFkZHJlc3NQYXJzZXMJM2RmNzRkMTU4N2Q2NTk2OTNiYmY2Nzk5YjMwYTk5NjY4NmZlNDgzNGRlM2U3YjZiYWNmMDdjODVhODQ4MjBkZgpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBUGxhbkFkZHJlc3NUYWtlc0FSZWxhdGl2ZUVuZAlkODFkN2NkMDE1ZTljMjdhYzZkMmE0YWRmYTgyYzFlNTQzODExNDBlMjBkODY1Mjg0OTE3Y2ZlZGRmODc0NmYzCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFRdW90ZUluc2lkZUFQYXR0ZXJuU3Vydml2ZXNUaGVIZWFkZXIJNGRmMWVkNjliMTIwYjBmNGUyOTAzYjg1NDllNzI2OTZlZDZkODUxZTAxZTU0Nzg3MjcwNWZiYTdlNWNjN2NlOApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBUmVsYXRpdmVFbmRJc1JlZnVzZWRXaGVyZUl0V291bGRCZUlnbm9yZWQJMzAyYzMxZjJjZTgwOTJlNzA1MmI4YzQ5NmRmYzk0NDBhMGVkNzI5MTNhMmVkMTc0ODQyZDUxMmYxNjE4MjA2NQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBUmVwZWF0ZWRHdWFyZEtleUlzUmVmdXNlZAkxNWZmMDUxOTgwYmM2ZDg5MzU2MmZlYTQ2MzUwNzliYmQ1NzlhZmY4Mjc5YTBkZTA1NGIyZjY5NmJlYzkwY2IzCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFTYXRpc2ZpZWRCb2R5Q291bnROYW1lc1RoZUV4dHJhTGluZXMJZmYxY2M4ZDY3YWZlZTRlYmUzMDM2YTI1YTg3N2Q3Y2QwNzQzOGUyYjJkNmU5MTBiYTYxNjRlNWVlZDY4NDlhYgpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBU2luZ2xlUXVvdGVkQW5jaG9yUGFyc2VzCWQxYWUxOTBlYmM0NWE3Yjc1ODUyY2NmNzQ0NzQ2YTJjMTAyMjViOGYzMzFhNzE1YzYyODNhN2EzNjU1NjlkNzYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVRyYWlsaW5nVG9rZW5BZnRlckFRdW90ZWRBbmNob3JOYW1lc0RvdWJsZVF1b3RlcwlkMmEzMDczODg4Y2EyYTRjYjg4YzQ1MzhmNmIyNjYzOWQ2ZjIyMGZjZDY2ZWFlNGM2YzFiOWFiMTc1ODU0MTEyCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFVVEY4Qk9NRG9lc05vdERpc3F1YWxpZnlUaGVGaXJzdEhlYWRlcgljYmEyNDBkMDliZjA2YmM5Mzg1NDhmNzc5ZGE2ZGE4ZWRkMWMzZDQyNTIzM2UxNjg4MzE3NDMxNGYyYzhhNTNhCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFuQWRkcmVzc1JlbmRlcnNCYWNrQXNUaGVDYWxsZXJXcm90ZUl0CTRkZjRmOTkxYzRlNjE1ODY0YTY4ZjBiMDlmZjQ2MWUwOTBlNTY5ZjRkNmU2NzUyNDdiZDRlMzY5MDk5ZjJhN2MKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QW5VbnF1b3RlZFNwYWNlZEFuY2hvclBhcnNlc1VudGlsVGhlTmV4dEtleQk0OTU3YzkzOGVhNmM1Zjc5MzE4Yzg1Y2VjZjNlZjM3M2ZjYjE2NWEyMTJiNzY1YzFmZmFkNDc2OTQzZTJlYmY1CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdENvbmNhdGVuYXRlZEJPTUZyYWdtZW50c1N0YXlUd29IdW5rcwkxMTg3YjU0YWE3ZDhhYWUzNTM2YzA3Mjc0OGEzM2IzM2UyODdhNjRiNmIzZGQ4MDJlMzc5ZjlmMmU1YzczOWIzCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdERlbGV0ZUJvZHlEb2VzTm90V2Vha2VuVGhlT3RoZXJPcHMJYzE0MTBjYjBlYzM4YzkwZTQyYTVkNTgwMWIzMWMxYWE2OTIwMTk2ZTkyNmFkMzk0NTIzZjk3MjQzM2Y3NWQ1Zgpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3REZWxldGVCb2R5SXNOb0xvbmdlckFQYXJzZUVycm9yCWJjODRmNzBkMmFmNjExN2EyODc3ZWY2YjIzZDRlM2I1ZTBlZTQ4ODAwOGY3YjlkY2RmMTk4NmFmZGQ1YTM0NGQKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0RGVsZXRlSXNUaGVPbmx5UmFuZ2VDb25zdW1pbmdPcFRoYXROZWVkc05vQm9keQkzNTM3NzM5NmU5NjkzNWIyNmNhZjk2ZmMxMDg5ODczOTZmODM5NjkxYmZlYmE4Mjk2MTIzOGUwNTNhODRjOGRkCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdERpc3RpbmN0R3VhcmRLZXlzU3RpbGxQYXJzZQk3NjIzYmY1ZDNhYjQ5MjUyMzAwNTVlMGMxZmNiMWI5Yzc1YWYwMmNlNzEzODJlMjYzMDI1NzNiNDQwMTBlMWJjCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEV2ZXJ5RXhpc3RpbmdBZGRyZXNzRm9ybUlzVW5jaGFuZ2VkCTBiOTU4MWE1ZjljMmEwMGFlMjkwYjhiZTI4NTQ2N2ZkMDZjZGE0MjFkMjYxNzBmMWE3MTdiM2U5NDRjNGJiNDYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0RXhwbGljaXRCb2R5TGVuZ3RoUHJvdGVjdHNIZWFkZXJMaWtlTGluZXMJMDI1OGM2ZWY1YWYzZjBhYzM1ZDE5OTBkZWIyNzY4MGQ2N2RkZDE4ZmQzY2VhZWNiOTNmMWRhNWM2Njk5NzViNwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RQYXJzZUFkZHIJM2E1OWExZjQzMzMzNjliNjgyYTFkNTc3MTFhMDNlOGZlNTVhMjc0OWUwNTI4OTM1NjhmODllNjgwZTZmOWE0YQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RQYXJzZU11bHRpcGxlRmlsZXNBbmRPcHMJNjE1ZTNlNWMzNWMyYWFhZjQwOTRjOTM2OTM3NmNhMDRiY2IwNzkyNjRjYzhmNTE5ZDI5NTEyMDhlZDFmNDIwZgpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RQYXJzZVJlamVjdHNCYWRQbGFucwk5NDBlNWJmNTBlMzc0N2YxNDg4YjY3MWRmYjJjYjI4N2ViMjI5NWQ0NWZkYzM4YzFkMTc1MDllNmU3MTFiNGFhCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdFBhcnNlUmVwb3J0c0V2ZXJ5RXJyb3IJZTg1MTY2Yjg4NTU1MTY0N2I0Mjc1YWE2Y2E2YTBiODdjYTQ2NmIyMWUzMTMxODMzZTExNjUxZTNkMzFjNGE2OQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RRdW90ZWRGaWVsZHNTdXJ2aXZlCTFlMjhmMjE4MWI5MjYyNjQ3MzVmZjVkYzk0MTMxNTIxYzViMDA1ZDA5MmQwNzY5NzQwNDUzZGE5YzMwNjYzZWMKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0UmF3V2l0aG91dEJvZHlJc1JlZnVzZWQJMmVmNjE1YWU1Y2RkYmUzZWVhYWM0NTU4ZWI1OTRhYjEyNjYwNzVhOWU1YjdmY2EyZGNjNmMyM2EzN2I4YmU3OQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RTaGFNdXN0QWN0dWFsbHlCZUhleGFkZWNpbWFsCTcwMjA5ZWY0OTc0NjRmZmEyNDQwZDkwZTg2ZmQyNTA2OGJjYTRjZmM1MTQ1OWQzNmMwYzE1MGE2YTY3OTkwZTgKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0VGhlSGludHNTdGF5UXVpZXRPbk9yZGluYXJ5RmFpbHVyZXMJYzJmNTQzMDc1NzFiYWIyNzk4Y2E2MDY0ZWUxNzZkZTBiNDNkOTFjMzg2MGIwYjgxMWFmZTA3YWE5ZjRiMjcwNApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RVbnF1b3RlZEFuY2hvcldpdGhFbWJlZGRlZFF1b3Rlc0lzUmVmdXNlZAlkMTY2OGFjYThlNGVjMWE2ZTNkMTliNDQ0OGQ1NDAzNDFjNDFmODlhNGQ3YTVkMTBkNGNlOGY3NDc1MjI1NWI0CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JcXVvdGVkCTkyOGQxNTY1Y2IwMzNkNDdiMDU3MDMxMGI4YTFkNWNiMzRiMzViNWJlMmI5ZTQyNGIxYjhjZjE1ZTlhMTQzMGIKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwl1bnF1b3RlZAk0NDc2MmM5ZWI2ZGE4ZjA1ZDNjMDU1NDFjZTQ5MDk1MmEyYmYxZTE2NjVmNzA2OWVkZmY1YWZhYTdlMTdjNDM2
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused/unquoted
      plan_test.go:768: unquoted anchor= with embedded quotes parsed; it must refuse
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused/quoted
  --- FAIL: TestUnquotedAnchorWithEmbeddedQuotesIsRefused (0.00s)
      --- FAIL: TestUnquotedAnchorWithEmbeddedQuotesIsRefused/unquoted (0.00s)
      --- PASS: TestUnquotedAnchorWithEmbeddedQuotesIsRefused/quoted (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.179s
  FAIL
  ```
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:a3d8b1d7770600f184669906ffa0f6cf64a073fa4a821e5749a297622404e1b6 · ms:294
  ```
  --- last 10 line(s) of stdout
  === RUN   TestAnUnquotedSpacedAnchorParsesUntilTheNextKey
  --- PASS: TestAnUnquotedSpacedAnchorParsesUntilTheNextKey (0.00s)
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused/unquoted
  === RUN   TestUnquotedAnchorWithEmbeddedQuotesIsRefused/quoted
  --- PASS: TestUnquotedAnchorWithEmbeddedQuotesIsRefused (0.00s)
      --- PASS: TestUnquotedAnchorWithEmbeddedQuotesIsRefused/unquoted (0.00s)
      --- PASS: TestUnquotedAnchorWithEmbeddedQuotesIsRefused/quoted (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.164s
  ```
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · ms:750
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · ms:410
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:de17f2f384118cbd64848825ea5a7624093872c60744b242d0d8362839e400c7 · ms:404
