# Task ADR-124-T1: {dirs} in scoped_check, refused in a step, taught

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `check.dirsOf`, the `{dirs}` placeholder
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `{dirs} in scoped_check, refused in a step, taught`

## Goal

`scoped_check` substitutes `{dirs}`, a `{files}`/`{dirs}` template runs scoped for any language, and a step holding `{dirs}` is refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `placeholders`, `command`, `dirsOf` |
| `internal/check/check_test.go` | edit | the tests |
| `AGENTS.md`, `README.md`, `internal/guide/guide.go`, `cmd/mrw/teach094_test.go` | edit | the step rule names `{dirs}`; the portable forms |
| `scripts/contract.sh` | edit | §222 |

## Ordered Steps

1. [S1] Write `TestDirsPlaceholderNamesEachEditedDirectoryOnce` and `TestAStepHoldingDirsIsRefused`. Confirm RED. [proof: mutation]
2. [S2] The token and the helper. Mutants: `{dirs}` out of `placeholders`; a `{dirs}`-only template falling back; `dirsOf` returning the paths. [proof: mutation]
3. [S3] Docs and contract §222. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestDirsPlaceholderNamesEachEditedDirectoryOnce|TestAStepHoldingDirsIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestDirsPlaceholderNamesEachEditedDirectoryOnce \(' "$out" \
  && grep -qE '^--- PASS: TestAStepHoldingDirsIsRefused \(' "$out" \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 222\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDirsPlaceholderNamesEachEditedDirectoryOnce` | `internal/check/check_test.go` | each directory once, sorted, `./dir` or `.`; a named directory; scoped for any language | none | S1, S2 |
| `TestAStepHoldingDirsIsRefused` | `internal/check/check_test.go` | `Placeholder` finds `{dirs}` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `dirsOf` |
| 2 — something selects it | `command` substitutes it |
| 3 — the caller can discover it | AGENTS.md, README, `mrw instructions` |
| 4 — it is used | the gap list's "check maps packages for Go only" |

## Mutation Log
- 2026-10-03 · 1c84cfd* · mutant killed · exit 1 · `internal/check/check.go` · S2: a step holding {dirs} is refused · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f
- 2026-10-03 · 1c84cfd* · mutant killed · exit 1 · `internal/check/check.go` · S2: a {dirs}-only template runs scoped for any language · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f
- 2026-10-03 · 1c84cfd* · mutant killed · exit 1 · `internal/check/check.go` · S2: {dirs} names directories, not the paths · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f

## Invariants

- `{packages}` and `{files}` expand exactly as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a Go project's scoped check changes.

## Out of Scope

- Per-ecosystem mapping (permanent: boundary: Zy chose the neutral token)

## Verification Log
- 2026-10-03 · 1c84cfd* · exit 1 · `set -o pipefail …` · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f · ms:459 · test-lock-sha256:c822a80f2e8a4f0790ed97e39fb5494b18394f7b1f4f63370777d273a88c17bf · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdENhbm5vdFN0YXJ0RGlkTm90UnVuCWQxM2E3MGJhNzI3NjE1MjQzZjhmNzA5NGMxMDJiMDlhM2ExNjRkMjNhNDMwYzMzM2U1MWU4YzI2ZmJmY2Y2MjIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBRGlyZWN0b3J5VGhhdENhbm5vdEJlUmVhZElzUmVmdXNlZE5vdFRyZWF0ZWRBc0VtcHR5CTY3MjM3YTkzYTUxMjZiNzhiY2VhYjAyMmFmZDM5MTQwMzQwOWFkMThmNjViNGQ2NWVlMjI4NjhmZDFmNmEyMDIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBRmFpbGluZ0NoZWNrS2VlcHNJdHNMb2cJOTkxZjQyMWYwZTc3YWI4NDBmMzI5NDJlMjZiYjY0YWFiYzcwOWU0YzRjZTBjMDliZmI5YzhkZmI1ZjFjODBlZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQYWRkZWRDaGVja0lzU3RpbGxUaGVEZWNsYXJlZENoZWNrCTk5ZTAyNjI2OTJmMzExZWRhNWM0Y2ZmNWEzMGQxYzA4ZjI4MWM5Yjk1NTMyODFhNzZmYTFkZmJhYTcxYmM4MTUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFkZGVkU2NvcGVkQ2hlY2tJc1N0aWxsRGVjbGFyZWQJNmM2ZWU4MGQ4M2IyZDZiZTY3NmMzZjljMDI3YWYzYmEwYjlkMDYyMjgwOTJlYTViYmMwYTNjMjVkNWU5YzgwNwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQYXNzaW5nQ2hlY2tMZWF2ZXNOb0xvZ0JlaGluZAk3NTllN2MzZDRjYzQxMTFjMTFhNDNjMzhiYzA1MWVhMjM3MzUyNjU4NGYzN2I1MzY5MDE5OThlNjRmYWFiMDcwCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVByZXNlbnRVbnBsYWNlYWJsZUluUm9vdFNjb3BlU3RpbGxGYWxsc0JhY2sJNTZhMDJkZjg2MTAyZjdlOWJjYmVlMzg4ZjA0MmI1NjdjYjA4ODczMTE1YjQ5MjYyYWE3ZjVhZTFkZTdmOGRiOQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFSZWFkYWJsZURpcmVjdG9yeVN0aWxsU2NvcGVzCWM2ZWFjYTY3YmQxMjZiODQ3NDljOWQ2YTNlZWYwMmUzMjY0MjQzODNjNWVhYTI4ZmY5MDk0ZjkwZWIwNzZjZDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUmVmdXNlZFNjb3BlRG9lc05vdEluaGVyaXRUaGVSb290c1ZlcmRpY3QJYmEwYjEzMzI3ZTEzNmNlZDVjNTI3OGMxNmVmMDIyMTc1MmIwNGU2YzBjYzc0YzRlZWM2OGUwMjQxZTNhM2Q1Ngpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFTY29wZU91dHNpZGVUaGVSb290SXNSZWZ1c2VkTm90RmFsbGVuQmFja1RvCWFiNzdmMTA4N2YwODVhZTRiN2Y3MjJjZjdjZDY2MzAyZjZkZTc3MmU3NmE3ODI1ZWNmZjU3MmUzNjdkNmMzYzUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBU2hlbGxJbmplY3RlZFNjb3BlU3RpbGxGYWlscwkxYWVhNjA3MzE4ZjM0YTFiMDhiZjU0MjUzNGJjYTY1NTZmMWM3ZjA3ZmFmZWNlZjRmMWY2NDdhOTFiNThlYzBkCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVN0ZXBIb2xkaW5nRGlyc0lzUmVmdXNlZAk5NDlmODc5NmYxOTZmNjcwMDZmZTRkMjJiZGRmZjc0MTNjMDQyNGY5YzQ1NDc0NTE0ZGM0MTczYzI2ODE0MDlmCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVdoaXRlc3BhY2VPbmx5Q2hlY2tJc05vdERlY2xhcmVkCWE5MDdlZTQzOWYxNGRjMmU0OGQyZTQxMzgwMTA4NzkxNTMzYTRhNGYwOGUyNGU4ZDBlZjJkNGIzNTc5MWVhNDQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBV2hpdGVzcGFjZU9ubHlTY29wZWRDaGVja0lzTm90RGVjbGFyZWQJZjZiZjBjNTQ5ZDY0NmE3NTA3NTBmZmE1MTBjOThmNjkzNTE1OGY4NWIzNzIyNjBlODg2MjUzNWI5YjRhYTUxYgpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFuSW5Sb290TWlzc0lzUmVmdXNlZE5vdEFTaWxlbnRQYXNzCTAxNzRhOTY2MTU4ZjJhOGQ2YTk5MjExZDcwZWUxYjEwZTBjMGY4MDg1NzhjMTFmYTg3NDYzMDcwMDViNDljZDEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBbk92ZXJsYXJnZVRpbWVvdXRJc0NsYW1wZWROb3RPdmVyZmxvd2VkCTYyMjkzNTRkOTUzMmZkMGFhNTY1YjMwYjQ1NGRkNzdlZGJmZDU5ZDViY2Q2NmI3NGNjZDE5OWRkMjIzODViYjQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3REaXJzUGxhY2Vob2xkZXJOYW1lc0VhY2hFZGl0ZWREaXJlY3RvcnlPbmNlCThlMWMyMjhmMDZkNDg5YWI2MDEzZWJjOGYyMGU1ZWFlNGUzNDQ5NWE3Yzk4M2Q5ZDc1YzhlMWE0OWIxOTNkMDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc09ubHlXaXRoTm9QYXRoc1N0aWxsRmFsbHNCYWNrCTAxOGIxNjg1NGUxNTdmZDM3YThlODBjYThiYjNmZTU5ZDY1MzY5ZDVkY2E5N2EyMmFkZDQ0ZDM2NTk2ZTI0ZWIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc1BsYWNlaG9sZGVyCWZhNGJhMGE2ZDU0MzczNjAwZjAwMjQ3OWIwOThhNzUxMTIxOWEyNTQwOGM2YTY5ODQxNTRkZmUyZTA1OTM3YzEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc1BsYWNlaG9sZGVyRG9lc05vdE5lZWRQYWNrYWdlcwk4MWNmZDQzNTc1MzQ4MGU5NmRkZTE1MjE3YmVlMmFhODU5MDYwOWVkNjNlMGVmNDc4MDhmNDU1OTA0ZjU1NjdkCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TG9hZEluZmVyc0dvQ2hlY2tCdXRTYXlzU28JMDk5ZTZiZmYzZmU3N2EzMzkwNzUzYzVlZmJjN2UyMTU1ZDNiNjQwNjlhOTliMmUyZGUxZWVmZGQzOWViMzMxYQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRPbkFCYXJlRGlyZWN0b3J5SGFzTm9DaGVjawlmNTVlMmIzZmVlZWYzNzhmYmM5MjU5NGVmYzRmYjViZmQ2MzM0YjM3MzU5ODUzYjM1M2MzZjgyODVjYzE0ZTYzCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TG9hZFByZWZlcnNUaGVEZWNsYXJlZENoZWNrCTI1ZDFmN2NlNTlkZTUzZjhkYmNjODZmODhlYmI1NWU1Y2UzNGU0ZTZlZTI5NzFmYmFkODNlMGZlNmNlMjkwOWUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RNaXhlZFBsYWNlaG9sZGVyc0ZhbGxCYWNrV2hlblVubWFwcGVkCTFhYzc3MzhjOTc5YzcwODFiNzFhMjI4OTA2MjYzMDQ2OTg2ZWNmNTJkM2I5YzYxNTcyZWRkOGM5N2FkM2QzMzcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3ROb0NoZWNrSXNOb3RBUGFzcwlhM2IzM2RhZTQyN2VhNmM1YTZiYjA4OTk0MDgyMmRlNWFjOGJlOTQyYWViMDIxOTBiYTVhODQ2MDNiMDU4YWI2CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UGFja2FnZXNPbmx5Tm9uR29TdGlsbEZhbGxzQmFjawkxYmEwYjI1MzBiZmM5M2Y0ODY3MTBlOTAyZTYyOTY0YTM3OGRlNjFlMDNjN2RkZDZiNGVmOGY3YTlmNDU4MzllCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UnVuUGFzc2VzCThlYjcwZmY4Zjg5MWM5MDNjYmU2ZDhiOTIwMjVjYzk5NzQ3NTU2YTkyYmRjYTMwYzhjNGE2MTA4ZjVlNThmYTcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RSdW5SZXBvcnRzVGhlUmVhbEV4aXRDb2RlCWEwYTRjMGY2MTBkOTVjZjQ0MWMwMGQ4NWJkODI0YTg1ZDUzOTY2MGFiMjdkNjNiOTBkMjQ2ODQxYzE1N2FkZDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RTY29wZURlcml2YXRpb24JNjRlOGFiZjc4YzVlMzgzNTY2YWZkMWU1MjliM2UwYWNjNWFlYjZmZWVjOGYxYzI4ZTQwN2E3YTM1OTI0M2UzNApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFN1YnN0aXR1dGVkUGF0aHNBcmVPbmVTaGVsbEFyZ3VtZW50RWFjaAllYzNlZGVlZTk1NWJkYmMwN2NhOGRkYmVjMjU0YTI1ODMwMzFjMzJkNGIwMjBmNjY2Mzc4NTIxZDQ5Y2Q2MTU2CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0VGFpbEFubm91bmNlc1doYXRJdExlZnRPdXQJNDc0OTQzZGM2MTUzOWEyZWI0NWE4Yzc5MzM4YjllOTY4ZjhiY2QyZWIwOTBlMTdlNzBiODdkYzhjYmM4NmQ4NQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFRpbWVvdXRJc1JlcG9ydGVkQXNBRmFpbHVyZU5vdEFQYXNzCWZmN2QwNjBjODc1ZWQ3ZTVkOWY2Y2JiMTMyMjJjYWJjZTQyMjlkZjcwNmYyYzBjYzJkMGE1OTdlOTQ0YWUyYWU
  ```
  --- last 9 line(s) of stdout
  === RUN   TestDirsPlaceholderNamesEachEditedDirectoryOnce
      check_test.go:788: got "FULL" (scoped false), want pytest . ./a ./b/c
  --- FAIL: TestDirsPlaceholderNamesEachEditedDirectoryOnce (0.00s)
  === RUN   TestAStepHoldingDirsIsRefused
      check_test.go:800: Placeholder = "", want {dirs}
  --- FAIL: TestAStepHoldingDirsIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.198s
  FAIL
  ```
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f · ms:37731
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f · ms:37365
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:32672792fc8ebcf6310ecb08fd4f2fad965374513cbdf310d6a4d8da44c5230f · ms:37370
