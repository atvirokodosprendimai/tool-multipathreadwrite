# Task ADR-111-T2: MCP's receipt keys are listed and held

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** MCP receipts held to `docs/receipts.txt`
**Consumes:** `docs/receipts.txt` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `MCP's receipt keys are listed and held`

## Goal

MCP's receipts — `mrw_write`'s structured value and the receipt `mrw_read` carries in `content[1]` — are held to `docs/receipts.txt` as the CLI's are. `mrw_write`'s is reflected from `writeReceipt`. `mrw_read`'s has variants — served, paged, no match, index — built as maps, so its keys are taken from `readSchema`, which `TestTheReadReceiptMatchesItsSchema` holds to a real answer of each variant both ways; the fence runs both tests (the Codex review of #308).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema_test.go` | read | `readSchema`, the declared union of `mrw_read`'s receipt keys, and the test that holds it to every variant |
| `internal/mcp/receipts111_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestNoShippedMCPReceiptFieldDisappears`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: the write receipt's `elided` key renamed; the paged receipt's `next_read` key renamed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestNoShippedMCPReceiptFieldDisappears|TestTheReadReceiptMatchesItsSchema' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestTheReadReceiptMatchesItsSchema \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoShippedMCPReceiptFieldDisappears` | `internal/mcp/receipts111_test.go` | every path of `writeReceipt`, and every receipt key and observation key `readSchema` declares, is in `docs/receipts.txt`, and every listed path of those receipts is still declared | — | S1, S2 |
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | (existing) every key a served, paged, grep, index or index-page answer carries is declared by `readSchema`, and every declared key is carried by one of them | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every receipt the CLI or MCP prints |
| 3 — the caller can discover it | `docs/receipts.txt` names every key a caller may rely on |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B4) |

## Mutation Log
- 2026-10-01 · 5ad968f* · mutant killed · exit 1 · `internal/mcp/tools.go` · the mcp_write elided key renamed · acceptance-sha256:1e92c441533c4f4a640ddde60ac6d4958283ae5eb4675330d5dad5f2661fb855
- 2026-10-01 · 5ad968f* · mutant killed · exit 1 · `internal/mcp/tools.go` · the mcp_read problems key renamed · acceptance-sha256:1e92c441533c4f4a640ddde60ac6d4958283ae5eb4675330d5dad5f2661fb855
- 2026-10-01 · 24c5778* · mutant killed · exit 1 · `internal/mcp/tools.go` · the mcp_write elided key renamed · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a
- 2026-10-01 · 24c5778* · mutant killed · exit 1 · `internal/mcp/tools.go` · the paged receipt key renamed: the schema test sees an undeclared key and an uncarried one · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a

## Invariants

- Every receipt's JSON is byte-for-byte what it was.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-111 task, in its own file.

## Verification Log
- 2026-10-01 · 5ad968f* · exit 1 · `set -o pipefail …` · acceptance-sha256:1e92c441533c4f4a640ddde60ac6d4958283ae5eb4675330d5dad5f2661fb855 · ms:140 · test-lock-sha256:61bc1ee12b590ea98f226636918e499cfc6e51ab41ab8ec23d295b8b041a2833 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlY2VpcHRzMTExX3Rlc3QuZ28JVGVzdE5vU2hpcHBlZE1DUFJlY2VpcHRGaWVsZERpc2FwcGVhcnMJOWQzMjYxMzRiNDFkY2ZiNTNiNDMzODM1ZmI0NzJhM2ExZmVmODQwZTY4YzBiOGUwOTczZGM5MWJmYTgzNTg5ZA
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/receipts111_test.go:17:31: undefined: readReceipt
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-01 · 5ad968f* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e92c441533c4f4a640ddde60ac6d4958283ae5eb4675330d5dad5f2661fb855 · ms:1417
- 2026-10-01 · 5ad968f* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e92c441533c4f4a640ddde60ac6d4958283ae5eb4675330d5dad5f2661fb855 · ms:1210
- 2026-10-01 · 24c5778* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a · ms:0 · test-lock-sha256:0e827b1db93ec24d1f36440bb4ba65a20ea92e79404db09d250d95329aef2f44 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlY2VpcHRzMTExX3Rlc3QuZ28JVGVzdE5vU2hpcHBlZE1DUFJlY2VpcHRGaWVsZERpc2FwcGVhcnMJM2RmYzgwZGE4YmIxY2QwNzI4OTdmZjkyOWI3N2VlNjlkODk5ZGYwN2M3NDZkMjg1YWFhNzg1MTcwZjIxMDc3Ygpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QUN1c3RvbU1hcnNoYWxsZXJNZWFuc05vUmVxdWlyZWRQcm9taXNlCTIzOTQ4ZDk0ZjlmMTA5ZTlhZmE4YzFiMjkwMzBjNjUzNjM3NWE2OGE2OWFhY2I0OTRiY2ZjMzgzNzViOGY5ZDgKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFEZXNjcmliZWRQcm9wZXJ0eVRoYXROb0xvbmdlckV4aXN0c0lzUmVmdXNlZAk5ZGI4N2VlYzM2MzRkN2I0YWNhYmI0M2E4N2MxNTQ0Zjc2MTcwZmE3NzdlNmMxNzM1YjUxMTEyZTQyZmUyM2QxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBTmlsU2xpY2VJc0FkbWl0dGVkQnlUaGVTY2hlbWEJYzgwOWRiYjQ0NjI1NWJkMDM4ODY5OWM1NGMyNzEzY2I5ZTgwMDNlNmMzZjkzZWM4YmJlMTYwMThkMTVjYmY4OApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QVByb3BlcnR5bGVzc09iamVjdFNjaGVtYUlzUmVmdXNlZAliZmRmOTllMzhlMzM4YjUwNzhmM2QyZDBiOWQzYTczYjBkYmYxNWZhNGVkOGVjZWVkNGI2N2Y4NzQ5MjJjNGYxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVSZWFkUmVjZWlwdE1hdGNoZXNJdHNTY2hlbWEJODkwYTFkMzEzOTlhNmJhMWM4MmMyM2Q2YmI3NjkwZDQ1NzM2N2NkMzhmODlhMDZmOGIzMThiODlmMTBjYTg3Ygpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0VGhlU2NoZW1hTmFtZXNUaGVGaWVsZHNPZlRoZVJlc3VsdAk1YjNhOGNkMTg5MjFhYTRmY2Y4YWFmMmY3NWQwMTIwMDk1YThmOWI4MTJiMTc3NGFlZTJjNjg0ZDIyOWM2NGI2 · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red mrw_read's keys come from readSchema, held to every receipt variant by TestTheReadReceiptMatchesItsSchema, which the fence now runs, instead of a readReceipt type that covered only the served variant (the Codex review of #308)
- 2026-10-01 · 24c5778* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a · ms:1333
- 2026-10-01 · 24c5778* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a · ms:1286
- 2026-10-02 · e1c7646* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1c09bf30457eea4c357942d4c403301743a4a619716d0c1bc252c6afa2888d2a · ms:0 · test-lock-sha256:57413d2bc485b46d5c154ce253aa7c676247fc655fefb7925e9ce72a10704ce4 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlY2VpcHRzMTExX3Rlc3QuZ28JVGVzdE5vU2hpcHBlZE1DUFJlY2VpcHRGaWVsZERpc2FwcGVhcnMJNWI2ZmI2OWQwNmRmM2YzODA2MDUzMDk5Y2NjZThkMzQ5Nzg1Mjc3OGQ4MjYwMTY1OTFiMmJhMDg1NGZlNThkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QUN1c3RvbU1hcnNoYWxsZXJNZWFuc05vUmVxdWlyZWRQcm9taXNlCTIzOTQ4ZDk0ZjlmMTA5ZTlhZmE4YzFiMjkwMzBjNjUzNjM3NWE2OGE2OWFhY2I0OTRiY2ZjMzgzNzViOGY5ZDgKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFEZXNjcmliZWRQcm9wZXJ0eVRoYXROb0xvbmdlckV4aXN0c0lzUmVmdXNlZAk5ZGI4N2VlYzM2MzRkN2I0YWNhYmI0M2E4N2MxNTQ0Zjc2MTcwZmE3NzdlNmMxNzM1YjUxMTEyZTQyZmUyM2QxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBTmlsU2xpY2VJc0FkbWl0dGVkQnlUaGVTY2hlbWEJYzgwOWRiYjQ0NjI1NWJkMDM4ODY5OWM1NGMyNzEzY2I5ZTgwMDNlNmMzZjkzZWM4YmJlMTYwMThkMTVjYmY4OApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QVByb3BlcnR5bGVzc09iamVjdFNjaGVtYUlzUmVmdXNlZAliZmRmOTllMzhlMzM4YjUwNzhmM2QyZDBiOWQzYTczYjBkYmYxNWZhNGVkOGVjZWVkNGI2N2Y4NzQ5MjJjNGYxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVSZWFkUmVjZWlwdE1hdGNoZXNJdHNTY2hlbWEJZjM1NzY2MzE4Y2U5NWFkNTJkZjEzYTBmMjk3YmU0NzVlMzNlNTRhYzBlNmFiMWMxOWExZTljMzZiOWM4MjEyNQpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0VGhlU2NoZW1hTmFtZXNUaGVGaWVsZHNPZlRoZVJlc3VsdAk1YjNhOGNkMTg5MjFhYTRmY2Y4YWFmMmY3NWQwMTIwMDk1YThmOWI4MTJiMTc3NGFlZTJjNjg0ZDIyOWM2NGI2 · test-lock-kind:replace
