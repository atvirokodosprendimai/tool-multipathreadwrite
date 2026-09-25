# Task ADR-069-T11: The guards agree with the parser on random argv; an iter verb is not judged

**Depends-on:** T10
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `TestTheGuardsAgreeWithTheParserOnRandomArgv` with its parser oracle; the iter verb exempt from the positional check
**Consumes:** T5–T10's guards
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guards agree with the parser`, `an iter verb is not judged`, `no engine file changes`

## Goal

Six Codex rounds each found one more edge of the parser that T5–T10 model. Instead of a seventh, the model is checked against the parser itself. `TestTheGuardsAgreeWithTheParserOnRandomArgv` drives random argv — every subcommand and iter verb, every flag own and inherited, padded names and values, attached values, `--`, `-`, a dash before a digit, unicode whitespace — through the real guards, and through the same command tree with every Action replaced by a recorder, so urfave itself reports the strings mrw would act on. The guard must refuse exactly when the parser trims a token and acts on it, or consumes an attached value that ends in whitespace (Decision item 5); a token's provenance is settled by re-parsing with a sentinel in its place, and whether the parser read a token as a flag by re-parsing with an unknown flag there. Pools are enumerated from the live command tree. The run found one gap: an iter verb was judged like a path, so `iter 'add ' x` was refused; the verb is exempt now. 40,000 cases over eight seeds agree at the time of writing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/paddedstress_test.go` | new | the differential test and its oracle |
| `cmd/mrw/paddedflag_test.go` | edit | the verb regression test |
| `cmd/mrw/main.go` | edit | `refusePaddedArgs`: the verb is not judged |

## Ordered Steps

1. [S1] Write the differential test; run it against the T10 guards; read every mismatch as a generator or oracle defect first, a guard defect second. [proof: acceptance]
2. [S2] Write the verb regression test; confirm RED; exempt the verb; GREEN. [proof: mutation]
   Mutants: the verb judged again (kills through the named test AND the random test); T10's stop rule reverted, T6's trimmed lookup reverted, T5's value skip reverted (each must be caught by the random test alone).
3. [S3] Eight seeds of 5000 clean, unpiped, before the record claims agreement. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v -run 'TestAnIterVerbIsNotJudgedAsAPath' 2>&1 | tee /tmp/adr069-T11.out \
  && grep -q '^--- PASS: TestAnIterVerbIsNotJudgedAsAPath ' /tmp/adr069-T11.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T11.out \
  && MRW_STRESS_N=1500 MRW_STRESS_SEED=1 go test ./cmd/mrw/ -count=1 -run 'TestTheGuardsAgreeWithTheParserOnRandomArgv' > /tmp/adr069-T11-s1.out 2>&1 \
  && MRW_STRESS_N=1500 MRW_STRESS_SEED=2 go test ./cmd/mrw/ -count=1 -run 'TestTheGuardsAgreeWithTheParserOnRandomArgv' > /tmp/adr069-T11-s2.out 2>&1 \
  && grep -q '^ok' /tmp/adr069-T11-s1.out && grep -q '^ok' /tmp/adr069-T11-s2.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheGuardsAgreeWithTheParserOnRandomArgv` | `cmd/mrw/paddedstress_test.go` | on random argv the guard refuses iff the parser would trim and act, or consume a padded attached value | — | S1, S3 |
| `TestAnIterVerbIsNotJudgedAsAPath` | `cmd/mrw/paddedflag_test.go` | `iter 'add ' x` is accepted; `iter add 'x '` is still refused | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `go test ./...` runs 300 random cases on every run |
| 3 — the caller can discover it | a mismatch prints seed, case, argv, what the parser delivered and what mrw said |
| 4 — it is used | it found the verb gap and retracted a false alarm (a padded ` --` is the terminator) before any Codex round did |

## Verification Log
(empty until execute)
- 2026-09-25 · edcd296* · exit 1 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1054 · test-lock-sha256:501048bd469a2d50b16b2d02b8d6dacc0525e2122c018eecf89a11bacc1e89ca · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJMWYxY2UzYzA5OTk0MThjNGU4MjdkOTA5ZGViNTlmOWE0NmEyYWJkNDY2ZTBhZDNkYjg2MDE2Y2Y2OGNkYjUxZApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAliYmIwMTJmMDBlZTMwN2E5ZjUxMmIyMDliZGVhNmRkZjQzNWY0MTU4OWUwNTYzOGIxZGFjZGYyMThmMGViYzY1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJMmM3ODk2ZDIxMGQxZTliMzg4N2Q2ZTRlMzI5NWEwODc5YTNhM2I4ODQwYzlkOTMzMDI0ODhjMzc0ZTcwOGEyNApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CWRiM2JhOGMwNGU4MjFhODlmZjc2ZTJmOWYxYjM4NDIxODkyOTk4YTI1ODcyODgwNzc2ZjRiNzE4YjhiMDMwMTAKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWRkOTE1ZDczNTJiMGZhYWE1ZDMzZDY0NzhkMTUxOGNkZWVhMDg0NDJkNzc2YmY2MDczZTNiOTlhZTNlNWYwM2EKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAkyZjM2YThlMzEwNGVhMDlhYzhkYjllNzBmNGU0NDNkM2NhYmM2MDJhMjA4YmQ3NDU5NDQ2YTJhYmQzMzhlN2Q1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCTgzYmE5MjZlZmE5ZGM1OWU1YzgzMTk2YzJhN2U5NzU5OWFiMjJmYzE2ZDI4NDQ1MDVhNjczNzkzNzFlMTA4OTgKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzgKYm9keQljbWQvbXJ3L3BhZGRlZHN0cmVzc190ZXN0LmdvCVRlc3RUaGVHdWFyZHNBZ3JlZVdpdGhUaGVQYXJzZXJPblJhbmRvbUFyZ3YJYTQ0NzIwMWRmNzNjMjFkNGZmODdjMDZmZjg1MDRkM2E1Mjc2NWY4OGRlNTczNjJkYmZhYzUyNDkwNWRlYzgzMg
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAnIterVerbIsNotJudgedAsAPath
      paddedflag_test.go:299: iter 'add ' x was refused, exit 2:
          'add ' has edge whitespace the argument parser strips; put -- before the path: mrw iter add -- 'add '
  --- FAIL: TestAnIterVerbIsNotJudgedAsAPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.307s
  FAIL
  ```
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:2238
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1780
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1763
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1782
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1569
- 2026-09-25 · edcd296* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:1791
- 2026-09-25 · 5cf522c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:0 · test-lock-sha256:61036fae709231af312bc8b5c85c4371881358d964cac69bbf140a1e8a8f63d8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QUxvbmVEYXNoRW5kc1RoZVBhcnNlQW5kVGhlR3VhcmQJMWYxY2UzYzA5OTk0MThjNGU4MjdkOTA5ZGViNTlmOWE0NmEyYWJkNDY2ZTBhZDNkYjg2MDE2Y2Y2OGNkYjUxZApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUGFkZGVkQm9vbGVhbkZsYWdOYW1lRG9lc05vdEhpZGVBUGFkZGVkUGF0aAliYmIwMTJmMDBlZTMwN2E5ZjUxMmIyMDliZGVhNmRkZjQzNWY0MTU4OWUwNTYzOGIxZGFjZGYyMThmMGViYzY1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdEFQYWRkZWRMb25lRGFzaElzUmVmdXNlZEJlZm9yZVRoZUd1YXJkU3RvcHMJMmM3ODk2ZDIxMGQxZTliMzg4N2Q2ZTRlMzI5NWEwODc5YTNhM2I4ODQwYzlkOTMzMDI0ODhjMzc0ZTcwOGEyNApib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUHJlc2VydmVkU3RvcFRva2VuSXNOb3RKdWRnZWRBZ2FpbnN0QVBvc2l0aW9uYWwJOGI0ZWU4Zjg5MzczYjYyMjkyYzVmYTllYTEzZWI0NDgzMWY1NmY2Mjc4NDEzOGRiNWYzYzhiMTFkZWMxYWRhYwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBUm9vdFRlcm1pbmF0b3JEb2VzTm90RW5kVGhlR3VhcmRCZWxvd0l0CWRiM2JhOGMwNGU4MjFhODlmZjc2ZTJmOWYxYjM4NDIxODkyOTk4YTI1ODcyODgwNzc2ZjRiNzE4YjhiMDMwMTAKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QVNpbmdsZURhc2hOb25MZXR0ZXJUb2tlblN0b3BzVGhlUGFyc2VyQW5kVGhlR3VhcmQJMjU0NTUxNTA4ZTM0YzJhOTAxZmUwYmRjNTczMjg1NjQzYmVjZGY5Y2E2YTY2M2QwYjlhNGUxNzNmNDVmNWEwNgpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkRmxhZ1ZhbHVlV2l0aFRyYWlsaW5nU3BhY2VJc1JlZnVzZWQJNTY0YTYxM2Y3M2E4OGFhMzY5MmU5OTk0ODFiODRlMTUxNTUyMTIxMjA4ZDBhNzBhZGY1NmRiYjIxOTA4YTI1Ywpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkF0dGFjaGVkVmFsdWVFbmRpbmdJbkFueVdoaXRlc3BhY2VJc1JlZnVzZWQJNjZjODBkZjA0YmJjMDM5MzQxNjEwNzgzNWMyMjA4ZGUwMzQzOGFiNzhkYmFlODM0ZjY4Mjc5NzMzNDBhODdjOQpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBbkluaGVyaXRlZFJvb3RGbGFnSXNSZWFkQnlCb3RoR3VhcmRzCWRkOTE1ZDczNTJiMGZhYWE1ZDMzZDY0NzhkMTUxOGNkZWVhMDg0NDJkNzc2YmY2MDczZTNiOTlhZTNlNWYwM2EKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5JdGVyVmVyYklzTm90SnVkZ2VkQXNBUGF0aAkyZjM2YThlMzEwNGVhMDlhYzhkYjllNzBmNGU0NDNkM2NhYmM2MDJhMjA4YmQ3NDU5NDQ2YTJhYmQzMzhlN2Q1CmJvZHkJY21kL21ydy9wYWRkZWRmbGFnX3Rlc3QuZ28JVGVzdFRoZUl0ZXJSZWZ1c2FsS2VlcHNUaGVWZXJiCTgzYmE5MjZlZmE5ZGM1OWU1YzgzMTk2YzJhN2U5NzU5OWFiMjJmYzE2ZDI4NDQ1MDVhNjczNzkzNzFlMTA4OTgKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlV2hvbGVBcmd2R3VhcmRSZWFkc0ZsYWdWYWx1ZXNBbmRUaGVUZXJtaW5hdG9yCWE1Y2I0NGIyNzcwMmM3MWYyNGQ4M2E3MjE1ZmI2ZGY5NDViYzMyM2M5NjdkYTIyNDRmYjZlNDhkMzYwNjVkNzgKYm9keQljbWQvbXJ3L3BhZGRlZHN0cmVzc190ZXN0LmdvCVRlc3RUaGVHdWFyZHNBZ3JlZVdpdGhUaGVQYXJzZXJPblJhbmRvbUFyZ3YJZWM3M2IzMmRkMDhlNGVmNTNjY2MyODlmNWEyNTlkNzBhOWI3ODE3ZGNkZDBhMjBmNGQxMjUwYjQ3NzQyNjFmMQ · test-lock-kind:replace
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · ms:2356

## Mutation Log
(empty until execute)
- 2026-09-25 · edcd296* · mutant killed · exit 1 · `cmd/mrw/main.go` · the verb is judged again: iter add-space x is refused; the named test and the random test go red · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · covers:an iter verb is not judged
- 2026-09-25 · edcd296* · mutant killed · exit 1 · `cmd/mrw/main.go` · T10 reverted: a preserved stop token is judged against a sibling; only the random test can see it here · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · covers:the guards agree with the parser
- 2026-09-25 · edcd296* · mutant killed · exit 1 · `cmd/mrw/main.go` · T7 reverted: an inherited --root is unknown to both guards; only the random test can see it here · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · covers:the guards agree with the parser
- 2026-09-25 · edcd296* · mutant killed · exit 1 · `cmd/mrw/main.go` · T6 reverted: a newline-ended attached value opens the trimmed name; only the random test can see it here · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · covers:the guards agree with the parser
- 2026-09-25 · edcd296* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:9f39cf83fc3f67536c6bb447e8d44888e451d94c45dcd11cb4f943b60f9349c0 · covers:no engine file changes

## Invariants

- Every T5–T10 test stays green.
- The oracle is written from the Decision and the parser's source, never from the guard code.

## Risks

- The oracle's pools are the generator's: a shape outside them (a grouped short flag, a third-level subcommand) is not exercised. Widen the pool when a reviewer names one, and prove the widened pool goes red against the reverted fix.
- The parser is the oracle, so a parser upgrade that changes a rule moves the oracle with it; the guard must follow, and this test says where.

## Out of Scope

- Driving the built binary through the same matrix (arm 3): the guards run in the Actions and in `main`, both of which this test calls in-process, and §128–§139 already drive the binary per shape (permanent: boundary: the exit-code oracle is the same predicate; citation: this task's Acceptance)

## Stop Condition

Stop if a mismatch needs a format change or an exit-code change.
