# Task ADR-090-T1: the shipped examples run on a fixture

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `internal/mcp/testdata/example/`; the pattern-addressed `examplePlan`; `TestTheShippedReadExampleServesEverySpec`
**Consumes:** `mrw_read`, `mrw_write` (ADR-010)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the plan's guards meet real code`, `a pattern address resolves exactly once`, `the plan is a multi-site plan`, `the plan shows both address forms`, `every read spec is served`, `the tree is gofmt-clean`, `no engine package changes`

## Goal

The examples a caller copies are proved on a tree that was not built from them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/testdata/example/internal/store/store.go` | new | the fixture the plan's first hunk and the read specs address |
| `internal/mcp/testdata/example/cmd/app/main.go` | new | the fixture the plan's pattern hunk and `$` address |
| `internal/mcp/instructions.go` | edit | the second hunk becomes `/^import \($/` |
| `internal/mcp/conformance_test.go` | edit | `dryRunExample` runs on a copy of the fixture, demands two hunks over two paths and a present `failed`; `treeFor` deleted |
| `internal/mcp/examples_test.go` | new | `TestTheShippedReadExampleServesEverySpec` |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerated | the one address changed in two places |
| `internal/mcp/era_test.go` | edit | the comment names this regeneration |
| `scripts/contract.sh` | edit | §43 copies the fixture, reads the published specs, dry-runs the plan, and refuses a pattern that matches nothing |

## Ordered Steps

1. [S1] The fixture, and `dryRunExample` on a copy of it with the two-hunk, two-path and checked-`failed` assertions; red on `add6807`, whose example says `12 insert-after` against an eleven-line `main.go`. [proof: mutation]
2. [S2] `examplePlan`'s second hunk becomes `/^import \($/`; the golden is regenerated and its diff is that address in two places. [proof: mutation]
3. [S3] `TestTheShippedReadExampleServesEverySpec`. [proof: mutation]
4. [S4] Contract §43 drives the built binary on the fixture, paired with the refused pattern. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestEveryEmbeddedExamplePlanReallyApplies|TestTheShippedReadExampleServesEverySpec|TestTheHandshakeShowsBodyOnAHeader' -v 2>&1 | tee /tmp/adr090-T1.out \
  && missing=$(for t in TestEveryEmbeddedExamplePlanReallyApplies TestTheShippedReadExampleServesEverySpec TestTheHandshakeShowsBodyOnAHeader; do grep -qE "^--- PASS: $t \(" /tmp/adr090-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && ! grep -q 'func treeFor' internal/mcp/conformance_test.go \
  && grep -q 'internal/mcp/testdata/example' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only add6807 -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryEmbeddedExamplePlanReallyApplies` | `internal/mcp/conformance_test.go` | every shipped plan dry-runs clean on a copy of the fixture, with two hunks over two paths, a line and a pattern address, and a present `failed` | — | S1, S2 |
| `TestTheShippedReadExampleServesEverySpec` | `internal/mcp/examples_test.go` | one `mrw_read` of the published specs serves every one of them on the fixture | — | S3 |
| `TestTheHandshakeShowsBodyOnAHeader` | `internal/mcp/bodyheader_test.go` | ADR-070's `body=4` survives the edit | — | S2 |

`TestALegacyResultIsUnchangedByTheModernPath` (`internal/mcp/era_test.go`) holds the regenerated golden
and is run by `go test ./...`, NOT by this fence: any change to an example changes the golden, so in
the fence it would kill every mutant for the wrong reason.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `examplePlan`, `exampleReadSpecs`, the fixture |
| 2 — something selects it | `instructionsText()` and `tools()` publish both examples |
| 3 — the caller can discover it | `initialize` instructions and `tools/list`, read off the built binary by §43 |
| 4 — it is used | telemetry is refused (ADR-009); the evidence for building it is the surviving anchor mutant of 2026-09-04 and ADR-013:210 |

## Verification Log
(empty until execute)
- 2026-09-28 · add6807* · exit 1 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:712 · test-lock-sha256:e4b3e46bca107b06faaf9cd2563c40b82c9c6a034e8a53bc847b2001513fbdec · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9ib2R5aGVhZGVyX3Rlc3QuZ28JVGVzdFRoZUhhbmRzaGFrZVNob3dzQm9keU9uQUhlYWRlcgk4NGUyNDgxMGFjZWJhOTYwZDA0Yjg3MGIzYWM0ZjUzMWUyNDVmYTZhMzZjMmEzMDdiODMwOTVkYzVmNDI3MmEzCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEFSZWFkUmVzdWx0Q2Fycmllc05vU3RydWN0dXJlZENvbnRlbnQJNWI0MTUwZTVjMmUwNTlkYzcxMTM0NDg2ZjJhOWQ2MGJiYTRhZGI1M2FjMTBhOWMxYmY0MWM5OTQ2ZGUzNTliOApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeURlY2xhcmVkT3V0cHV0U2NoZW1hVmFsaWRhdGVzQVJlYWxSZXNwb25zZQljNjBlYTkyNGU2MWIwM2JhYzUxMDUyNjA2ZjhiODJkNmMyMmJmN2Y2OTE0MmQyMmQ1ZDE0NzM3YTA0ODJlZWY5CmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEV2ZXJ5RW1iZWRkZWRFeGFtcGxlUGxhblJlYWxseUFwcGxpZXMJOWFhZmE3MjE3MmI2Y2E1YmM1MjkwNzg2YTJhMzZlOWE5NDlkNGVjYjIzNGIwMTZhYWM5MWI0YTRjNTM0OGFlNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeU91dHB1dFNjaGVtYVByb3BlcnR5SXNEZXNjcmliZWQJNjlmYjBlOGQ4MjNiOWEyZTFkMWIzOTEyMDYxMDcwMDBmNjNhY2YxNDM1ODI4OTMyYzE5YzQyNzZmYmM3N2MzNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RUaGVBbm5vdGF0aW9uc01hdGNoV2hhdFRoZVRvb2xEb2VzCTU1ZWNlMTZjMDhhNjEwMGM1MTlmZjFjZDY1MWY1ZmQxYzk3NjAzM2Y4M2U4OTJhOGM3ZDU0M2JhNWY4Njg4MGEKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0VGhlRmlyc3RDb250ZW50QmxvY2tJc1RoZVNlcmlhbGl6ZWRTdHJ1Y3R1cmVkQ29udGVudAlhNTliOGVmMDUxMWJkMTI1MjRlM2QwMzE2YjYxZWUwMjc1N2U4Mzg3YTgyYzU4MjI0OWQzMzE1N2QxNjY4ZThjCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdFRoZVN0YXR1c0Rlc2NyaXB0aW9uTmFtZXNUaGVWYWx1ZXNUaGVFbmdpbmVTZW5kcwllNTZiZDIzZWExZTc4ZjFhZTFlZTI3MjUxM2FiODU5NTE3ZTU3Y2MxMzkxZmY4NWY0MzFmZjUwNmZjNWE5Zjg4CmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTGVnYWN5UmVzdWx0SXNVbmNoYW5nZWRCeVRoZU1vZGVyblBhdGgJZjc3MTEwMWFmYzdmNWEyY2Q5ZmM2ZDFiMTkyZGYzMjFmNGUxN2ZiZjg3MzQ1YzhhNDljYmM3OTU4YTBkNWIwYgpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVyblJlYWRTdGF5c1dpdGhpblRoZUNlaWxpbmcJODQ4YzJiMTI1YzYwN2FmYzMwYzhkMTk3NWMyODdmZmE1NDU5YWVlNDQ1OGI1ODAwNWUyYzc0ZWY3ZjVlYmY5Ywpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVyblJlcXVlc3RGb3JBblVua25vd25WZXJzaW9uSXNSZWZ1c2VkV2l0aEl0c1N1cHBvcnRlZExpc3QJMTg1MzlhMWI0YjM0NTRjNmMyNDFhZGUzNTQyNTZmMmU3MjgxNDdlOGMzNTJkOWRhMDk2YzVmY2ZiYTNkYTQyOQpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVyblJlcXVlc3RXaXRob3V0Q2xpZW50Q2FwYWJpbGl0aWVzSXNJbnZhbGlkUGFyYW1zCTQxMzAzZmRjZGM4NGQ2Y2U5NTYzOTg1NTU3OGJjNDRhN2U5ZWY0M2E5YjFlOGUyMWJkMjYxZDNjZTcxOTU4NzIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXN1bHRDYXJyaWVzUmVzdWx0VHlwZUFuZFNlcnZlckluZm8JOGQ0YmVlZmI3YWQwMzFmYzAxMzc3Y2M0MmEyZDA1MmNmMTgwMGVmZmYzOGMwYTU0M2IzMDhkMWU5ZjViNzMzMwpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVyblRvb2xzTGlzdENhcnJpZXNDYWNoaW5nSGludHMJYzU0NWVjOWM3Nzk2NjM5ZTIxNjY1YzJiZTM1YjVmNDAxOGIxOTk5NTI4NGU4YmY0MTFkOGVkMDEwODlhNWVhNQpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVybldyaXRlUmVjZWlwdElzQnVkZ2V0ZWRXaXRoSXRzRGVjb3JhdGlvbglmNjJiZmEzZTFjMDJhNjg2ZjQwMGIzY2JkZDc2Mzg1NzA5OGMyMDRlYWMyMWQ4MmMzNzdkMGMxYjA2NzkzOGE4CmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RTZXJ2ZXJEaXNjb3Zlck5hbWVzRXZlcnlTdXBwb3J0ZWRWZXJzaW9uCWE5ZmY2OTAzMGU1NWU2ZGMwYThjYzhkMGZjMTFmMTMxYTYzMTc4OWEwMzBkMGE1NzlmNGEyMWYwMTk1Yzc0YWEKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdFRoZUVyYVByZWNlZGVuY2VJc0ZpeGVkCTU2NjlhMmIyYzgyM2ViZDAxYmZiN2RmOTBmMWZhODljYjk4NGUyNDlmNjdkYTMzOWNkMDdiMjAyZDRhYjJmZmQKYm9keQlpbnRlcm5hbC9tY3AvZXhhbXBsZXNfdGVzdC5nbwlUZXN0VGhlU2hpcHBlZFJlYWRFeGFtcGxlU2VydmVzRXZlcnlTcGVjCTdlNzRiODQ5NTJiNDZhODNmOGQ2ZDE0ODNmNzE0ZDE4MjgyZTM5NTEwZWMyOGJlMjdjOTBkYjU0OTY2ZWU1MDg
  ```
  --- last 10 line(s) of stdout (of 53 after folding 53 raw)
          @@ 11-11
          -- ck 25c14a57b6d314c1 open lines 11-11 (1 lines follow)
             11| }
          -- ck 25c14a57b6d314c1 close
          -- This serve licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
  --- FAIL: TestTheShippedReadExampleServesEverySpec (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.109s
  FAIL
  ```
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:828
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:397
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:382
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:335
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:373
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · ms:356
- 2026-09-28 · add6807* · exit 1 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:927
  ```
  --- last 10 line(s) of stdout (of 65 after folding 65 raw)
          @@ 11-11
          -- ck 758afd6f5601f665 open lines 11-11 (1 lines follow)
             11| }
          -- ck 758afd6f5601f665 close
          -- This serve licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
  --- FAIL: TestTheShippedReadExampleServesEverySpec (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.184s
  FAIL
  ```
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:742
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:348
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:373
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:398
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:346
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:309
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:306
- 2026-09-28 · add6807* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:0 · test-lock-sha256:195a7da52dfd2699cfde015e9139407188ad95e75f39dfec017a174bcbf31c68 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9ib2R5aGVhZGVyX3Rlc3QuZ28JVGVzdFRoZUhhbmRzaGFrZVNob3dzQm9keU9uQUhlYWRlcgk4NGUyNDgxMGFjZWJhOTYwZDA0Yjg3MGIzYWM0ZjUzMWUyNDVmYTZhMzZjMmEzMDdiODMwOTVkYzVmNDI3MmEzCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEFSZWFkUmVzdWx0Q2Fycmllc05vU3RydWN0dXJlZENvbnRlbnQJNWI0MTUwZTVjMmUwNTlkYzcxMTM0NDg2ZjJhOWQ2MGJiYTRhZGI1M2FjMTBhOWMxYmY0MWM5OTQ2ZGUzNTliOApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeURlY2xhcmVkT3V0cHV0U2NoZW1hVmFsaWRhdGVzQVJlYWxSZXNwb25zZQljNjBlYTkyNGU2MWIwM2JhYzUxMDUyNjA2ZjhiODJkNmMyMmJmN2Y2OTE0MmQyMmQ1ZDE0NzM3YTA0ODJlZWY5CmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdEV2ZXJ5RW1iZWRkZWRFeGFtcGxlUGxhblJlYWxseUFwcGxpZXMJOWFhZmE3MjE3MmI2Y2E1YmM1MjkwNzg2YTJhMzZlOWE5NDlkNGVjYjIzNGIwMTZhYWM5MWI0YTRjNTM0OGFlNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeU91dHB1dFNjaGVtYVByb3BlcnR5SXNEZXNjcmliZWQJNjlmYjBlOGQ4MjNiOWEyZTFkMWIzOTEyMDYxMDcwMDBmNjNhY2YxNDM1ODI4OTMyYzE5YzQyNzZmYmM3N2MzNApib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RUaGVBbm5vdGF0aW9uc01hdGNoV2hhdFRoZVRvb2xEb2VzCTU1ZWNlMTZjMDhhNjEwMGM1MTlmZjFjZDY1MWY1ZmQxYzk3NjAzM2Y4M2U4OTJhOGM3ZDU0M2JhNWY4Njg4MGEKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0VGhlRmlyc3RDb250ZW50QmxvY2tJc1RoZVNlcmlhbGl6ZWRTdHJ1Y3R1cmVkQ29udGVudAlhNTliOGVmMDUxMWJkMTI1MjRlM2QwMzE2YjYxZWUwMjc1N2U4Mzg3YTgyYzU4MjI0OWQzMzE1N2QxNjY4ZThjCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdFRoZVN0YXR1c0Rlc2NyaXB0aW9uTmFtZXNUaGVWYWx1ZXNUaGVFbmdpbmVTZW5kcwllNTZiZDIzZWExZTc4ZjFhZTFlZTI3MjUxM2FiODU5NTE3ZTU3Y2MxMzkxZmY4NWY0MzFmZjUwNmZjNWE5Zjg4CmJvZHkJaW50ZXJuYWwvbWNwL2V4YW1wbGVzX3Rlc3QuZ28JVGVzdFRoZURlc2NyaXB0aW9uV2Fsa05hbWVzSXRzU2VudGluZWxzCTMxYmVkNjMyNjljNzQ5OWI4MGViYWY2NTA4YWZiZGRhM2Y5ZGMwMzU5ZTMwY2M1MzJlOGZlMmVjNGMwYTliMTgKYm9keQlpbnRlcm5hbC9tY3AvZXhhbXBsZXNfdGVzdC5nbwlUZXN0VGhlU2hpcHBlZFJlYWRFeGFtcGxlU2VydmVzRXZlcnlTcGVjCTdlNzRiODQ5NTJiNDZhODNmOGQ2ZDE0ODNmNzE0ZDE4MjgyZTM5NTEwZWMyOGJlMjdjOTBkYjU0OTY2ZWU1MDg · test-lock-kind:replace
- 2026-09-28 · human-observed · Claude (session 614518c9) observed 2026-09-28: the relock follows the fence dropping TestALegacyResultIsUnchangedByTheModernPath, which changed with every example mutant and so killed them for the wrong reason; the golden still runs in go test ./..., and the locked tests' bodies are unchanged. Approved as a narrowing of what the fence selects, not a weakening of any test.
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:802
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · ms:510

## Mutation Log
(empty until execute)
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the anchor names text the fixture does not hold · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:the plan's guards meet real code
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the pattern matches no line · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:a pattern address resolves exactly once
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the pattern matches two lines · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:a pattern address resolves exactly once
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · both hunks address one file, so the plan is no longer multi-site · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:the plan is a multi-site plan
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · a read spec matches nothing on the tree · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:every read spec is served
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the receiver parens unescaped, as shipped before ADR-090 · acceptance-sha256:c081948e000abc0527b78c0cd4f0ee8e4e4fdf07132b78a6883bd85c586bbb52 · covers:every read spec is served
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the anchor names text the fixture does not hold · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:the plan's guards meet real code
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the pattern matches no line · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:a pattern address resolves exactly once
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the pattern matches two lines · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:a pattern address resolves exactly once
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · both hunks address one file · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:the plan is a multi-site plan
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the pattern hunk goes back to a line number, which still applies · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:the plan shows both address forms
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · a read spec matches nothing on the tree · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:every read spec is served
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the receiver parens unescaped, as shipped before ADR-090 · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:every read spec is served
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/testdata/example/cmd/app/main.go` · the fixture is not gofmt-clean · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:the tree is gofmt-clean
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:23c05b82ad686b0ff172e81b699e82007f7bd708d58420b293a0822e3f263c56 · covers:no engine package changes

## Invariants

- The handshake stays within 4,096 bytes (ADR-037).
- `examplePlan` keeps `body=4` on its first header (ADR-070).

## Risks

- None beyond the record's.

## Out of Scope

- The anchor guard of a CLI example (permanent: boundary: `internal/guide` examples are proved by that package's tests)

## Stop Condition

The fence exits 0.
