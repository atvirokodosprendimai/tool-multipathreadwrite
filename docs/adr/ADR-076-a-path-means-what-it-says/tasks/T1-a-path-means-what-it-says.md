# Task ADR-076-T1: a path means what it says

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `rooted.EndsInSeparator`, `rooted.ErrNotADirectory`, the missing-root refusal, `win32Device` and `opensDevice`
**Consumes:** `rooted.Resolve`, `rooted.Abs`, `refuseFile`, `planPathOp`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a trailing separator names a directory`, `a plan names files`, `a rename names the file it makes`, `a missing root is named`, `a device is asked of the OS`, `a contract row drives the binary`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

`a.txt/` reached a.txt on read and write, a rename to `d/` made a file `d`, a missing root was a path
"outside the root", and `NUL` was read as an empty file on Windows. Each names what it is instead.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `EndsInSeparator`, `ErrNotADirectory`; Resolve refuses the spelling on a file and a device; Abs names a missing root |
| `internal/rooted/links.go`, `links_windows.go`, `links_other.go` | edit | `win32Device` (Go's rule) and `opensDevice` (the OS's answer) |
| `internal/read/read.go` | edit | `UNREADABLE` for the spelling; the absolute branch keeps the separator |
| `internal/apply/apply.go`, `internal/apply/pathop.go` | edit | a hunk path and a rename destination spelled as a directory are refused |
| `internal/rooted/paths076_test.go`, `internal/read/trailing076_test.go`, `internal/apply/paths076_test.go` | new | the tests below |
| `cmd/mrw/device_windows_test.go` | new | `NUL` and `CON` refused through the CLI on Windows; `nul.bin` written where the OS makes it a file |
| `scripts/contract.sh` | edit | §151 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `EndsInSeparator` returns false; Resolve's directory clause dropped; apply's hunk-path refusal dropped; the rename-destination refusal dropped; Abs's missing-root clause dropped; `win32Device` returns "".
3. [S3] Contract §151: `a.go/` read and written, a rename to `d/` and to `d/b.go`, a read and a create under a missing root. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ ./internal/read/ ./internal/apply/ -count=1 -timeout 180s -run 'TestADeviceNameIsRefusedNotReadAsAnEmptyFile|TestAPathEndingInASeparatorMustNameADirectory|TestARootThatDoesNotExistIsNamedAsMissing|TestADeviceNameIsACandidateByGosRule|TestASpecEndingInASeparatorIsNotServedAsAFile|TestAHunkPathEndingInASeparatorIsRefused|TestARenameToADirectorySpellingIsRefused' -v 2>&1 | tee /tmp/adr076-T1.out \
  && missing=$(for t in TestAPathEndingInASeparatorMustNameADirectory TestARootThatDoesNotExistIsNamedAsMissing TestADeviceNameIsACandidateByGosRule TestASpecEndingInASeparatorIsNotServedAsAFile TestAHunkPathEndingInASeparatorIsRefused TestARenameToADirectorySpellingIsRefused; do grep -qE "^--- PASS: $t \(" /tmp/adr076-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^func TestADeviceNameIsRefusedNotReadAsAnEmptyFile(' cmd/mrw/device_windows_test.go \
  && grep -q '^# 151\. ' scripts/contract.sh \
  && GOOS=windows go vet ./internal/rooted/ ./internal/apply/ ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/check internal/state internal/lines internal/iter internal/seen internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/check internal/state internal/lines internal/iter internal/seen internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPathEndingInASeparatorMustNameADirectory` | `internal/rooted/paths076_test.go` | `a.txt/` refused with the sentinel; `sub/`, `missing/`, `./` resolve | — | S1, S2 |
| `TestARootThatDoesNotExistIsNamedAsMissing` | `internal/rooted/paths076_test.go` | Abs and Resolve name a missing root, never "outside"; a file root is "not a directory" | — | S1, S2 |
| `TestADeviceNameIsACandidateByGosRule` | `internal/rooted/paths076_test.go` | the device names, with extensions and spaces, are candidates; look-alikes and mid-path names are not | — | S1, S2 |
| `TestASpecEndingInASeparatorIsNotServedAsAFile` | `internal/read/trailing076_test.go` | relative and absolute `a.go/` are UNREADABLE and serve no line; `a.go` is served; a grep under `sub/` walks | — | S1, S2 |
| `TestAHunkPathEndingInASeparatorIsRefused` | `internal/apply/paths076_test.go` | `a.txt/` replace and `new/` create refused, nothing written; `a.txt` applies | — | S1, S2 |
| `TestARenameToADirectorySpellingIsRefused` | `internal/apply/paths076_test.go` | `d/` (missing) and `e/` (existing) refused naming `d/a.txt`; `d/a.txt` lands | — | S1, S2 |
| `TestADeviceNameIsRefusedNotReadAsAnEmptyFile` | `cmd/mrw/device_windows_test.go` | Windows CI: `read NUL`, `@@ NUL 0 create`, a rename to `CON` refused naming the device; `nul.bin` written where it is a file | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.EndsInSeparator`, `win32Device`, `opensDevice` |
| 2 — something selects it | every read spec and every plan path, CLI and MCP |
| 3 — the caller can discover it | each refusal names the spelling and what to write |
| 4 — it is used | the v1.25.1 round and the review of #228 hit each of these |

## Verification Log
(empty until execute)
- 2026-09-26 · c30a546* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:3779
- 2026-09-26 · c30a546* · exit 1 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:785 · test-lock-sha256:e3b9f5971d3da46ecece19d69d9a09b25fdb80173d1667231ffb6ad3e9e27ddb · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZGV2aWNlX3dpbmRvd3NfdGVzdC5nbwlUZXN0QURldmljZU5hbWVJc1JlZnVzZWROb3RSZWFkQXNBbkVtcHR5RmlsZQkwZmU4ZjkwNGVkMjQ5Y2QwMmI5NzFhNGY2MjVlZTQ4ZjViMmUxOWJmYWY4NzE5NmViODhkNDdiM2FlOWExNGU4CmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QUh1bmtQYXRoRW5kaW5nSW5BU2VwYXJhdG9ySXNSZWZ1c2VkCTI4YThiZTJhZDcxOGUzNWRmZDZkZmE5M2ZiN2ZhMTU1ZmYxYzFlNDk5ZmQwMjQ0ZWNjY2MyNDcxMTgyMzI3YzUKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBUmVhZE9ubHlGaWxlSXNSZWZ1c2VkRm9yRXZlcnlPcFRoYXRDaGFuZ2VzSXQJYWYwMWY0Nzk4ODIwOWRlYWQ3NWNjYmRlMjk5MWI2ZTdmZTcwNDkxOWM4YTIyY2NhYWRhOGNmZWRmNGU1YjdiOApib2R5CWludGVybmFsL2FwcGx5L3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFSZW5hbWVUb0FEaXJlY3RvcnlTcGVsbGluZ0lzUmVmdXNlZAkxZTNmN2U0Yjc5MGUyNWI3OTU2MTEzMDRjMGJmMzMwMGViMmViNzY2OGVhMzc2ODgxNDFmODc5MDE4MTE3NmE4CmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QVdyaXRlVGhyb3VnaEFTeW1saW5rTmFtZXNJdHNUYXJnZXQJNmEzYzE5NzE2MjY0Y2JmMzk5NTY3MjdmZDQ5ODU4YjE0ZTc4NGJlMThmZmE2NjE0MjQ3Y2Y4NTJhNzdmMDgwMApib2R5CWludGVybmFsL2FwcGx5L3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFuSW5zZXJ0SW50b0FuRW1wdHlGaWxlRW5kc0l0c0xpbmVXaXRoQU5ld2xpbmUJZWMxYmRmZjM1NDI3YTlmOTY1NmM1MDM0YWVhNDI0M2JlZTU2OGQ4MTU3NDNlMDFmMTQ2OWRmYjAzNzUyYWQzMApib2R5CWludGVybmFsL2FwcGx5L3BhdGhzMDc2X3Rlc3QuZ28JVGVzdFRoZVBsYW5OYW1lc1RoZURpcmVjdG9yaWVzSXRNYWRlCTQwODVlZTRmM2I0YzQ1ZmQyOGE0ZWZjODQ4MmYyNDczMDBlMDdmMjI3ZmNiZGRiM2U3ZGMwMWQ3ZmZjMDgwNmMKYm9keQlpbnRlcm5hbC9yZWFkL3RyYWlsaW5nMDc2X3Rlc3QuZ28JVGVzdEFTcGVjRW5kaW5nSW5BU2VwYXJhdG9ySXNOb3RTZXJ2ZWRBc0FGaWxlCWFmNGM1YTc2OTU5MGM4Mjc0ZWJmODNiN2U2YTc5MjE2Yjk5M2RjY2Q0NDJhMTZkMjNmODI3NzIwY2FkZjJiODYKYm9keQlpbnRlcm5hbC9yb290ZWQvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QURldmljZU5hbWVJc0FDYW5kaWRhdGVCeUdvc1J1bGUJYzEzZjIxNDYyNWFiZDdiMmU4ZGJlNmZlYTczZmU1MWU2MzIwNTcwZWEzZDM2MDIzODQ5ZWFkYTM1OTY0ZDljMwpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBUGF0aEVuZGluZ0luQVNlcGFyYXRvck11c3ROYW1lQURpcmVjdG9yeQlkNTBhYzRjMDZjZjYwY2I2YzkyODg1MDYzODYzMTI2YzJmYjc2YTExMjhjYzdjMTYxZWE0NGY2OTUxZDhmYzExCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFSb290VGhhdERvZXNOb3RFeGlzdElzTmFtZWRBc01pc3NpbmcJM2ViMzc2ZjlhNDRmMTg5NDZiYzVmOTYzZjY4ZjU0ZDVlM2FiYTViNzUwZDY4N2Y5ZWJmZTU5MDA2OGZlMjFlOA
  ```
  --- last 10 line(s) of stdout (of 67 after folding 67 raw)
              5| }
              6| 
              7| func Bar() {
              8| 	return
              9| }
  --- FAIL: TestASpecEndingInASeparatorIsNotServedAsAFile (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.172s
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-09-26 · c30a546* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:423
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:837
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:404
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:395
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:424
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:397
- 2026-09-26 · 86e21bd* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · ms:348

## Mutation Log
(empty until execute)
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/rooted/rooted.go` · EndsInSeparator never sees a separator · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a trailing separator names a directory
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/rooted/rooted.go` · Resolve refuses a directory and passes a file · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a trailing separator names a directory
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/rooted/rooted.go` · Abs falls back to the spelling of a missing root · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a missing root is named
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/apply/apply.go` · apply accepts a hunk path spelled as a directory · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a plan names files
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/apply/pathop.go` · a rename to d/ makes a file d again · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a rename names the file it makes
- 2026-09-26 · 86e21bd* · mutant killed · exit 1 · `internal/rooted/links.go` · no name is a device candidate · acceptance-sha256:a6c724f648454d0b392272fc4ce10d168405ea428dfe4ea7c7ac746fe2079eaa · covers:a device is asked of the OS

## Invariants

- A spelling that names a directory never reaches a file, on read or write; a root is named when it is missing; on Windows a name the OS opens as a device is never read or written.

## Risks

- The Windows half — `opensDevice` — can only be exercised on the Windows CI shard; `TestADeviceNameIsRefusedNotReadAsAnEmptyFile` is its evidence there.

## Out of Scope

- A trailing separator inside an apply_patch document (permanent: fact: it compiles to a native plan and meets the same refusal)

## Stop Condition

The fence exits 0, every Mutation Log row reads `killed`, and the Windows CI shard is green.
