# Task ADR-071-T3: A Win32 alias is refused on Windows

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `rooted.win32Alias`, its Windows-only call in `Resolve` and `Abs`
**Consumes:** `followLinks` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a trailing dot or space is an alias`, `a colon is a stream`, `. and .. are not aliases`, `the Windows tests compile and exist`, `the engine packages are unchanged`

## Goal

On Windows `b.txt.`, `sp.txt ` and `b.txt::$DATA` reach `b.txt`, and `-C 'dir '` reaches `dir`: a
write lands through a name that is not on disk and the receipt names the alias. Refuse a
component that Win32 would remap, in the caller's path and in the root.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/links.go` | edit | `win32Alias`: the first component Windows would read differently, and what it would read |
| `internal/rooted/rooted.go` | edit | `Resolve` and `Abs` refuse an alias when `followLinks` |
| `internal/rooted/links_test.go` | edit | the pure function, on every platform |
| `internal/rooted/links_windows_test.go` | edit | `Resolve` refuses `b.txt.` and a root `dir ` on Windows |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: the trailing-dot branch dropped; the `:` branch dropped; `.` and `..` no longer exempt (a cleaned `./a.txt` would be refused).

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ -count=1 -timeout 120s -run 'TestWin32Alias|TestResolveRefusesAWin32Alias' -v 2>&1 | tee /tmp/adr071-T3.out \
  && missing=$(for t in TestWin32AliasNamesTheComponentWindowsWouldRemap TestWin32AliasLeavesDotAndDotDotAlone; do grep -qE "^--- PASS: $t \(" /tmp/adr071-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && GOOS=windows go vet ./internal/rooted/ \
  && grep -q '^func TestResolveRefusesAWin32Alias(' internal/rooted/links_windows_test.go \
&& git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWin32AliasNamesTheComponentWindowsWouldRemap` | `internal/rooted/links_test.go` | `b.txt.`, `sp.txt `, `b.txt::$DATA`, `dir /a.txt`, a volume and a UNC prefix | — | S1, S2 |
| `TestWin32AliasLeavesDotAndDotDotAlone` | `internal/rooted/links_test.go` | `.`, `..`, `a/./b` and ordinary names are not aliases | — | S1, S2 |
| `TestResolveRefusesAWin32Alias` | `internal/rooted/links_windows_test.go` | the real `Resolve` and `Abs` on Windows | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `win32Alias` |
| 2 — something selects it | `Resolve` and `Abs` on Windows |
| 3 — the caller can discover it | the refusal names the component and what Windows reads |
| 4 — it is used | the Windows filesystem session wrote and unlinked through these names |

## Verification Log
(empty until execute)
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · ms:1642
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · ms:301
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · ms:277
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · ms:272
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:183a5b6502283b923f1f9fa94a1ef98d39998f2217a7d01e0733cd4517d8a78d · ms:284
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · ms:1075
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · ms:719
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · ms:6769
- 2026-09-25 · 77a408a* · exit 1 · `set -o pipefail …` · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · ms:1124 · test-lock-sha256:bea6d38722d5f498ac4fd0efc32668a0bdc35e46662f68f647d97ae67a88390c · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc190ZXN0LmdvCVRlc3RUaGVMaW5rV2Fsa0JvdW5kc0FMb29wCWU0NmE1YmQyOWE3ZmVkYzc5OTRmOWQ1ZmFjMzBiMGQzMjI2MDUwZmJiZGM1MGJiMDMxODY3OTZlMGRiNmVmMzYKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0VGhlTGlua1dhbGtMZWF2ZXNBUGxhY2Vob2xkZXJBbG9uZQk0N2E4N2RkYjhiY2U1ODhkNzljZTdhYTlhMTNkNGRhZTFlODkzYWFiYjY5YTYxODQ3MzlmZmM4NTg2MTQ1NTk2CmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrUmVmdXNlc0FMaW5rSXRDYW5ub3RSZWFkCTMzZTcyMzEwYjUwNzRhMGMxNjYwM2ExZDU4ZGYxMDc5ZDM0NWIyYWQ3ODQ4YmIxNDExNzEwYmJjNDRiNTFkNmIKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0VGhlTGlua1dhbGtSZXBsYWNlc0FMaW5rV2l0aEl0c1RhcmdldAliY2RjNGJmMjRlMmM4NDAzZjRhMzQ5ODA3YmZiZGM1ZDNmNjgzOTUyYmYyNzMwNTY2NmI3NzVkMWNjNGI4NTJjCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrU3RvcHNBdEFNaXNzaW5nQ29tcG9uZW50CTFjZGEyYTFjNTFkNmU1N2Q0ZDAwZDQ3MjFjMGI3NWYxMGVlNWFmNWFiOGY0NWJiNTE5YWJmM2Q5NTY2NDRhNGUKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0V2luMzJBbGlhc0xlYXZlc0RvdEFuZERvdERvdEFsb25lCTA0YmIzY2Y5Nzg5ZGYxNGE1MzIxMDI5ZjJlZjkyNmExNDVlYTY0NGU3ZTQxMjY5M2IwZWQ4MzZlYTRlYjk2MTAKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0V2luMzJBbGlhc05hbWVzVGhlQ29tcG9uZW50V2luZG93c1dvdWxkUmVtYXAJYTQwY2I4YTdiZWJhZDFiNTE1MDI1MjdjM2M1NWM4NDU4MTY3MDU5NjU4ZDE0ODQ0NTMwMjMwMWRiMDMxZDUzZQpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc193aW5kb3dzX3Rlc3QuZ28JVGVzdFJlc29sdmVSZWZ1c2VzQUp1bmN0aW9uT3V0T2ZUaGVSb290CTUwZjlmMjdkYzg4MWQyMDkxM2ExYjk1ZjU4NzhiZWJjZDhlODI0NGZjMGI0MGFhZWVkOTEzM2JjMTc3OTA2OTQKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3Nfd2luZG93c190ZXN0LmdvCVRlc3RSZXNvbHZlUmVmdXNlc0FXaW4zMkFsaWFzCTNlYjRkMmMwMzJkODkyN2JlNzI3M2Q4MWU1ZTE3NTdiZjk1ZDVkZmJmNWIyYzk0YjczYWQ3ZTk3ZmZmODI2ZDk
  ```
  --- last 10 line(s) of stdout (of 13 after folding 13 raw)
      links_test.go:178: win32Alias("b.txt::$DATA") = ("", ""), want ("b.txt::$DATA", "b.txt")
      links_test.go:178: win32Alias("dir /a.txt") = ("", ""), want ("dir ", "dir")
      links_test.go:178: win32Alias("sub\\dir.\\a.go") = ("", ""), want ("dir.", "dir")
      links_test.go:178: win32Alias("...") = ("", ""), want ("...", "")
  --- FAIL: TestWin32AliasNamesTheComponentWindowsWouldRemap (0.00s)
  === RUN   TestWin32AliasLeavesDotAndDotDotAlone
  --- PASS: TestWin32AliasLeavesDotAndDotDotAlone (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.240s
  FAIL
  ```
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · ms:795
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · ms:345
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · ms:271
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · ms:333
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · ms:299

## Mutation Log
(empty until execute)
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a trailing dot is no longer an alias, so b.txt. reaches b.txt · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · covers:a trailing dot or space is an alias
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a colon is no longer a stream, so b.txt::$DATA reaches b.txt · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · covers:a colon is a stream
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · dot and dot-dot are no longer exempt, so a cleaned ./a.txt would be refused on Windows · acceptance-sha256:2b64847f9b6fc227582f9e7caf51047a524066d40430efa1a692eedc934ad96f · covers:. and .. are not aliases
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a trailing dot is no longer an alias, so b.txt. reaches b.txt · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · covers:a trailing dot or space is an alias
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a colon is no longer a stream, so b.txt::$DATA reaches b.txt · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · covers:a colon is a stream
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · dot and dot-dot are no longer exempt, so a cleaned ./a.txt would be refused on Windows · acceptance-sha256:d1b5bac5a3cf13917be9d9fc9b7722eb8771301fb0796651fdf47ce153b2fad5 · covers:. and .. are not aliases
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · a trailing dot is no longer an alias, so b.txt. reaches b.txt · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · covers:a trailing dot or space is an alias
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · a colon is no longer a stream, so b.txt::$DATA reaches b.txt · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · covers:a colon is a stream
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · dot and dot-dot are no longer exempt, so a cleaned ./a.txt would be refused on Windows · acceptance-sha256:5b5df0c361a553a1117383c05ca0dee0e777ce37414a32ee5640695a79606d1d · covers:. and .. are not aliases

## Invariants

- POSIX is unchanged: `followLinks` is false there.

## Risks

- A real Windows name ending in a space (created through `\\?\`) becomes unreachable; Win32 tools cannot open it either.

## Out of Scope

- 8.3 short names (permanent: fact: they resolve to one file and the ledger matches it; citation: file `internal/apply/apply.go:1752`)

## Stop Condition

Stop if the check needs to run on POSIX.
