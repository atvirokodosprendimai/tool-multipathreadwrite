# Task ADR-091-T1: the receipt's keys are slash-spelled

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `slashKeys`; `TestAReceiptKeyIsSpelledWithSlashes`
**Consumes:** `read.Run`'s observation map
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a backslashed key is slash-spelled`, `the value is carried unchanged`, `the receipt carries the converted map`, `the tree is gofmt-clean`, `no engine package changes`

## Goal

A Windows caller finds a served file in `observed` under the path it would write in a plan.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/wirepath.go` | new | `slashKeys` |
| `internal/mcp/wirepath_test.go` | new | drives it with `\` on any platform, and a real read's receipt |
| `internal/mcp/tools.go` | edit | the three receipt builders pass `observed` through `slashKeys` |
| `internal/mcp/schema_test.go` | edit | `observed`'s description names the spelling |
| `internal/mcp/examples_test.go` | edit | ADR-090's read test compares keys as they come |

## Ordered Steps

1. [S1] `TestAReceiptKeyIsSpelledWithSlashes`; red before `slashKeys` exists. [proof: mutation]
2. [S2] `slashKeys`, called by the three builders. [proof: mutation]
3. [S3] The description, and ADR-090's read test without its ToSlash, so windows-shard 1 proves the wiring. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAReceiptKeyIsSpelledWithSlashes|TestTheShippedReadExampleServesEverySpec|TestTheReadReceiptMatchesItsSchema' -v 2>&1 | tee /tmp/adr091-T1.out \
  && missing=$(for t in TestAReceiptKeyIsSpelledWithSlashes TestTheShippedReadExampleServesEverySpec TestTheReadReceiptMatchesItsSchema; do grep -qE "^--- PASS: $t \(" /tmp/adr091-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 194f588 -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReceiptKeyIsSpelledWithSlashes` | `internal/mcp/wirepath_test.go` | `slashKeys` turns `\` keys into `/` with the value unchanged, and a served read's receipt carries slash keys | — | S1, S2 |
| `TestTheShippedReadExampleServesEverySpec` | `internal/mcp/examples_test.go` | the receipt's keys, as they come, name the paths the specs asked for | — | S3 |
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | the receipt still matches its described schema | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `slashKeys` |
| 2 — something selects it | the three receipt builders in `tools.go` |
| 3 — the caller can discover it | `observed`'s description |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is windows-shard 1 on #261 and the Codex review of #261 |

## Verification Log
(empty until execute)
- 2026-09-28 · 194f588* · exit 1 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:190 · test-lock-sha256:d505af59b6f9ceed2a55f709530139d7bc5bf7740eccc02beaa3d3e9c1836880 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9leGFtcGxlc190ZXN0LmdvCVRlc3RUaGVEZXNjcmlwdGlvbldhbGtOYW1lc0l0c1NlbnRpbmVscwkzMWJlZDYzMjY5Yzc0OTliODBlYmFmNjUwOGFmYmRkYTNmOWRjMDM1OWUzMGNjNTMyZThmZTJlYzRjMGE5YjE4CmJvZHkJaW50ZXJuYWwvbWNwL2V4YW1wbGVzX3Rlc3QuZ28JVGVzdFRoZVNoaXBwZWRSZWFkRXhhbXBsZVNlcnZlc0V2ZXJ5U3BlYwk4MDAxMGVhNzBhNjE3NzA1NjExODAxOTdkNDAyNDRkOTEwMTk3MzM3YzMyMmM4NjAyMjE4ZDIzN2ExMzU2NTEyCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBQ3VzdG9tTWFyc2hhbGxlck1lYW5zTm9SZXF1aXJlZFByb21pc2UJMjM5NDhkOTRmOWYxMDllOWFmYThjMWIyOTAzMGM2NTM2Mzc1YTY4YTY5YWFjYjQ5NGJjZmMzODM3NWI4ZjlkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QURlc2NyaWJlZFByb3BlcnR5VGhhdE5vTG9uZ2VyRXhpc3RzSXNSZWZ1c2VkCTlkYjg3ZWVjMzYzNGQ3YjRhY2FiYjQzYTg3YzE1NDRmNzYxNzBmYTc3N2U2YzE3MzViNTExMTJlNDJmZTIzZDEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFOaWxTbGljZUlzQWRtaXR0ZWRCeVRoZVNjaGVtYQljODA5ZGJiNDQ2MjU1YmQwMzg4Njk5YzU0YzI3MTNjYjllODAwM2U2YzNmOTNlYzhiYmUxNjAxOGQxNWNiZjg4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBUHJvcGVydHlsZXNzT2JqZWN0U2NoZW1hSXNSZWZ1c2VkCWJmZGY5OWUzOGUzMzhiNTA3OGYzZDJkMGI5ZDNhNzNiMGRiZjE1ZmE0ZWQ4ZWNlZWQ0YjY3Zjg3NDkyMmM0ZjEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdFRoZVJlYWRSZWNlaXB0TWF0Y2hlc0l0c1NjaGVtYQk4OTBhMWQzMTM5OWE2YmExYzgyYzIzZDZiYjc2OTBkNDU3MzY3Y2QzOGY4OWEwNmY4YjMxOGI4OWYxMGNhODdiCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVTY2hlbWFOYW1lc1RoZUZpZWxkc09mVGhlUmVzdWx0CTViM2E4Y2QxODkyMWFhNGZjZjhhYWYyZjc1ZDAxMjAwOTVhOGY5YjgxMmIxNzc0YWVlMmM2ODRkMjI5YzY0YjYKYm9keQlpbnRlcm5hbC9tY3Avd2lyZXBhdGhfdGVzdC5nbwlUZXN0QVJlY2VpcHRLZXlJc1NwZWxsZWRXaXRoU2xhc2hlcwliOTFlMTJhZDBkMTMyNjE3Y2Y0M2Q3M2E5MGU5MzY0OTgzMzZjYzAxYjUwYTVhMGZiZDY4YTRlZjAyZWQ2NWY4
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/wirepath_test.go:22:9: undefined: slashKeys
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:1201
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:436
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:546
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:534
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:647
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:813
- 2026-09-28 · 194f588* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:0 · test-lock-sha256:44bedb4727f03671aa296a420b884a8b860b9a08a7ca7dc83b28bba7ae2e1bdc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9leGFtcGxlc190ZXN0LmdvCVRlc3RUaGVEZXNjcmlwdGlvbldhbGtOYW1lc0l0c1NlbnRpbmVscwkzMWJlZDYzMjY5Yzc0OTliODBlYmFmNjUwOGFmYmRkYTNmOWRjMDM1OWUzMGNjNTMyZThmZTJlYzRjMGE5YjE4CmJvZHkJaW50ZXJuYWwvbWNwL2V4YW1wbGVzX3Rlc3QuZ28JVGVzdFRoZVNoaXBwZWRSZWFkRXhhbXBsZVNlcnZlc0V2ZXJ5U3BlYwlhZTAwNzhkMWVjMzZjZWNhYzdkNmVmOGUzNWQ4NGE0ZmY4YjhjYTkwN2YxYzUwN2U5N2Y0YzRmOGYwODdmMjBiCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBQ3VzdG9tTWFyc2hhbGxlck1lYW5zTm9SZXF1aXJlZFByb21pc2UJMjM5NDhkOTRmOWYxMDllOWFmYThjMWIyOTAzMGM2NTM2Mzc1YTY4YTY5YWFjYjQ5NGJjZmMzODM3NWI4ZjlkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QURlc2NyaWJlZFByb3BlcnR5VGhhdE5vTG9uZ2VyRXhpc3RzSXNSZWZ1c2VkCTlkYjg3ZWVjMzYzNGQ3YjRhY2FiYjQzYTg3YzE1NDRmNzYxNzBmYTc3N2U2YzE3MzViNTExMTJlNDJmZTIzZDEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFOaWxTbGljZUlzQWRtaXR0ZWRCeVRoZVNjaGVtYQljODA5ZGJiNDQ2MjU1YmQwMzg4Njk5YzU0YzI3MTNjYjllODAwM2U2YzNmOTNlYzhiYmUxNjAxOGQxNWNiZjg4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBUHJvcGVydHlsZXNzT2JqZWN0U2NoZW1hSXNSZWZ1c2VkCWJmZGY5OWUzOGUzMzhiNTA3OGYzZDJkMGI5ZDNhNzNiMGRiZjE1ZmE0ZWQ4ZWNlZWQ0YjY3Zjg3NDkyMmM0ZjEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdFRoZVJlYWRSZWNlaXB0TWF0Y2hlc0l0c1NjaGVtYQk4OTBhMWQzMTM5OWE2YmExYzgyYzIzZDZiYjc2OTBkNDU3MzY3Y2QzOGY4OWEwNmY4YjMxOGI4OWYxMGNhODdiCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVTY2hlbWFOYW1lc1RoZUZpZWxkc09mVGhlUmVzdWx0CTViM2E4Y2QxODkyMWFhNGZjZjhhYWYyZjc1ZDAxMjAwOTVhOGY5YjgxMmIxNzc0YWVlMmM2ODRkMjI5YzY0YjYKYm9keQlpbnRlcm5hbC9tY3Avd2lyZXBhdGhfdGVzdC5nbwlUZXN0QVJlY2VpcHRLZXlJc1NwZWxsZWRXaXRoU2xhc2hlcwliOTFlMTJhZDBkMTMyNjE3Y2Y0M2Q3M2E5MGU5MzY0OTgzMzZjYzAxYjUwYTVhMGZiZDY4YTRlZjAyZWQ2NWY4 · test-lock-kind:replace
- 2026-09-28 · human-observed · Claude (session 614518c9) observed 2026-09-28: ADR-091 S3 made TestTheShippedReadExampleServesEverySpec compare the receipt's observed keys as they come instead of through filepath.ToSlash, because ADR-091 now spells them with / on every platform; the checked assertion stays. Approved: stricter — on Windows it now also proves the receipt builders call slashKeys.
- 2026-09-28 · 194f588* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · ms:476

## Mutation Log
(empty until execute)
- 2026-09-28 · 194f588* · mutant inconclusive · exit 1 · `internal/mcp/wirepath.go` · keys pass through unconverted · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:a backslashed key is slash-spelled
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-28 · 194f588* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · the spans are dropped in conversion · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:the value is carried unchanged
- 2026-09-28 · 194f588* · mutant killed · exit 1 · `internal/mcp/tools.go` · the served receipt stops carrying the converted map · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:the receipt carries the converted map
- 2026-09-28 · 194f588* · mutant killed · exit 1 · `internal/mcp/testdata/example/cmd/app/main.go` · the tree is not gofmt-clean · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:the tree is gofmt-clean
- 2026-09-28 · 194f588* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:no engine package changes
- 2026-09-28 · 194f588* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · the separator is never replaced, so keys pass through unconverted · acceptance-sha256:0ab44bb957f93f5763dfb281db871872c61a3d47586b07273ace013ab1720e27 · covers:a backslashed key is slash-spelled

## Invariants

- The ledger, the ack store and `read.Run` keep their OS keys.

## Risks

- None beyond the record's.

## Out of Scope

- The wiring mutant on a non-Windows host (permanent: boundary: `filepath.Separator` is `/` there, so dropping the call changes nothing locally; windows-shard 1 runs the read test that proves it)

## Stop Condition

The fence exits 0.
