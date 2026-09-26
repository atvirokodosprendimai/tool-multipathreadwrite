# Task ADR-069-T7: Both guards read the flags the parser accepts, ancestors included, and stop where it stops; contract §136

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `flagKinds`, `tokenRole` with `roleStop`; both guards on them
**Consumes:** T6's `takesValue`, `flagRole`, `refusePaddedFlagValues`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an inherited flag is read by both guards`, `the guard stops where the parser stops`, `the binary refuses them`, `no engine file changes`

## Goal

The second Codex review of PR #222 found two gaps in T6. A parent's persistent flag is accepted by a subcommand that has no flag of the same name (`command_parse.go:43-57`): `write` and `iter` take `--root` after the verb, `read`, whose `-C` is context, does not. Neither guard read the inherited flag, so `write --root -- --root='dir '` ended the walk at the `--` the root flag consumed and reached `dir`, and `iter --root ' x' add x` was falsely refused. And a single dash before a non-letter is where the parser stops and keeps every remaining token as given (`command_parse.go:134-138`), so a file named ` -1= ` was served by v1.25.0 and refused by T6's walker as an attached value. The fix: `flagKinds` computes the flags the parser accepts for a command from its lineage, an ancestor's flag skipped whole when any of its names is the command's own; `flagRole` reports the parser's stop, and both guards end there.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `flagKinds`, `tokenRole`, `flagRole`, `refusePaddedArgs`, `refusePaddedFlagValues` |
| `cmd/mrw/paddedflag_test.go` | edit | the two tests |
| `scripts/contract.sh` | edit | §136 |

## Ordered Steps

1. [S1] Write the two tests; confirm RED on assertions. [proof: mutation]
   - `refusePaddedFlagValues` refuses `write --root -- --root='dir ' p.mrw` and accepts `iter --root -- add x`; the binary accepts `iter --root ' x' add x` and refuses `iter --root ' x' add 'x '`.
   - The binary serves ` -1= ` as given and still refuses `--max-lines='1 '`.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutants: ancestors skipped in `flagKinds`; the stop rule dropped in `flagRole`.
3. [S3] §136 through the binary: write, iter and read. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 136\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAnInheritedRootFlagIsReadByBothGuards|TestASingleDashNonLetterTokenStopsTheParserAndTheGuard' 2>&1 | tee /tmp/adr069-T7.out \
  && grep -q '^--- PASS: TestAnInheritedRootFlagIsReadByBothGuards ' /tmp/adr069-T7.out \
  && grep -q '^--- PASS: TestASingleDashNonLetterTokenStopsTheParserAndTheGuard ' /tmp/adr069-T7.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T7.out \
  && ./scripts/contract.sh > /tmp/adr069-T7-contract.out 2>&1 \
  && grep -q '^  PASS  an inherited root flag after the verb is read by the whole-argv guard' /tmp/adr069-T7-contract.out \
  && grep -q '^  PASS  an inherited root value equal to a positional after its trim is not refused' /tmp/adr069-T7-contract.out \
  && grep -q '^  PASS  a single dash before a non-letter stops the parser and the guard' /tmp/adr069-T7-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInheritedRootFlagIsReadByBothGuards` | `cmd/mrw/paddedflag_test.go` | the whole-argv guard and the subcommand guard both read an inherited `--root` | — | S1, S2 |
| `TestASingleDashNonLetterTokenStopsTheParserAndTheGuard` | `cmd/mrw/paddedflag_test.go` | ` -1= ` is served as given; an attached padded value is still refused | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §136 |
| 2 — something selects it | every CLI call passes `main` and its subcommand's Action |
| 3 — the caller can discover it | the refusal names the spelling that works |
| 4 — it is used | §136 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:36164
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:35125
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:37153
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:35772
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:39832
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:34243
- 2026-09-25 · 50becfb* · exit 1 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:838 · test-lock-sha256:ed541815f9cdb485939b128f01c010bff05792883d5863f3924ebe6374614ab9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVBhZGRlZEJvb2xlYW5GbGFnTmFtZURvZXNOb3RIaWRlQVBhZGRlZFBhdGgJYmJiMDEyZjAwZWUzMDdhOWY1MTJiMjA5YmRlYTZkZGY0MzVmNDE1ODllMDU2MzhiMWRhY2RmMjE4ZjBlYmM2NQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBU2luZ2xlRGFzaE5vbkxldHRlclRva2VuU3RvcHNUaGVQYXJzZXJBbmRUaGVHdWFyZAkyNTQ1NTE1MDhlMzRjMmE5MDFmZTBiZGM1NzMyODU2NDNiZWNkZjljYTZhNjYzZDBiOWE0ZTE3M2Y0NWY1YTA2CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuQXR0YWNoZWRGbGFnVmFsdWVXaXRoVHJhaWxpbmdTcGFjZUlzUmVmdXNlZAk1NjRhNjEzZjczYTg4YWEzNjkyZTk5OTQ4MWI4NGUxNTE1NTIxMjEyMDhkMGE3MGFkZjU2ZGJiMjE5MDhhMjVjCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuQXR0YWNoZWRWYWx1ZUVuZGluZ0luQW55V2hpdGVzcGFjZUlzUmVmdXNlZAk2NmM4MGRmMDRiYmMwMzkzNDE2MTA3ODM1YzIyMDhkZTAzNDM4YWI3OGRiYWU4MzRmNjgyNzk3MzM0MGE4N2M5CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFuSW5oZXJpdGVkUm9vdEZsYWdJc1JlYWRCeUJvdGhHdWFyZHMJZGQ5MTVkNzM1MmIwZmFhYTVkMzNkNjQ3OGQxNTE4Y2RlZWEwODQ0MmQ3NzZiZjYwNzNlM2I5OWFlM2U1ZjAzYQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RUaGVJdGVyUmVmdXNhbEtlZXBzVGhlVmVyYgk4M2JhOTI2ZWZhOWRjNTllNWM4MzE5NmMyYTdlOTc1OTlhYjIyZmMxNmQyODQ0NTA1YTY3Mzc5MzcxZTEwODk4CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZVdob2xlQXJndkd1YXJkUmVhZHNGbGFnVmFsdWVzQW5kVGhlVGVybWluYXRvcglhNWNiNDRiMjc3MDJjNzFmMjRkODNhNzIxNWZiNmRmOTQ1YmMzMjNjOTY3ZGEyMjQ0ZmI2ZTQ4ZDM2MDY1ZDc4
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
      paddedflag_test.go:180: iter --root ' x' add x was refused, exit 2:
          ' x' has edge whitespace the argument parser strips; put -- before the path: mrw iter add -- ' x'
  --- FAIL: TestAnInheritedRootFlagIsReadByBothGuards (0.00s)
  === RUN   TestASingleDashNonLetterTokenStopsTheParserAndTheGuard
      paddedflag_test.go:198: read ' -1= ' exited 2 or did not serve the file:
          ' -1= ' ends in whitespace the argument parser strips; pass the value as its own argument:  -1 ' '
  --- FAIL: TestASingleDashNonLetterTokenStopsTheParserAndTheGuard (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.273s
  FAIL
  ```
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:32969
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · ms:0 · test-lock-sha256:5efa6abd1c1d8f55bff1f72cfd2bbf1ad96c9f5f9395e28ecf487b932929e340 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTRkNmViMTNjMzY5MjI2N2JmODEwNjVjNTk3ZjkzNWM1Y2EwN2YyZDA2OTVkZjAwNzNlMTg0NmUyMGU1MTE1Y2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJZjY0MDkzYzE1NjgwZmE4NzIwZDVhYzRlYTAwNGYzZjI4Mzg0NzY2ZTc3NTlkNzMzODUyODYwMDUwMjM0MjUyMQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAk3MTU4OTA4N2Q0NjQxODhkMjA1YjA2NTEwYmVmNzc1M2VlMTQ0Y2IyNDdmNWJiNDU4N2M4ZWU5NTQ5YjcyODM3CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJNTRjOTg2NGYzODI2ODEwMTdlOTM2ZGMwODM0NWQxNjM1NzczYzY5ZWNhNjRiOTQ4Njg0ZTFkNTBlZGU4YmJiMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CThkMTA5MzVhNjNhNjJkODhlOWI3ODY2OTY4MTE3MTU3YTMyMDk4YzJiZmY4ZDAzYWExNjYyNzg1Yjk3MDljNmQKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWI4MzYzMjA0Y2E2Y2YwZDQyNGU5MDUyOGY1Y2EzMDkxZDg5MDEzNmNmYTJkMGY0N2EzNWM2YjA3YzRkZGMyYWIKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAk3NGQzZWY3ZTMyNDBkNTYwZGYxMzZhODYwOTI3MzhjNGZlMmZkMTQ0Y2ZkMmM1NTk2ZWQxNzBkMTc4YzE3MDEwCmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCWYxY2IzMTgyNWJkYzY3YjBjM2YwZDY1ZTVjNDkyNWFjYzQxY2I0MGFmNTFmNzMxNmIxOWJhZDZmNWVmMGU0MjEKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzg · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `cmd/mrw/main.go` · ancestors are skipped: an inherited --root is unknown to both guards, write --root -- ends the walk and iter --root x-space add x is falsely refused · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · covers:an inherited flag is read by both guards
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `cmd/mrw/main.go` · the parser stop is not modelled: -1=space is read as an attached flag value and refused · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · covers:the guard stops where the parser stops
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `cmd/mrw/main.go` · the subcommand guard reads only its own flags: iter --root x-space add x is refused through the binary, §136 goes red · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · covers:the binary refuses them
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:fd0716c09016cffa7fd0037fc14c9892a7e59e5625301a45ea0c0f337ebb3bce · covers:no engine file changes

## Invariants

- Every T5 and T6 test stays green.
- A separate flag value is never refused; a token the parser keeps as given is never refused.

## Risks

- A future subcommand with its own subcommands inherits through the same lineage walk; `subcommand` descends one level per positional, as the parser does.

## Out of Scope

- Grouped short flags (`-nq`): `UseShortOptionHandling` is off on every command, so the parser refuses them by name (permanent: fact: urfave v3.11.0 `useShortOptionHandling`; citation: `command_parse.go:127`)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
