# Task ADR-090-T2: the description walk names its sentinels

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestTheDescriptionWalkNamesItsSentinels`; §44's `files.written` line
**Consumes:** `describedPaths`, `readSchema()` (test helpers), `tools()`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `each sentinel is required by exact path`, `the tree is gofmt-clean`, `no engine package changes`

## Goal

Losing a whole container of described fields fails a test, not only the floor of twenty.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/examples_test.go` | edit | `TestTheDescriptionWalkNamesItsSentinels` |
| `scripts/contract.sh` | edit | §44 asserts the walk reaches `mrw_write:files.written` |

## Ordered Steps

1. [S1] `TestTheDescriptionWalkNamesItsSentinels` requires `mrw_write:hunks.status`, `mrw_write:files.written`, `mrw_write:pattern.fires` and `mrw_read receipt:observed.Spans` by exact path. Every one exists today, so red is shown by mutants that drop each. [proof: mutation]
2. [S2] §44 names `files.written`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestTheDescriptionWalkNamesItsSentinels|TestEveryOutputSchemaPropertyIsDescribed' -v 2>&1 | tee /tmp/adr090-T2.out \
  && missing=$(for t in TestTheDescriptionWalkNamesItsSentinels TestEveryOutputSchemaPropertyIsDescribed; do grep -qE "^--- PASS: $t \(" /tmp/adr090-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q 'mrw_write:files.written' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only add6807 -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheDescriptionWalkNamesItsSentinels` | `internal/mcp/examples_test.go` | four exact paths are walked and described | — | S1 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | unchanged; still passes | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four properties in `schema.go` and `readSchema()` |
| 2 — something selects it | `tools()` declares `mrw_write`'s outputSchema |
| 3 — the caller can discover it | `tools/list`, walked by §44 on the built binary |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is ADR-012 follow-up (b), a Codex finding |

## Verification Log
(empty until execute)
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:637
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:308
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:322
- 2026-09-28 · add6807* · exit 1 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:946 · test-lock-sha256:5c9cc6ad4fc790e981279ad1a2570233dc9d60a10268fb2dcf4be581fef795c4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RBUmVhZFJlc3VsdENhcnJpZXNOb1N0cnVjdHVyZWRDb250ZW50CTViNDE1MGU1YzJlMDU5ZGM3MTEzNDQ4NmYyYTlkNjBiYmE0YWRiNTNhYzEwYTljMWJmNDFjOTk0NmRlMzU5YjgKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0RXZlcnlEZWNsYXJlZE91dHB1dFNjaGVtYVZhbGlkYXRlc0FSZWFsUmVzcG9uc2UJYzYwZWE5MjRlNjFiMDNiYWM1MTA1MjYwNmY4YjgyZDZjMjJiZjdmNjkxNDJkMjJkNWQxNDczN2EwNDgyZWVmOQpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RFdmVyeUVtYmVkZGVkRXhhbXBsZVBsYW5SZWFsbHlBcHBsaWVzCTlhYWZhNzIxNzJiNmNhNWJjNTI5MDc4NmEyYTM2ZTlhOTQ5ZDRlY2IyMzRiMDE2YWFjOTFiNGE0YzUzNDhhZTQKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0RXZlcnlPdXRwdXRTY2hlbWFQcm9wZXJ0eUlzRGVzY3JpYmVkCTY5ZmIwZThkODIzYjlhMmUxZDFiMzkxMjA2MTA3MDAwZjYzYWNmMTQzNTgyODkzMmMxOWM0Mjc2ZmJjNzdjMzQKYm9keQlpbnRlcm5hbC9tY3AvY29uZm9ybWFuY2VfdGVzdC5nbwlUZXN0VGhlQW5ub3RhdGlvbnNNYXRjaFdoYXRUaGVUb29sRG9lcwk1NWVjZTE2YzA4YTYxMDBjNTE5ZmYxY2Q2NTFmNWZkMWM5NzYwMzNmODNlODkyYThjN2Q1NDNiYTVmODY4ODBhCmJvZHkJaW50ZXJuYWwvbWNwL2NvbmZvcm1hbmNlX3Rlc3QuZ28JVGVzdFRoZUZpcnN0Q29udGVudEJsb2NrSXNUaGVTZXJpYWxpemVkU3RydWN0dXJlZENvbnRlbnQJYTU5YjhlZjA1MTFiZDEyNTI0ZTNkMDMxNmI2MWVlMDI3NTdlODM4N2E4MmM1ODIyNDlkMzMxNTdkMTY2OGU4Ywpib2R5CWludGVybmFsL21jcC9jb25mb3JtYW5jZV90ZXN0LmdvCVRlc3RUaGVTdGF0dXNEZXNjcmlwdGlvbk5hbWVzVGhlVmFsdWVzVGhlRW5naW5lU2VuZHMJZTU2YmQyM2VhMWU3OGYxYWUxZWUyNzI1MTNhYjg1OTUxN2U1N2NjMTM5MWZmODVmNDMxZmY1MDZmYzVhOWY4OApib2R5CWludGVybmFsL21jcC9leGFtcGxlc190ZXN0LmdvCVRlc3RUaGVEZXNjcmlwdGlvbldhbGtOYW1lc0l0c1NlbnRpbmVscwkzMWJlZDYzMjY5Yzc0OTliODBlYmFmNjUwOGFmYmRkYTNmOWRjMDM1OWUzMGNjNTMyZThmZTJlYzRjMGE5YjE4CmJvZHkJaW50ZXJuYWwvbWNwL2V4YW1wbGVzX3Rlc3QuZ28JVGVzdFRoZVNoaXBwZWRSZWFkRXhhbXBsZVNlcnZlc0V2ZXJ5U3BlYwk3ZTc0Yjg0OTUyYjQ2YTgzZjhkNmQxNDgzZjcxNGQxODI4MmUzOTUxMGVjMjhiZTI3YzkwZGI1NDk2NmVlNTA4
  ```
  --- last 6 line(s) of stdout
  === RUN   TestEveryOutputSchemaPropertyIsDescribed
  --- PASS: TestEveryOutputSchemaPropertyIsDescribed (0.00s)
  === RUN   TestTheDescriptionWalkNamesItsSentinels
  --- PASS: TestTheDescriptionWalkNamesItsSentinels (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.377s
  ```
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:387
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:301
- 2026-09-28 · add6807* · exit 0 · `set -o pipefail …` · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · ms:596

## Mutation Log
(empty until execute)
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/schema.go` · the pattern container is dropped from the schema and its descriptions together · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · covers:each sentinel is required by exact path
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/schema.go` · a file's written flag is dropped from the schema and its description together · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · covers:each sentinel is required by exact path
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/schema.go` · the hunks container is dropped from the schema and its descriptions together · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · covers:each sentinel is required by exact path
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/mcp/testdata/example/cmd/app/main.go` · the fixture is not gofmt-clean · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · covers:the tree is gofmt-clean
- 2026-09-28 · add6807* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:816f7fcf1544f45b663a55f1622679e4ce6f0a72e698e60d6e705c17341ecc9e · covers:no engine package changes

## Invariants

- `TestEveryOutputSchemaPropertyIsDescribed`'s body is unchanged; three task locks hash it.

## Risks

- None beyond the record's.

## Out of Scope

- Counting per container (permanent: boundary: ADR-012 offered it as the alternative to exact sentinels; one is enough)

## Stop Condition

The fence exits 0.
