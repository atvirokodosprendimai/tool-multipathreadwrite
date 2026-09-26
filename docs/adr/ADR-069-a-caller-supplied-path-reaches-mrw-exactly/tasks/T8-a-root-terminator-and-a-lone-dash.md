# Task ADR-069-T8: A root `--` does not end the guard below it; a lone `-` ends it; contract §137

**Depends-on:** T7
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** per-level option termination in the whole-argv walk; the parser's lone-dash stop; the note exemption narrowed to the note's words
**Consumes:** T7's `flagKinds`, `flagRole`, `tokenRole`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a root terminator does not end the guard below it`, `a lone dash ends the parse and the guard`, `the binary refuses them`, `no engine file changes`

## Goal

The third Codex review of PR #222 found two gaps in T7. A `--` before the subcommand ends the ROOT's options only: the parser still dispatches the subcommand, which parses its own flags (`command_run.go:282-315`). The whole-argv guard ended its walk there and the iter guard exempted a note entirely, so `-- iter note --root='dir ' x` and `-- stats --root='dir '` reached `dir`. And a lone `-` ends the parse: the parser keeps it as a positional and drops every token after it (`command_parse.go:123-125`), so the guard, reading on, refused `write - '--format=plan '`, which v1.25.0 accepted. The fix: after a root `--` the whole-argv walk treats the rest as positional until it descends into the subcommand, where its options are in force again; the note exemption skips the note's words and not the flags beside them; and `flagRole` reports a lone `-` as the parser's stop, where both guards end.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs`, `flagRole`, `refusePaddedFlagValues` |
| `cmd/mrw/paddedflag_test.go` | edit | the two tests |
| `scripts/contract.sh` | edit | §137 |

## Ordered Steps

1. [S1] Write the two tests; confirm RED on assertions. [proof: mutation]
   - `refusePaddedFlagValues` refuses `-- iter note --root='dir ' x` and `-- stats --root='dir '`, accepts `-- read -- --root='x '`; the binary refuses `iter note --root='dir ' x` and accepts `iter note 'revised '`.
   - `refusePaddedFlagValues` accepts `write - '--format=plan '` and refuses `write '--format=plan ' -`; the binary does not refuse `read - '--max-lines=1 '`.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutants: a root `--` ends the walk again; the lone-dash stop dropped; the note exemption widened back.
3. [S3] §137 through the binary: iter note, stats and write. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 137\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestARootTerminatorDoesNotEndTheGuardBelowIt|TestALoneDashEndsTheParseAndTheGuard' 2>&1 | tee /tmp/adr069-T8.out \
  && grep -q '^--- PASS: TestARootTerminatorDoesNotEndTheGuardBelowIt ' /tmp/adr069-T8.out \
  && grep -q '^--- PASS: TestALoneDashEndsTheParseAndTheGuard ' /tmp/adr069-T8.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T8.out \
  && ./scripts/contract.sh > /tmp/adr069-T8-contract.out 2>&1 \
  && grep -q '^  PASS  a -- before the subcommand does not end the guard for its flags' /tmp/adr069-T8-contract.out \
  && grep -q '^  PASS  a subcommand with no guard of its own is covered by the whole-argv guard' /tmp/adr069-T8-contract.out \
  && grep -q '^  PASS  a lone - ends the parse and the guard: a token the parser drops is not refused' /tmp/adr069-T8-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARootTerminatorDoesNotEndTheGuardBelowIt` | `cmd/mrw/paddedflag_test.go` | the walk continues below the subcommand after a root `--`; a note's flags are judged, its words are not | — | S1, S2 |
| `TestALoneDashEndsTheParseAndTheGuard` | `cmd/mrw/paddedflag_test.go` | a token after a lone `-` is not refused; the same token before it is | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §137 |
| 2 — something selects it | every CLI call passes `main` and its subcommand's Action |
| 3 — the caller can discover it | the refusal names the spelling that works |
| 4 — it is used | §137 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · 97f8a02* · exit 1 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:1039 · test-lock-sha256:e2e2b0b7c189b8d006a7093badb3211980bb61b0d006021d480f2badc0c82535 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJMWYxY2UzYzA5OTk0MThjNGU4MjdkOTA5ZGViNTlmOWE0NmEyYWJkNDY2ZTBhZDNkYjg2MDE2Y2Y2OGNkYjUxZApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAliYmIwMTJmMDBlZTMwN2E5ZjUxMmIyMDliZGVhNmRkZjQzNWY0MTU4OWUwNTYzOGIxZGFjZGYyMThmMGViYzY1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFSb290VGVybWluYXRvckRvZXNOb3RFbmRUaGVHdWFyZEJlbG93SXQJZGIzYmE4YzA0ZTgyMWE4OWZmNzZlMmY5ZjFiMzg0MjE4OTI5OThhMjU4NzI4ODA3NzZmNGI3MThiOGIwMzAxMApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBU2luZ2xlRGFzaE5vbkxldHRlclRva2VuU3RvcHNUaGVQYXJzZXJBbmRUaGVHdWFyZAkyNTQ1NTE1MDhlMzRjMmE5MDFmZTBiZGM1NzMyODU2NDNiZWNkZjljYTZhNjYzZDBiOWE0ZTE3M2Y0NWY1YTA2CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuQXR0YWNoZWRGbGFnVmFsdWVXaXRoVHJhaWxpbmdTcGFjZUlzUmVmdXNlZAk1NjRhNjEzZjczYTg4YWEzNjkyZTk5OTQ4MWI4NGUxNTE1NTIxMjEyMDhkMGE3MGFkZjU2ZGJiMjE5MDhhMjVjCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuQXR0YWNoZWRWYWx1ZUVuZGluZ0luQW55V2hpdGVzcGFjZUlzUmVmdXNlZAk2NmM4MGRmMDRiYmMwMzkzNDE2MTA3ODM1YzIyMDhkZTAzNDM4YWI3OGRiYWU4MzRmNjgyNzk3MzM0MGE4N2M5CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuSW5oZXJpdGVkUm9vdEZsYWdJc1JlYWRCeUJvdGhHdWFyZHMJZGQ5MTVkNzM1MmIwZmFhYTVkMzNkNjQ3OGQxNTE4Y2RlZWEwODQ0MmQ3NzZiZjYwNzNlM2I5OWFlM2U1ZjAzYQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVJdGVyUmVmdXNhbEtlZXBzVGhlVmVyYgk4M2JhOTI2ZWZhOWRjNTllNWM4MzE5NmMyYTdlOTc1OTlhYjIyZmMxNmQyODQ0NTA1YTY3Mzc5MzcxZTEwODk4CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZVdob2xlQXJndkd1YXJkUmVhZHNGbGFnVmFsdWVzQW5kVGhlVGVybWluYXRvcglhNWNiNDRiMjc3MDJjNzFmMjRkODNhNzIxNWZiNmRmOTQ1YmMzMjNjOTY3ZGEyMjQ0ZmI2ZTQ4ZDM2MDY1ZDc4
  ```
  --- last 10 line(s) of stdout (of 15 after folding 15 raw)
          0 entr(ies), 0 file(s)
  --- FAIL: TestARootTerminatorDoesNotEndTheGuardBelowIt (0.00s)
  === RUN   TestALoneDashEndsTheParseAndTheGuard
      paddedflag_test.go:239: a token the parser drops after a lone - was refused: '--format=plan ' ends in whitespace the argument parser strips; pass the value as its own argument: --format 'plan '
      paddedflag_test.go:246: read - '--max-lines=1 ' was refused by the subcommand guard (exit 2):
          '--max-lines=1 ' ends in whitespace the argument parser strips; pass the value as its own argument: --max-lines '1 '
  --- FAIL: TestALoneDashEndsTheParseAndTheGuard (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.254s
  FAIL
  ```
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:35177
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:33858
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:35433
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:34317
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:33736
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:33591
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:32968
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · ms:0 · test-lock-sha256:5efa6abd1c1d8f55bff1f72cfd2bbf1ad96c9f5f9395e28ecf487b932929e340 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTRkNmViMTNjMzY5MjI2N2JmODEwNjVjNTk3ZjkzNWM1Y2EwN2YyZDA2OTVkZjAwNzNlMTg0NmUyMGU1MTE1Y2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJZjY0MDkzYzE1NjgwZmE4NzIwZDVhYzRlYTAwNGYzZjI4Mzg0NzY2ZTc3NTlkNzMzODUyODYwMDUwMjM0MjUyMQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAk3MTU4OTA4N2Q0NjQxODhkMjA1YjA2NTEwYmVmNzc1M2VlMTQ0Y2IyNDdmNWJiNDU4N2M4ZWU5NTQ5YjcyODM3CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJNTRjOTg2NGYzODI2ODEwMTdlOTM2ZGMwODM0NWQxNjM1NzczYzY5ZWNhNjRiOTQ4Njg0ZTFkNTBlZGU4YmJiMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CThkMTA5MzVhNjNhNjJkODhlOWI3ODY2OTY4MTE3MTU3YTMyMDk4YzJiZmY4ZDAzYWExNjYyNzg1Yjk3MDljNmQKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWI4MzYzMjA0Y2E2Y2YwZDQyNGU5MDUyOGY1Y2EzMDkxZDg5MDEzNmNmYTJkMGY0N2EzNWM2YjA3YzRkZGMyYWIKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAk3NGQzZWY3ZTMyNDBkNTYwZGYxMzZhODYwOTI3MzhjNGZlMmZkMTQ0Y2ZkMmM1NTk2ZWQxNzBkMTc4YzE3MDEwCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCWYxY2IzMTgyNWJkYzY3YjBjM2YwZDY1ZTVjNDkyNWFjYzQxY2I0MGFmNTFmNzMxNmIxOWJhZDZmNWVmMGU0MjEKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzg · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `cmd/mrw/main.go` · a root -- ends the walk again: -- iter note --root=dir-space reaches dir · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:a root terminator does not end the guard below it
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `cmd/mrw/main.go` · a lone - is a positional again: the token the parser drops is refused · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:a lone dash ends the parse and the guard
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `cmd/mrw/main.go` · the note exemption is whole again: iter note --root=dir-space is not refused by the subcommand guard · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:a root terminator does not end the guard below it
- 2026-09-25 · 97f8a02* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · every terminator ends the whole-argv walk: §137 iter note and stats rows go red through the binary · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:the binary refuses them
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:no engine file changes
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `cmd/mrw/main.go` · options stay off below the subcommand after a root --: -- stats --root=dir-space is not refused, the §137 stats row goes red through the binary · acceptance-sha256:c5cd9f2ed9d26cf3b0a35a22a5d4c34c2c8903ea57dbbda60402338a23a558de · covers:the binary refuses them

## Invariants

- Every T5, T6 and T7 test stays green.
- A token the parser keeps as given, or never sees, is never refused.

## Risks

- A subcommand named with padding after a root `--` (`-- 'read '`) is handed on as given and refused by the parser as unknown; the walk ends at the lookup, which is the parser's verdict.

## Out of Scope

- A default command (`DefaultCommand`): none is configured, so the parser refuses an unknown first positional (permanent: fact: `rootCommand` sets no DefaultCommand; citation: `cmd/mrw/main.go`)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
