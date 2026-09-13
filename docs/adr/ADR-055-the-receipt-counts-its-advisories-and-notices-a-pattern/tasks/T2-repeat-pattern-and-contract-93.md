# Task ADR-055-T2: Recent-window ring and the pattern line; contract §93

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `authoring.Recent` + pattern line (T2)
**Consumes:** `apply.Result.Advisories` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `ring holds last 10`, `no paths or plan text`, `line at 3 of 10`, `fails open`, `stats shows the window`

## Goal

Every write `Record` site also appends `<unix-seconds> write <advisories>` to `recent` beside the tally, keeping the last 10. After a CLI write, when ≥3 of the last 10 carried an advisory, `report` prints `pattern: K of your last N writes carried a balance advisory — read past the range before the next one`. `mrw stats` prints the same line under the same condition and `recent: N write(s) in the window` always. Thresholds are named constants in `internal/authoring`. Unreadable or absent `recent` is empty, never an error.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/authoring/authoring.go` | edit | `RecordRecent`, `Recent`, `Pattern` and the two constants. Vocabulary untouched. |
| `internal/authoring/authoring_test.go` | edit | Red: ring keeps 10; no path survives a write; `Pattern` at 3 of 10; fail-open on garbage. |
| `cmd/mrw/main.go` | edit | Call `RecordRecent` beside every write `Record`; `report` pattern line; `stats` window lines. The selector. |
| `internal/mcp/tools.go` | edit | `RecordRecent` beside its `Record` (feeds the ring; prints nothing). |
| `cmd/mrw/advisory_test.go` | edit | Red: three advisory writes then a fourth prints the line; two do not; `stats` shows the window. |
| `scripts/contract.sh` | edit | **§93** — pair: three `{`-only replaces then the receipt prints `pattern:` / two do not / `stats` prints `recent:`; `strings recent` shows no path. |

## Ordered Steps

1. [S1] Write `TestRecentKeepsTheLastTenWritesAndNoPaths` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestPatternFiresAtThreeOfTenAndNotAtTwo` and `TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory` — RED. [proof: mutation]
3. [S3] Implement the ring, `Pattern`, and the two call sites. Confirm GREEN. Deleting the CLI `RecordRecent` call must fail S2; lowering the threshold to 2 must fail the not-at-two half. [proof: mutation]
4. [S4] Write §93 RED then GREEN, including the `strings` check that no path reached disk. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 93\. ' scripts/contract.sh \
  && go test ./internal/authoring/ ./cmd/mrw/ -count=1 -v \
    -run 'TestRecentKeepsTheLastTenWritesAndNoPaths|TestPatternFiresAtThreeOfTenAndNotAtTwo|TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory' 2>&1 | tee /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestRecentKeepsTheLastTenWritesAndNoPaths' /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestPatternFiresAtThreeOfTenAndNotAtTwo' /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory' /tmp/adr055-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/authoring/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestRecentKeepsTheLastTenWritesAndNoPaths`~~ | `internal/authoring/authoring_test.go` | not yet written — Proposed; T2 S1 writes it | — | S1, S3 |
| ~~`TestPatternFiresAtThreeOfTenAndNotAtTwo`~~ | `internal/authoring/authoring_test.go` | not yet written — Proposed; T2 S2 writes it | — | S2, S3 |
| ~~`TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory`~~ | `cmd/mrw/advisory_test.go` | not yet written — Proposed; T2 S2 writes it | — | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §93 |
| 2 — something selects it | every write `Record` site calls `RecordRecent`; deleting the CLI call fails S2 and §93 |
| 3 — the caller can discover it | the line arrives unasked on the receipt; `stats` shows the window; T4 |
| 4 — it is used | the field run's fourth repeat is the case; ADR-009 refused telemetry |

## Mutation Log

(empty until execute)

## Invariants

- ADR-009: counts and a timestamp only; no path, plan text or address in `recent`. Fails open. Never fails a write.
- Five authoring names unchanged; no sixth Outcome.
- MCP feeds the ring and prints nothing.

## Risks

- Two writers appending concurrently (CLI + MCP) can interleave lines; the ring is advisory input and a torn line is skipped by the reader, never an error.

## Stop Condition

If the only way to go green is to record a path or a plan line, stop — ADR-009's boundary.

## Out of Scope

- The advisory count itself (T1)
- `--strict-balance` (T3)
- Pattern line on the MCP receipt (BACKLOG)

## Verification Log

(empty until execute)
