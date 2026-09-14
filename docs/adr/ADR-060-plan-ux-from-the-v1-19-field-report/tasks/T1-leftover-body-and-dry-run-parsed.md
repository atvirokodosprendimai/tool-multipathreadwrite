# Task ADR-060-T1: leftover `body=` names extra lines; dry-run prints parsed bodies

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** leftover extra-count message; `parsed:` dry-run lines; §100
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `leftover names extra count`, `one error per hunk`, `dry-run prints parsed body=`

## Goal

A satisfied `body=N` followed by extra lines names N and how many extras. `--dry-run` prints each parsed hunk's body count.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | Count extras; name declared vs extra. |
| `internal/plan/plan_test.go` | edit | Red: extra count in the message. |
| `cmd/mrw/main.go` | edit | `--dry-run` human `parsed:` lines. |
| `cmd/mrw/dryrun_parsed_test.go` | create | Red: dry-run prints `body=`. |
| `scripts/contract.sh` | edit | **§100**. |

## Ordered Steps

1. [S1] Write the failing tests. [proof: mutation]
2. [S2] Leftover message names extra count. S1 leftover GREEN. [proof: mutation]
3. [S3] Dry-run prints `parsed:`. S1 dry-run GREEN. [proof: mutation]
4. [S4] §100 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 100\. ' scripts/contract.sh \
  && go test ./internal/plan/ ./cmd/mrw/ -count=1 -v \
    -run 'TestASatisfiedBodyCountNamesTheExtraLines|TestDryRunPrintsParsedHunkBodies' 2>&1 | tee /tmp/adr060-t1.out \
  && grep -q '^--- PASS: TestASatisfiedBodyCountNamesTheExtraLines' /tmp/adr060-t1.out \
  && grep -q '^--- PASS: TestDryRunPrintsParsedHunkBodies' /tmp/adr060-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/ ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASatisfiedBodyCountNamesTheExtraLines` | `internal/plan/plan_test.go` | `body=1` plus two extra lines names `body=1` and `2 extra`; still one error; still `is not part of any hunk` | — | S1, S2 |
| `TestDryRunPrintsParsedHunkBodies` | `cmd/mrw/dryrun_parsed_test.go` | `--dry-run` prints `parsed:` with `body=1` | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | Parse leftover branch; write `--dry-run`. Deleting either leaves S1 red |
| 3 — the caller can discover it | §100; T5 teach |
| 4 — it is used | quality-blueprints leftover refusals; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · 708adf2* · mutant inconclusive · exit 1 · `internal/plan/plan.go` · leftover extra count dropped so TestASatisfiedBodyCountNamesTheExtraLines must go red · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · covers:leftover names extra count
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · leftover extra count renamed so TestASatisfiedBodyCountNamesTheExtraLines must go red · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · covers:leftover names extra count
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `cmd/mrw/main.go` · dry-run parsed: prefix dropped so TestDryRunPrintsParsedHunkBodies must go red · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · covers:dry-run prints parsed body=
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · one leftover error becomes one per extra line so TestASatisfiedBodyCountNamesTheExtraLines must go red · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · covers:one error per hunk

## Invariants

- One leftover error per hunk (`TestAnUnaccountedBlockIsReportedOncePerHunk`).
- `--json --dry-run` does not require `parsed:` (human only).
- Apply still writes nothing under `--dry-run`.

## Risks

- Message change breaks a grep of `is not part of any hunk`. Keep that phrase.

## Stop Condition

Stop if leftover reporting needs a second body grammar (ed `.`).

## Out of Scope

- `body=@path` (T4).
- Read neighbour hint (T2).

## Verification Log
_(tool-written)_
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · ms:833 · test-lock-sha256:4fff850430920ad52dd44d185a86aa01ee108a38cec82dd45930cbbe4480153e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZHJ5cnVuX3BhcnNlZF90ZXN0LmdvCVRlc3REcnlSdW5QcmludHNQYXJzZWRIdW5rQm9kaWVzCTRmNjI4N2U5NjM0MGU0YzZjMTEwYWQxMTVjYWJjM2I5YjJkOWE4NmNhMTNhYjg3NzMzOTllMjc0MGNiM2I5ZDYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QUJPTUluQUJvZHlJc1dyaXR0ZW5CYWNrVmVyYmF0aW0JNTQyZmVmMTRiNjAxNTZjZGVkYTY2MWU4MzlkNjM0MDE4MDFhN2I2NTkzMmYzM2E2ZTM3MDdmZmNiMjA0NGFjZQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBQk9NSW5zaWRlQVBsYW5Jc0NvbnRlbnROb3RTeW50YXgJMzJiZTRhYzU1ZGIwNzBjYmNjOWM5Y2RhNjcxZDc3NTNlMGU0MDg2ZjJhNzNkMDM0OTgyYmJkNmU0MzllZTU5OApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBQm9keUxpbmVUaGF0TG9va3NMaWtlQUhlYWRlclNheXNTbwllZDExM2Q0OWVmMTg2MGRlMTU5NGMxODY3NGVkNzkwNjU5NjFjNjJkNzZkOGZhZTI3ZmU5OWQxYmU4ZjA5NGVlCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFDcmVhdGVXaXRoTm9Cb2R5SXNSZWZ1c2VkVW5sZXNzSXRTYXlzQm9keVplcm8JYzlkM2NhOGNjZDIzNzIzZGNlMzRkOTU3YWUyNTYyYmI2YWQxZDNjOTU2OTQ0Y2M1ODY1NGNkODY3NTFhMzdkOApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkUGF0dGVybklzUmVmdXNlZEF0UGFyc2VUaW1lCTFlNDUzNWJlOWI2M2M0OTJmMzM4ZWI0MWE0M2E3ZDQ1YzQxNzAxMzQ5MmYxZThkZWJhZDZjZjY3ODFjMWUxOGMKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVBhdHRlcm5BZGRyZXNzUGFyc2VzCTNkZjc0ZDE1ODdkNjU5NjkzYmJmNjc5OWIzMGE5OTY2ODZmZTQ4MzRkZTNlN2I2YmFjZjA3Yzg1YTg0ODIwZGYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVBsYW5BZGRyZXNzVGFrZXNBUmVsYXRpdmVFbmQJZDgxZDdjZDAxNWU5YzI3YWM2ZDJhNGFkZmE4MmMxZTU0MzgxMTQwZTIwZDg2NTI4NDkxN2NmZWRkZjg3NDZmMwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBUXVvdGVJbnNpZGVBUGF0dGVyblN1cnZpdmVzVGhlSGVhZGVyCTRkZjFlZDY5YjEyMGIwZjRlMjkwM2I4NTQ5ZTcyNjk2ZWQ2ZDg1MWUwMWU1NDc4NzI3MDVmYmE3ZTVjYzdjZTgKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kSXNSZWZ1c2VkV2hlcmVJdFdvdWxkQmVJZ25vcmVkCTMwMmMzMWYyY2U4MDkyZTcwNTJiOGM0OTZkZmM5NDQwYTBlZDcyOTEzYTJlZDE3NDg0MmQ1MTJmMTYxODIwNjUKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVJlcGVhdGVkR3VhcmRLZXlJc1JlZnVzZWQJMTVmZjA1MTk4MGJjNmQ4OTM1NjJmZWE0NjM1MDc5YmJkNTc5YWZmODI3OWEwZGUwNTRiMmY2OTZiZWM5MGNiMwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBU2F0aXNmaWVkQm9keUNvdW50TmFtZXNUaGVFeHRyYUxpbmVzCWZmMWNjOGQ2N2FmZWU0ZWJlMzAzNmEyNWE4NzdkN2NkMDc0MzhlMmIyZDZlOTEwYmE2MTY0ZTVlZWQ2ODQ5YWIKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0QVNpbmdsZVF1b3RlZEFuY2hvclBhcnNlcwlkMWFlMTkwZWJjNDVhN2I3NTg1MmNjZjc0NDc0NmEyYzEwMjI1YjhmMzMxYTcxNWM2MjgzYTdhMzY1NTY5ZDc2CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFUcmFpbGluZ1Rva2VuQWZ0ZXJBUXVvdGVkQW5jaG9yTmFtZXNEb3VibGVRdW90ZXMJZDJhMzA3Mzg4OGNhMmE0Y2I4OGM0NTM4ZjZiMjY2MzlkNmYyMjBmY2Q2NmVhZTRjNmMxYjlhYjE3NTg1NDExMgpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBVVRGOEJPTURvZXNOb3REaXNxdWFsaWZ5VGhlRmlyc3RIZWFkZXIJY2JhMjQwZDA5YmYwNmJjOTM4NTQ4Zjc3OWRhNmRhOGVkZDFjM2Q0MjUyMzNlMTY4ODMxNzQzMTRmMmM4YTUzYQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RBbkFkZHJlc3NSZW5kZXJzQmFja0FzVGhlQ2FsbGVyV3JvdGVJdAk0ZGY0Zjk5MWM0ZTYxNTg2NGE2OGYwYjA5ZmY0NjFlMDkwZTU2OWY0ZDZlNjc1MjQ3YmQ0ZTM2OTA5OWYyYTdjCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEFuVW5xdW90ZWRTcGFjZWRBbmNob3JQYXJzZXNVbnRpbFRoZU5leHRLZXkJNDk1N2M5MzhlYTZjNWY3OTMxOGM4NWNlY2YzZWYzNzNmY2IxNjVhMjEyYjc2NWMxZmZhZDQ3Njk0M2UyZWJmNQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RDb25jYXRlbmF0ZWRCT01GcmFnbWVudHNTdGF5VHdvSHVua3MJMTE4N2I1NGFhN2Q4YWFlMzUzNmMwNzI3NDhhMzNiMzNlMjg3YTY0YjZiM2RkODAyZTM3OWY5ZjJlNWM3MzliMwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3REZWxldGVCb2R5RG9lc05vdFdlYWtlblRoZU90aGVyT3BzCWMxNDEwY2IwZWMzOGM5MGU0MmE1ZDU4MDFiMzFjMWFhNjkyMDE5NmU5MjZhZDM5NDUyM2Y5NzI0MzNmNzVkNWYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0RGVsZXRlQm9keUlzTm9Mb25nZXJBUGFyc2VFcnJvcgliYzg0ZjcwZDJhZjYxMTdhMjg3N2VmNmIyM2Q0ZTNiNWUwZWU0ODgwMDhmN2I5ZGNkZjE5ODZhZmRkNWEzNDRkCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdERlbGV0ZUlzVGhlT25seVJhbmdlQ29uc3VtaW5nT3BUaGF0TmVlZHNOb0JvZHkJMzUzNzczOTZlOTY5MzViMjZjYWY5NmZjMTA4OTg3Mzk2ZjgzOTY5MWJmZWJhODI5NjEyMzhlMDUzYTg0YzhkZApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3REaXN0aW5jdEd1YXJkS2V5c1N0aWxsUGFyc2UJNzYyM2JmNWQzYWI0OTI1MjMwMDU1ZTBjMWZjYjFiOWM3NWFmMDJjZTcxMzgyZTI2MzAyNTczYjQ0MDEwZTFiYwpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RFdmVyeUV4aXN0aW5nQWRkcmVzc0Zvcm1Jc1VuY2hhbmdlZAkwYjk1ODFhNWY5YzJhMDBhZTI5MGI4YmUyODU0NjdmZDA2Y2RhNDIxZDI2MTcwZjFhNzE3YjNlOTQ0YzRiYjQ2CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdEV4cGxpY2l0Qm9keUxlbmd0aFByb3RlY3RzSGVhZGVyTGlrZUxpbmVzCTAyNThjNmVmNWFmM2YwYWMzNWQxOTkwZGViMjc2ODBkNjdkZGQxOGZkM2NlYWVjYjkzZjFkYTVjNjY5OTc1YjcKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0UGFyc2VBZGRyCTNhNTlhMWY0MzMzMzY5YjY4MmExZDU3NzExYTAzZThmZTU1YTI3NDllMDUyODkzNTY4Zjg5ZTY4MGU2ZjlhNGEKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0UGFyc2VNdWx0aXBsZUZpbGVzQW5kT3BzCTYxNWUzZTVjMzVjMmFhYWY0MDk0YzkzNjkzNzZjYTA0YmNiMDc5MjY0Y2M4ZjUxOWQyOTUxMjA4ZWQxZjQyMGYKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0UGFyc2VSZWplY3RzQmFkUGxhbnMJOTQwZTViZjUwZTM3NDdmMTQ4OGI2NzFkZmIyY2IyODdlYjIyOTVkNDVmZGMzOGMxZDE3NTA5ZTZlNzExYjRhYQpib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCVRlc3RQYXJzZVJlcG9ydHNFdmVyeUVycm9yCWU4NTE2NmI4ODU1NTE2NDdiNDI3NWFhNmNhNmEwYjg3Y2E0NjZiMjFlMzEzMTgzM2UxMTY1MWUzZDMxYzRhNjkKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0UXVvdGVkRmllbGRzU3Vydml2ZQkxZTI4ZjIxODFiOTI2MjY0NzM1ZmY1ZGM5NDEzMTUyMWM1YjAwNWQwOTJkMDc2OTc0MDQ1M2RhOWMzMDY2M2VjCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdFJhd1dpdGhvdXRCb2R5SXNSZWZ1c2VkCTJlZjYxNWFlNWNkZGJlM2VlYWFjNDU1OGViNTk0YWIxMjY2MDc1YTllNWI3ZmNhMmRjYzZjMjNhMzdiOGJlNzkKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0U2hhTXVzdEFjdHVhbGx5QmVIZXhhZGVjaW1hbAk3MDIwOWVmNDk3NDY0ZmZhMjQ0MGQ5MGU4NmZkMjUwNjhiY2E0Y2ZjNTE0NTlkMzZjMGMxNTBhNmE2Nzk5MGU4CmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JVGVzdFRoZUhpbnRzU3RheVF1aWV0T25PcmRpbmFyeUZhaWx1cmVzCWMyZjU0MzA3NTcxYmFiMjc5OGNhNjA2NGVlMTc2ZGUwYjQzZDkxYzM4NjBiMGI4MTFhZmUwN2FhOWY0YjI3MDQKYm9keQlpbnRlcm5hbC9wbGFuL3BsYW5fdGVzdC5nbwlUZXN0VW5xdW90ZWRBbmNob3JXaXRoRW1iZWRkZWRRdW90ZXNJc1JlZnVzZWQJZDE2NjhhY2E4ZTRlYzFhNmUzZDE5YjQ0NDhkNTQwMzQxYzQxZjg5YTRkN2E1ZDEwZDRjZThmNzQ3NTIyNTViNApib2R5CWludGVybmFsL3BsYW4vcGxhbl90ZXN0LmdvCXF1b3RlZAk5MjhkMTU2NWNiMDMzZDQ3YjA1NzAzMTBiOGExZDVjYjM0YjM1YjViZTJiOWU0MjRiMWI4Y2YxNWU5YTE0MzBiCmJvZHkJaW50ZXJuYWwvcGxhbi9wbGFuX3Rlc3QuZ28JdW5xdW90ZWQJNDQ3NjJjOWViNmRhOGYwNWQzYzA1NTQxY2U0OTA5NTJhMmJmMWUxNjY1ZjcwNjllZGZmNWFmYWE3ZTE3YzQzNg
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.168s
  === RUN   TestDryRunPrintsParsedHunkBodies
      dryrun_parsed_test.go:21: dry-run does not print parsed body=1:
          ok   a.go 2 replace  -1 +1
          1 hunk(s), 1 file(s), 0 failed, 0 advisories — dry run, nothing written
  --- FAIL: TestDryRunPrintsParsedHunkBodies (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.296s
  FAIL
  ```
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · ms:1409
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · ms:933
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · ms:602
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:e1cd42c454c34f837cfe41d5fe4eb5dec79651dda41b4eab7aa3db738f995d30 · ms:598
