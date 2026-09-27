# Task ADR-086-T1: a plan header keeps every byte

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a byte-preserving `splitHeader`
**Consumes:** nothing
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a path keeps its bytes`, `an anchor keeps its bytes`, `valid text splits as before`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

`@@ bad\xffname.txt 0 create` created `bad\xef\xbf\xbdname.txt` at exit 0: the header walk replaced the invalid byte.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/bytes086_test.go` | new | a path and an anchor keep an invalid byte; a pattern and quotes still split |
| `internal/plan/plan.go` | edit | `splitHeader` walks bytes |
| `scripts/contract.sh` | edit | §168 |

## Ordered Steps

1. [S1] Write `TestAHeaderKeepsEveryByteOfItsPath`; it fails on `537b896`, where the path comes back with U+FFFD. [proof: mutation]
2. [S2] Walk bytes in `splitHeader`. [proof: mutation]
3. [S3] Contract §168: the name lands as written or is refused with nothing written, never under another name. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -count=1 -timeout 180s -run 'TestAHeaderKeepsEveryByteOfItsPath' -v 2>&1 | tee /tmp/adr086-T1.out \
  && missing=$(for t in TestAHeaderKeepsEveryByteOfItsPath; do grep -qE "^--- PASS: $t \(" /tmp/adr086-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 168\. ' scripts/contract.sh \
  && ! grep -q 'rs := \[\]rune(s)' internal/plan/plan.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAHeaderKeepsEveryByteOfItsPath` | `internal/plan/bytes086_test.go` | an invalid byte survives in a path and an anchor; a pattern and quoted values split as before | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `splitHeader` |
| 2 — something selects it | every plan header, CLI and MCP, and both foreign formats' compiled headers |
| 3 — the caller can discover it | the receipt names the file the author wrote |
| 4 — it is used | chaos seed 25 and this record's reproduction; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 537b896* · exit 1 · `set -o pipefail …` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:412 · test-lock-sha256:c0f6d520f3ce4e669276762e372441dc90e9f22b48f03e755439f791bb655c79 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3BsYW4vYnl0ZXMwODZfdGVzdC5nbwlUZXN0QUhlYWRlcktlZXBzRXZlcnlCeXRlT2ZJdHNQYXRoCTEwNmNjYjdhNDk3MzNiMTZlOGY3NmFiMjQzNzZmNzhmYzM4MjcyZTRiODkzYTY3NmY0MmFiNWY1YWY1MGU4MjE
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan.test]
  internal/plan/bytes086_test.go:26:28: invalid operation: hs[2].Addr.StartPat != "^f\xfe (s)" (mismatched types *regexp.Regexp and untyped string)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [build failed]
  FAIL
  ```
- 2026-09-27 · 537b896* · exit 1 · `set -o pipefail …` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:438
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAHeaderKeepsEveryByteOfItsPath
      bytes086_test.go:22: path = "bad�name.txt", want "bad\xffname.txt"
      bytes086_test.go:25: anchor = "x�y z", want "x\xfey z"
  --- FAIL: TestAHeaderKeepsEveryByteOfItsPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.134s
  FAIL
  ```
- 2026-09-27 · 537b896* · exit 1 · `set -o pipefail …` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:239
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAHeaderKeepsEveryByteOfItsPath
  --- PASS: TestAHeaderKeepsEveryByteOfItsPath (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.075s
  ```
- 2026-09-27 · 537b896* · exit 0 · `set -o pipefail …` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:2850
- 2026-09-27 · 537b896* · exit 0 · `set -o pipefail …` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:368
- 2026-09-27 · 537b896* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · ms:0 · test-lock-sha256:293227084421fc3881cb07e65f2b3245c8269489e37f7e1c30d0cbdaf0b4c5a4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3BsYW4vYnl0ZXMwODZfdGVzdC5nbwlUZXN0QUhlYWRlcktlZXBzRXZlcnlCeXRlT2ZJdHNQYXRoCTM2NTM1NGRiNDc4YmViZWYxMTZkY2MyZjdmZTEyYzI2ZDI4MDdjYTE1ZDVhYmJjMThkZWNjZDU3NTZiZjY3MzI · test-lock-kind:replace
- 2026-09-27 · human-observed · relock 2026-09-27: the first red row was a compile failure (StartPat is a *regexp.Regexp, and regexp refuses an invalid byte); the test's third case was corrected to a valid multibyte pattern BEFORE any implementation, then went red for the real reason (path and anchor rewritten to U+FFFD); the path and anchor assertions are unchanged

## Mutation Log
(empty until execute)
- 2026-09-27 · 537b896* · mutant killed · exit 1 · `internal/plan/plan.go` · the header walk rewrites an invalid byte to U+FFFD again · acceptance-sha256:f300a16bdef860873bdcb117e571c70d29dcee221a5c238cc83565a3810b7dfb · covers:a path keeps its bytes

## Invariants

- Every header that splits today splits the same way.

## Risks

- None beyond the record's.

## Out of Scope

- Body lines (permanent: boundary: a body line is never walked as runes, so only the header needs the change)

## Stop Condition

The fence exits 0 and contract §168 passes.
