# Task ADR-086-T3: Windows refuses a component it would read as U+FFFD

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the invalid-UTF-8 case in `rooted.win32Alias`
**Consumes:** the byte-preserving `splitHeader` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an invalid byte is named an alias`, `valid UTF-8 is not`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

Go's Windows file calls convert a path to UTF-16 and map a byte that is not valid UTF-8 to U+FFFD without an error, so after T1 a create of `bad\xffname.txt` still landed as the replacement name on Windows (Codex review of #254).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/utf8086_test.go` | new | `win32Alias` names the component and what Windows would open |
| `internal/rooted/links.go` | edit | the invalid-UTF-8 case |
| `internal/rooted/rooted.go` | edit | the refusal names the third remap |

## Ordered Steps

1. [S1] Write `TestWin32AliasNamesAComponentThatIsNotValidUTF8`; it fails without the case. [proof: mutation]
2. [S2] Name such a component in `win32Alias`; `rooted.Resolve` already refuses every alias on Windows (ADR-071 Decision 4). [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ -count=1 -timeout 180s -run 'TestWin32AliasNamesAComponentThatIsNotValidUTF8' -v 2>&1 | tee /tmp/adr086-T3.out \
  && missing=$(for t in TestWin32AliasNamesAComponentThatIsNotValidUTF8; do grep -qE "^--- PASS: $t \(" /tmp/adr086-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && GOOS=windows go vet ./internal/rooted/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWin32AliasNamesAComponentThatIsNotValidUTF8` | `internal/rooted/utf8086_test.go` | an invalid byte is named with the U+FFFD name; valid UTF-8 is not | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `win32Alias` |
| 2 — something selects it | `rooted.Resolve` on Windows, for every read spec, plan path and rename destination |
| 3 — the caller can discover it | the refusal names the component and the name Windows would use |
| 4 — it is used | the Codex review of #254 traced it; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 318a8c4* · exit 1 · `set -o pipefail …` · acceptance-sha256:bdafffd6b8178a76378d4a2aea5018f02ef503044eaf5533bde3d26c9b07c6bf · ms:1607 · test-lock-sha256:11c4a1bc6593935d38ff63e3d4b9b3000b34e9fecb8cb1fc4f4038521933a4e7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3Jvb3RlZC91dGY4MDg2X3Rlc3QuZ28JVGVzdFdpbjMyQWxpYXNOYW1lc0FDb21wb25lbnRUaGF0SXNOb3RWYWxpZFVURjgJZWMxNmRjZWJjMzZhNTAyYThmZDE2NGMzYmI4ZGMyZDE4MmIzZTU5NDk3MWNiMWEwMmQwZTgxNzM1ZTBjMzQxMQ
  ```
  --- last 6 line(s) of stdout
  === RUN   TestWin32AliasNamesAComponentThatIsNotValidUTF8
      utf8086_test.go:17: win32Alias = ("", ""), want ("bad\xffname.txt", "bad�name.txt")
  --- FAIL: TestWin32AliasNamesAComponentThatIsNotValidUTF8 (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.183s
  FAIL
  ```
- 2026-09-27 · 318a8c4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bdafffd6b8178a76378d4a2aea5018f02ef503044eaf5533bde3d26c9b07c6bf · ms:916
- 2026-09-27 · 318a8c4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bdafffd6b8178a76378d4a2aea5018f02ef503044eaf5533bde3d26c9b07c6bf · ms:567

## Mutation Log
(empty until execute)
- 2026-09-27 · 318a8c4* · mutant killed · exit 1 · `internal/rooted/links.go` · an invalid byte is not named a Win32 alias · acceptance-sha256:bdafffd6b8178a76378d4a2aea5018f02ef503044eaf5533bde3d26c9b07c6bf · covers:an invalid byte is named an alias

## Invariants

- Off Windows nothing changes: `win32Alias` is consulted only where `followLinks` is set.

## Risks

- The refusal itself runs only on Windows; the Windows CI shard runs the package's tests, and the pure test runs everywhere.

## Out of Scope

- A Windows end-to-end fixture for the refusal (permanent: boundary: `followLinks` is a build-time constant, so only the Windows shard drives `Resolve` there, as ADR-071 Decision 4 already is)

## Stop Condition

The fence exits 0 and the Windows CI shard passes.
