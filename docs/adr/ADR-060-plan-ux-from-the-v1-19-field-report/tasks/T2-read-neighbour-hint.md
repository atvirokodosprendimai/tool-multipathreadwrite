# Task ADR-060-T2: `read` hints a multi-line replace needs the next line

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** neighbour hint on ranged read; §101
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `multi-line non-EOF span prints the hint`, `whole-file read is quiet`, `single-line span is quiet`

## Goal

A served span of two or more lines that is not through last line prints one note that a multi-line replace of that span needs a served line after End.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | Print the note after `@@ start-end` body. |
| `internal/read/read_test.go` | edit | Red: hint present / absent. |
| `scripts/contract.sh` | edit | **§101**. |

## Ordered Steps

1. [S1] Write the failing tests. [proof: mutation]
2. [S2] Print the hint. S1 GREEN. [proof: mutation]
3. [S3] §101 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 101\. ' scripts/contract.sh \
  && go test ./internal/read/ -count=1 -v \
    -run 'TestReadHintsAMultiLineReplaceNeedsTheNextLine' 2>&1 | tee /tmp/adr060-t2.out \
  && grep -q '^--- PASS: TestReadHintsAMultiLineReplaceNeedsTheNextLine' /tmp/adr060-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadHintsAMultiLineReplaceNeedsTheNextLine` | `internal/read/read_test.go` | `a.go:3-5` on a longer file prints the note naming 3-5 and after 5; whole-file read has no note; `a.go:3` has no note | — | S1, S2 |

The three cases are subtests of that one name so the fence's mutant still runs the absences.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `read.Run` after `@@ start-end`. Deleting the print leaves S1 red |
| 3 — the caller can discover it | §101; T5 teach |
| 4 — it is used | quality-blueprints first multi-line replace of a fresh range; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/read/read.go` · neighbour hint says beyond not after, so TestReadHintsAMultiLineReplaceNeedsTheNextLine ranged subtest must go red · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · covers:multi-line non-EOF span prints the hint
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/read/read.go` · whole-file multi-line span also prints the hint so the whole subtest must go red · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · covers:whole-file read is quiet
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/read/read.go` · single-line span also prints the hint so the single subtest must go red · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · covers:single-line span is quiet

## Invariants

- ADR-052 licence is unchanged.
- `--stat` does not print spans, so no hint.
- Hint is not a checker; a write without the neighbour still refuses at apply.

## Risks

- Existing read tests that compare full output. Additive note; update if an exact-equality test fails.

## Stop Condition

Stop if the hint requires changing the neighbour licence.

## Out of Scope

- Single-line neighbour licence (BACKLOG).
- `--echo-pad` (ADR-052).

## Verification Log
_(tool-written)_
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · ms:355 · test-lock-sha256:4cfc511bcaa43861193a4231a4ffb3809439adda2f90845938e8c477f8fac1ba · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBUmFuZ2VBZ2FpbnN0QW5FbXB0eUZpbGVJc1JlcG9ydGVkCWQ4OWRhYTQyZjk5ODliZDY1YmQxMzVkZGQwMGViNzQxNDc0MjA4NWM1ODUxYmFhZjNmZDcxNjEwYjVlMjVkZjYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJhbmdlVGhhdE1hdGNoZXNOb3RoaW5nT2JzZXJ2ZXNOb3RoaW5nCWRmZTYyYTgzYzUxYjlkODEyMmI3ZjlmMzc3N2JlYmI5YzJlODc2N2UyYjJkOWJmYWQzNTRjNDI1YWZjNWE5OWIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJldmVyc2VkUmFuZ2VXcml0dGVuV2l0aERvbGxhcklzTm90U2VydmVkCTgzYjgwNDU5NzE1NjIzNzA4YWY1ZTZlZmE2YWMwNmM5ZTVmNWQ4MWQ4Mzg4N2FiZmUyN2I5MDZkY2QyYWNjMjMKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0Q29udGV4dEFyb3VuZFNpbmdsZVBhdHRlcm4JYTA2YWQ4MDdiMWE1OWE4ZThhMWYwOWY2MDM1N2RiZmU5NGE0MjVmY2VkYTY3MzFjOGI4Y2UxMWQzOTAzY2MyZgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3REZXNjZW5kaW5nUmFuZ2VzQXJlU29ydGVkQW5kTWVyZ2VkCTcwNjRiNDFhNzg5MTNiN2ZkNzBhNzliNjBlYjU1ZDAxMjJiOGUzMWNkZGE0NjNhMmEwODFlMzYwZWZkY2VmNTAKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0RG9sbGFyQXNBbkVuZFN0aWxsUnVuc1RvRU9GCWY5NjA2NWY0Yjk2Yzc1Nzc5MWVlYTc5MTMwZGNlZmJmOGI2ZWFjY2QzMzdjZmY4YzNkMjM5NTkwNzIzZDY2NDUKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0RG9sbGFySXNUaGVMYXN0TGluZU5vdFRoZVdob2xlRmlsZQk0YjM0ZjMxMWFmYjU2MDI5ZjdjNTFiMDllMDg0YmY4MjNkYTI0ODBhMDhlMzJlYjZhOWMwNzEwOWE2N2YyZjdmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdE1hbnlGaWxlc09uZUNhbGwJM2RmNTQ4NGRkNTJiNjc3ZThiOTY1ZmZiZjRiNDQ4NmNiNTI3OTNiNzY1NjJhMDk0Y2M0ZjNmNTA0MjgzNGEwYwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RNYW55VW5zb3J0ZWRSYW5nZXNNZXJnZUNvcnJlY3RseQlkZDMyZDVmZGMwNDY4OTdlMmMzNWE3Y2MzYzk4YTk2MzNjNzhlYjVhMzQxMTczM2Q1ZDg0YjkyNzAxNzg2ZjBiCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdE1heExpbmVzUmVwb3J0c1doYXRJdFdpdGhoZWxkCTY0ZDJiMWRiYjIxMzQxMTgxNzcxNTI3YjAzM2ZjNTU3NjNjZDYzM2NiNWI5OWI2MGUyODExMGQ3MzY3NTNlN2EKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0TnVtZXJpY1Jhbmdlc0FuZEhlYWRlcgliZDdlNTM5MDkxMjY5YjRkNzIwZmUzYmE1ZWFiYmQ4Y2UxMzVhYTkxMTQwMWNkYWI4NTg1YzRjMmMxMGI5YzdmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdE92ZXJsYXBwaW5nUmFuZ2VzQXJlTWVyZ2VkTm90UmVwZWF0ZWQJN2U5M2YyNDg0MzljZDUwOThmZWVjY2IzNmE5ZmFiZWJhNDg5Y2FiNzkzNDg4MzY2YjNiM2E0NGQ4YzBlNjBlMQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RQYXJzZVNwZWMJZTdjZDVlZmIwMmExYzdmYTkxMmRiZDZjMzIxZGMwY2ZjZWRkMTFjZjc1MzFkMWVjYjNiY2U4MDc4YjY1ZDNjYwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RSZWFkSGludHNBTXVsdGlMaW5lUmVwbGFjZU5lZWRzVGhlTmV4dExpbmUJODExOWNhYjJjODkwNDk5MDg0ZTRmMWM4ODAwMTJiYjMwYmYzNTYxM2M3MTg5NzI5M2I2NzcyOWI1YTc3ZDk2Zgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RSZWdleHBSYW5nZQkyMzAzNDFiM2Q5OWM3ODc3N2EzNWI5MjlmZDY2MDU2MTE2MTNiNDRkOWZkZDQwNDFjMDcwYTE3YTY0ZjI3MmY1CmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFN0YXRBc2tzRm9yVGhlRmFjdE5vdFRoZUFydGlmYWN0CTQwZWJmNjVhYzBiYWFiNThkNTA1MjdmMTBkZGEzNzVmZGMwNDlhYTEyZDFlNzUyZDA4NzUzOTg3N2EyY2ZlZWIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0VW5tYXRjaGVkUGF0dGVybklzUmVwb3J0ZWQJZThlYTM3MWNjZmI4YzczODBiODE5ZGY5ZThjODUwOWMxMjlmZWUxNDZlNDc5NzdjZGEwOGFiMGFkOTc5YzJjMQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RXaG9sZUFuZFBhcnRpYWxPYnNlcnZhdGlvbnNBcmVVbmNoYW5nZWQJNmI2ZTM0YzIwNGM2ZjQxZTAyYTA1YjJmYTFlZjQwYTliNTM2YWExM2QzMjNhOGIwNGQ3MGJkYjRjMGZkNzgzNgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCXJhbmdlZAljMTAyZjg4NGJkZTBhODg1ZmM3YWUxYzI5YzQ3NTRmMTA3MGNkNjk4MzQ0ZGM5MjU0NTZjMmM2NmRlMmVhYWMyCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28Jc2luZ2xlCWY4NDIzMWY3MzRhM2ZmMjg1YmI5YTdjZWFjNjgyNjFmMmVmZTI4OWY0MGFlYWY5OTQwY2M3YzY1MDZiMmFiZWYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwl3aG9sZQkzZDJlNTllOTNkMDUzNGUwMmIyNWIwODhiNWZkNGMyMGVkNDJiNGMzY2QzYjJlMmE5Y2NhNDM1MmI2OWU1Yjdl
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
              5| }
  === RUN   TestReadHintsAMultiLineReplaceNeedsTheNextLine/whole
  === RUN   TestReadHintsAMultiLineReplaceNeedsTheNextLine/single
  --- FAIL: TestReadHintsAMultiLineReplaceNeedsTheNextLine (0.00s)
      --- FAIL: TestReadHintsAMultiLineReplaceNeedsTheNextLine/ranged (0.00s)
      --- PASS: TestReadHintsAMultiLineReplaceNeedsTheNextLine/whole (0.00s)
      --- PASS: TestReadHintsAMultiLineReplaceNeedsTheNextLine/single (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.186s
  FAIL
  ```
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · ms:539
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · ms:441
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:6d211cb19da384f319e8dae702c6f971ef0ee09f9a9c4231d254ba4967f6c675 · ms:429
