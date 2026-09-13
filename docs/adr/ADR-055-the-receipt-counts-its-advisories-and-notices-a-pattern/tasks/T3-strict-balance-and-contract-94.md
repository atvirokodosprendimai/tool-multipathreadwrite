# Task ADR-055-T3: `--strict-balance` opt-in refusal of the wrap-tail signature; contract §94

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `Options.StrictBalance` / `--strict-balance` (T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `signature refused`, `balanced replace applies`, `multi-line untouched`, `prose exempt`, `off by default`, `mcp flag`

## Goal

With `--strict-balance` (MCP `strict_balance`), a hunk is refused when `SrcOp == "replace"`, `Start == End`, the path is not prose, and some family among `{}` `()` `[]` has a non-zero net in the consumed line that the body does not match. The reason names the family and both nets. Refused means failed: siblings skip, nothing written, exit 1 (ADR-001). Off by default; the same plan without the flag applies with the balance row. Multi-line addresses are ADR-052's and untouched. Default stays off — T4 pre-registers the campaign that could change that.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options.StrictBalance`; the check beside the ADR-052 licence at the same resolve site. |
| `internal/apply/apply_test.go` | edit | Red: signature refused with both nets named; balanced single-line applies; multi-line address not touched by this check; `.md` exempt; off by default applies with Balance. |
| `cmd/mrw/main.go` | edit | `--strict-balance` flag into `Options`. The CLI selector. |
| `internal/mcp/tools.go` | edit | `strict_balance` argument into `Options`. The MCP selector (a flag on the existing tool, ADR-044). |
| `internal/mcp/schema.go` | edit | Describe `strict_balance`. |
| `scripts/contract.sh` | edit | **§94** — pair: flag + signature → exit 1, file unchanged, reason names `{ +1` / same plan no flag → exit 0 with `balance` row / flag + balanced body → exit 0 / flag + `.md` → exit 0. |

## Ordered Steps

1. [S1] Write `TestStrictBalanceRefusesTheWrapTailSignature` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestStrictBalanceLeavesBalancedMultiLineAndProseAlone` and `TestStrictBalanceIsOffByDefault` — RED until the option exists. [proof: mutation]
3. [S3] Implement the check and both selectors. Confirm GREEN. Deleting the `Start == End` guard must fail the multi-line half; deleting the net compare must fail S1. [proof: mutation]
4. [S4] Write §94 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 94\. ' scripts/contract.sh \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
    -run 'TestStrictBalanceRefusesTheWrapTailSignature|TestStrictBalanceLeavesBalancedMultiLineAndProseAlone|TestStrictBalanceIsOffByDefault|TestEveryOutputSchemaPropertyIsDescribed' 2>&1 | tee /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceRefusesTheWrapTailSignature' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceLeavesBalancedMultiLineAndProseAlone' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestStrictBalanceIsOffByDefault' /tmp/adr055-t3.out \
  && grep -q '^--- PASS: TestEveryOutputSchemaPropertyIsDescribed' /tmp/adr055-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestStrictBalanceRefusesTheWrapTailSignature`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T3 S1 writes it | — | S1, S3 |
| ~~`TestStrictBalanceLeavesBalancedMultiLineAndProseAlone`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T3 S2 writes it | — | S2, S3 |
| ~~`TestStrictBalanceIsOffByDefault`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T3 S2 writes it | — | S2, S3 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | the MCP input schema describes `strict_balance` | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §94 |
| 2 — something selects it | `--strict-balance` and `strict_balance` set `Options.StrictBalance`; deleting either selector fails its half of §94 |
| 3 — the caller can discover it | T4 on `write --help`; the refusal reason itself |
| 4 — it is used | opt-in; the pre-registered campaign (T4) is how use is measured |

## Mutation Log

(empty until execute)

## Invariants

- ADR-001: a refusal writes nothing; siblings skip; exit 1.
- ADR-048: arithmetic on one hunk; no parser.
- ADR-052: multi-line addresses are the licence's, not this check's.
- ADR-054: prose exempt; the balance row on an unflagged run is unchanged.
- Default off.

## Risks

- Braces in string literals refuse a correct single-line replace when opted in. The reason names both nets; the caller drops the flag for that plan. This is the bargain the flag states.

## Stop Condition

If the only way to go green is to refuse without the flag, or to touch multi-line addresses, stop.

## Out of Scope

- Advisory count (T1), pattern line (T2), teaching (T4)
- A default (BACKLOG pre-registration, T4)

## Verification Log

(empty until execute)
