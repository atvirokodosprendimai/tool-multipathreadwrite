# Task ADR-054-T3: stats always prints `failed_check`; landed line; contract §91

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** stats five names + landed line (T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `zeros print`, `no sixth Outcome`, `landed denominator`

## Goal

`mrw stats` always prints the five vocabulary names, including `failed_check 0`. After the five rows, one derived line: `landed writes: N; failed_check F of those (x%).` with `N = applied + failed_check + check_not_run`. `check_not_run` is in N because that Outcome means the write stands (exit 2: written, but no check could run). Landed is not "wrote and was checked". JSON always includes the five keys plus `landed` and `failed_check_of_landed`. Do not add a sixth Outcome.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `statsCmd` rendering. The selector. |
| `cmd/mrw/planpath_test.go` | edit | Or a dedicated stats test: zeros print; derived line; json keys. |
| `scripts/contract.sh` | edit | **§91** — a checkout with only `applied` still prints `failed_check 0` and the landed line. |

## Ordered Steps

1. [S1] Write `TestStatsPrintsFailedCheckEvenWhenZero` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestStatsLandedLineUsesAppliedPlusFailedCheckPlusCheckNotRun` — RED until the derived line exists. [proof: mutation]
3. [S3] Render all five names from the closed vocabulary (not `t.Names()` alone). Un-strike the Tests rows when the funcs exist. Confirm S1–S2 GREEN. Deleting the zero-print must fail S1. [proof: mutation]
4. [S4] Write §91 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 91\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestStatsPrintsFailedCheckEvenWhenZero|TestStatsLandedLineUsesAppliedPlusFailedCheckPlusCheckNotRun' 2>&1 | tee /tmp/adr054-t3.out \
  && grep -q '^--- PASS: TestStatsPrintsFailedCheckEvenWhenZero' /tmp/adr054-t3.out \
  && grep -q '^--- PASS: TestStatsLandedLineUsesAppliedPlusFailedCheckPlusCheckNotRun' /tmp/adr054-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestStatsPrintsFailedCheckEvenWhenZero`~~ | `cmd/mrw/planpath_test.go` | not yet written — Proposed; T3 S1 writes it | — | S1, S3 |
| ~~`TestStatsLandedLineUsesAppliedPlusFailedCheckPlusCheckNotRun`~~ | `cmd/mrw/planpath_test.go` | not yet written — Proposed; T3 S2 writes it | — | S2, S3 |
| `§91` | `scripts/contract.sh` | Built binary: zeros and landed line | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §91 |
| 2 — something selects it | `statsCmd`; deleting the zero-print fails S1 |
| 3 — the caller can discover it | T4; `mrw stats` itself |
| 4 — it is used | the Zeus population is this repository's next `mrw stats` |

## Mutation Log

(empty until execute)

## Invariants

- ADR-009: no plan text, no paths; five names only.
- `--no-check` still records `applied`.
- `internal/authoring` vocabulary stays five names.
- The derived line's comment or test name must not call N "verified writes".

## Risks

- Scripts that parse exactly three rows break. Mitigation: five names were already the vocabulary; they were just omitted at zero.

## Stop Condition

If the only way to go green is a sixth Outcome, stop.

## Out of Scope

- Default check (T1) — T3 is useful with zeros even before T1
- Balance (T2)
- Teaching (T4)

## Verification Log

(empty until execute)
