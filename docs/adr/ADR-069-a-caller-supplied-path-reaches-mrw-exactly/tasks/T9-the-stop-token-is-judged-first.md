# Task ADR-069-T9: The stop token is judged before the guard stops; contract §138

**Depends-on:** T8
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the positional check ahead of the stop in the subcommand guard
**Consumes:** T8's lone-dash stop
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the stop token is judged first`, `the binary refuses it`, `no engine file changes`

## Goal

The fourth Codex review of PR #222 found one gap T8 opened. The parser trims a lone `-` and KEEPS it as the positional that ends its parse (`command_parse.go:123-125`), so `write ' - '` read stdin once the guard stopped at the `-` before judging the token — which T7's guard had refused. The fix: the subcommand guard judges the stop token like any positional first, then ends its walk. After a `--` the parser keeps ` - ` as given, and a bare `-` is stdin.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs` |
| `cmd/mrw/paddedflag_test.go` | edit | the test |
| `scripts/contract.sh` | edit | §138 |

## Ordered Steps

1. [S1] Write the test; confirm RED on assertions. [proof: mutation]
   `write ' - '` and `read ' - ' x` are refused naming the `--`; `write -- ' - '` is not refused as padded.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutant: the stop token skipped by the positional check again.
3. [S3] §138 through the binary: write. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 138\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAPaddedLoneDashIsRefusedBeforeTheGuardStops' 2>&1 | tee /tmp/adr069-T9.out \
  && grep -q '^--- PASS: TestAPaddedLoneDashIsRefusedBeforeTheGuardStops ' /tmp/adr069-T9.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T9.out \
  && ./scripts/contract.sh > /tmp/adr069-T9-contract.out 2>&1 \
  && grep -q '^  PASS  a padded lone - is refused before the guard stops' /tmp/adr069-T9-contract.out \
  && grep -q '^  PASS  and a bare - after -- is stdin' /tmp/adr069-T9-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPaddedLoneDashIsRefusedBeforeTheGuardStops` | `cmd/mrw/paddedflag_test.go` | ` - ` is refused as a padded positional; after `--` it is kept as given | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §138 |
| 2 — something selects it | every CLI call passes its subcommand's Action |
| 3 — the caller can discover it | the refusal names the `--` that reaches the path |
| 4 — it is used | §138 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · a04ee52* · exit 1 · `set -o pipefail …` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:1191 · test-lock-sha256:7c1b1ce4a20eb1ebb2185df863442ad1a533abd2fd6618a599477332af4d57ca · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJMWYxY2UzYzA5OTk0MThjNGU4MjdkOTA5ZGViNTlmOWE0NmEyYWJkNDY2ZTBhZDNkYjg2MDE2Y2Y2OGNkYjUxZApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAliYmIwMTJmMDBlZTMwN2E5ZjUxMmIyMDliZGVhNmRkZjQzNWY0MTU4OWUwNTYzOGIxZGFjZGYyMThmMGViYzY1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJMmM3ODk2ZDIxMGQxZTliMzg4N2Q2ZTRlMzI5NWEwODc5YTNhM2I4ODQwYzlkOTMzMDI0ODhjMzc0ZTcwOGEyNApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CWRiM2JhOGMwNGU4MjFhODlmZjc2ZTJmOWYxYjM4NDIxODkyOTk4YTI1ODcyODgwNzc2ZjRiNzE4YjhiMDMwMTAKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWRkOTE1ZDczNTJiMGZhYWE1ZDMzZDY0NzhkMTUxOGNkZWVhMDg0NDJkNzc2YmY2MDczZTNiOTlhZTNlNWYwM2EKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlSXRlclJlZnVzYWxLZWVwc1RoZVZlcmIJODNiYTkyNmVmYTlkYzU5ZTVjODMxOTZjMmE3ZTk3NTk5YWIyMmZjMTZkMjg0NDUwNWE2NzM3OTM3MWUxMDg5OApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVXaG9sZUFyZ3ZHdWFyZFJlYWRzRmxhZ1ZhbHVlc0FuZFRoZVRlcm1pbmF0b3IJYTVjYjQ0YjI3NzAyYzcxZjI0ZDgzYTcyMTVmYjZkZjk0NWJjMzIzYzk2N2RhMjI0NGZiNmU0OGQzNjA2NWQ3OA
  ```
  --- last 10 line(s) of stdout
  === RUN   TestAPaddedLoneDashIsRefusedBeforeTheGuardStops
      paddedflag_test.go:260: write ' - ' exited 2, want 2 refusing ' - ' as a padded positional:
          <stdin>: plan is empty: no @@ headers found
      paddedflag_test.go:266: read ' - ' x was not refused as a padded positional:
          ==> -  UNREADABLE  open /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPaddedLoneDashIsRefusedBeforeTheGuardStops905072647/001/-: no such file or directory
          1 range(s) could not be served
  --- FAIL: TestAPaddedLoneDashIsRefusedBeforeTheGuardStops (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.282s
  FAIL
  ```
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:34313
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:34311
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:33091
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:33151
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · ms:0 · test-lock-sha256:5efa6abd1c1d8f55bff1f72cfd2bbf1ad96c9f5f9395e28ecf487b932929e340 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTRkNmViMTNjMzY5MjI2N2JmODEwNjVjNTk3ZjkzNWM1Y2EwN2YyZDA2OTVkZjAwNzNlMTg0NmUyMGU1MTE1Y2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJZjY0MDkzYzE1NjgwZmE4NzIwZDVhYzRlYTAwNGYzZjI4Mzg0NzY2ZTc3NTlkNzMzODUyODYwMDUwMjM0MjUyMQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAk3MTU4OTA4N2Q0NjQxODhkMjA1YjA2NTEwYmVmNzc1M2VlMTQ0Y2IyNDdmNWJiNDU4N2M4ZWU5NTQ5YjcyODM3CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJNTRjOTg2NGYzODI2ODEwMTdlOTM2ZGMwODM0NWQxNjM1NzczYzY5ZWNhNjRiOTQ4Njg0ZTFkNTBlZGU4YmJiMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CThkMTA5MzVhNjNhNjJkODhlOWI3ODY2OTY4MTE3MTU3YTMyMDk4YzJiZmY4ZDAzYWExNjYyNzg1Yjk3MDljNmQKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWI4MzYzMjA0Y2E2Y2YwZDQyNGU5MDUyOGY1Y2EzMDkxZDg5MDEzNmNmYTJkMGY0N2EzNWM2YjA3YzRkZGMyYWIKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAk3NGQzZWY3ZTMyNDBkNTYwZGYxMzZhODYwOTI3MzhjNGZlMmZkMTQ0Y2ZkMmM1NTk2ZWQxNzBkMTc4YzE3MDEwCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCWYxY2IzMTgyNWJkYzY3YjBjM2YwZDY1ZTVjNDkyNWFjYzQxY2I0MGFmNTFmNzMxNmIxOWJhZDZmNWVmMGU0MjEKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzg · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-25 · a04ee52* · mutant killed · exit 1 · `cmd/mrw/main.go` · the stop token skips the positional check again: write space-dash-space reads stdin, the test and §138 go red · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · covers:the stop token is judged first
- 2026-09-25 · a04ee52* · mutant killed · exit 1 · `cmd/mrw/main.go` · every positional ends the walk: the padded path after a plain one is no longer judged, T1 rows go red through the binary · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · covers:the binary refuses it
- 2026-09-25 · a04ee52* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:64d13c5fea1b55fb8486c49deab955417741422c1ee74a1b8c49d915a79bb635 · covers:no engine file changes

## Invariants

- Every T5–T8 test stays green.
- A token the parser keeps as given, or never sees, is never refused.

## Risks

- None known beyond the parser edges the earlier tasks record.

## Out of Scope

- A padded lone `-` at the ROOT (`mrw ' - '`): the parser trims it and finds no such command, exit 2 by name (permanent: fact: `CommandNotFound` in `rootCommand`; citation: `cmd/mrw/main.go`)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
