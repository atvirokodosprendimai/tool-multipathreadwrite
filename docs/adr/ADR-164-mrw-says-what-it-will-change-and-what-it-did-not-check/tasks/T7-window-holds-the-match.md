# Task ADR-164-T7: a --max-cols window holds the match

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the window placement in `internal/read/maxcols.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a match that fits is whole`, `a longer match starts the window`, `no match starts the line`, `the binary prints it`

## Goal

Decision 7: `window` centres on the match itself, so a match no longer than N is never cut.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/maxcols.go` | edit | `window` takes the match's end too and centres on its middle; a match longer than N starts the window |
| `internal/read/maxcols164_test.go` | add | the test |
| `internal/read/maxcols137_test.go` | read only | `TestTheWindowCentresOnTheMatch` already asserts a mid-line, front, back and no-pattern case; it must stay green |
| `scripts/contract.sh` | edit | §279 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAMatchThatFitsIsWholeInItsWindow (`window("Alpha NEEDLE x", 8, NEEDLE)` holds `NEEDLE` whole; a 30-character match with N 20 starts the window at the match; with N equal to the match's length the window is exactly the match; an empty match does not panic). [proof: acceptance]
2. [S2] Centre on the match. [proof: mutation]
3. [S3] Contract §279: `mrw read --grep NEEDLE --max-cols 8 f` prints `NEEDLE` whole; the pair: the line is still not licensed (a plan to it is refused). [proof: human: the executor runs ./scripts/contract.sh once and reads the §279 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAMatchThatFitsIsWholeInItsWindow$|TestTheWindowCentresOnTheMatch$|TestAnOverlongLineIsCutToAWindowAndLicensesNothing$|TestTheWindowCentresOnTheEndOfAStartEndRange$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAMatchThatFitsIsWholeInItsWindow \(' "$out" \
  && grep -qE '^--- PASS: TestTheWindowCentresOnTheMatch \(' "$out" \
  && grep -qE '^--- PASS: TestAnOverlongLineIsCutToAWindowAndLicensesNothing \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 279\. ' scripts/contract.sh > /tmp/adr164-t7-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWindowCentresOnTheMatch` | `internal/read/maxcols137_test.go` | ADR-137's window cases unchanged | none | — |
| `TestAnOverlongLineIsCutToAWindowAndLicensesNothing` | `internal/read/maxcols137_test.go` | a cut line still licenses nothing | none | — |

Planned, named without backticks until it exists: TestAMatchThatFitsIsWholeInItsWindow in `internal/read/maxcols164_test.go` (S1, S2).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `window`'s placement |
| 2 — something selects it | every `mrw read --max-cols N` with a pattern; contract §279 drives the binary |
| 3 — the caller can discover it | the `[cols A-B of C]` marker |
| 4 — it is used | nothing measures this; met in this session (2026-10-20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The window is N characters wide whenever the line is longer than N; a cut line licenses nothing (ADR-137); `mrw_read`'s `max_cols` (ADR-154) gets the same placement.

## Risks

- None beyond the record's.

## Out of Scope

- Windows round every match on a line (permanent: boundary: ADR-137 cuts round the first match)

## Stop Condition

Stop and bring it to Zy if `TestTheWindowCentresOnTheEndOfAStartEndRange` (a `/a/,/b/` range centres on its end) cannot hold beside a centred match.
