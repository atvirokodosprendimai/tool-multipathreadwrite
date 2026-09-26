# Task ADR-080-T2: a check interrupted before it starts says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `check.Interrupted` for a check that never started; the exit-3 branches
**Consumes:** `check.Run`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a cancelled check says interrupted`, `a write exits 3`, `mrw check exits 3`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A signal that landed before the check's process started reported "could not start … declare a check", exit 2.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Interrupted` for a never-started cancelled check |
| `cmd/mrw/main.go` | edit | exit 3 on both commands |
| `internal/check/logs080_test.go`, `cmd/mrw/checklog080_test.go` | new | the tests below |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: the `ctx.Err() == context.Canceled` branch removed; the write's exit-3 branch removed.
3. [S3] No contract row: no shell lands a signal in a microseconds-wide window; a context cancelled beforehand reaches it deterministically, which is what the tests do. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 240s -run 'TestACheckCancelledBeforeItStartsSaysInterrupted|TestAMissingShellUnderACancelIsStillCouldNotStart|TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted|TestMrwCheckCancelledBeforeItStartsSaysInterrupted|TestAnInterruptedCheckSaysSo' -v 2>&1 | tee /tmp/adr080-T2.out \
  && missing=$(for t in TestACheckCancelledBeforeItStartsSaysInterrupted TestAMissingShellUnderACancelIsStillCouldNotStart TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted TestMrwCheckCancelledBeforeItStartsSaysInterrupted TestAnInterruptedCheckSaysSo; do grep -qE "^--- PASS: $t \(" /tmp/adr080-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/lines internal/iter internal/seen internal/state internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/lines internal/iter internal/seen internal/state internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACheckCancelledBeforeItStartsSaysInterrupted` | `internal/check/logs080_test.go` | a pre-cancelled check never runs, says interrupted, keeps no log | — | S1, S2 |
| `TestAMissingShellUnderACancelIsStillCouldNotStart` | `internal/check/logs080_test.go` | a missing shell under a cancelled context is "could not start", not interrupted | — | S2 |
| `TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted` | `cmd/mrw/checklog080_test.go` | the write lands and exits 3, "interrupted before it started" | — | S1, S2 |
| `TestMrwCheckCancelledBeforeItStartsSaysInterrupted` | `cmd/mrw/checklog080_test.go` | `mrw check` exits 3, "interrupted before it started", not "declare one" | — | S2 |
| `TestAnInterruptedCheckSaysSo` | `internal/check/group_unix_test.go` | ADR-072's pair: interrupted mid-run | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every check run and every `--ast-grep` read |
| 3 — the caller can discover it | the receipt says what it removed and names what it kept |
| 4 — it is used | the review of #232 and the 3,103 logs on one machine |

## Verification Log
(empty until execute)
- 2026-09-26 · 6835512* · exit 1 · `set -o pipefail …` · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · ms:346 · test-lock-sha256:b5004d362cfcae25640fae96d1b8ee0ac3eaefedd422f90721235ad52f776a54 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvY2hlY2tsb2cwODBfdGVzdC5nbwlUZXN0QVRpbWVkT3V0Q2hlY2tOYW1lc0l0c0xvZwkyYzA2NDE0YTAxYzJhYzQzZjI5MjE0YjRkMzk4ZTNkZjJmZmUwMDNkZTEzNjUwNDVkMzI3OGJlZGFiOWU1ZmMyCmJvZHkJY21kL21ydy9jaGVja2xvZzA4MF90ZXN0LmdvCVRlc3RBV3JpdGVXaG9zZUNoZWNrSXNDYW5jZWxsZWRCZWZvcmVJdFN0YXJ0c1NheXNJbnRlcnJ1cHRlZAkwY2ZjZjAzOTFjN2E2ZmMwYTE0MGVkYzg4NTAyNGIzZjI3NWM2ZGU4NjBlMzlhOGMyMzMwYTMzMGM5YmMxNWQxCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0xlYXZlc05vR3JhbmRjaGlsZAkzMzRiYjUwMzlhNjYxZTE0YzdiZDE0Njk5OWM2ZmIwMzFkNWU3M2M3NjYwZDhmMzIyZjM4ZWM2NWRmMjBlZmMwCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBbkludGVycnVwdGVkQ2hlY2tTYXlzU28JNGYwOWU2YjY5ZGI2YTY3NWM2YWFmNjVlZGRiYmMzYjNiYTI3ZTk2M2M5MzA2NWE5NDcxNjQxZjQ1YTdlZDQwYQpib2R5CWludGVybmFsL2NoZWNrL2dyb3VwX3VuaXhfdGVzdC5nbwlUZXN0VGhlQ2hlY2tTdG9wc09uSGFuZ3VwVW5sZXNzSGFuZ3VwSXNJZ25vcmVkCWE0NjkyYTI0MWZmZGEyZmNmOTNkNmMwNGVjMDFjYjAwZWQ3MDlhYWEwZWVkZTVjZTMxNWQzOTBkZWZjNTA0MTUKYm9keQlpbnRlcm5hbC9jaGVjay9sb2dzMDgwX3Rlc3QuZ28JVGVzdEFDaGVja0NhbmNlbGxlZEJlZm9yZUl0U3RhcnRzU2F5c0ludGVycnVwdGVkCTAyYjBkMDY1MTBmYzNkMWJjZjZiMTVmODFkZTM3ZDM1YTEyOTJkNjczMzY2ODIyNzhiNzRjYTBjYTFmNzkzNDQKYm9keQlpbnRlcm5hbC9jaGVjay9sb2dzMDgwX3Rlc3QuZ28JVGVzdEFDaGVja1JlbW92ZXNJdHNPd25Mb2dzT2xkZXJUaGFuQVdlZWsJNDJlZThmNzBlMTFmZTc2ZDQ5OWU1ZGVhMzFhYzA4ODhlMDM4Mjc3MGNlM2RlODQ0OWNiNTFkM2JmOGE0NzUzYQpib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QVRpbWVkT3V0Q2hlY2tLZWVwc0l0c0xvZwk4ZGZhMzE5ZTBjYWRkMzU1ODFjNzQ3MmQwZTBkNjY5NjMwY2I3NmQzZjM4NmY5YWI5MDM4ODM4NDdmNjcwNjJj
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted
      checklog080_test.go:60: want exit 3, interrupted before it started: the write applied but no check could run: could not start: context canceled — declare one in .quality-harness.json
          ok   a.go 2 replace  -1 +1
          wrote a.go  2L -> 2L  sha 9ff0b535
          1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
          check SKIPPED: could not start: context canceled
  --- FAIL: TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.082s
  FAIL
  ```
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · ms:629
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · ms:646
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · ms:610
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · ms:634
- 2026-09-26 · 6835512* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:0 · test-lock-sha256:8329ae81274e0a4cb9a94ee2050871216bd074d32c83585cb7b1eece593e5026 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvY2hlY2tsb2cwODBfdGVzdC5nbwlUZXN0QVRpbWVkT3V0Q2hlY2tOYW1lc0l0c0xvZwkyYzA2NDE0YTAxYzJhYzQzZjI5MjE0YjRkMzk4ZTNkZjJmZmUwMDNkZTEzNjUwNDVkMzI3OGJlZGFiOWU1ZmMyCmJvZHkJY21kL21ydy9jaGVja2xvZzA4MF90ZXN0LmdvCVRlc3RBV3JpdGVXaG9zZUNoZWNrSXNDYW5jZWxsZWRCZWZvcmVJdFN0YXJ0c1NheXNJbnRlcnJ1cHRlZAkwY2ZjZjAzOTFjN2E2ZmMwYTE0MGVkYzg4NTAyNGIzZjI3NWM2ZGU4NjBlMzlhOGMyMzMwYTMzMGM5YmMxNWQxCmJvZHkJY21kL21ydy9jaGVja2xvZzA4MF90ZXN0LmdvCVRlc3RNcndDaGVja0NhbmNlbGxlZEJlZm9yZUl0U3RhcnRzU2F5c0ludGVycnVwdGVkCWI2OWNkODM0ZjMyY2FiNzRhMDdkNzNkNzY4YjM5N2RlZmRkNDk0ZDc3NDhhMWU1YzdiOTc5ZDJlMGE4OGQ3YmEKYm9keQlpbnRlcm5hbC9jaGVjay9ncm91cF91bml4X3Rlc3QuZ28JVGVzdEFUaW1lZE91dENoZWNrTGVhdmVzTm9HcmFuZGNoaWxkCTMzNGJiNTAzOWE2NjFlMTRjN2JkMTQ2OTk5YzZmYjAzMWQ1ZTczYzc2NjBkOGYzMjJmMzhlYzY1ZGYyMGVmYzAKYm9keQlpbnRlcm5hbC9jaGVjay9ncm91cF91bml4X3Rlc3QuZ28JVGVzdEFuSW50ZXJydXB0ZWRDaGVja1NheXNTbwk0ZjA5ZTZiNjlkYjZhNjc1YzZhYWY2NWVkZGJiYzNiM2JhMjdlOTYzYzkzMDY1YTk0NzE2NDFmNDVhN2VkNDBhCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RUaGVDaGVja1N0b3BzT25IYW5ndXBVbmxlc3NIYW5ndXBJc0lnbm9yZWQJYTQ2OTJhMjQxZmZkYTJmY2Y5M2Q2YzA0ZWMwMWNiMDBlZDcwOWFhYTBlZWRlNWNlMzE1ZDM5MGRlZmM1MDQxNQpib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QUNoZWNrQ2FuY2VsbGVkQmVmb3JlSXRTdGFydHNTYXlzSW50ZXJydXB0ZWQJMDJiMGQwNjUxMGZjM2QxYmNmNmIxNWY4MWRlMzdkMzVhMTI5MmQ2NzMzNjY4MjI3OGI3NGNhMGNhMWY3OTM0NApib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QUNoZWNrUmVtb3Zlc0l0c093bkxvZ3NPbGRlclRoYW5BV2Vlawk0MmVlOGY3MGUxMWZlNzZkNDk5ZTVkZWEzMWFjMDg4OGUwMzgyNzcwY2UzZGU4NDQ5Y2I1MWQzYmY4YTQ3NTNhCmJvZHkJaW50ZXJuYWwvY2hlY2svbG9nczA4MF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0tlZXBzSXRzTG9nCThkZmEzMTllMGNhZGQzNTU4MWM3NDcyZDBlMGQ2Njk2MzBjYjc2ZDNmMzg2ZjlhYjkwMzg4Mzg0N2Y2NzA2MmM · test-lock-kind:replace
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:857
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:634
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:564
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:563
- 2026-09-26 · 52debba* · exit 0 · `set -o pipefail …` · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · ms:2892
- 2026-09-26 · 31fd88c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:0 · test-lock-sha256:bff230c286da1bafbd1ea15fd085bf0cbf191a8761a90a42443e07dfe87dc938 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvY2hlY2tsb2cwODBfdGVzdC5nbwlUZXN0QVRpbWVkT3V0Q2hlY2tOYW1lc0l0c0xvZwkyYzA2NDE0YTAxYzJhYzQzZjI5MjE0YjRkMzk4ZTNkZjJmZmUwMDNkZTEzNjUwNDVkMzI3OGJlZGFiOWU1ZmMyCmJvZHkJY21kL21ydy9jaGVja2xvZzA4MF90ZXN0LmdvCVRlc3RBV3JpdGVXaG9zZUNoZWNrSXNDYW5jZWxsZWRCZWZvcmVJdFN0YXJ0c1NheXNJbnRlcnJ1cHRlZAkwY2ZjZjAzOTFjN2E2ZmMwYTE0MGVkYzg4NTAyNGIzZjI3NWM2ZGU4NjBlMzlhOGMyMzMwYTMzMGM5YmMxNWQxCmJvZHkJY21kL21ydy9jaGVja2xvZzA4MF90ZXN0LmdvCVRlc3RNcndDaGVja0NhbmNlbGxlZEJlZm9yZUl0U3RhcnRzU2F5c0ludGVycnVwdGVkCWI2OWNkODM0ZjMyY2FiNzRhMDdkNzNkNzY4YjM5N2RlZmRkNDk0ZDc3NDhhMWU1YzdiOTc5ZDJlMGE4OGQ3YmEKYm9keQlpbnRlcm5hbC9jaGVjay9ncm91cF91bml4X3Rlc3QuZ28JVGVzdEFUaW1lZE91dENoZWNrTGVhdmVzTm9HcmFuZGNoaWxkCTMzNGJiNTAzOWE2NjFlMTRjN2JkMTQ2OTk5YzZmYjAzMWQ1ZTczYzc2NjBkOGYzMjJmMzhlYzY1ZGYyMGVmYzAKYm9keQlpbnRlcm5hbC9jaGVjay9ncm91cF91bml4X3Rlc3QuZ28JVGVzdEFuSW50ZXJydXB0ZWRDaGVja1NheXNTbwk0ZjA5ZTZiNjlkYjZhNjc1YzZhYWY2NWVkZGJiYzNiM2JhMjdlOTYzYzkzMDY1YTk0NzE2NDFmNDVhN2VkNDBhCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RUaGVDaGVja1N0b3BzT25IYW5ndXBVbmxlc3NIYW5ndXBJc0lnbm9yZWQJYTQ2OTJhMjQxZmZkYTJmY2Y5M2Q2YzA0ZWMwMWNiMDBlZDcwOWFhYTBlZWRlNWNlMzE1ZDM5MGRlZmM1MDQxNQpib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QUNoZWNrQ2FuY2VsbGVkQmVmb3JlSXRTdGFydHNTYXlzSW50ZXJydXB0ZWQJMDJiMGQwNjUxMGZjM2QxYmNmNmIxNWY4MWRlMzdkMzVhMTI5MmQ2NzMzNjY4MjI3OGI3NGNhMGNhMWY3OTM0NApib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QUNoZWNrUmVtb3Zlc0l0c093bkxvZ3NPbGRlclRoYW5BV2Vlawk0MmVlOGY3MGUxMWZlNzZkNDk5ZTVkZWEzMWFjMDg4OGUwMzgyNzcwY2UzZGU4NDQ5Y2I1MWQzYmY4YTQ3NTNhCmJvZHkJaW50ZXJuYWwvY2hlY2svbG9nczA4MF90ZXN0LmdvCVRlc3RBTWlzc2luZ1NoZWxsVW5kZXJBQ2FuY2VsSXNTdGlsbENvdWxkTm90U3RhcnQJZWRiMGFjYmQ5ODI4MmEzYzg5ZTg3ODc1MGUzNmVhNjYxOWM3NDc2MzRmMDk5NTg0ZTYxM2U2NmFiMTc0Njk5MQpib2R5CWludGVybmFsL2NoZWNrL2xvZ3MwODBfdGVzdC5nbwlUZXN0QVBydW5lTGlzdHNUaGVEaXJlY3RvcnlJdFdhc0dpdmVuCTY3MDgwZjFhNTU0NWI4NzcyY2RhN2NjM2IzNWNkMTc3YzMyMTYzODQyNWFlMTg3MmE3ZDM3NWMwMzc1ZmY3MzAKYm9keQlpbnRlcm5hbC9jaGVjay9sb2dzMDgwX3Rlc3QuZ28JVGVzdEFUaW1lZE91dENoZWNrS2VlcHNJdHNMb2cJOGRmYTMxOWUwY2FkZDM1NTgxYzc0NzJkMGUwZDY2OTYzMGNiNzZkM2YzODZmOWFiOTAzODgzODQ3ZjY3MDYyYw · test-lock-kind:replace
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:1175
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:898
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:860
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:1143
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:2409
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · ms:2221

## Mutation Log
(empty until execute)
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/check/check.go` · a cancel before start reads as could not start · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · covers:a cancelled check says interrupted
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `cmd/mrw/main.go` · the write exit-3 branch removed · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · covers:a write exits 3
- 2026-09-26 · 6835512* · mutant survived · exit 0 · `cmd/mrw/main.go` · mrw check exit-3 branch removed · acceptance-sha256:2294ae36534beff07552b71987a39d7719ccb97f4cc7b7dfeaa839767acb9bf4 · covers:mrw check exits 3
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/check/check.go` · a cancel before start reads as could not start · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · covers:a cancelled check says interrupted
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `cmd/mrw/main.go` · the write exit-3 branch removed · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · covers:a write exits 3
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw check exit-3 branch removed · acceptance-sha256:4f7ebe78f736f4636881590c375eaaff68e352bf1e584baf1e0c07ea6af7dc94 · covers:mrw check exits 3
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `internal/check/check.go` · a cancel before start reads as could not start · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · covers:a cancelled check says interrupted
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `cmd/mrw/main.go` · the write exit-3 branch removed · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · covers:a write exits 3
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw check exit-3 branch removed · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · covers:mrw check exits 3
- 2026-09-26 · 31fd88c* · mutant inconclusive · exit 1 · `internal/check/check.go` · the context, not the start, decides interrupted · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · covers:a cancelled check says interrupted
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `internal/check/check.go` · the context, not the start, decides interrupted · acceptance-sha256:fcf4bea40fa4ce5ce7da63207ccc661362066b7a9d70fb2c8b3a7792aff4c769 · covers:a cancelled check says interrupted

## Invariants

- One signal gets one verdict and one exit, whenever it lands.

## Risks

- A tally counts the write as check_not_run, since nothing ran.

## Out of Scope

- A shell-level contract row (permanent: fact: the window is microseconds wide; the cancelled context reaches it)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
