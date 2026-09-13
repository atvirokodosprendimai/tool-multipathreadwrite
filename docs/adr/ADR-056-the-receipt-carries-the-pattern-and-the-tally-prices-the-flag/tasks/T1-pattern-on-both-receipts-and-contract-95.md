# Task ADR-056-T1: `pattern` on both JSON receipts; contract §95

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `authoring.PatternInfo`; `pattern` on `mrw write --json` and `mrw_write`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `pattern always present`, `both transports`, `fires on the third`, `schema describes it`

## Goal

The CLI prints a `pattern:` line; neither JSON receipt carries the fact. Put a `pattern` object `{advisory_writes, window, fires}` on both, on every write, read from the ring after this write joined it. The transport-equality contract row must keep passing, which is why it is both and not one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/authoring/authoring.go` | edit | `PatternInfo` + `PatternOf(root)`. |
| `cmd/mrw/main.go` | edit | `receipt.Pattern`, set after `RecordRecent`. |
| `internal/mcp/tools.go` | edit | `writeReceipt.Pattern`, set after `RecordRecent`. |
| `internal/mcp/schema.go` | edit | Describe `pattern` and its three members; the conformance test refuses an undescribed key. |
| `internal/mcp/pattern_test.go` | create | Red: three delta writes over MCP; first receipt `fires:false`, third `fires:true, advisory_writes:3, window:3`; a refused plan still carries the object. |
| `cmd/mrw/advisory_test.go` | edit | Red: `--json` receipt has `pattern`. |
| `scripts/contract.sh` | edit | **§95** — next free after §94. |

## Ordered Steps

1. [S1] Write `TestTheMCPReceiptCarriesThePattern` and `TestTheCLIJSONReceiptCarriesThePattern` — RED. [proof: mutation]
2. [S2] Implement `PatternInfo`, both fields, four schema rows. S1 GREEN. [proof: mutation]
3. [S3] Write §95 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 95\. ' scripts/contract.sh \
  && go test ./internal/mcp/ ./cmd/mrw/ -count=1 -v \
    -run 'TestTheMCPReceiptCarriesThePattern|TestTheCLIJSONReceiptCarriesThePattern|TestEveryOutputSchemaPropertyIsDescribed' 2>&1 | tee /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestTheMCPReceiptCarriesThePattern' /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestTheCLIJSONReceiptCarriesThePattern' /tmp/adr056-t1.out \
  && grep -q '^--- PASS: TestEveryOutputSchemaPropertyIsDescribed' /tmp/adr056-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr056-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/mcp/ ./cmd/mrw/ ./internal/authoring/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheMCPReceiptCarriesThePattern` | `internal/mcp/pattern_test.go` | present on write 1 with `fires:false`; write 3 `fires:true, 3, 3`; present on a refused plan | — | S1, S2 |
| `TestTheCLIJSONReceiptCarriesThePattern` | `cmd/mrw/advisory_test.go` | `--json` has `pattern.window == 1` after one write and `fires` false | — | S1, S2 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | the four new keys are described | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §95 |
| 2 — something selects it | every write sets it; deleting the assignment fails S1 |
| 3 — the caller can discover it | the output schema describes it; T3 names it |
| 4 — it is used | ADR-055's motivating caller read the JSON and not the row |

## Mutation Log

_(tool-written)_

## Invariants

- `pattern` is present on every receipt, both transports, whether or not it fires.
- The human `pattern:` line is unchanged.
- The existing transport-equality contract row is not edited.

## Risks

| Risk | Mitigation |
|------|------------|
| Ring state differs between the CLI and MCP fixtures in the equality row | both are fresh roots with one landed write: `{1 or 0, 1, false}` on both |

## Stop Condition

Stop if the equality row cannot pass with the field on both sides: that means the two transports record the ring differently, which is a T2-of-ADR-055 bug, not this task's.

## Out of Scope

- Anything about the pricing (T2).

## Verification Log

_(tool-written)_
