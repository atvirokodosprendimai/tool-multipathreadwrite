# Task ADR-060-T5: failing check prints last error; teach

**Depends-on:** T1, T2, T3, T4
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `check last:`; §104; AGENTS/help/BACKLOG
**Consumes:** T1–T4 served-path behaviour
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `FAIL prints check last above full output`, `PASS has no check last`

## Goal

Exit 3 prints the check's last non-empty Tail line as `check last:` immediately above `full output:`. Help and AGENTS name all five Decision items. BACKLOG records the field report as shipped.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `reportCheck` last-error line; `write --help` `body=@`. |
| `cmd/mrw/writecheck_test.go` | edit | Red: FAIL has `check last:` above the log path. |
| `internal/guide/guide.go` | edit | `body=@path`; unquoted `"`; leftover. |
| `cmd/mrw/writehelp_test.go` | edit | Help still names quoting and now `body=@`. |
| `AGENTS.md` | edit | The five items. |
| `docs/adr/BACKLOG.md` | edit | Inventory row shipped. |
| `scripts/contract.sh` | edit | **§104**. |

## Ordered Steps

1. [S1] Write the failing check-last test. [proof: mutation]
2. [S2] Print `check last:`. S1 GREEN. [proof: mutation]
3. [S3] §104 RED then GREEN. [proof: mutation]
4. [S4] Teach help, guide, AGENTS, BACKLOG. [proof: acceptance]
5. [S5] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 104\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestFailedCheckPrintsTheLastErrorLine|TestWriteHelpNamesBodyAtPath' 2>&1 | tee /tmp/adr060-t5.out \
  && grep -q '^--- PASS: TestFailedCheckPrintsTheLastErrorLine' /tmp/adr060-t5.out \
  && grep -q '^--- PASS: TestWriteHelpNamesBodyAtPath' /tmp/adr060-t5.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t5.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestFailedCheckPrintsTheLastErrorLine` | `cmd/mrw/writecheck_test.go` | a failing declared check's receipt contains `check last:` with the last error, then `full output:`; a passing check does not print `check last:` | — | S1, S2 |
| `TestWriteHelpNamesBodyAtPath` | `cmd/mrw/writehelp_test.go` | `write --help` names `body=@` | — | S4 |

PASS absence is a subtest of that name.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `reportCheck` on `!r.OK()`. Deleting the print leaves S1 red |
| 3 — the caller can discover it | §104; AGENTS; `write --help` |
| 4 — it is used | quality-blueprints exit 3 / missing `node_modules`; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `cmd/mrw/main.go` · FAIL prints last check: not check last: so TestFailedCheckPrintsTheLastErrorLine must go red · acceptance-sha256:0f7b601f6b894654cf21244e9b59ed3551a524867eca9229a215c8c44b8237ec · covers:FAIL prints check last above full output
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `cmd/mrw/main.go` · check last prints on PASS not FAIL so both subtests of TestFailedCheckPrintsTheLastErrorLine must go red · acceptance-sha256:0f7b601f6b894654cf21244e9b59ed3551a524867eca9229a215c8c44b8237ec · covers:PASS has no check last

## Invariants

- Verdict still from the process exit (ADR-003).
- Tail dump of up to 30 lines stays.
- `writeCmd` Description stays a raw string without nested backticks.

## Risks

- A check with empty Tail. Skip `check last:` rather than printing a blank.

## Stop Condition

Stop if printing the last line requires changing `check.Run` or Tail length.

## Out of Scope

- Changing default Tail (record Out of Scope).
- Closing the palace inbox drawer is a memory write after merge, not a repo file.

## Verification Log
_(tool-written)_
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:0f7b601f6b894654cf21244e9b59ed3551a524867eca9229a215c8c44b8237ec · ms:419 · test-lock-sha256:280e240c80784ed8bd77414390f46495cd70ad857c7382aacba4ebb1b773dd82 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RDaGVja0FuZE5vQ2hlY2tUb2dldGhlcklzVXNhZ2UJMTEyOTVhMzRkNzY5NThiZWU5ZjAwYzdhYzk2MDlhMjczYmUxYTE5OGY4Y2U4ZmEyNDU5YzdjM2IxZjEzNTE4MQpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RFeHBsaWNpdENoZWNrU3RpbGxSdW5zT25Qcm9zZQk0OGM1MWUzYTY1YTIzZWNiZDE2ZjhjYTI1NjlmYjNlMTg1NDA4NGRhNDQzODVhNmUwNjg0ZWEzYzBkMmUzYzljCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdEZhaWxlZENoZWNrUHJpbnRzVGhlTGFzdEVycm9yTGluZQkxMDI4MmVmM2I5MjQ1M2U2YmQwYTQ1OTgzMzUwODFlODI2YmQwNDcwMTE3YTZlZGMwMzE5YjFlMDY2OWVjYTNmCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdE5vQ2hlY2tPcHRzT3V0CTljNmJmYzVlODNmNmUzZGJhODExODE1YjdlNDYzZjQ0ODQ5ODVlZTFiYjIyZmFmNGQ5NTZjN2JjZjRiZDA0MGMKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVPZlByb3NlRG9lc05vdFJ1blRoZURlZmF1bHRDaGVjawllNzg5ZTk2ZGIwYTY2NDc2NGEwOWI0ZGUyMzNiZWQyYWUwN2ZjOWMzODFhMWY0MmM5ZTQwMzIwYmE5ZWJjZmQxCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdFdyaXRlUnVuc1RoZUNoZWNrQnlEZWZhdWx0CTg5MWM2M2I5ZmZlN2ZiMTUzZWU2NGYwODAzNmU2NjUzZDE2NWY3ODA0ODdhMTdlNzE0Njg5ZGY5MjBkYzNjMmUKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVXaXRob3V0QUNoZWNrQ29tbWFuZFN0aWxsQXBwbGllcwlkYTJmZDk3MDY0ZTEyNDA3NTA3NzFmZDRhMjRkYzA4ZGJlNDA4ZWE5MTE3NDYyNWExZjM3NjhjZjdkMzFlODRlCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JZmFpbAljODBlYmM3MWQyOTQwMTZmOWRhNDg4NTUzMzg4MWU2OWZiZWJlNmYyMTE1MDE3ZDc1MTExOWVkNjRhNDNhNjYzCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JcGFzcwlhYjBlZmM5NzQ3YTIzNWRlZGNiOTczMjA5MGU0OGVjZDc2MzM5NzI1NDk3NTg4ZDI0MTljMDdjMmU3MTM5NjhiCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNBZHZpc29yaWVzQW5kU3RyaWN0QmFsYW5jZQlmMmNjZDNlMDliMzE0YjYyMDUwYjQwY2UwYzc3N2IxNDYxYWNjNDcyZmM5MDVmMWVkMmQ5ZWYyMTFjOGI2YjQyCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNBcHBseVBhdGNoRm9ybWF0CWMzODY1OTc2NTI2NjE1YWM0OWJiOTczOTI4NmNjZjI3YmE3MTM2ZTIzZGIyNzUwYzE5NmY0OTNjNjNhYmM3NWUKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc0JvZHlBdFBhdGgJODA0MTcxMWI1NjZkZjVhMDlkN2EzNmQ2YWQ2ZjU5ZDljMDY4MTFmMzI0NjhlYTQ4YWRmOTkwZjA2YWZmMDM1Mgpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzRWNob1BhZAliNDhjMjA5MjJjYjIyZDJlMGI2OTQ4NjhjYTczNGIyMjg3YTc1YWQ5OWVkZDE1ZDk2YzMyMzQwZmI0ZWFmMTgwCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNIb3dUb1F1b3RlQUhlYWRlck9wdGlvbgk1NzY5ODY1MGQ0OTRiZjY4MzRiMzM4MGM0MWM4ODI5YmUyNzAzODVmY2E3ZTMwM2YwOGQ3MDZjMmUyZjk2ZjJlCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNOb0NoZWNrCTgwOTMwNDc1NTRiN2EyZGNiYzY2OGE5YzlkNjllN2EyNDQ2NDMxMWIwOTFkOTdlN2ZmZjdkNWE2YzhlYTFjOTYKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc1RoZVBhdHRlcm5GaWVsZAkyMmI2ZmI1NDZjNGUwNGQ4NWM3NDMyOGNiMGY0MmUwOTI0ODY3ZTNjZWQxZmY4Nzc2OGMwZDE3YzIwZTgzNjFj
  ```
  --- last 10 line(s) of stdout (of 83 after folding 83 raw)
          shape) as a failed hunk — exit 1, nothing is written. Off by default; braces
          inside strings count, so drop it for that plan. Both JSON receipts (--json
          and mrw_write) carry "pattern" {advisory_writes, window, fires} on every
          write, and mrw stats prints a "strict-balance pricing" block as writes land —
          how many the flag would have refused, and whether those writes then broke,
          held or went unchecked.
  --- FAIL: TestWriteHelpNamesBodyAtPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.209s
  FAIL
  ```
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f7b601f6b894654cf21244e9b59ed3551a524867eca9229a215c8c44b8237ec · ms:523
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f7b601f6b894654cf21244e9b59ed3551a524867eca9229a215c8c44b8237ec · ms:523
