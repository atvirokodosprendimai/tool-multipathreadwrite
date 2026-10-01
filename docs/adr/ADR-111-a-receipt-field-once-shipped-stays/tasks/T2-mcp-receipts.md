# Task ADR-111-T2: MCP's receipt keys are listed and held

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `readReceipt`
**Consumes:** `docs/receipts.txt` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `MCP's receipt keys are listed and held`

## Goal

MCP's receipts — `mrw_write`'s structured value and the receipt `mrw_read` carries in `content[1]` — are held to `docs/receipts.txt` as the CLI's are; the read receipt becomes a named type with the same JSON.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `readReceipt` |
| `internal/mcp/receipts111_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestNoShippedMCPReceiptFieldDisappears`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: the write receipt's `elided` key renamed; the read receipt's `problems` key renamed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestNoShippedMCPReceiptFieldDisappears' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoShippedMCPReceiptFieldDisappears` | `internal/mcp/receipts111_test.go` | every path of `writeReceipt` and `readReceipt` is in `docs/receipts.txt`, and every listed path of those receipts is still in a type | — | S1, S2 |

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
