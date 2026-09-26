# Task ADR-069-T10: A preserved stop token is not judged against a sibling; contract §139

**Depends-on:** T9
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the positional check limited to the lone-dash stop
**Consumes:** T9's stop-token check
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a preserved stop token is not judged`, `the binary serves it`, `no engine file changes`

## Goal

The fifth Codex review of PR #222 found one gap T9 opened. Of the two tokens the parser stops at, only the lone `-` is trimmed and kept (`command_parse.go:123-125`); a single dash before a non-letter is kept as given with everything after it (`:134-138`). T9 judged both before stopping, so `read ' -1= ' '-1='`, both files present, was refused because the first's trimmed spelling is the second. The fix: only the lone `-` is judged before the stop.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs` |
| `cmd/mrw/paddedflag_test.go` | edit | the test |
| `scripts/contract.sh` | edit | §139 |

## Ordered Steps

1. [S1] Write the test; confirm RED on assertions. [proof: mutation]
   `read ' -1= ' '-1='` serves both as given; `read ' - ' x` is still refused.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutant: every stop token judged again.
3. [S3] §139 through the binary. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 139\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAPreservedStopTokenIsNotJudgedAgainstAPositional' 2>&1 | tee /tmp/adr069-T10.out \
  && grep -q '^--- PASS: TestAPreservedStopTokenIsNotJudgedAgainstAPositional ' /tmp/adr069-T10.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T10.out \
  && ./scripts/contract.sh > /tmp/adr069-T10-contract.out 2>&1 \
  && grep -q '^  PASS  a preserved stop token is not judged against a sibling' /tmp/adr069-T10-contract.out \
  && grep -q '^  PASS  and the padded lone - is still refused' /tmp/adr069-T10-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPreservedStopTokenIsNotJudgedAgainstAPositional` | `cmd/mrw/paddedflag_test.go` | ` -1= ` beside `-1=` is served as given; ` - ` is still refused | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §139 |
| 2 — something selects it | every CLI call passes its subcommand's Action |
| 3 — the caller can discover it | the served header names the path as given |
| 4 — it is used | §139 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · 2517b6a* · exit 1 · `set -o pipefail …` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:631 · test-lock-sha256:2f9b1f60ffa0cfc25272d4f9de5241df48745ef172ef4bf26108b8be64358e28 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJMWYxY2UzYzA5OTk0MThjNGU4MjdkOTA5ZGViNTlmOWE0NmEyYWJkNDY2ZTBhZDNkYjg2MDE2Y2Y2OGNkYjUxZApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAliYmIwMTJmMDBlZTMwN2E5ZjUxMmIyMDliZGVhNmRkZjQzNWY0MTU4OWUwNTYzOGIxZGFjZGYyMThmMGViYzY1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJMmM3ODk2ZDIxMGQxZTliMzg4N2Q2ZTRlMzI5NWEwODc5YTNhM2I4ODQwYzlkOTMzMDI0ODhjMzc0ZTcwOGEyNApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CWRiM2JhOGMwNGU4MjFhODlmZjc2ZTJmOWYxYjM4NDIxODkyOTk4YTI1ODcyODgwNzc2ZjRiNzE4YjhiMDMwMTAKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWRkOTE1ZDczNTJiMGZhYWE1ZDMzZDY0NzhkMTUxOGNkZWVhMDg0NDJkNzc2YmY2MDczZTNiOTlhZTNlNWYwM2EKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlSXRlclJlZnVzYWxLZWVwc1RoZVZlcmIJODNiYTkyNmVmYTlkYzU5ZTVjODMxOTZjMmE3ZTk3NTk5YWIyMmZjMTZkMjg0NDUwNWE2NzM3OTM3MWUxMDg5OApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVXaG9sZUFyZ3ZHdWFyZFJlYWRzRmxhZ1ZhbHVlc0FuZFRoZVRlcm1pbmF0b3IJYTVjYjQ0YjI3NzAyYzcxZjI0ZDgzYTcyMTVmYjZkZjk0NWJjMzIzYzk2N2RhMjI0NGZiNmU0OGQzNjA2NWQ3OA
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPreservedStopTokenIsNotJudgedAgainstAPositional
      paddedflag_test.go:285: read ' -1= ' '-1=' exited 2 or did not serve both as given:
          ' -1= ' has edge whitespace the argument parser strips; put -- before the path: mrw read -- ' -1= '
  --- FAIL: TestAPreservedStopTokenIsNotJudgedAgainstAPositional (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.194s
  FAIL
  ```
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:34657
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:38030
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:34431
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:33565
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · ms:0 · test-lock-sha256:5efa6abd1c1d8f55bff1f72cfd2bbf1ad96c9f5f9395e28ecf487b932929e340 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTRkNmViMTNjMzY5MjI2N2JmODEwNjVjNTk3ZjkzNWM1Y2EwN2YyZDA2OTVkZjAwNzNlMTg0NmUyMGU1MTE1Y2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJZjY0MDkzYzE1NjgwZmE4NzIwZDVhYzRlYTAwNGYzZjI4Mzg0NzY2ZTc3NTlkNzMzODUyODYwMDUwMjM0MjUyMQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAk3MTU4OTA4N2Q0NjQxODhkMjA1YjA2NTEwYmVmNzc1M2VlMTQ0Y2IyNDdmNWJiNDU4N2M4ZWU5NTQ5YjcyODM3CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJNTRjOTg2NGYzODI2ODEwMTdlOTM2ZGMwODM0NWQxNjM1NzczYzY5ZWNhNjRiOTQ4Njg0ZTFkNTBlZGU4YmJiMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CThkMTA5MzVhNjNhNjJkODhlOWI3ODY2OTY4MTE3MTU3YTMyMDk4YzJiZmY4ZDAzYWExNjYyNzg1Yjk3MDljNmQKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWI4MzYzMjA0Y2E2Y2YwZDQyNGU5MDUyOGY1Y2EzMDkxZDg5MDEzNmNmYTJkMGY0N2EzNWM2YjA3YzRkZGMyYWIKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAk3NGQzZWY3ZTMyNDBkNTYwZGYxMzZhODYwOTI3MzhjNGZlMmZkMTQ0Y2ZkMmM1NTk2ZWQxNzBkMTc4YzE3MDEwCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCWYxY2IzMTgyNWJkYzY3YjBjM2YwZDY1ZTVjNDkyNWFjYzQxY2I0MGFmNTFmNzMxNmIxOWJhZDZmNWVmMGU0MjEKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzg · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-25 · 2517b6a* · mutant killed · exit 1 · `cmd/mrw/main.go` · every stop token is judged again: dash-1-equals beside its twin is refused, the test and §139 go red · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · covers:a preserved stop token is not judged
- 2026-09-25 · 2517b6a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the lone dash is no longer judged either: §139 and §138 go red through the binary · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · covers:the binary serves it
- 2026-09-25 · 2517b6a* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:644fce83339eb88ba5e29ba65d21cfb7fa09daeb4632e314862eb83acd1d522b · covers:no engine file changes

## Invariants

- Every T5–T9 test stays green.
- A token the parser keeps as given, or never sees, is never refused.

## Risks

- None known beyond the parser edges the earlier tasks record.

## Out of Scope

- The parser's own handling of a token that begins with a digit after two dashes (`--1`): it is a flag name to the parser and refused by name (permanent: fact: `command_parse.go:131-133`; citation: urfave/cli v3.11.0)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
