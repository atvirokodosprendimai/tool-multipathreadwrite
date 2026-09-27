# Task ADR-088-T1: production holds nothing only a test reaches

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `check.Shell` with three results; `readSchema`, `checkSignals`, `maxInstructionsChars` in test files; `TestTheReadReceiptMatchesItsSchema`
**Consumes:** nothing
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `deadcode reports nothing`, `U1000 without tests reports nothing`, `the read receipt matches its schema`, `the Windows build still vets`

## Goal

Five production identifiers were reachable only from tests (`deadcode ./...`, `staticcheck -tests=false -checks U1000 ./...` at `d352ba7`).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/shell.go`, `internal/check/check.go` | edit | `Shell()` is the three-result function `Run` calls; `checkSignals` goes |
| `internal/check/group_unix_test.go`, `internal/check/shell082_test.go`, `cmd/mrw/shell082_test.go`, `internal/adversarial/shell082_test.go` | edit | the helpers the tests use |
| `internal/mcp/schema.go`, `internal/mcp/schema_test.go` | edit | `readSchema` and `readDescriptions` move to the test file |
| `internal/mcp/instructions.go`, `internal/mcp/instructions_test.go` | edit, new | `maxInstructionsChars` moves |
| `internal/mcp/tools.go` | edit | comments that called `readSchema` production |
| `docs/adr/ADR-019-*/tasks/T3-*.md`, `docs/adr/ADR-039-*/tasks/T3-*.md` | edit | their fences grep the moved literal |

## Ordered Steps

1. [S1] Write `TestTheReadReceiptMatchesItsSchema`; the fence fails on `d352ba7`, where deadcode reports three functions. [proof: mutation]
2. [S2] Move or fold the five identifiers. [proof: mutation]
3. [S3] Re-fence ADR-019 T3 and ADR-039 T3 to `instructions_test.go` and re-verify each. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
dc=$(go run golang.org/x/tools/cmd/deadcode@v0.50.0 ./...) && [ -z "$dc" ] \
  && go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 -tests=false -checks U1000 ./... \
  && go test ./internal/mcp/ -count=1 -timeout 180s -run 'TestTheReadReceiptMatchesItsSchema' -v 2>&1 | tee /tmp/adr088-T1.out \
  && grep -qE '^--- PASS: TestTheReadReceiptMatchesItsSchema \(' /tmp/adr088-T1.out \
  && GOOS=windows go vet ./internal/check/ ./cmd/mrw/ ./internal/adversarial/ \
  && [ -z "$(gofmt -l .)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/lines internal/iter internal/subproc \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | a real `mrw_read` receipt has exactly the keys `readSchema` declares | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `check.Shell` |
| 2 — something selects it | `check.Run` calls `Shell()` |
| 3 — the caller can discover it | not applicable: internal |
| 4 — it is used | every `mrw check` and default write check |

## Verification Log
(empty until execute)
- 2026-09-27 · d352ba7* · exit 1 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:2169 · test-lock-sha256:60cf201a68006ac3f61492b4ffcbb2745b552d0b89133eb701b21c86330a2369 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QUN1c3RvbU1hcnNoYWxsZXJNZWFuc05vUmVxdWlyZWRQcm9taXNlCTIzOTQ4ZDk0ZjlmMTA5ZTlhZmE4YzFiMjkwMzBjNjUzNjM3NWE2OGE2OWFhY2I0OTRiY2ZjMzgzNzViOGY5ZDgKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFEZXNjcmliZWRQcm9wZXJ0eVRoYXROb0xvbmdlckV4aXN0c0lzUmVmdXNlZAk5ZGI4N2VlYzM2MzRkN2I0YWNhYmI0M2E4N2MxNTQ0Zjc2MTcwZmE3NzdlNmMxNzM1YjUxMTEyZTQyZmUyM2QxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBTmlsU2xpY2VJc0FkbWl0dGVkQnlUaGVTY2hlbWEJYzgwOWRiYjQ0NjI1NWJkMDM4ODY5OWM1NGMyNzEzY2I5ZTgwMDNlNmMzZjkzZWM4YmJlMTYwMThkMTVjYmY4OApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QVByb3BlcnR5bGVzc09iamVjdFNjaGVtYUlzUmVmdXNlZAliZmRmOTllMzhlMzM4YjUwNzhmM2QyZDBiOWQzYTczYjBkYmYxNWZhNGVkOGVjZWVkNGI2N2Y4NzQ5MjJjNGYxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVSZWFkUmVjZWlwdE1hdGNoZXNJdHNTY2hlbWEJODkwYTFkMzEzOTlhNmJhMWM4MmMyM2Q2YmI3NjkwZDQ1NzM2N2NkMzhmODlhMDZmOGIzMThiODlmMTBjYTg3Ygpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0VGhlU2NoZW1hTmFtZXNUaGVGaWVsZHNPZlRoZVJlc3VsdAk1YjNhOGNkMTg5MjFhYTRmY2Y4YWFmMmY3NWQwMTIwMDk1YThmOWI4MTJiMTc3NGFlZTJjNjg0ZDIyOWM2NGI2
  ```
  ```
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:7199
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:4924
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:4189
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:4037
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · ms:5674

## Mutation Log
(empty until execute)
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `internal/check/check.go` · a production function nothing calls · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · covers:deadcode reports nothing
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `internal/mcp/instructions.go` · a production const nothing reads · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · covers:U1000 without tests reports nothing
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `internal/mcp/tools.go` · a paged receipt renames a key readSchema declares · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · covers:the read receipt matches its schema
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `cmd/mrw/shell082_test.go` · a caller of Shell left on the old two results · acceptance-sha256:2aa582b7467f1b123d156b1af96ae880345a076565a2be1db03b9454092c43f5 · covers:the Windows build still vets

## Invariants

- Every refusal's text and every exit code is unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- A runtime check of the 4096 bound (permanent: boundary: it would be behaviour that never fires in a shipped binary; the contract asserts the bound on the built binary)

## Stop Condition

The fence exits 0.
