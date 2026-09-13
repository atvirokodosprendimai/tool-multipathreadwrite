# Task ADR-057-T3: contract §98–§99; teach; BACKLOG

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** §98, §99; AGENTS/README/skill; BACKLOG shipped
**Consumes:** T1, T2
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `native unlink applies`, `sibling fail restores`, `Delete File applies`

## Goal

The binary does what the unit tests say. The docs name the two ops. BACKLOG's deferred Delete File row becomes shipped.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | §98 native; §99 apply_patch. |
| `cmd/mrw/main.go` | edit | Ledger Drop; receipt `removed`. |
| `cmd/mrw/unlink_receipt_test.go` | create | Red: receipt names `removed`; ledger drops. |
| `internal/mcp/tools.go` | edit | Ledger Drop. |
| `internal/mcp/schema.go` | edit | `files.removed`, `files.renamed_to`. |
| `AGENTS.md` | edit | Ops list. |
| `README.md` | edit | Ops list. |
| `docs/adr/BACKLOG.md` | edit | Shipped. |
| `.claude/skills/mrw/SKILL.md` | edit | If it enumerates ops. |

## Ordered Steps

1. [S1] Write a red test that the human receipt names `removed` for an unlink, then wire ledger Drop, the receipt verb, and schema. [proof: mutation]
2. [S2] §98 and §99 RED then GREEN. [proof: mutation]
3. [S3] Teach and BACKLOG. [proof: acceptance]
4. [S4] `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 98\. ' scripts/contract.sh \
  && grep -q '^# 99\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ ./internal/mcp/ -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestUnlinkReceiptNamesRemoved` | `cmd/mrw/unlink_receipt_test.go` | human receipt says `removed`; ledger drops the path | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §98 §99 |
| 2 — something selects it | `$MRW write` |
| 3 — the caller can discover it | AGENTS.md |
| 4 — it is used | Zeus `rm` is the caller; ADR-009 refuses telemetry |

## Mutation Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `cmd/mrw/main.go` · receipt verb stays wrote, so TestUnlinkReceiptNamesRemoved must go red · acceptance-sha256:7b595d48064a4598c01aa9a11778e5b30b3757f97c4df37ab188654b61b4117a · covers:native unlink applies
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `cmd/mrw/main.go` · Drop is skipped, so TestUnlinkReceiptNamesRemoved must go red · acceptance-sha256:7b595d48064a4598c01aa9a11778e5b30b3757f97c4df37ab188654b61b4117a · covers:native unlink applies

## Invariants

- Exit codes unchanged.
- MCP still does not run `--check` (ADR-044).

## Risks

| Risk | Mitigation |
|------|------------|
| Schema test fails on new JSON keys | describe them in S1 before the fence |

## Stop Condition

Stop if §98 cannot restore on sibling fail: that is T1's defect.

## Out of Scope

- Per-extension check skip (ADR-059).

## Verification Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · exit 1 · `set -o pipefail …` · acceptance-sha256:7b595d48064a4598c01aa9a11778e5b30b3757f97c4df37ab188654b61b4117a · ms:50 · test-lock-sha256:7120e34336cf096e7152d8b9452da39d3d40a9a6fde18b740b8b272bcb5b0b44 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdW5saW5rX3JlY2VpcHRfdGVzdC5nbwlUZXN0VW5saW5rUmVjZWlwdE5hbWVzUmVtb3ZlZAk4YmQyMjJkZjMzNzhmZmJlMmYyNDgzNDFjMmJhYTg5NmVlMzk1Zjc2Mjg5MjkyMWY0YWViMDdkNWIwNWJlMGJh
  ```
  ```
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b595d48064a4598c01aa9a11778e5b30b3757f97c4df37ab188654b61b4117a · ms:17151
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b595d48064a4598c01aa9a11778e5b30b3757f97c4df37ab188654b61b4117a · ms:8808
