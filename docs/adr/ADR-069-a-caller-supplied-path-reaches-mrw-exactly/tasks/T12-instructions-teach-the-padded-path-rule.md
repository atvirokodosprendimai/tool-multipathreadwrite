# Task ADR-069-T12: `mrw instructions` teaches the padded-path rule; contract §140

**Depends-on:** T11
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one line in `guide.CLI`
**Consumes:** T1's refusal and T5's attached-value refusal
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `instructions teach the rule`, `the binary prints it`, `no engine file changes`

## Goal

The v1.25.1 field test of the downloaded Windows asset found that `mrw instructions` said nothing about the padded-path refusal or the `--` escape, so a caller with only the binary learned the rule at the refusal. ADR-063 promised the read side's traps there. The fix: one line in `guide.CLI` — a padded path goes after `--`; an attached value ending in whitespace is passed as its own argument.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | the line |
| `internal/guide/guide_test.go` | edit | the test |
| `scripts/contract.sh` | edit | §140 |

## Ordered Steps

1. [S1] Write the test; confirm RED. [proof: mutation]
2. [S2] Add the line; GREEN. [proof: mutation]
   Mutant: the line teaches the trim as the rule.
3. [S3] §140 through the binary. [proof: mutation]

## Acceptance

```bash
set -o pipefail
grep -q '^# 140\. ' scripts/contract.sh \
  && go test ./internal/guide/ -count=1 -v -run 'TestInstructionsTeachThePaddedPathRule' 2>&1 | tee /tmp/adr069-T12.out \
  && grep -q '^--- PASS: TestInstructionsTeachThePaddedPathRule ' /tmp/adr069-T12.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T12.out \
  && ./scripts/contract.sh > /tmp/adr069-T12-contract.out 2>&1 \
  && grep -q '^  PASS  instructions teach that a padded path goes after --' /tmp/adr069-T12-contract.out \
  && grep -q '^  PASS  and they do not teach the trim as the rule' /tmp/adr069-T12-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/guide)" ] \
  && go vet ./internal/guide/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsTeachThePaddedPathRule` | `internal/guide/guide_test.go` | `CLI()` carries the `--` rule and the attached-value rule | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §140 |
| 2 — something selects it | `mrw instructions` prints `CLI()` verbatim |
| 3 — the caller can discover it | it is the discovery path |
| 4 — it is used | a Windows field tester read `instructions` looking for exactly this and did not find it |

## Verification Log
(empty until execute)
- 2026-09-25 · 5cf522c* · exit 1 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:351 · test-lock-sha256:41fc4078025abb0d152786584f3ef8d78cbfc352e64740f1a5d41532e6e6e152 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSUNvbnRhaW5zU2hhcmVkQW5kVGhlT3BlcmF0b3JUcmFwcwk2NTZiMWY2MzU0ZDRjYzY0NmEzNjAxZDBkYWUzY2Y4MmNkNzlhMGNhMGY2MTM0NzQ0ZTJkODU1ZjQwY2FlNzU1CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc0Fsd2F5c0FuZEFQbGFuCTg3NmQ4ZDVlZWU2MGIwODk5ZjI0ODhmYzdhOGU3YzM4YzQ5Mzk1MWY0ZTRmODcxNDcxZGIzMGM1ZjBmN2U3MTEKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzVGhlUmVhZFNpZGUJMDFmYzAzYTE1MGMyMTdjMTExZDNlZjE2MDAyMDliMjNlYmY2YTBlZTY1NzA4ZjRiMGNiMmIxNzAzN2QxZjlkMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEV2ZXJ5U3VyZmFjZUNvbnRhaW5zVGhlU2hhcmVkU2VudGVuY2VzCWM0ZTMxNjkzMWUzNWYxNWJlODljZmU0ZWEzYTRmNWQ1ODZiNzI3NDE0OTE5ODYwYjNkYTMxMzY3MTUzY2Y3NzAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNUZWFjaFRoZVBhZGRlZFBhdGhSdWxlCTg2NTgyMDMwNjU2ZTg2ZTdkN2U3MGZiY2Y3NGU1ZDNhMTRiNDQxN2I3NGE2YzY1NjA3NTVjY2I4ZTc5NTI4YTcKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3ROb1NlcnZlZFRleHRQcm9taXNlc05vdGhpbmdXcml0dGVuT25BbnlGYWlsdXJlCTBjMzUzN2I1OTc0ODZmZmY2OTM5NTEwZDI2NjgzOTc1NDIzYmI2MzgwZjEzOTJjNGVmZjRjNGFhYTY3MGI5ZGQKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RTaGFyZWRTYXlzQUZhaWxlZENvbW1pdElzUmVwb3J0ZWRQYXJ0aWFsCWRmOTFhNzIyZTU1ODExMDcyNzFhMDkxZDFhZDEyMzYwMWJlZDlhMjBhNzA5NWJmM2ZmODc1M2Q1OGJiNDE4ZGM
  ```
  --- last 8 line(s) of stdout
  === RUN   TestInstructionsTeachThePaddedPathRule
      guide_test.go:150: mrw instructions lacks "goes after --"
      guide_test.go:150: mrw instructions lacks "mrw read -- 'x '"
      guide_test.go:150: mrw instructions lacks "--files-from 'list '"
  --- FAIL: TestInstructionsTeachThePaddedPathRule (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.166s
  FAIL
  ```
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:66978
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:47094
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:33570
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:34992
- 2026-10-22 · be843d71* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:0 · test-lock-sha256:5bfb63b5348dec9e700583611070bbc93266c8c73d35eef8c462f40ce5d01e1c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJQ29udGFpbnNTaGFyZWRBbmRUaGVPcGVyYXRvclRyYXBzCTY1NmIxZjYzNTRkNGNjNjQ2YTM2MDFkMGRhZTNjZjgyY2Q3OWEwY2EwZjYxMzQ3NDRlMmQ4NTVmNDBjYWU3NTUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzQWx3YXlzQW5kQVBsYW4JODc2ZDhkNWVlZTYwYjA4OTlmMjQ4OGZjN2E4ZTdjMzhjNDkzOTUxZjRlNGY4NzE0NzFkYjMwYzVmMGY3ZTcxMQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNUaGVSZWFkU2lkZQlkZGM1YzUzNGZlNjRmMGE0NjZkZGJjOWI3MmM4OTVkNDg3ZDNhMDE1ZDIzMzFlM2Y2Yjg0NzVkNDI2Zjg2MTNkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlQ29udGFpbnNUaGVTaGFyZWRTZW50ZW5jZXMJYzRlMzE2OTMxZTM1ZjE1YmU4OWNmZTRlYTNhNGY1ZDU4NmI3Mjc0MTQ5MTk4NjBiM2RhMzEzNjcxNTNjZjc3MApib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0xlYWRXaXRoVGhlUmVjaXBlcwk1MjU2YjQ0YWU1MWIyMjE4MWVlYzVhNDdkNjdlODA2YTEyY2M3ZTg0MjI5YTI4NGFjZGUxOTY5MjAyYmFkM2YzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zU2F5QVdyaXR0ZW5GaWxlSXNXaG9sbHlLbm93bgkyYThlODZkMjg0Y2E5Y2IzZWZmNDYxMjc2MTg3OWIwODQxMDYzMWJkNWNjZmIzMjY2OTY0ZjMwNDEzYzkwYmJlCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hUaGVQYWRkZWRQYXRoUnVsZQk4NjU4MjAzMDY1NmU4NmU3ZDdlNzBmYmNmNzRlNWQzYTE0YjQ0MTdiNzRhNmM2NTYwNzU1Y2NiOGU3OTUyOGE3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Tm9TZXJ2ZWRUZXh0UHJvbWlzZXNOb3RoaW5nV3JpdHRlbk9uQW55RmFpbHVyZQkwYzM1MzdiNTk3NDg2ZmZmNjkzOTUxMGQyNjY4Mzk3NTQyM2JiNjM4MGYxMzkyYzRlZmY0YzRhYWE2NzBiOWRkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0U2hhcmVkU2F5c0FGYWlsZWRDb21taXRJc1JlcG9ydGVkUGFydGlhbAlkZjkxYTcyMmU1NTgxMTA3MjcxYTA5MWQxYWQxMjM2MDFiZWQ5YTIwYTcwOTViZjNmZjg3NTNkNThiYjQxOGRj · test-lock-kind:replace
- 2026-10-22 · f862d8a3* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:0 · test-lock-sha256:dee436843dc76889ecaef90cca83262b63d2c651de59a32512fb6f43b3d1c20c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJQ29udGFpbnNTaGFyZWRBbmRUaGVPcGVyYXRvclRyYXBzCTY1NmIxZjYzNTRkNGNjNjQ2YTM2MDFkMGRhZTNjZjgyY2Q3OWEwY2EwZjYxMzQ3NDRlMmQ4NTVmNDBjYWU3NTUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzQWx3YXlzQW5kQVBsYW4JODc2ZDhkNWVlZTYwYjA4OTlmMjQ4OGZjN2E4ZTdjMzhjNDkzOTUxZjRlNGY4NzE0NzFkYjMwYzVmMGY3ZTcxMQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNUaGVSZWFkU2lkZQkxMGY2M2E5OThmYzUyMzhmN2QwN2Y1OWQxMjk1MTJiMmJhMzFhMmRkZWM5YzY1YTA5MjVkNzkwMTZmZmQyNjQyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlQ29udGFpbnNUaGVTaGFyZWRTZW50ZW5jZXMJYzRlMzE2OTMxZTM1ZjE1YmU4OWNmZTRlYTNhNGY1ZDU4NmI3Mjc0MTQ5MTk4NjBiM2RhMzEzNjcxNTNjZjc3MApib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0xlYWRXaXRoVGhlUmVjaXBlcwk1MjU2YjQ0YWU1MWIyMjE4MWVlYzVhNDdkNjdlODA2YTEyY2M3ZTg0MjI5YTI4NGFjZGUxOTY5MjAyYmFkM2YzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zU2F5QVdyaXR0ZW5GaWxlSXNXaG9sbHlLbm93bgkyYThlODZkMjg0Y2E5Y2IzZWZmNDYxMjc2MTg3OWIwODQxMDYzMWJkNWNjZmIzMjY2OTY0ZjMwNDEzYzkwYmJlCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hUaGVQYWRkZWRQYXRoUnVsZQk4NjU4MjAzMDY1NmU4NmU3ZDdlNzBmYmNmNzRlNWQzYTE0YjQ0MTdiNzRhNmM2NTYwNzU1Y2NiOGU3OTUyOGE3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Tm9TZXJ2ZWRUZXh0UHJvbWlzZXNOb3RoaW5nV3JpdHRlbk9uQW55RmFpbHVyZQkwYzM1MzdiNTk3NDg2ZmZmNjkzOTUxMGQyNjY4Mzk3NTQyM2JiNjM4MGYxMzkyYzRlZmY0YzRhYWE2NzBiOWRkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0U2hhcmVkU2F5c0FGYWlsZWRDb21taXRJc1JlcG9ydGVkUGFydGlhbAlkZjkxYTcyMmU1NTgxMTA3MjcxYTA5MWQxYWQxMjM2MDFiZWQ5YTIwYTcwOTViZjNmZjg3NTNkNThiYjQxOGRj · test-lock-kind:replace
- 2026-10-22 · b03796e7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:0 · test-lock-sha256:4bf0134a48d71ce0db7c0a7c9629bcad523e8f9ca41a0db81c7e81219c1536aa · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJQ29udGFpbnNTaGFyZWRBbmRUaGVPcGVyYXRvclRyYXBzCTY1NmIxZjYzNTRkNGNjNjQ2YTM2MDFkMGRhZTNjZjgyY2Q3OWEwY2EwZjYxMzQ3NDRlMmQ4NTVmNDBjYWU3NTUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzQWx3YXlzQW5kQVBsYW4JODc2ZDhkNWVlZTYwYjA4OTlmMjQ4OGZjN2E4ZTdjMzhjNDkzOTUxZjRlNGY4NzE0NzFkYjMwYzVmMGY3ZTcxMQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNUaGVSZWFkU2lkZQkxMGY2M2E5OThmYzUyMzhmN2QwN2Y1OWQxMjk1MTJiMmJhMzFhMmRkZWM5YzY1YTA5MjVkNzkwMTZmZmQyNjQyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlQ29udGFpbnNUaGVTaGFyZWRTZW50ZW5jZXMJYzRlMzE2OTMxZTM1ZjE1YmU4OWNmZTRlYTNhNGY1ZDU4NmI3Mjc0MTQ5MTk4NjBiM2RhMzEzNjcxNTNjZjc3MApib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0xlYWRXaXRoVGhlUmVjaXBlcwkxN2Y3MTUyZjY1NzdkYWU1MTMyZGUwYTA0OWUwNTJiMDdkYWQxOWFiN2M1ZWRjZjUwZmM1OWI3ZDFkZWI2MTNkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zU2F5QVdyaXR0ZW5GaWxlSXNXaG9sbHlLbm93bgkyYThlODZkMjg0Y2E5Y2IzZWZmNDYxMjc2MTg3OWIwODQxMDYzMWJkNWNjZmIzMjY2OTY0ZjMwNDEzYzkwYmJlCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hUaGVQYWRkZWRQYXRoUnVsZQk4NjU4MjAzMDY1NmU4NmU3ZDdlNzBmYmNmNzRlNWQzYTE0YjQ0MTdiNzRhNmM2NTYwNzU1Y2NiOGU3OTUyOGE3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Tm9TZXJ2ZWRUZXh0UHJvbWlzZXNOb3RoaW5nV3JpdHRlbk9uQW55RmFpbHVyZQkwYzM1MzdiNTk3NDg2ZmZmNjkzOTUxMGQyNjY4Mzk3NTQyM2JiNjM4MGYxMzkyYzRlZmY0YzRhYWE2NzBiOWRkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0U2hhcmVkU2F5c0FGYWlsZWRDb21taXRJc1JlcG9ydGVkUGFydGlhbAlkZjkxYTcyMmU1NTgxMTA3MjcxYTA5MWQxYWQxMjM2MDFiZWQ5YTIwYTcwOTViZjNmZjg3NTNkNThiYjQxOGRj · test-lock-kind:replace
- 2026-10-23 · 4bfd6600* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:0 · test-lock-sha256:5d89b763f3e34b9e69a43b222a4d923b03684e809218c3f70faf2d0b77430e58 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJQ29udGFpbnNTaGFyZWRBbmRUaGVPcGVyYXRvclRyYXBzCTY1NmIxZjYzNTRkNGNjNjQ2YTM2MDFkMGRhZTNjZjgyY2Q3OWEwY2EwZjYxMzQ3NDRlMmQ4NTVmNDBjYWU3NTUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzQWx3YXlzQW5kQVBsYW4JODc2ZDhkNWVlZTYwYjA4OTlmMjQ4OGZjN2E4ZTdjMzhjNDkzOTUxZjRlNGY4NzE0NzFkYjMwYzVmMGY3ZTcxMQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNUaGVSZWFkU2lkZQkxMGY2M2E5OThmYzUyMzhmN2QwN2Y1OWQxMjk1MTJiMmJhMzFhMmRkZWM5YzY1YTA5MjVkNzkwMTZmZmQyNjQyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlQ29udGFpbnNUaGVTaGFyZWRTZW50ZW5jZXMJYzRlMzE2OTMxZTM1ZjE1YmU4OWNmZTRlYTNhNGY1ZDU4NmI3Mjc0MTQ5MTk4NjBiM2RhMzEzNjcxNTNjZjc3MApib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0xlYWRXaXRoVGhlUmVjaXBlcwk1MTE5NTkzOTc1NDFlMTEwYTgwMmM3NGY2N2Q0MjMwMjY0ODBjYmYzZGI1OWQ2NWQxZDU4NjNlNzA1OTkwNWI0CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zU2F5QVdyaXR0ZW5GaWxlSXNXaG9sbHlLbm93bgkyYThlODZkMjg0Y2E5Y2IzZWZmNDYxMjc2MTg3OWIwODQxMDYzMWJkNWNjZmIzMjY2OTY0ZjMwNDEzYzkwYmJlCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hUaGVQYWRkZWRQYXRoUnVsZQk4NjU4MjAzMDY1NmU4NmU3ZDdlNzBmYmNmNzRlNWQzYTE0YjQ0MTdiNzRhNmM2NTYwNzU1Y2NiOGU3OTUyOGE3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Tm9TZXJ2ZWRUZXh0UHJvbWlzZXNOb3RoaW5nV3JpdHRlbk9uQW55RmFpbHVyZQkwYzM1MzdiNTk3NDg2ZmZmNjkzOTUxMGQyNjY4Mzk3NTQyM2JiNjM4MGYxMzkyYzRlZmY0YzRhYWE2NzBiOWRkCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0U2hhcmVkU2F5c0FGYWlsZWRDb21taXRJc1JlcG9ydGVkUGFydGlhbAlkZjkxYTcyMmU1NTgxMTA3MjcxYTA5MWQxYWQxMjM2MDFiZWQ5YTIwYTcwOTViZjNmZjg3NTNkNThiYjQxOGRj · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed for ADR-169: the moved locks are additive edits to shared tests — TestInstructionsLeadWithTheRecipes counts 11 recipes (ADR-169 adds the re-read guard), TestTheReadReceiptMatchesItsSchema declares the unchanged list, TestNoShippedMCPReceiptFieldDisappears reads a list's items; no assertion removed or weakened; approved · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f

## Mutation Log
(empty until execute)
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/guide/guide.go` · the line teaches the trim as the rule: the test and §140 go red · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:instructions teach the rule
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/guide/guide.go` · the attached-value half is dropped: the test and §140 go red through the binary · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:the binary prints it
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:no engine file changes

## Invariants

- Every earlier `guide` test stays green; `mrw instructions` still exits 0 with no flags.

## Risks

- None known.

## Out of Scope

- Teaching the rule in the MCP handshake: the MCP tools take specs as JSON and never pass through the argument parser (permanent: boundary: ADR-068 keeps a spec exactly on that surface; citation: `internal/mcp/tools.go`)

## Stop Condition

Stop if the line needs a format change or an exit-code change.
