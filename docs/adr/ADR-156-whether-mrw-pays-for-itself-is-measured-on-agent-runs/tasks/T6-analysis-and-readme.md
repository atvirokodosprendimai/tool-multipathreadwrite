# Task ADR-156-T6: analysis against the registered bar, and README

**Depends-on:** T3, T4, T5
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `bench analyze`, `docs/bench/RESULTS.md`, the README sentence
**Consumes:** the receipts of T3, T4 and T5
**Data dependency:** the stage receipts
**Proof map:** v1
**Rests-on:** `pairs by task and keeps every attempt`, `admits only by the registered rules`, `the verdict follows the bar`, `README states no more than the verdict`

## Goal

One verdict per model and task source, computed exactly as BACKLOG registered it, and README saying that and no more.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/analyze.go` | add | pairing, the bootstrap (seed 156, 10,000 resamples), the verdict |
| `internal/bench/analyze_test.go` | add | the tests |
| `cmd/bench/main.go` | edit | `analyze` |
| `docs/bench/RESULTS.md` | add | every verdict, every silent mis-edit, every exclusion |
| `README.md` | edit | one sentence per verdict, with the link |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestTheAnalysisPairsByTaskAndKeepsEveryAttempt`, `TestOnlyTheRegisteredExclusionsAreAdmitted`, `TestTheVerdictFollowsTheRegisteredBar` (constructed receipts that must yield PAYS, LOSES and INCONCLUSIVE, and one at each boundary).
2. [S2] `bench analyze`. [proof: mutation]
3. [S3] `docs/bench/RESULTS.md` from `bench analyze` output, not by hand. [proof: acceptance]
4. [S4] README, one sentence per verdict, a LOSES or INCONCLUSIVE as plainly as a PAYS. [proof: human: Zy reads the README sentence against RESULTS.md]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -run 'TestTheAnalysisPairsByTaskAndKeepsEveryAttempt|TestOnlyTheRegisteredExclusionsAreAdmitted|TestTheVerdictFollowsTheRegisteredBar' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheAnalysisPairsByTaskAndKeepsEveryAttempt \(' "$out" \
  && grep -qE '^--- PASS: TestOnlyTheRegisteredExclusionsAreAdmitted \(' "$out" \
  && grep -qE '^--- PASS: TestTheVerdictFollowsTheRegisteredBar \(' "$out" \
  && go run ./cmd/bench analyze docs/bench > "$out.results" \
  && cmp -s "$out.results" docs/bench/RESULTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheAnalysisPairsByTaskAndKeepsEveryAttempt` | `internal/bench/analyze_test.go` | pairing within model and source; nothing dropped | none | S1, S2 |
| `TestOnlyTheRegisteredExclusionsAreAdmitted` | `internal/bench/analyze_test.go` | only the registered exclusions | none | S1, S2 |
| `TestTheVerdictFollowsTheRegisteredBar` | `internal/bench/analyze_test.go` | PAYS, LOSES, INCONCLUSIVE and their boundaries | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `bench analyze` |
| 2 — something selects it | the fence regenerates RESULTS.md and compares |
| 3 — the caller can discover it | README's sentence and link |
| 4 — it is used | README readers |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The bar in BACKLOG is never edited after T3 starts; an amendment is a new registration before a new run, as ADR-119's were.

## Risks

- A verdict read as a general claim: README names the models, the hosts and the task sources it holds for.

## Out of Scope

- A claim about hosts other than Claude Code (permanent: boundary: Decision 1 runs one host)

## Stop Condition

The fence exits 0 and Zy has read the README sentence.
