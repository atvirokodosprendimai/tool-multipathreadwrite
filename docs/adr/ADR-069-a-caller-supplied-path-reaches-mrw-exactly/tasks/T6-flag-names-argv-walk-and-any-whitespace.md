# Task ADR-069-T6: A flag name is read trimmed; the whole-argv guard reads values and the terminator; any trailing whitespace refuses an attached value; contract §135

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `takesValue`, `flagRole`, `subcommand`; `refusePaddedFlagValues(root, argv)`; `padAttached` on `unicode.IsSpace`
**Consumes:** T5's guards
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a flag name is classified trimmed`, `the whole-argv guard reads values and the terminator`, `any trailing whitespace refuses an attached value`, `the binary refuses them`, `no engine file changes`

## Goal

The Codex review of PR #222 found three gaps in T5. First, `read '--no-numbers ' 'x '` served `x`: the guard looked the flag name up as typed, so a padded boolean name read as value-taking and the padded path after it was skipped. Second, the whole-argv check read every `-` token as a flag, so `--files-from '--list= '` was falsely refused though the parser keeps it, and it stopped at any `--`, including one a root flag consumed, so `--root -- --root='dir '` reached `dir`. Third, `padAttached` checked for space and tab while the parser's trim is `strings.TrimSpace`, so `--files-from=$'list\n'` opened `list`. The fix: a flag name is classified trimmed, as the parser reads it; the whole-argv check walks argv the way the parser does (root flags and their values, the subcommand, its flags and their values; only a bare `--` ends it); and any trailing whitespace on an attached value is refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs`, `takesValue`, `flagRole`, `padAttached`, `refusePaddedFlagValues`, `subcommand`, `main` |
| `cmd/mrw/paddedflag_test.go` | edit | the three tests; T5's direct calls take the root command |
| `scripts/contract.sh` | edit | §135 |

## Ordered Steps

1. [S1] Write the three tests; confirm RED on assertions. [proof: mutation]
   - `read '--no-numbers ' 'x '` exits 2 naming `'x '`; `read --no-numbers x` serves `x`.
   - `refusePaddedFlagValues` accepts `read --files-from '--list= '` and a path after a bare `--`; refuses `--root -- --root='dir '` and `read --grep -- --exclude='x '`; the binary serves `--files-from '--list= '`.
   - `--files-from=` ending in `\n` or U+00A0 exits 2 saying to pass the value as its own argument; the separate spelling serves `x `.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutants: the name looked up untrimmed; the walker never consumes a value; `padAttached` back to space and tab.
3. [S3] §135 through the binary, root flag included. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 135\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath|TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator|TestAnAttachedValueEndingInAnyWhitespaceIsRefused' 2>&1 | tee /tmp/adr069-T6.out \
  && grep -q '^--- PASS: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath ' /tmp/adr069-T6.out \
  && grep -q '^--- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator ' /tmp/adr069-T6.out \
  && grep -q '^--- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused ' /tmp/adr069-T6.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T6.out \
  && ./scripts/contract.sh > /tmp/adr069-T6-contract.out 2>&1 \
  && grep -q '^  PASS  a padded boolean flag name does not hide a padded path' /tmp/adr069-T6-contract.out \
  && grep -q '^  PASS  a separate value that looks like an attached flag is served' /tmp/adr069-T6-contract.out \
  && grep -q '^  PASS  a -- consumed by a root flag does not end the whole-argv guard' /tmp/adr069-T6-contract.out \
  && grep -q '^  PASS  and the newline refusal names the separate spelling' /tmp/adr069-T6-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath` | `cmd/mrw/paddedflag_test.go` | a padded boolean flag name is classified as the parser reads it | — | S1, S2 |
| `TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator` | `cmd/mrw/paddedflag_test.go` | a separate value is never judged; a consumed `--` does not end the walk; a bare one does | — | S1, S2 |
| `TestAnAttachedValueEndingInAnyWhitespaceIsRefused` | `cmd/mrw/paddedflag_test.go` | newline and U+00A0 refuse an attached value; the separate spelling serves | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §135 |
| 2 — something selects it | every CLI call passes `main` and its subcommand's Action |
| 3 — the caller can discover it | the refusal names the spelling that works |
| 4 — it is used | §135 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:262 · test-lock-sha256:021f8bace3bef0dde10575602cc68171a2576fe176627532965c749804465c25 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVBhZGRlZEJvb2xlYW5GbGFnTmFtZURvZXNOb3RIaWRlQVBhZGRlZFBhdGgJYTNmNzVhNTQyYTMzOTJhMDlkZjcxMDk4YWMyYjM0MjM4NzFhYjcyOTA3ZGRjMGJhOTYwMjQ5MmY0ZTY1NWNjNwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVJdGVyUmVmdXNhbEtlZXBzVGhlVmVyYgk4M2JhOTI2ZWZhOWRjNTllNWM4MzE5NmMyYTdlOTc1OTlhYjIyZmMxNmQyODQ0NTA1YTY3Mzc5MzcxZTEwODk4CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZVdob2xlQXJndkd1YXJkUmVhZHNGbGFnVmFsdWVzQW5kVGhlVGVybWluYXRvcglhNWNiNDRiMjc3MDJjNzFmMjRkODNhNzIxNWZiNmRmOTQ1YmMzMjNjOTY3ZGEyMjQ0ZmI2ZTQ4ZDM2MDY1ZDc4
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  cmd/mrw/main.go:34:2: "unicode" imported and not used
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [build failed]
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:944
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  --- FAIL: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
      paddedflag_test.go:147: --files-from="/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused640813013/002/list\n" exited 2, want 2 saying to pass the value as its own argument:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused640813013/002/list: no such file or directory
      paddedflag_test.go:147: --files-from="/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused640813013/002/list\u00a0" exited 2, want 2 saying to pass the value as its own argument:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused640813013/002/list: no such file or directory
  --- FAIL: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.317s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:567
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.00s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.204s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:284
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.00s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.093s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:309
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.00s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.095s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:404
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.01s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.100s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:379
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.00s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.102s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:330
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      paddedflag_test.go:100: read '--exclude ' ' x' x exited 2 or did not serve x:
          --exclude without --grep: there is nothing to exclude from
  --- FAIL: TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath (0.00s)
  === RUN   TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator
  --- PASS: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
  --- PASS: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.120s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:0 · test-lock-sha256:13a73fb9d21d8c44c550516bb06f476e7ad786058f76f82a8f7c3038db3efb06 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVBhZGRlZEJvb2xlYW5GbGFnTmFtZURvZXNOb3RIaWRlQVBhZGRlZFBhdGgJYmJiMDEyZjAwZWUzMDdhOWY1MTJiMjA5YmRlYTZkZGY0MzVmNDE1ODllMDU2MzhiMWRhY2RmMjE4ZjBlYmM2NQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVJdGVyUmVmdXNhbEtlZXBzVGhlVmVyYgk4M2JhOTI2ZWZhOWRjNTllNWM4MzE5NmMyYTdlOTc1OTlhYjIyZmMxNmQyODQ0NTA1YTY3Mzc5MzcxZTEwODk4CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZVdob2xlQXJndkd1YXJkUmVhZHNGbGFnVmFsdWVzQW5kVGhlVGVybWluYXRvcglhNWNiNDRiMjc3MDJjNzFmMjRkODNhNzIxNWZiNmRmOTQ1YmMzMjNjOTY3ZGEyMjQ0ZmI2ZTQ4ZDM2MDY1ZDc4 · test-lock-kind:replace
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:1074
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  --- FAIL: TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator (0.00s)
  === RUN   TestAnAttachedValueEndingInAnyWhitespaceIsRefused
      paddedflag_test.go:148: --files-from="/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused3598136347/002/list\n" exited 2, want 2 saying to pass the value as its own argument:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused3598136347/002/list: no such file or directory
      paddedflag_test.go:148: --files-from="/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused3598136347/002/list\u00a0" exited 2, want 2 saying to pass the value as its own argument:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAnAttachedValueEndingInAnyWhitespaceIsRefused3598136347/002/list: no such file or directory
  --- FAIL: TestAnAttachedValueEndingInAnyWhitespaceIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.246s
  FAIL
  ```
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:35300
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:32576
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:32251
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:32239
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:32889
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:34911
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:33858
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:32640
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · ms:0 · test-lock-sha256:5efa6abd1c1d8f55bff1f72cfd2bbf1ad96c9f5f9395e28ecf487b932929e340 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTRkNmViMTNjMzY5MjI2N2JmODEwNjVjNTk3ZjkzNWM1Y2EwN2YyZDA2OTVkZjAwNzNlMTg0NmUyMGU1MTE1Y2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJZjY0MDkzYzE1NjgwZmE4NzIwZDVhYzRlYTAwNGYzZjI4Mzg0NzY2ZTc3NTlkNzMzODUyODYwMDUwMjM0MjUyMQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAk3MTU4OTA4N2Q0NjQxODhkMjA1YjA2NTEwYmVmNzc1M2VlMTQ0Y2IyNDdmNWJiNDU4N2M4ZWU5NTQ5YjcyODM3CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJNTRjOTg2NGYzODI2ODEwMTdlOTM2ZGMwODM0NWQxNjM1NzczYzY5ZWNhNjRiOTQ4Njg0ZTFkNTBlZGU4YmJiMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CThkMTA5MzVhNjNhNjJkODhlOWI3ODY2OTY4MTE3MTU3YTMyMDk4YzJiZmY4ZDAzYWExNjYyNzg1Yjk3MDljNmQKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWI4MzYzMjA0Y2E2Y2YwZDQyNGU5MDUyOGY1Y2EzMDkxZDg5MDEzNmNmYTJkMGY0N2EzNWM2YjA3YzRkZGMyYWIKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAk3NGQzZWY3ZTMyNDBkNTYwZGYxMzZhODYwOTI3MzhjNGZlMmZkMTQ0Y2ZkMmM1NTk2ZWQxNzBkMTc4YzE3MDEwCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCWYxY2IzMTgyNWJkYzY3YjBjM2YwZDY1ZTVjNDkyNWFjYzQxY2I0MGFmNTFmNzMxNmIxOWJhZDZmNWVmMGU0MjEKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzg · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `cmd/mrw/main.go` · the flag name is looked up as typed again: a padded --exclude name is unknown and x is falsely refused · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:a flag name is classified trimmed
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `cmd/mrw/main.go` · root flags consume nothing: --root -- --root=dir-space ends at the -- again · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:the whole-argv guard reads values and the terminator
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `cmd/mrw/main.go` · subcommand flags consume nothing: --files-from --list=space is judged as a flag and falsely refused · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:the whole-argv guard reads values and the terminator
- 2026-09-25 · f2661ae* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · space and tab only again: a newline-ended attached value opens the trimmed name · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:any trailing whitespace refuses an attached value
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `cmd/mrw/main.go` · the subcommand guard forgets which flags take values: --grep -- x-space ends the guard, T5 and §135 go red · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:the binary refuses them
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `cmd/mrw/main.go` · space and tab only again: a newline-ended attached value opens the trimmed name · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:any trailing whitespace refuses an attached value
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:069999b5a295a620578988d8980d8d1d0a852ff479b08a59d41614a83900d40c · covers:no engine file changes

## Invariants

- Every T5 test stays green; `read -- 'x '` and every unpadded call behave as before.
- A separate flag value is never refused.

## Risks

- A flag the parser accepts in a form this walker does not model (a grouped short flag) would be classified by its first character; urfave v3.11.0 refuses `-C1` by name, so no such form reaches an Action.

## Out of Scope

- Recovering an attached value instead of refusing it (permanent: boundary: the parser has already trimmed it; ADR-069 refuses rather than guesses)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
