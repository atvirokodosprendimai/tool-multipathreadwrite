# Task ADR-078-T3: the MCP answer says what to do next

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `read.Options.Stop`, `tooLongLine`, `lineTooLongMessage`
**Consumes:** `capped`, `servedLineNumber`, `encodedTextLen`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a read the server refuses stops paying for the rest`, `the line that cannot fit is named`, `the ranges around it are offered`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

100,000 named specs held the server past two minutes, and a line that alone encodes past the ceiling was advised a range that held it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.Stop`, asked before each spec |
| `internal/mcp/tools.go` | edit | the stop on a named multi-spec read; `tooLongLine`, `lineTooLongMessage` in both refusals |
| `internal/mcp/protocol078_test.go`, `internal/read/stop078_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §157, §158 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `Run` ignores `Stop`; the MCP read sets no `Stop`; `overflowMessage` skips `tooLongLine`; `renderedFitMessage` skips it.
3. [S3] Contract §157 and §158. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ ./internal/mcp/ -count=1 -timeout 240s -run 'TestRunStopsBetweenSpecsWhenAskedTo|TestAManySpecReadStopsOnceItIsOverTheCeiling|TestTheRefusalNamesTheLineThatEncodesPastTheCeiling|TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt|TestALineThatEncodesPastTheCeilingIsSentToTheCLI|TestARenderedFitRefusalNamesTheLine' -v 2>&1 | tee /tmp/adr078-T3.out \
  && missing=$(for t in TestRunStopsBetweenSpecsWhenAskedTo TestAManySpecReadStopsOnceItIsOverTheCeiling TestTheRefusalNamesTheLineThatEncodesPastTheCeiling TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt TestALineThatEncodesPastTheCeilingIsSentToTheCLI TestARenderedFitRefusalNamesTheLine; do grep -qE "^--- PASS: $t \(" /tmp/adr078-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 157\. ' scripts/contract.sh \
  && grep -q '^# 158\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRunStopsBetweenSpecsWhenAskedTo` | `internal/read/stop078_test.go` | Stop true after the first spec serves one | — | S1, S2 |
| `TestAManySpecReadStopsOnceItIsOverTheCeiling` | `internal/mcp/protocol078_test.go` | 100,000 specs refused within 30 s: "would have returned more than", advises fewer files, names no long line | — | S1, S2 |
| `TestTheRefusalNamesTheLineThatEncodesPastTheCeiling` | `internal/mcp/protocol078_test.go` | line 1 named with `f.svg:2-`, which is served | — | S1, S2 |
| `TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt` | `internal/mcp/protocol078_test.go` | line 5 named with `f.svg:1-4` and `f.svg:6-` | — | S1, S2 |
| `TestALineThatEncodesPastTheCeilingIsSentToTheCLI` | `internal/mcp/escaped_page_test.go` | ADR-074's pair: no `:1-1`, names `mrw read` | — | S2 |
| `TestARenderedFitRefusalNamesTheLine` | `internal/mcp/protocol078_test.go` | a line that renders inside the limit and encodes past it is named, with `f.svg:2-` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every CLI call, or every MCP request |
| 3 — the caller can discover it | each refusal names the fix |
| 4 — it is used | the v1.25.1 adversarial round and the review of #232 hit each |

## Verification Log
(empty until execute)
- 2026-09-26 · 6b17481* · exit 1 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:10973 · test-lock-sha256:a862ac0cfaf89b23141b0f935f2e9c0ef5a076e7fb88705ca538f15a8b4bbfcd · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUNsb3NlZE1hcmt1cFJhbmdlUmVmdXNhbE5hbWVzVGhlRW5jb2RpbmcJMjNkNmMxNWZhMDdiYWFmNDc2ZWZhOTY1YmU3YzM2ODFiN2I5ODA4ZDFiMzI4ZDkwZjNkNGMwYzA3ZTNhYmM0MApib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUZpbGVXaG9zZUVzY2FwZWRIYWxmQ29tZXNGaXJzdFBhZ2VzVG9JdHNFbmQJODA2YzhiZTZkMzliYmVmZDU4NTlkZTExNWUwYmU3ZTA2OGU1OWQ5NTkzNzc2YjY0YTZhNGU3ZjIyNTZkNWVlZQpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nSXNTZW50VG9UaGVDTEkJMTViMTk4YjYwZGFiNjg1ZmFkMzExMjMyYzIyOTM2MTY0OThlNzJlMjExYjFkYjkzNDVjZDNiMjc0YTJlMmQwNwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QU1hcmt1cEZpbGVQYWdlc0J5SXRzRW5jb2RlZFNpemUJMzkzNTc0YmE1NGYyM2I0YzQxMGE0ZGQwZTkxNmZkOTJiMmJjZTZkMDdlYWFkZWFjY2FmNzE3ZWUyM2M2MGY3MQpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBQmFkRXhjbHVkZUdsb2JJc1JlZnVzZWRPdmVyTUNQCWJhYTI2NGY2YTYyMmI4ZTgwZmVhYjZhMzhmZWUzOWYxYmYxYWU5NTRhN2IyMzc4Y2Y4ZDZkN2JlMDEzYzUwZDMKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QUxvbmdMaW5lSW5UaGVNaWRkbGVJc05hbWVkV2l0aFRoZVJhbmdlc0Fyb3VuZEl0CWI4YjU5ZWJiNTAyOWI3MmI4OGY4Yzc1Zjk1MDMyZmMwNmNjMTlhNDM2NmRkOGU5MzVlZDk1Nzc5NzA0YTIxODcKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QU1hbnlTcGVjUmVhZFN0b3BzT25jZUl0SXNPdmVyVGhlQ2VpbGluZwljNTI5YzQ1ZGNlMDhlYmFlMmNkZjI2MjNiOTNiNzE2MGNjNzdiZGM5OGQxNjJiOWQ0OGFlYWExYzVmODY4NWU2CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZXF1ZXN0SWRUaGF0SXNOZWl0aGVyQVN0cmluZ05vckFuSW50ZWdlcklzSW52YWxpZAlkNWQ4YzY0NTc0ODZkYzgxNTBhNjMxZjljYjdmYzA5MWVhMDEwNzYwMGFkNzgwOWUzOWQ3ODFmODU5NGRlNWViCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFyZ3VtZW50c1RoYXRBcmVOb3RVVEY4QXJlUmVmdXNlZEJ5TmFtZQliNGEwZjZiOWNjYjc1OTllMjBmMGI0ODUwOTMwYWI4Y2Q0MDY2YjY5ODZiMGYxOGRkNGJhY2U1MDQ0MjdiZGE4CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxOYW1lc1RoZUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nCTg1NTU2NWM3ZDMzYmQ0OTQ2OGIyZDFlZmIzYjBmOTI2NjZkM2VjMmZjMjExZjhiZmRiMDhmOGU2MWZiNmQ4YWMKYm9keQlpbnRlcm5hbC9yZWFkL3N0b3AwNzhfdGVzdC5nbwlUZXN0Q2hlY2tFeGNsdWRlUmVmdXNlc1doYXRDYW5OZXZlck1hdGNoCTViOWZiZmFlYTdhNGZmNDZkZTZkZjQ5ZjA5NGRlZWU3MmQ4YTJlZmViNTJkYTc3MWI2YjcxMWExNTczNzQxZDUKYm9keQlpbnRlcm5hbC9yZWFkL3N0b3AwNzhfdGVzdC5nbwlUZXN0UnVuU3RvcHNCZXR3ZWVuU3BlY3NXaGVuQXNrZWRUbwllODQyYzVmODQ0ZjYxYTU3Zjg5NTk0N2JhMmM2Mjc0ZWJhMmE3MmVmNzk2MWZkMTRjMTBmMTQ5ZjhiMTc4MDNi
  ```
  --- last 10 line(s) of stdout (of 36 after folding 36 raw)
          Nothing was read and nothing was recorded, so no write is licensed by it.
          Ask for narrower ranges — around 101 lines per file at this file's line length — or name fewer files in one call.
      protocol078_test.go:157: the refusal lacks "mrw read f.svg:5":
          that read would have returned about 228058 bytes and the limit is 200000.
          Nothing was read and nothing was recorded, so no write is licensed by it.
          Ask for narrower ranges — around 101 lines per file at this file's line length — or name fewer files in one call.
  --- FAIL: TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	10.726s
  FAIL
  ```
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:398
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:367
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:386
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:390
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · ms:357
- 2026-09-26 · 6b17481* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:0 · test-lock-sha256:7d851e795ba5815fd82ebab9c143c502d8ef58aea24b820d8c4cec5ec7868865 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUNsb3NlZE1hcmt1cFJhbmdlUmVmdXNhbE5hbWVzVGhlRW5jb2RpbmcJMjNkNmMxNWZhMDdiYWFmNDc2ZWZhOTY1YmU3YzM2ODFiN2I5ODA4ZDFiMzI4ZDkwZjNkNGMwYzA3ZTNhYmM0MApib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUZpbGVXaG9zZUVzY2FwZWRIYWxmQ29tZXNGaXJzdFBhZ2VzVG9JdHNFbmQJODA2YzhiZTZkMzliYmVmZDU4NTlkZTExNWUwYmU3ZTA2OGU1OWQ5NTkzNzc2YjY0YTZhNGU3ZjIyNTZkNWVlZQpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nSXNTZW50VG9UaGVDTEkJMTViMTk4YjYwZGFiNjg1ZmFkMzExMjMyYzIyOTM2MTY0OThlNzJlMjExYjFkYjkzNDVjZDNiMjc0YTJlMmQwNwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QU1hcmt1cEZpbGVQYWdlc0J5SXRzRW5jb2RlZFNpemUJMzkzNTc0YmE1NGYyM2I0YzQxMGE0ZGQwZTkxNmZkOTJiMmJjZTZkMDdlYWFkZWFjY2FmNzE3ZWUyM2M2MGY3MQpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBQmFkRXhjbHVkZUdsb2JJc1JlZnVzZWRPdmVyTUNQCWJhYTI2NGY2YTYyMmI4ZTgwZmVhYjZhMzhmZWUzOWYxYmYxYWU5NTRhN2IyMzc4Y2Y4ZDZkN2JlMDEzYzUwZDMKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QUxvbmdMaW5lSW5UaGVNaWRkbGVJc05hbWVkV2l0aFRoZVJhbmdlc0Fyb3VuZEl0CWI4YjU5ZWJiNTAyOWI3MmI4OGY4Yzc1Zjk1MDMyZmMwNmNjMTlhNDM2NmRkOGU5MzVlZDk1Nzc5NzA0YTIxODcKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QU1hbnlTcGVjUmVhZFN0b3BzT25jZUl0SXNPdmVyVGhlQ2VpbGluZwk4YzcyZGQ1MzUzNTAxNmQ4MjAwZjk1NDg0YjJkMTVhOWVkMTQzYzZlNGQyMTNmNmUzZTI2YWFjODkxYTg3ODAwCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZW5kZXJlZEZpdFJlZnVzYWxOYW1lc1RoZUxpbmUJYjNkYjM1ZGJjOThlNzYxZDVmZWFjNjVkMWQ1ZmMwY2M4YzZjMzY2YTI3ODM1MDczYWVkYTc0ODAxNDM0M2UxYwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBUmVxdWVzdElkVGhhdElzTmVpdGhlckFTdHJpbmdOb3JBbkludGVnZXJJc0ludmFsaWQJZDVkOGM2NDU3NDg2ZGM4MTUwYTYzMWY5Y2I3ZmMwOTFlYTAxMDc2MDBhZDc4MDllMzlkNzgxZjg1OTRkZTVlYgpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBcmd1bWVudHNUaGF0QXJlTm90VVRGOEFyZVJlZnVzZWRCeU5hbWUJYjRhMGY2YjljY2I3NTk5ZTIwZjBiNDg1MDkzMGFiOGNkNDA2NmI2OTg2YjBmMThkZDRiYWNlNTA0NDI3YmRhOApib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RUaGVSZWZ1c2FsTmFtZXNUaGVMaW5lVGhhdEVuY29kZXNQYXN0VGhlQ2VpbGluZwk4NTU1NjVjN2QzM2JkNDk0NjhiMmQxZWZiM2IwZjkyNjY2ZDNlYzJmYzIxMWY4YmZkYjA4ZjhlNjFmYjZkOGFjCmJvZHkJaW50ZXJuYWwvcmVhZC9zdG9wMDc4X3Rlc3QuZ28JVGVzdENoZWNrRXhjbHVkZVJlZnVzZXNXaGF0Q2FuTmV2ZXJNYXRjaAk1YjlmYmZhZWE3YTRmZjQ2ZGU2ZGY0OWYwOTRkZWVlNzJkOGEyZWZlYjUyZGE3NzFiNmI3MTFhMTU3Mzc0MWQ1CmJvZHkJaW50ZXJuYWwvcmVhZC9zdG9wMDc4X3Rlc3QuZ28JVGVzdFJ1blN0b3BzQmV0d2VlblNwZWNzV2hlbkFza2VkVG8JZTg0MmM1Zjg0NGY2MWE1N2Y4OTU5NDdiYTJjNjI3NGViYTJhNzJlZjc5NjFmZDE0YzEwZjE0OWY4YjE3ODAzYg · test-lock-kind:replace
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:1082
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:462
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:422
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:680
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · ms:402

## Mutation Log
(empty until execute)
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/read/read.go` · Run ignores Stop · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 6b17481* · mutant survived · exit 0 · `internal/mcp/tools.go` · the MCP read never asks to stop · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · covers:a read the server refuses stops paying for the rest
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · the overflow refusal advises a range that holds the line · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · covers:the line that cannot fit is named
- 2026-09-26 · 6b17481* · mutant survived · exit 0 · `internal/mcp/tools.go` · the rendered-fit refusal says no range can help · acceptance-sha256:5810249148da615291cdda1ebf13ca8a0722a898525d9c48248767d6fe35dbcf · covers:the ranges around it are offered
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/read/read.go` · Run ignores Stop · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP read never asks to stop · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · the overflow refusal advises a range that holds the line · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · covers:the line that cannot fit is named
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · the rendered-fit refusal says no range can help · acceptance-sha256:dd49a5aabb5309ecbdc50b36ac51c2ebc6e7e01cef138dbd6da206edccee4c74 · covers:the ranges around it are offered

## Invariants

- A refusal over MCP names a next call that can succeed, or says there is none.

## Risks

- A timing bound in the contract row; it is 30 s against the two minutes it replaced.

## Out of Scope

- Paging a named multi-spec read (permanent: fact: firstPage pages one open-ended spec; several are refused whole, ADR-014)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
