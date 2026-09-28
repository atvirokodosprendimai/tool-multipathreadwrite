# Task ADR-091-T2: the write receipt's paths are slash-spelled

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `slashResult`; `TestAWriteReceiptPathIsSpelledWithSlashes`
**Consumes:** `apply.Result`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every write receipt path is slash-spelled`, `the engine's result is not changed`, `the write receipt carries the converted result`, `the tree is gofmt-clean`, `no engine package changes`

## Goal

A Windows caller matches a write receipt's paths to the plan lines that produced them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/wirepath.go` | edit | `slashResult` |
| `internal/mcp/wirepath_test.go` | edit | drives it with `\` on any platform, and a real write's receipt |
| `internal/mcp/tools.go` | edit | `boundedReceipt` converts first |
| `internal/mcp/schema.go` | edit | five path descriptions name the spelling |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerated | those descriptions, in `tools/list` |
| `internal/mcp/era_test.go` | edit | the comment names this regeneration |

## Ordered Steps

1. [S1] `TestAWriteReceiptPathIsSpelledWithSlashes`; red before `slashResult` exists. [proof: mutation]
2. [S2] `slashResult`, called first in `boundedReceipt`. [proof: mutation]
3. [S3] The descriptions and the regenerated golden. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAWriteReceiptPathIsSpelledWithSlashes|TestEveryOutputSchemaPropertyIsDescribed|TestEveryDeclaredOutputSchemaValidatesARealResponse' -v 2>&1 | tee /tmp/adr091-T2.out \
  && missing=$(for t in TestAWriteReceiptPathIsSpelledWithSlashes TestEveryOutputSchemaPropertyIsDescribed TestEveryDeclaredOutputSchemaValidatesARealResponse; do grep -qE "^--- PASS: $t \(" /tmp/adr091-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 194f588 -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteReceiptPathIsSpelledWithSlashes` | `internal/mcp/wirepath_test.go` | `slashResult` turns `\` into `/` in all five fields, leaves `root` and the input untouched, and a real write's receipt names the path its plan did | — | S1, S2 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | the changed descriptions still describe every property | — | S3 |
| `TestEveryDeclaredOutputSchemaValidatesARealResponse` | `internal/mcp/conformance_test.go` | a converted receipt still validates against the declared schema | — | S2 |

`TestALegacyResultIsUnchangedByTheModernPath` holds the regenerated golden and is run by
`go test ./...`, not by this fence, for the reason ADR-090 T1 records: a description change moves
the golden, so in the fence it would kill mutants for the wrong reason.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `slashResult` |
| 2 — something selects it | `boundedReceipt`, the one path an `apply.Result` takes to the wire |
| 3 — the caller can discover it | the five descriptions in `tools/list` |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #262 |

## Verification Log
(empty until execute)
- 2026-09-28 · daff486* · exit 1 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:279 · test-lock-sha256:02f80b00ba5a4eb736ed321accc2277ff7582d96e6b90ae226fb5b68610aead3 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RBUmVhZFJlc3VsdENhcnJpZXNOb1N0cnVjdHVyZWRDb250ZW50CTViNDE1MGU1YzJlMDU5ZGM3MTEzNDQ4NmYyYTlkNjBiYmE0YWRiNTNhYzEwYTljMWJmNDFjOTk0NmRlMzU5YjgKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0RXZlcnlEZWNsYXJlZE91dHB1dFNjaGVtYVZhbGlkYXRlc0FSZWFsUmVzcG9uc2UJYzYwZWE5MjRlNjFiMDNiYWM1MTA1MjYwNmY4YjgyZDZjMjJiZjdmNjkxNDJkMjJkNWQxNDczN2EwNDgyZWVmOQpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeUVtYmVkZGVkRXhhbXBsZVBsYW5SZWFsbHlBcHBsaWVzCTlhYWZhNzIxNzJiNmNhNWJjNTI5MDc4NmEyYTM2ZTlhOTQ5ZDRlY2IyMzRiMDE2YWFjOTFiNGE0YzUzNDhhZTQKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0RXZlcnlPdXRwdXRTY2hlbWFQcm9wZXJ0eUlzRGVzY3JpYmVkCTY5ZmIwZThkODIzYjlhMmUxZDFiMzkxMjA2MTA3MDAwZjYzYWNmMTQzNTgyODkzMmMxOWM0Mjc2ZmJjNzdjMzQKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0VGhlQW5ub3RhdGlvbnNNYXRjaFdoYXRUaGVUb29sRG9lcwk1NWVjZTE2YzA4YTYxMDBjNTE5ZmYxY2Q2NTFmNWZkMWM5NzYwMzNmODNlODkyYThjN2Q1NDNiYTVmODY4ODBhCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdFRoZUZpcnN0Q29udGVudEJsb2NrSXNUaGVTZXJpYWxpemVkU3RydWN0dXJlZENvbnRlbnQJYTU5YjhlZjA1MTFiZDEyNTI0ZTNkMDMxNmI2MWVlMDI3NTdlODM4N2E4MmM1ODIyNDlkMzMxNTdkMTY2OGU4Ywpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RUaGVTdGF0dXNEZXNjcmlwdGlvbk5hbWVzVGhlVmFsdWVzVGhlRW5naW5lU2VuZHMJZTU2YmQyM2VhMWU3OGYxYWUxZWUyNzI1MTNhYjg1OTUxN2U1N2NjMTM5MWZmODVmNDMxZmY1MDZmYzVhOWY4OApib2R5CWludGVybmFsL21jcC93aXJlcGF0aF90ZXN0LmdvCVRlc3RBUmVjZWlwdEtleUlzU3BlbGxlZFdpdGhTbGFzaGVzCWI5MWUxMmFkMGQxMzI2MTdjZjQzZDczYTkwZTkzNjQ5ODMzNmNjMDFiNTBhNWEwZmJkNjhhNGVmMDJlZDY1ZjgKYm9keQlpbnRlcm5hbC9tY3Avd2lyZXBhdGhfdGVzdC5nbwlUZXN0QVdyaXRlUmVjZWlwdFBhdGhJc1NwZWxsZWRXaXRoU2xhc2hlcwkzMDZiYTFkMWVmZjYxODdkZjg3OWE2ZTU4YjY1MmVkNWM4ZmNhODgxYzU2YWI0ODMzMGFhYWUyYWU1NTI1MzVh
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/wirepath_test.go:58:9: undefined: slashResult
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:674
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:339
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:439
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:364
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:426
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:313
- 2026-09-28 · daff486* · exit 0 · `set -o pipefail …` · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · ms:356

## Mutation Log
(empty until execute)
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · a hunk path is not converted · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:every write receipt path is slash-spelled
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · a symlink's target is not converted · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:every write receipt path is slash-spelled
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · a created directory is not converted · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:every write receipt path is slash-spelled
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · the hunks slice is shared, so the engine's result is rewritten · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:the engine's result is not changed
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/tools.go` · the write receipt stops carrying the converted result · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:the write receipt carries the converted result
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/mcp/testdata/example/cmd/app/main.go` · the tree is not gofmt-clean · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:the tree is gofmt-clean
- 2026-09-28 · daff486* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:a971d59f5bd2f814d05c3f4f965fe94db5f659040d4af6078ce13ba2e5b2269f · covers:no engine package changes

## Invariants

- The ledger and the engine's `apply.Result` keep their OS paths; `root` stays an OS path.

## Risks

- None beyond the record's.

## Out of Scope

- The wiring mutant on a non-Windows host (permanent: boundary: `filepath.Separator` is `/` there; windows-shard 1 runs the real-write half of the test that proves it)

## Stop Condition

The fence exits 0.
