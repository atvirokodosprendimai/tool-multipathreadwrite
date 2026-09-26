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

100,000 named specs held the server past 120 s in the v1.25.1 round, and a line past the ceiling was advised a range that held it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.Stop`, asked before each spec |
| `internal/mcp/tools.go` | edit | the stop on a named multi-spec read; `tooLongLine`, `lineTooLongMessage` in both refusals |
| `internal/mcp/protocol078_test.go`, `internal/read/stop078_test.go` | new | the tests below |
| `internal/mcp/tools_test.go` | edit | `TestTheUnservableLineIsDiagnosedPerFile` pins the crowded case |
| `scripts/contract.sh` | edit | §157, §158 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `Run` ignores `Stop`; the MCP read sets no `Stop`; `overflowMessage` skips `tooLongLine`; `renderedFitMessage` skips it.
3. [S3] Contract §157 and §158. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ ./internal/mcp/ -count=1 -timeout 240s -run 'TestRunStopsBetweenSpecsWhenAskedTo|TestAManySpecReadStopsOnceItIsOverTheCeiling|TestTheRefusalNamesTheLineThatEncodesPastTheCeiling|TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt|TestALineThatEncodesPastTheCeilingIsSentToTheCLI|TestARenderedFitRefusalNamesTheLine|TestTheUnservableLineIsDiagnosedPerFile' -v 2>&1 | tee /tmp/adr078-T3.out \
  && missing=$(for t in TestRunStopsBetweenSpecsWhenAskedTo TestAManySpecReadStopsOnceItIsOverTheCeiling TestTheRefusalNamesTheLineThatEncodesPastTheCeiling TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt TestALineThatEncodesPastTheCeilingIsSentToTheCLI TestARenderedFitRefusalNamesTheLine TestTheUnservableLineIsDiagnosedPerFile; do grep -qE "^--- PASS: $t \(" /tmp/adr078-T3.out || echo "$t"; done) \
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
| `TestAManySpecReadStopsOnceItIsOverTheCeiling` | `internal/mcp/protocol078_test.go` | 100,000 specs refused within 30 s, "more than N bytes" with N under twice the limit, advising fewer files and naming no long line | — | S1, S2 |
| `TestTheRefusalNamesTheLineThatEncodesPastTheCeiling` | `internal/mcp/protocol078_test.go` | an escaped line, a plain line past the ceiling, and two such lines in one file are each named with the open range after them, which is served | — | S1, S2 |
| `TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt` | `internal/mcp/protocol078_test.go` | line 5 named; `f.svg:1-4` "holds the lines before it" and `f.svg:6-` "reads on", neither promised | — | S1, S2 |
| `TestALineThatEncodesPastTheCeilingIsSentToTheCLI` | `internal/mcp/escaped_page_test.go` | ADR-074's pair: no `:1-1`, names `mrw read` | — | S2 |
| `TestARenderedFitRefusalNamesTheLine` | `internal/mcp/protocol078_test.go` | a line that renders inside the limit and encodes past it is named, with `f.svg:2-` | — | S1, S2 |
| `TestTheUnservableLineIsDiagnosedPerFile` | `internal/mcp/tools_test.go` | a file after others that completed no line is named and sent to be read alone, which settles it; long servable lines still get a range | — | S2 |

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
- 2026-09-26 · 97c7d92* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:0 · test-lock-sha256:836ae07663bba31a0ae0e568725f99ae6a2d0e8c4675e6b33286ea3424c71494 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUNsb3NlZE1hcmt1cFJhbmdlUmVmdXNhbE5hbWVzVGhlRW5jb2RpbmcJMjNkNmMxNWZhMDdiYWFmNDc2ZWZhOTY1YmU3YzM2ODFiN2I5ODA4ZDFiMzI4ZDkwZjNkNGMwYzA3ZTNhYmM0MApib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUZpbGVXaG9zZUVzY2FwZWRIYWxmQ29tZXNGaXJzdFBhZ2VzVG9JdHNFbmQJODA2YzhiZTZkMzliYmVmZDU4NTlkZTExNWUwYmU3ZTA2OGU1OWQ5NTkzNzc2YjY0YTZhNGU3ZjIyNTZkNWVlZQpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nSXNTZW50VG9UaGVDTEkJMTViMTk4YjYwZGFiNjg1ZmFkMzExMjMyYzIyOTM2MTY0OThlNzJlMjExYjFkYjkzNDVjZDNiMjc0YTJlMmQwNwpib2R5CWludGVybmFsL21jcC9lc2NhcGVkX3BhZ2VfdGVzdC5nbwlUZXN0QU1hcmt1cEZpbGVQYWdlc0J5SXRzRW5jb2RlZFNpemUJMzkzNTc0YmE1NGYyM2I0YzQxMGE0ZGQwZTkxNmZkOTJiMmJjZTZkMDdlYWFkZWFjY2FmNzE3ZWUyM2M2MGY3MQpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBQmFkRXhjbHVkZUdsb2JJc1JlZnVzZWRPdmVyTUNQCTdhOTUzOWJmOTVkMzczMjMwZTU5OWJiZTBmYjFjZDM0NzRjODg0MjUzYTczMWFmMzdjNjhjZjRiMmIzNzcwOWIKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QUxvbmdMaW5lSW5UaGVNaWRkbGVJc05hbWVkV2l0aFRoZVJhbmdlc0Fyb3VuZEl0CWQzYjgwMDRlMThhNzRlNzUyYjcxNTkyZmQyNWJmYzk2YzIwYTBjYmFmODNlMGRiZmY0OWU0OGFjY2JhYWEzMmMKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QU1hbnlTcGVjUmVhZFN0b3BzT25jZUl0SXNPdmVyVGhlQ2VpbGluZwk3ZWE2MWJhYmE0NWI4Yjk1ZTA4MWQ2OWIyYmEyNjk5Njc3MzA1MWJhY2RlNGNmYjUzOGQzNmU5NzVjZGY4ZjI4CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZW5kZXJlZEZpdFJlZnVzYWxOYW1lc1RoZUxpbmUJYjNkYjM1ZGJjOThlNzYxZDVmZWFjNjVkMWQ1ZmMwY2M4YzZjMzY2YTI3ODM1MDczYWVkYTc0ODAxNDM0M2UxYwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBUmVxdWVzdElkVGhhdElzTmVpdGhlckFTdHJpbmdOb3JBbkludGVnZXJJc0ludmFsaWQJN2VhNTEzNDM2NTM5MmQzMTM4Mzc5ZjU4NzhiZDUxMDcwOGY1YzE1YmQ5NmIyOTcyZmE5YmNjMjdhZTI4OTY1MQpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBcmd1bWVudHNUaGF0QXJlTm90VVRGOEFyZVJlZnVzZWRCeU5hbWUJNzU2NjZiNjI5YzgzYjcxOTA1NDU2MGIzMGRmZjFiZjMyZDA1ZGRkY2RlYTcxODk3ZGZmYTZlMDBhNGEwOWQ1Ngpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RUaGVSZWZ1c2FsTmFtZXNUaGVMaW5lVGhhdEVuY29kZXNQYXN0VGhlQ2VpbGluZwkyMDQzNTBmODVhODE4NmRkMTBkOGM4ZDQ3MTgwZjI5YTgwNWMxNTBjM2VkN2ZmZWQxY2VjYThiZDE1Mzk3ODRjCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlSXNLbm93bkJ5SXRzU2VydmVkVGV4dAkwYzc1Zjg0YjYxMTFlZTk1MjU4NjIxY2E5NTU0N2JhYTZlNGI0NTU5NmYzZjkwNTBhNjdmMjRmNzY2ZmRmZDI5CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlSXNNZWFzdXJlZEFmdGVySXRzTWFya2Vyc0FuZEZvb3RlcgllZDYzMGI2ODhhOTIxY2UxOTRlODI1Zjc1MmVjOWZkZDhlZWRkODQxZDE4MGY1NzM5NjViM2YwOWU1M2QxNWE4CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlTGljZW5zZXNPbmx5V2hhdEl0U2VydmVkCTRhZDU3YWViOGY2NTU4ZDgyNzgzYmQ0MmI0MDQyZTVlYmZkOWJmMTUyNGE0ZDgwMDEzMWQ4YzU1MTVjZWQ1MzcKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVBhZ2VUaGF0Q2Fubm90Rml0SXNOb3RBUGFnZQk2NTc3ODc5M2YxZjVjYjJjOWY5MTg0Yzc3MWM3MDU2NDM2NGNkNzk4OWU0ZDViMDY4ZGU3OGNhMzU1NmRkNGY0CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlZEZvb3RlckNhcnJpZXNUaGVPbmVSdWxlCWE0ZDFlZDE0NDNhMGYyYmFlM2U0NTQxNmM2Mjk0Y2EyOWI3N2E0ZWVjODI2YzFjODcwOGQ3YWE4YTYwNWI2ZjYKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVBhZ2VkUmVhZFJlYXNzZW1ibGVzVGhlV2hvbGVGaWxlCTMzNTNmODUzNDZhYzMzYjA3MTRlY2Y4YjhkYjMxZjgwMzBmNWJiODBkZDM3ZTliMjFjNmQxMTIzNDE4NDA0NDcKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVJlYWRPdmVyVGhlTGltaXRJc1JlZnVzZWROb3RUcnVuY2F0ZWQJMDBmY2JlY2Q4ZGJjYmZmMDEyN2ViZjM2NDBhNDQzZDllN2UzZDdjYTEyYWM2NzA5ODI0MmYxMjI3NzczZGVmNgpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBUmVhZFRoYXRTZXJ2ZWROb3RoaW5nSXNBbkVycm9yCTZhN2E0NDM0NThlMGY2NWQyNWUzMGI0OTE1NjhlZWQ5MjRkYTU5MTVmMjM4ZWIwZDkxYzllNmNjY2NmZDcxYjkKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVJlYWRVbmRlclRoZUxpbWl0SXNVbmNoYW5nZWQJNzE0NzNhNWI5MTczMWRmZTE5MDk1MmFlNDI5ZTU1ZTQxZGUxYmE4NDEzYmViYWZiZjZiMzZiZjc1ZWNkNGU4Nwpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBV2Fsa1Byb2JsZW1Jc1JlcG9ydGVkQW5kTm90U3dhbGxvd2VkCTNlNmU2MjkwOGJlNmFiODlmNmI5YWI5MTUwMTA4MzRlZTk0NDEzZDUyZGIxMjBiNDc3NGVmOGRiNTUyNTk4MTUKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVdhbGtQcm9ibGVtU3Vydml2ZXNBVmFsaWRTaWJsaW5nCWJlZTcwODhlNWNjMjIxZTc3NTI2OTU0NTI0MDU1YzU4OWY4ZjAyMGU2Mjk4YzUwZjgxY2FjOTc1YzRhYjkxYmIKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVdyaXRlVG9BblVucmVhZEZpbGVJc1JlZnVzZWRPdmVyTUNQCWJiMmE0ZTFmYzI0MTRhM2NjNTA2MzE3NWEwMzRjOTcwY2VkOTA1YTc1N2UwYjZkYTJkZjA3ZWNhOWI5ZTk5YWMKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QWNrT25BUmVhZFByb21vdGVzVG9vCTc3Yjg3MWNjM2QzMjA5YTg3NTRjMjkxNjVlZjRkYWE5MGU3ZGY1MjFkNmFkMDU0ZTA4MzI2NjgwYjVlZDc1MDgKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QWZ0ZXJXaXRob3V0R3JlcElzUmVmdXNlZAk5MTc1MmIwN2JkZjY2NzAzOGI1YWNiOGZlODI4MjNmYzQyYzJiNzFmODU2MWJmYzQzYjA1ZDNiZTAxYTVjYjI4CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFuSW5kZXhUb29MYXJnZVRvU2VydmVQYWdlc0J5RmlsZQk2YjYyNDE0ZWNhOWQzODEyY2E1Njg0ZDExNDY5ZmU2MzU1NjBkZmVkZmM0YjdjMGExODFmNzFhYzA3NjEyYjdiCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFuTUNQUmVhZExpY2Vuc2VzQUNMSVdyaXRlCTkzNmUxZGI4ZDdmZTY1NmJlNzE1MjQ1MzkxMzUxNzVlZWMyYjkwMjkyYmQzOTIxNjE3NjNkNzk3YWFkNmNmZDEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QW5PdmVyc2l6ZWRHcmVwUmV0dXJuc1RoZUluZGV4QW5kTm90QURlYWRFbmQJNzNkNTJkNzA1YjI4NzVhMjFjM2M0MzdiZTFmNWE3YzY1MTU0NzM3YzMxYzQyNWZiZDkwNDBkOTZhYzJjZWQ1MQpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBbk92ZXJzaXplZFJlYWRTdGlsbFJlYWRzQXNJbmNvbXBsZXRlCTM4ZjI1NGI2NWYzZGJjMDFmOGQyZmYxZTY4YzQ4NWQzOTg2MTU1OGM5NTZkZjRlNjlmMDUzMDA4MjczODg5YzEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0Qm90aFRvb2xzQWR2ZXJ0aXNlQWNrCTlhMzYxM2UzMGFmNjE1ZWY2MDg0YjRmMzBmNGY0ZjBlZDFmMjEzNmYzNjYxMzZjZmY0MGQ5ZTllYjE1MzRjYjkKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0Q29uY3VycmVudFRvb2xDYWxsc0RvTm90TG9zZUFMZWRnZXJFbnRyeQk3Mjc1ZGUxYzdkNTE1MTdlZTE5NzliMGVmNjljMTE3NTY4MmU1NDVhMTMwZDdkN2YyNmMwOGY1Y2M3YjllYzAyCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEdyZXBSZWZ1c2VzQVJhbmdlZFNwZWMJODQ2MTliOGE0ZTJjM2I5OTI2MmM3NDU1MmVlYTQyMDc3YjRmMzBmYjNlODA4ZGQ4Mzk4ZWEzMTc4MTZmOTIzNApib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RHcmVwU2VydmVzV2hhdEl0RmluZHNBbmRSZWNvcmRzSXQJNTI5MTcxM2I4MGNmYzg4ZjIzODkyZGVkZjgwMzQwZTYyY2FmODYxNjYxZGE2ZjIxNGZiMWI0NjBjMTFkNjZmMgpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3ROb0dyZXBBbnN3ZXJFeGNlZWRzVGhlRGVjbGFyZWRDYXAJMGUzMmYzN2MyNTAzNDRlN2Q3YWQ1NjI0ODJmNjNkZmIzZmJiYWExMDBkODQ0MThkOGZlNTY5NDE4ZTg3YjZhMwpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVDTElSZWFkSXNVbmFmZmVjdGVkQnlUaGVNQ1BMaW1pdAk0ZTU4MGViMThlZTYxNDBlNjBlZGZmMzM0NzY5N2U2ZjBlZjY1MzUyMjhjZTM0ZDNkODk1Mjc0OTkxZTFkZmY1CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdFRoZUNhcHBlZFdyaXRlclJldGFpbnNOb01vcmVUaGFuSXRzTGltaXQJNTc3N2MxMjU3YjRhZWFlODc3ODU0OTk0Njc0MzUyYjBhNjdlM2I1NjFiZmEzODYyNTY4YTU4YmMxZjdmOTgxMQpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVJbmRleFN1cnZpdmVzQVBhdHRlcm5UaGF0TG9va3NMaWtlQVJhbmdlCWIyNjA4MjFkOTVlNTVlNDdkYThiN2NmMzcyOGM3NGJlNjEwZDUyOTgwMjA4ZDQzNDcxMGRlMTJlZDc1MDQ3ZTEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlUmVhZFRvb2xPYnNlcnZlc1doYXRUaGVDTElXb3VsZE9ic2VydmUJYmE0NmJiYmUyMTk5YjkwMzk5MWEwOTQ2MWExMzQ0ODNlMjNhN2Q1ZjIxNzg1MzRlNmI2OGZjOTA0NjdiYTA0Zgpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVSZWZ1c2FsRG9lc05vdEludmVudEFuSW52YWxpZFNwZWMJMWJmNWZlYWRkYWRhODIyZDJlZDRjNTYzNjhjNTY3NDc4N2U3NWZjYjg4MTM3ODEyNmZjYWQwZjhlYWMxODIwMQpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVSZWZ1c2FsTmFtZXNUaGVMaW1pdEFuZEFSYW5nZVRvUmV0cnkJMzgxNmZhMDEyOTcyOTY0NTg0OTY3MWU5MzUzNjAyMWM0ZGY3Yzk5MTU3MDlkMzk5ZWZiZjM0MmE3NDMxNGQyOApib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVUb29sUmVzdWx0Q2Fycmllc0NvbnRlbnRBbmRTdHJ1Y3R1cmVkQ29udGVudAkwYWQyYjM3ZmY5MGQ4M2Y1YjUyODE3ZjE5YmMxNmZhOTBmMjBlNzFkMjI1ODg1Yzg0ZjZmYjQ0OWI4ZjliMWE3CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdFRoZVVuc2VydmFibGVMaW5lSXNEaWFnbm9zZWRQZXJGaWxlCTUzYzZiMTRlZWZkYjNjOWExZTU4MzcyYTZkYmY4NTc4OTg1YzdjMmYxNzRiNGE3ZGQ4YTMwNDYxNTdjNTdiODkKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlV3JpdGVUb29sUmV0dXJuc1RoZVNhbWVSZXN1bHRBc1RoZUNMSQlhYTI3MjBlNzAzZDhkMWFjYzNlMjhmMjAxMjRjZTQ4MzZjMzYzYTJiMTk3ZTA2ZTgzZDg0NGQwNDA0NTg4ZDM4CmJvZHkJaW50ZXJuYWwvcmVhZC9zdG9wMDc4X3Rlc3QuZ28JVGVzdENoZWNrRXhjbHVkZVJlZnVzZXNXaGF0Q2FuTmV2ZXJNYXRjaAlhMGQ0MDgxODg1ZTU5MGIwZjQxZDA3MTZiYjliODU0NDVhNTM5Yzc1ZjRkYzFjMTFjMDcyN2VlNzA1MzkyYjQwCmJvZHkJaW50ZXJuYWwvcmVhZC9zdG9wMDc4X3Rlc3QuZ28JVGVzdFJ1blN0b3BzQmV0d2VlblNwZWNzV2hlbkFza2VkVG8JZTg0MmM1Zjg0NGY2MWE1N2Y4OTU5NDdiYTJjNjI3NGViYTJhNzJlZjc5NjFmZDE0YzEwZjE0OWY4YjE3ODAzYg · test-lock-kind:replace
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:541
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:423
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:427
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:405
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:492
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:392
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:404
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:413
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · ms:386

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
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/read/read.go` · Run ignores Stop · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP read never asks to stop · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · the stop is said and not taken · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:a read the server refuses stops paying for the rest
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · the overflow refusal advises a range that holds the line · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:the line that cannot fit is named
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · a plain line past the ceiling gets no narrower range · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:the line that cannot fit is named
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · a file after others is diagnosed as unservable · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:the line that cannot fit is named
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · the rendered-fit refusal says no range can help · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:the ranges around it are offered
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/mcp/tools.go` · the prefix range is promised to serve · acceptance-sha256:7eee333cd1d2252deae850f631b377760020ea165da1945e68465a9d6d64fa7c · covers:the ranges around it are offered

## Invariants

- A refusal over MCP names a next call that can succeed, or says there is none.

## Risks

- A timing bound in the contract row, 30 s; the size bound beside it is what shows the stop.

## Out of Scope

- Paging a named multi-spec read (permanent: fact: firstPage pages one open-ended spec; several are refused whole, ADR-014)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
