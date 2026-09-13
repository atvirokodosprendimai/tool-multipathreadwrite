# Task ADR-056-T2: the tally prices `--strict-balance`; contract §96

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `apply.Result.StrictWouldRefuse`; `authoring.RecordPricing` / `LoadPricing` / `Pricing`; the stats block; `--json` `pricing`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `same predicate as the flag`, `flag-on writes not priced`, `one outcome per would-refuse write`, `counts only`, `stats prints and json carries`

## Goal

Make the pre-registered campaign runnable from mrw's own state: count, per landed flag-off write, whether the flag would have refused it and how the write ended (broke / held / unchecked). Counts only (ADR-009). `stats` prints the block and the bar it is measured against.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | Compute `wrapTailDelta` for ok single-line non-prose replaces when the flag is OFF; `Result.StrictWouldRefuse` (`json:"-"`). |
| `internal/apply/apply_test.go` | edit | Red: signature → 1 and absent from JSON; balanced → 0; flag on → 0; multi-line → 0; prose → 0. |
| `internal/authoring/authoring.go` | edit | `pricing` file; `PricingOutcome`; `RecordPricing`; `LoadPricing`; `Pricing.FalsePositiveRate`. |
| `internal/authoring/authoring_test.go` | edit | Red: counters after a sequence; file is `name N` lines only; garbage reads as zero; rate with no checked refusals. |
| `cmd/mrw/main.go` | edit | Record after the check verdict; stats block; `--json` `pricing`. |
| `internal/mcp/tools.go` | edit | Record `unchecked` on a landed write. |
| `cmd/mrw/planpath_test.go` | edit | Red: `stats` after a `--no-check` delta write, a checked-and-broke delta write, a clean write, and a `--strict-balance` write shows `C=3 R=2 broke 1 held 0 unchecked 1`; `--json` `pricing` matches. |
| `scripts/contract.sh` | edit | **§96**. |

## Ordered Steps

1. [S1] Write the engine test, the authoring test and the stats test — RED, every assertion in place. [proof: mutation]
2. [S2] Implement. S1 GREEN. [proof: mutation]
3. [S3] §96 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 96\. ' scripts/contract.sh \
  && go test ./internal/apply/ ./internal/authoring/ ./cmd/mrw/ -count=1 -v \
    -run 'TestStrictWouldRefuseIsCountedOnlyWhenTheFlagIsOff|TestPricingCountsCandidatesRefusalsAndOutcomes|TestStatsPricesStrictBalance' 2>&1 | tee /tmp/adr056-t2.out \
  && grep -q '^--- PASS: TestStrictWouldRefuseIsCountedOnlyWhenTheFlagIsOff' /tmp/adr056-t2.out \
  && grep -q '^--- PASS: TestPricingCountsCandidatesRefusalsAndOutcomes' /tmp/adr056-t2.out \
  && grep -q '^--- PASS: TestStatsPricesStrictBalance' /tmp/adr056-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr056-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./internal/authoring/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStrictWouldRefuseIsCountedOnlyWhenTheFlagIsOff` | `internal/apply/apply_test.go` | signature → 1, not in JSON; balanced / multi-line / prose → 0; flag on → 0 | — | S1, S2 |
| `TestPricingCountsCandidatesRefusalsAndOutcomes` | `internal/authoring/authoring_test.go` | five counters after a sequence; file shape; garbage → zero; rate text with no checked refusals | — | S1, S2 |
| `TestStatsPricesStrictBalance` | `cmd/mrw/planpath_test.go` | the block and `--json` after four real writes | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §96 |
| 2 — something selects it | every landed flag-off write records; the CLI attributes the check verdict |
| 3 — the caller can discover it | `mrw stats` prints the block with the bar; T3 names it |
| 4 — it is used | the BACKLOG pre-registration reads it |

## Mutation Log

_(tool-written)_

## Invariants

- The pricing predicate IS `wrapTailDelta`, the flag's predicate — one function.
- A write with `--strict-balance` on records nothing to pricing.
- Exactly one of broke/held/unchecked increments per would-refuse write.
- The `pricing` file holds `name N` lines and nothing else.

## Risks

| Risk | Mitigation |
|------|------------|
| Attributing the outcome before the check has run | record sits after the ADR-054 check block, beside `authoring.Record` |
| A survivor found after the red | every masking mechanism was asked "which assertion fails" before S1 |

## Stop Condition

Stop if pricing needs anything ADR-009 refuses to make the campaign meaningful.

## Out of Scope

- Changing the default. Any per-hunk receipt field.

## Verification Log

_(tool-written)_
