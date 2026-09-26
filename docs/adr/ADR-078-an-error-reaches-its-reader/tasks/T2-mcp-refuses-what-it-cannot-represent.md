# Task ADR-078-T2: the MCP server refuses what it cannot represent

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `validRequestID`, `nonUTF8Arg`, `read.CheckExclude`
**Consumes:** `json.RawMessage`, `pathExcluded`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an id is a string or an integer`, `invalid UTF-8 is refused by name`, `a glob that cannot match is refused on both surfaces`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

An id of any JSON type was echoed back, invalid UTF-8 became a path nobody sent, and a glob that can never match was ignored over MCP.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `validRequestID` |
| `internal/mcp/tools.go` | edit | `nonUTF8Arg`; `CheckExclude` on `mrw_read` |
| `internal/read/walk.go` | edit | `CheckExclude`; `excluded` calls `pathExcluded` |
| `cmd/mrw/main.go` | edit | the CLI calls `CheckExclude` |
| `internal/mcp/protocol078_test.go`, `internal/read/stop078_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §156 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `validRequestID` returns true; `nonUTF8Arg` returns ""; the MCP `CheckExclude` call removed; `CheckExclude` accepts a leading /.
3. [S3] Contract §156. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ ./internal/read/ ./cmd/mrw/ -count=1 -timeout 240s -run 'TestARequestIdThatIsNeitherAStringNorAnIntegerIsInvalid|TestArgumentsThatAreNotUTF8AreRefusedByName|TestABadExcludeGlobIsRefusedOverMCP|TestCheckExcludeRefusesWhatCanNeverMatch|TestTheDocumentedUsageErrorsAreErrors' -v 2>&1 | tee /tmp/adr078-T2.out \
  && missing=$(for t in TestARequestIdThatIsNeitherAStringNorAnIntegerIsInvalid TestArgumentsThatAreNotUTF8AreRefusedByName TestABadExcludeGlobIsRefusedOverMCP TestCheckExcludeRefusesWhatCanNeverMatch TestTheDocumentedUsageErrorsAreErrors; do grep -qE "^--- PASS: $t \(" /tmp/adr078-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 156\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARequestIdThatIsNeitherAStringNorAnIntegerIsInvalid` | `internal/mcp/protocol078_test.go` | `true`, `1.5`, `{}`, `[]`, `1e3`, `-0.0` refused `-32600` with a null id; strings and integers served | — | S1, S2 |
| `TestArgumentsThatAreNotUTF8AreRefusedByName` | `internal/mcp/protocol078_test.go` | raw `\xff` in `specs` and in `plan` refused by name; nothing created; UTF-8 passes | — | S1, S2 |
| `TestABadExcludeGlobIsRefusedOverMCP` | `internal/mcp/protocol078_test.go` | `[` and `/vendor` refused; `vendor` accepted | — | S1, S2 |
| `TestCheckExcludeRefusesWhatCanNeverMatch` | `internal/read/stop078_test.go` | the shared check | — | S1, S2 |
| `TestTheDocumentedUsageErrorsAreErrors` | `cmd/mrw/planpath_test.go` | the CLI's `--exclude '['` refusal is unchanged | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every CLI call, or every MCP request |
| 3 — the caller can discover it | each refusal names the fix |
| 4 — it is used | the v1.25.1 adversarial round and the review of #232 hit each |

## Verification Log
(empty until execute)
- 2026-09-26 · 6b17481* · exit 1 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:644 · test-lock-sha256:393fdff52fd6514295a77e37256010e87c6c7d3fe1f732a8c55ee79ed2104f94 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0QU1pc3NpbmdQbGFuRmlsZVNheXNXaGVyZUl0TG9va2VkCTFmNTY3MTc5OGU4ZDllZGNkMDYxMWZjZjIyNzY0ZjQ4ZDY1OWFhNDZiYjA0ZDljNjk0NGRiMGI5NGNiYTVkN2IKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVJc1NlcnZlZFRocm91Z2hHcmVwRGVzcGl0ZUV4Y2x1ZGUJMWVlODVmYmFmMmM4NzIxZjVjMjNjOTU3NjEyZDI3YzIwNmQyMWU5OWUzNzQyZDU3NjM4NjYxOTY0OGRlZDEzNQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0QVBsYW5Jc1JlYWRSZWxhdGl2ZVRvVGhlV29ya2luZ0RpcmVjdG9yeQljY2Q4OGVlZDBkMWY2NWY5ZmNjYWZiZWEzMGIxOTAyZjdhNjZjYWVhMzQyYzNkYTcyMmM0ZTA0NWE5YmU5MjcwCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RFeGNsdWRlRHJvcHNBTWF0Y2hpbmdGaWxlQW5kUHJ1bmVzQURpcmVjdG9yeQljMDBkMDU3YzBmZmNhZDA1ODU1NzViNDQ3ZTcwOTg3OGQ1OTEyNjlhMzFkZTNlZDBmMjc0OTAxYzYyZDExZGI0CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RGaWxlc0Zyb21SZWFkc1NwZWNzRnJvbVN0ZGluCTA5NzE4OGNlODY2NmZlZDllNmM2N2I5MmU0NjcwNjJjNGY4ZDg0YTYzNjFmYjVmOTBjMzI2ZDc0ZjUwYTBhOTIKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBIb25vdXJzQW5BYnNvbHV0ZVBhdGhJbnNpZGVUaGVSb290CWNjMWNlMjg4Njc1OTAxZjk4MzlkYjI1NDVmOTc5MzI1N2VjOTEzNWRjNTFjZmIzNWEyYzQ2MzJjM2ZkMzRhMWQKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBSZXBvcnRzQVBhdHRlcm5UaGF0TWF0Y2hlZE5vRmlsZQlkYTk1NTQ3ZGNhNThjMWM0Yzk3MGJlZmIzZmQ4MzFhZTUyZDRjMzU4MmU3ODhhOTQwZDY4NTY0MDQ5YWJiODcwCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RHcmVwUmVwb3J0c0FSZWZ1c2VkUGF0aEFuZFNlcnZlc1RoZVJlc3QJZjk0YjhmNTM0NzE2OWFhNmRhOWQ2ZjJlOTkwYjY3MGY3ZDcyY2UxZmQ0NzgyNjk3NTRhNjlmY2E1MjVlNDdiNwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0R3JlcFJlc29sdmVzV29ya2luZ1NldFBvaW50ZXJzCWMyM2VlZGEwNjhlMGJkYmM5YmQ0MmU1ZDM0MjgyNDEyYjUwY2Y4MjljMWE0ZjA0ODQyMzE1MTY0MWEyZmUyOGYKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBXaXRoTm9QYXRoc1dhbGtzVGhlUm9vdAllMTA0OWExOTEwNDIxNjhjNTk0YzIxZmVjYjNhOWEwODQ2M2JkYjdmNzE1NTEzMTY5OGFjYmMyNTZkYmM5MTAxCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3ROb0FyZ3VtZW50c1dpdGhvdXRHcmVwU3RpbGxSZWFkc1RoZVdvcmtpbmdTZXQJZjIzYzJjMTY3YzA2YzdlN2ZmYjRjODE5NGUzMjkyZGJmZjJhMGYxMzdlOWJmM2JlOTJiNTkwNzBlNWZiYTBjMwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0UHJlY2VkZW5jZVVzZXNGbGFnUHJlc2VuY2VOb3RFbXB0aW5lc3MJNWI2ZTRiNzFiZDU5NGRhOWMzOWY0MGMyZTVjNzdmYjQ4OGVjODdjZTcxMDRlNTc3ZjJkZDIwNWYzNDA2NjNiMQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNDb3VudHNBUmVjb3JkZWRXcml0ZQk3YWZiMDg4ZTBhZTQ0MTA2NWQ0YTgwYjZiNmQwMzVjYmU3ZjU2OTFiZTY0NGZkOTdiMjZlZmEwZWE3MGQ5YTRlCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c0pTT05QYXJzZXMJNDk5ZDcwZTZmYzgzN2ZiODFjMmUxNjExOWJiMGE0MWMwNGY0Yzk2NzhjOWY2MzExYTk4OWVmNTYxMzE3ODA2Mgpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNMYW5kZWRMaW5lVXNlc0FwcGxpZWRQbHVzRmFpbGVkQ2hlY2tQbHVzQ2hlY2tOb3RSdW4JMWUxZTM1ZjNhY2Y2N2VmYzE1M2NkNDg4M2Q1YTg1YzUyMmUzNTlmODM0YWM3YWMyMzI3NDJhMGZlMmNjYzFkZQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNPbkFGcmVzaENoZWNrb3V0U2F5c05vdGhpbmdJc1JlY29yZGVkWWV0CTEzYjAzZTA0ZDZkNmQ1NWRmYzZlZTJmZjgwODc5ODIwZTJkYmMzODNlMGUyNDJmNTNlYmU5ZmUyNGZiYWZiOTcKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUHJpY2VzU3RyaWN0QmFsYW5jZQk0NmM4MTcwODY1NmNjOWEyZDQxODAzMmYzNGI4ZDE1YzllMTA1OGEwNmUzYmY2ZjZhMzE5ZTUwYWZmZTAxZGQ2CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c1ByaW50c0ZhaWxlZENoZWNrRXZlbldoZW5aZXJvCTU5MWQwN2RjMTFiMzQ0MTNkNWIxMTA0N2RiOWZiZmMzZDIzZTQyOWVhNWNiNmE3ZjNiMGM3Y2QyNDc5NmJmNTcKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUHJpbnRzVGhlRGVub21pbmF0b3JCZXNpZGVUaGVSYXRlCWU4YjBlMjg2MjJmZGI0OGMyMjFkMWE1MWZlMTE2Y2Q5NDJiNWUwZTMxZmIyNzVlZTZiMGViZjEyMmEzY2Q0M2QKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUmVzZXRFbXB0aWVzQW5kUmVwb3J0c1doYXRJdERpc2NhcmRlZAllYTQ5MDYyYTE5ZjI2NGVjMWEzYzVjYmI3OGJhMzczYTc3MDZjNmE0ZDZmNmUxYWVlMmVhY2VkYWU2NWQxZmU3CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RUaGVEb2N1bWVudGVkVXNhZ2VFcnJvcnNBcmVFcnJvcnMJNDFjZjJhNDU2MjViYTRmNzc0NTdiNDg4ZWYxNjllYWM5OTA0Mzk3NzIxYzNhMjE2YzRlYjFhMjlkYWFhMmRjNgpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0VGhlUGxhbkVycm9yV3JhcHNUaGVDYXVzZQlmNjgzMmQyN2RjMWE5Zjg3YzFkMTQ0ODE1YmYxYzExMTdkMjM2YTMzMjVkNjY0MzNhNzNmNWYwZTM5NThmMzY3CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFCYWRFeGNsdWRlR2xvYklzUmVmdXNlZE92ZXJNQ1AJYmFhMjY0ZjZhNjIyYjhlODBmZWFiNmEzOGZlZTM5ZjFiZjFhZTk1NGE3YjIzNzhjZjhkNmQ3YmUwMTNjNTBkMwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTG9uZ0xpbmVJblRoZU1pZGRsZUlzTmFtZWRXaXRoVGhlUmFuZ2VzQXJvdW5kSXQJYjhiNTllYmI1MDI5YjcyYjg4ZjhjNzVmOTUwMzJmYzA2Y2MxOWE0MzY2ZGQ4ZTkzNWVkOTU3Nzk3MDRhMjE4Nwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTWFueVNwZWNSZWFkU3RvcHNPbmNlSXRJc092ZXJUaGVDZWlsaW5nCWM1MjljNDVkY2UwOGViYWUyY2RmMjYyM2I5M2I3MTYwY2M3N2JkYzk4ZDE2MmI5ZDQ4YWVhYTFjNWY4Njg1ZTYKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QVJlcXVlc3RJZFRoYXRJc05laXRoZXJBU3RyaW5nTm9yQW5JbnRlZ2VySXNJbnZhbGlkCWQ1ZDhjNjQ1NzQ4NmRjODE1MGE2MzFmOWNiN2ZjMDkxZWEwMTA3NjAwYWQ3ODA5ZTM5ZDc4MWY4NTk0ZGU1ZWIKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QXJndW1lbnRzVGhhdEFyZU5vdFVURjhBcmVSZWZ1c2VkQnlOYW1lCWI0YTBmNmI5Y2NiNzU5OWUyMGYwYjQ4NTA5MzBhYjhjZDQwNjZiNjk4NmIwZjE4ZGQ0YmFjZTUwNDQyN2JkYTgKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0VGhlUmVmdXNhbE5hbWVzVGhlTGluZVRoYXRFbmNvZGVzUGFzdFRoZUNlaWxpbmcJODU1NTY1YzdkMzNiZDQ5NDY4YjJkMWVmYjNiMGY5MjY2NmQzZWMyZmMyMTFmOGJmZGIwOGY4ZTYxZmI2ZDhhYwpib2R5CWludGVybmFsL3JlYWQvc3RvcDA3OF90ZXN0LmdvCVRlc3RDaGVja0V4Y2x1ZGVSZWZ1c2VzV2hhdENhbk5ldmVyTWF0Y2gJNWI5ZmJmYWVhN2E0ZmY0NmRlNmRmNDlmMDk0ZGVlZTcyZDhhMmVmZWI1MmRhNzcxYjZiNzExYTE1NzM3NDFkNQpib2R5CWludGVybmFsL3JlYWQvc3RvcDA3OF90ZXN0LmdvCVRlc3RSdW5TdG9wc0JldHdlZW5TcGVjc1doZW5Bc2tlZFRvCWU4NDJjNWY4NDRmNjFhNTdmODk1OTQ3YmEyYzYyNzRlYmEyYTcyZWY3OTYxZmQxNGMxMGYxNDlmOGIxNzgwM2I
  ```
  --- last 10 line(s) of stdout (of 46 after folding 46 raw)
           type:text] map[text:{"observed":{"a.txt":{"SHA":"87428fc522803d31065e7bce3cf03fe475096631e5e07bbd7a0fde60c4cf25c7","Spans":[[1,1]]}},"problems":0} type:text]]]
  --- FAIL: TestABadExcludeGlobIsRefusedOverMCP (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.171s
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  === RUN   TestTheDocumentedUsageErrorsAreErrors
  --- PASS: TestTheDocumentedUsageErrorsAreErrors (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.092s
  FAIL
  ```
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:729
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:403
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:322
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:351
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:350
- 2026-09-26 · 6b17481* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:0 · test-lock-sha256:f4d363ceb134ae013c6a86c70a1b49c597e616cc81a814fc860ad4f632c3195e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0QU1pc3NpbmdQbGFuRmlsZVNheXNXaGVyZUl0TG9va2VkCTFmNTY3MTc5OGU4ZDllZGNkMDYxMWZjZjIyNzY0ZjQ4ZDY1OWFhNDZiYjA0ZDljNjk0NGRiMGI5NGNiYTVkN2IKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVJc1NlcnZlZFRocm91Z2hHcmVwRGVzcGl0ZUV4Y2x1ZGUJMWVlODVmYmFmMmM4NzIxZjVjMjNjOTU3NjEyZDI3YzIwNmQyMWU5OWUzNzQyZDU3NjM4NjYxOTY0OGRlZDEzNQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0QVBsYW5Jc1JlYWRSZWxhdGl2ZVRvVGhlV29ya2luZ0RpcmVjdG9yeQljY2Q4OGVlZDBkMWY2NWY5ZmNjYWZiZWEzMGIxOTAyZjdhNjZjYWVhMzQyYzNkYTcyMmM0ZTA0NWE5YmU5MjcwCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RFeGNsdWRlRHJvcHNBTWF0Y2hpbmdGaWxlQW5kUHJ1bmVzQURpcmVjdG9yeQljMDBkMDU3YzBmZmNhZDA1ODU1NzViNDQ3ZTcwOTg3OGQ1OTEyNjlhMzFkZTNlZDBmMjc0OTAxYzYyZDExZGI0CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RGaWxlc0Zyb21SZWFkc1NwZWNzRnJvbVN0ZGluCTA5NzE4OGNlODY2NmZlZDllNmM2N2I5MmU0NjcwNjJjNGY4ZDg0YTYzNjFmYjVmOTBjMzI2ZDc0ZjUwYTBhOTIKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBIb25vdXJzQW5BYnNvbHV0ZVBhdGhJbnNpZGVUaGVSb290CWNjMWNlMjg4Njc1OTAxZjk4MzlkYjI1NDVmOTc5MzI1N2VjOTEzNWRjNTFjZmIzNWEyYzQ2MzJjM2ZkMzRhMWQKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBSZXBvcnRzQVBhdHRlcm5UaGF0TWF0Y2hlZE5vRmlsZQlkYTk1NTQ3ZGNhNThjMWM0Yzk3MGJlZmIzZmQ4MzFhZTUyZDRjMzU4MmU3ODhhOTQwZDY4NTY0MDQ5YWJiODcwCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RHcmVwUmVwb3J0c0FSZWZ1c2VkUGF0aEFuZFNlcnZlc1RoZVJlc3QJZjk0YjhmNTM0NzE2OWFhNmRhOWQ2ZjJlOTkwYjY3MGY3ZDcyY2UxZmQ0NzgyNjk3NTRhNjlmY2E1MjVlNDdiNwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0R3JlcFJlc29sdmVzV29ya2luZ1NldFBvaW50ZXJzCWMyM2VlZGEwNjhlMGJkYmM5YmQ0MmU1ZDM0MjgyNDEyYjUwY2Y4MjljMWE0ZjA0ODQyMzE1MTY0MWEyZmUyOGYKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBXaXRoTm9QYXRoc1dhbGtzVGhlUm9vdAllMTA0OWExOTEwNDIxNjhjNTk0YzIxZmVjYjNhOWEwODQ2M2JkYjdmNzE1NTEzMTY5OGFjYmMyNTZkYmM5MTAxCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3ROb0FyZ3VtZW50c1dpdGhvdXRHcmVwU3RpbGxSZWFkc1RoZVdvcmtpbmdTZXQJZjIzYzJjMTY3YzA2YzdlN2ZmYjRjODE5NGUzMjkyZGJmZjJhMGYxMzdlOWJmM2JlOTJiNTkwNzBlNWZiYTBjMwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0UHJlY2VkZW5jZVVzZXNGbGFnUHJlc2VuY2VOb3RFbXB0aW5lc3MJNWI2ZTRiNzFiZDU5NGRhOWMzOWY0MGMyZTVjNzdmYjQ4OGVjODdjZTcxMDRlNTc3ZjJkZDIwNWYzNDA2NjNiMQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNDb3VudHNBUmVjb3JkZWRXcml0ZQk3YWZiMDg4ZTBhZTQ0MTA2NWQ0YTgwYjZiNmQwMzVjYmU3ZjU2OTFiZTY0NGZkOTdiMjZlZmEwZWE3MGQ5YTRlCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c0pTT05QYXJzZXMJNDk5ZDcwZTZmYzgzN2ZiODFjMmUxNjExOWJiMGE0MWMwNGY0Yzk2NzhjOWY2MzExYTk4OWVmNTYxMzE3ODA2Mgpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNMYW5kZWRMaW5lVXNlc0FwcGxpZWRQbHVzRmFpbGVkQ2hlY2tQbHVzQ2hlY2tOb3RSdW4JMWUxZTM1ZjNhY2Y2N2VmYzE1M2NkNDg4M2Q1YTg1YzUyMmUzNTlmODM0YWM3YWMyMzI3NDJhMGZlMmNjYzFkZQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNPbkFGcmVzaENoZWNrb3V0U2F5c05vdGhpbmdJc1JlY29yZGVkWWV0CTEzYjAzZTA0ZDZkNmQ1NWRmYzZlZTJmZjgwODc5ODIwZTJkYmMzODNlMGUyNDJmNTNlYmU5ZmUyNGZiYWZiOTcKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUHJpY2VzU3RyaWN0QmFsYW5jZQk0NmM4MTcwODY1NmNjOWEyZDQxODAzMmYzNGI4ZDE1YzllMTA1OGEwNmUzYmY2ZjZhMzE5ZTUwYWZmZTAxZGQ2CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c1ByaW50c0ZhaWxlZENoZWNrRXZlbldoZW5aZXJvCTU5MWQwN2RjMTFiMzQ0MTNkNWIxMTA0N2RiOWZiZmMzZDIzZTQyOWVhNWNiNmE3ZjNiMGM3Y2QyNDc5NmJmNTcKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUHJpbnRzVGhlRGVub21pbmF0b3JCZXNpZGVUaGVSYXRlCWU4YjBlMjg2MjJmZGI0OGMyMjFkMWE1MWZlMTE2Y2Q5NDJiNWUwZTMxZmIyNzVlZTZiMGViZjEyMmEzY2Q0M2QKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzUmVzZXRFbXB0aWVzQW5kUmVwb3J0c1doYXRJdERpc2NhcmRlZAllYTQ5MDYyYTE5ZjI2NGVjMWEzYzVjYmI3OGJhMzczYTc3MDZjNmE0ZDZmNmUxYWVlMmVhY2VkYWU2NWQxZmU3CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RUaGVEb2N1bWVudGVkVXNhZ2VFcnJvcnNBcmVFcnJvcnMJNDFjZjJhNDU2MjViYTRmNzc0NTdiNDg4ZWYxNjllYWM5OTA0Mzk3NzIxYzNhMjE2YzRlYjFhMjlkYWFhMmRjNgpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0VGhlUGxhbkVycm9yV3JhcHNUaGVDYXVzZQlmNjgzMmQyN2RjMWE5Zjg3YzFkMTQ0ODE1YmYxYzExMTdkMjM2YTMzMjVkNjY0MzNhNzNmNWYwZTM5NThmMzY3CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFCYWRFeGNsdWRlR2xvYklzUmVmdXNlZE92ZXJNQ1AJYmFhMjY0ZjZhNjIyYjhlODBmZWFiNmEzOGZlZTM5ZjFiZjFhZTk1NGE3YjIzNzhjZjhkNmQ3YmUwMTNjNTBkMwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTG9uZ0xpbmVJblRoZU1pZGRsZUlzTmFtZWRXaXRoVGhlUmFuZ2VzQXJvdW5kSXQJYjhiNTllYmI1MDI5YjcyYjg4ZjhjNzVmOTUwMzJmYzA2Y2MxOWE0MzY2ZGQ4ZTkzNWVkOTU3Nzk3MDRhMjE4Nwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTWFueVNwZWNSZWFkU3RvcHNPbmNlSXRJc092ZXJUaGVDZWlsaW5nCThjNzJkZDUzNTM1MDE2ZDgyMDBmOTU0ODRiMmQxNWE5ZWQxNDNjNmU0ZDIxM2Y2ZTNlMjZhYWM4OTFhODc4MDAKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QVJlbmRlcmVkRml0UmVmdXNhbE5hbWVzVGhlTGluZQliM2RiMzVkYmM5OGU3NjFkNWZlYWM2NWQxZDVmYzBjYzhjNmMzNjZhMjc4MzUwNzNhZWRhNzQ4MDE0MzQzZTFjCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZXF1ZXN0SWRUaGF0SXNOZWl0aGVyQVN0cmluZ05vckFuSW50ZWdlcklzSW52YWxpZAlkNWQ4YzY0NTc0ODZkYzgxNTBhNjMxZjljYjdmYzA5MWVhMDEwNzYwMGFkNzgwOWUzOWQ3ODFmODU5NGRlNWViCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFyZ3VtZW50c1RoYXRBcmVOb3RVVEY4QXJlUmVmdXNlZEJ5TmFtZQliNGEwZjZiOWNjYjc1OTllMjBmMGI0ODUwOTMwYWI4Y2Q0MDY2YjY5ODZiMGYxOGRkNGJhY2U1MDQ0MjdiZGE4CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxOYW1lc1RoZUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nCTg1NTU2NWM3ZDMzYmQ0OTQ2OGIyZDFlZmIzYjBmOTI2NjZkM2VjMmZjMjExZjhiZmRiMDhmOGU2MWZiNmQ4YWMKYm9keQlpbnRlcm5hbC9yZWFkL3N0b3AwNzhfdGVzdC5nbwlUZXN0Q2hlY2tFeGNsdWRlUmVmdXNlc1doYXRDYW5OZXZlck1hdGNoCTViOWZiZmFlYTdhNGZmNDZkZTZkZjQ5ZjA5NGRlZWU3MmQ4YTJlZmViNTJkYTc3MWI2YjcxMWExNTczNzQxZDUKYm9keQlpbnRlcm5hbC9yZWFkL3N0b3AwNzhfdGVzdC5nbwlUZXN0UnVuU3RvcHNCZXR3ZWVuU3BlY3NXaGVuQXNrZWRUbwllODQyYzVmODQ0ZjYxYTU3Zjg5NTk0N2JhMmM2Mjc0ZWJhMmE3MmVmNzk2MWZkMTRjMTBmMTQ5ZjhiMTc4MDNi · test-lock-kind:replace
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · ms:857

## Mutation Log
(empty until execute)
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/mcp.go` · an id of any JSON type is dispatched · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · covers:an id is a string or an integer
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · invalid UTF-8 decodes to U+FFFD · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · covers:invalid UTF-8 is refused by name
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/mcp/tools.go` · MCP ignores a glob that cannot match · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · covers:a glob that cannot match is refused on both surfaces
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/read/walk.go` · a glob starting with / is accepted · acceptance-sha256:b9190356f5b2384aa1c3344c9c41f96a8f757d2716976af9a2d2d83dd41b54a3 · covers:a glob that cannot match is refused on both surfaces

## Invariants

- Nothing the MCP server answers pairs with a request it could not have received as sent.

## Risks

- A host that sends integer ids beyond 64 bits: the literal check accepts any integer.

## Out of Scope

- A `\\udc80` escape (permanent: fact: valid JSON text; refusing a decoded U+FFFD would refuse real names)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
