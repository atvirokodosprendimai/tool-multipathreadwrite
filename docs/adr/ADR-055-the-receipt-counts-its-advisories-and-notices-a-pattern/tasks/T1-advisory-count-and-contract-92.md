# Task ADR-055-T1: Advisory count on the summary line and the receipt; contract §92

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `apply.Result.Advisories` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `count always printed`, `zero printed`, `json and mcp carry advisories`, `skip and fail contribute nothing`

## Goal

`report` prints `N hunk(s), M file(s), F failed, A advisor(y|ies) — <state>` always, including `0 advisories`. `apply.Result.Advisories` counts `ok` hunks with non-empty `Balance`; JSON and the MCP receipt carry `advisories`. `--quiet` still prints the summary, so the count. Skipped and failed hunks contribute nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Result.Advisories` computed beside `Balance`. |
| `internal/apply/apply_test.go` | edit | Red: count on a delta hunk; zero on a clean plan; skipped hunk not counted. |
| `cmd/mrw/main.go` | edit | `report` summary clause. The selector a human reads. |
| `cmd/mrw/advisory_test.go` | create | Red: summary line carries `1 advisory` / `0 advisories`; `--quiet` carries it; `--json` carries `advisories`. |
| `internal/mcp/schema.go` | edit | `advisories` description — the conformance test refuses an undescribed property. |
| `scripts/contract.sh` | edit | **§92** — next free after §91. Pair: a `{`-only replace prints `1 advisory` / a clean replace prints `0 advisories` / the one existing summary grep is moved, not deleted. |

## Ordered Steps

1. [S1] Write `TestTheSummaryLineCountsAdvisories` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestAdvisoriesCountsOnlyOkHunksWithABalance` (engine) and `TestQuietAndJSONCarryTheAdvisoryCount` — RED. [proof: mutation]
3. [S3] Implement `Result.Advisories` and the summary clause. Confirm S1–S2 GREEN. Deleting the clause must fail S1; counting skipped hunks must fail S2. [proof: mutation]
4. [S4] Write §92 RED then GREEN; move the one grep that matched the old summary. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 92\. ' scripts/contract.sh \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
    -run 'TestTheSummaryLineCountsAdvisories|TestAdvisoriesCountsOnlyOkHunksWithABalance|TestQuietAndJSONCarryTheAdvisoryCount|TestEveryOutputSchemaPropertyIsDescribed' 2>&1 | tee /tmp/adr055-t1.out \
  && grep -q '^--- PASS: TestTheSummaryLineCountsAdvisories' /tmp/adr055-t1.out \
  && grep -q '^--- PASS: TestAdvisoriesCountsOnlyOkHunksWithABalance' /tmp/adr055-t1.out \
  && grep -q '^--- PASS: TestQuietAndJSONCarryTheAdvisoryCount' /tmp/adr055-t1.out \
  && grep -q '^--- PASS: TestEveryOutputSchemaPropertyIsDescribed' /tmp/adr055-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestTheSummaryLineCountsAdvisories`~~ | `cmd/mrw/advisory_test.go` | not yet written — Proposed; T1 S1 writes it | — | S1, S3 |
| ~~`TestAdvisoriesCountsOnlyOkHunksWithABalance`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| ~~`TestQuietAndJSONCarryTheAdvisoryCount`~~ | `cmd/mrw/advisory_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | the MCP receipt describes `advisories` | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §92 |
| 2 — something selects it | `report` prints the clause on every write; deleting it fails S1 and §92 |
| 3 — the caller can discover it | the summary line itself; T4 names it on `write --help` |
| 4 — it is used | the field run read the summary every time and nothing else; ADR-009 refused telemetry |

## Mutation Log

(empty until execute)

## Invariants

- ADR-001/ADR-054: the count never changes a verdict or a byte.
- `Echo` is not an advisory.
- Five authoring names unchanged.

## Risks

- The one contract grep on `0 failed — applied` moves; the fence must fail if the clause is dropped, not if the grep is edited.

## Stop Condition

If the only way to go green is to print the count only when non-zero, stop — that fork was rejected.

## Out of Scope

- The ring and the pattern line (T2)
- `--strict-balance` (T3)
- Teaching (T4)

## Verification Log

(empty until execute)
