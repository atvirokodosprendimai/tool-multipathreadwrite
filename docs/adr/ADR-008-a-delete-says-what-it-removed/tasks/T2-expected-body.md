# Task ADR-008-T2: A delete may say which lines it expects to remove

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `delete` with a body, meaning the expected removal
**Consumes:** none — T2's S2 removes the `delete takes no body` rejection itself
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Let a caller pin a destructive hunk with the thing only they have — their own
picture of the lines being removed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `validate` stops rejecting a body on `delete` |
| `internal/plan/plan_test.go` | edit | the parse-level tests |
| `internal/apply/apply.go` | edit | compare the body against the addressed lines and fail the hunk on a mismatch |
| `internal/adversarial/planformat_test.go` | edit | the promise, asserted from outside |
| `README.md` | edit | the guard table and one worked example |
| `scripts/contract.sh` | edit | a row for the match and a row for the mismatch |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a `delete` whose body matches
   the addressed lines applies; one whose body differs fails and writes nothing;
   a bodyless `delete` is unchanged.
2. [S2] Remove the `delete takes no body` rejection from `validate`, keeping the
   error for every other case it covers.
3. [S3] In `planFile`, when a delete carries a body, compare it line for line
   against `orig[start-1:end]` and fail with the first differing line named —
   the same shape as the anchor failure, which prints the anchor beside the real
   line.
4. [S4] Document in the README's guard table, with the worked example, and add
   both contract rows. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ ./internal/apply/ ./internal/adversarial/ -run 'DeleteBody|ExpectedRemoval|BodylessDelete|StillRejected' -v 2>&1 | tee /tmp/adr008-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr008-t2.out \
  && grep -q 'expects to remove' README.md \
  && go test ./... && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDeleteBodyIsNoLongerAParseError` | `internal/plan/plan_test.go` | S2 | — | S1, S2 |
| `TestADeleteWhoseExpectedRemovalMatchesApplies` | `internal/apply/apply_test.go` | the match path | — | S1, S3 |
| `TestADeleteWhoseExpectedRemovalDiffersWritesNothing` | `internal/adversarial/planformat_test.go` | the refusal, and that the whole plan is abandoned | — | S1, S3 |
| `TestAnExpectedRemovalIsNotCheckedAgainstAnUnseenFile` | `internal/apply/apply_test.go` | the comparison runs after the ledger check, so a refusal never quotes a file the caller has not read | — | S3 |
| `TestABodylessDeleteIsUnchanged` | `internal/apply/apply_test.go` | the opt-in stays opt-in | — | S1, S3 |
| `TestDeleteBodyDoesNotWeakenTheOtherOps` | `internal/plan/plan_test.go` | S2 removes one rejection, not the section it lived in | — | S1, S2 |
| `TestADeleteWhoseExpectedRemovalDiffersNamesTheLine` | `internal/apply/apply_test.go` | the refusal names the line, the expectation and what is there | — | S1, S3 |
| `TestAnExpectedRemovalDiffersOnlyInWhitespace` | `internal/apply/apply_test.go` | a whitespace-only mismatch is reported as two DIFFERENT strings, not two identical ones | — | S3 |
| `TestAnExpectedRemovalOfTheWrongLengthIsRejected` | `internal/apply/apply_test.go` | a body of the wrong length is the miscounted range this record is about | — | S1, S3 |
| `TestAReplaceWithNoBodyIsStillRejectedNowThatDeleteTakesOne` | `internal/adversarial/planformat_test.go` | ADR-006's mirror-image rule survives this change | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `planFile`'s delete branch; removing the comparison leaves `TestADeleteWhoseExpectedRemovalDiffersWritesNothing` red |
| 3 — the caller can discover it | the README guard table and two contract rows, asserted by the fence |
| 4 — it is used | nothing measures this yet |

## Mutation Log

- 2026-09-01 · 9d3e013* · mutant killed · exit 1 · `internal/apply/apply.go` · the expected-removal comparison is what refuses a wrong body; without it a mismatched delete applies silently · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b
- 2026-09-01 · 9d3e013* · mutant killed · exit 1 · `internal/apply/apply.go` · a body shorter than the range would otherwise compare only as far as it goes — the miscounted range this record is about · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b
- 2026-09-01 · 91d4268* · mutant killed · exit 1 · `internal/apply/apply.go` · re-recorded after the comparison moved below the ledger check: it is still what refuses a wrong body · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b
- 2026-09-01 · c8147c7* · mutant killed · exit 1 · `internal/apply/apply.go` · trimming both sides renders a whitespace-only mismatch as two identical strings — the exact defect probing found · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b
- 2026-09-01 · c8147c7* · mutant killed · exit 1 · `internal/apply/apply.go` · B2: with the range-level ledger gate gone the body comparison answers first and quotes an unserved line — this is the ordering the fixed fixture pins · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b

## Invariants

- A bodyless `delete` behaves exactly as it does today.
- A mismatch fails the HUNK, so the existing all-or-nothing rule abandons the
  whole plan — a partially applied plan is the worst outcome (ADR-001).
- `replace` still refuses an empty body (ADR-006); the two rules are symmetric
  and neither is weakened.

## Risks

- Callers may write the body from the file rather than from their picture,
  which asserts nothing. Unfixable in the tool — noted in the README so the
  reason to write it is legible.

## Stop Condition

Stop if the comparison cannot report a mismatch as precisely as the anchor
failure does — a refusal that does not say which line differed is worse than
the guard being absent.

## Out of Scope

- Requiring the body (deferred: docs/adr/BACKLOG.md)
- Any equivalent for `replace`, which already carries a body

## Verification Log
- 2026-09-01 · 9d3e013* · exit 1 · `set -o pipefail …` · acceptance-sha256:c4f106e9b5a53879ff889b8d31f7ecad2dc15737baad0806a8db9db8f8308b60 · ms:727
  ```
  --- last 10 line(s) of stdout (of 26 after folding 26 raw)
  === RUN   TestABodylessDeleteIsUnchanged
  --- PASS: TestABodylessDeleteIsUnchanged (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.022s
  === RUN   TestADeleteWhoseExpectedRemovalDiffersWritesNothing
      planformat_test.go:244: a delete whose expected removal was wrong did not fail: [{Path:b.go Addr:3 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast:} {Path:a.go Addr:4-5 Op:delete Status:ok Reason: Removed:2 Added:0 SrcLine:0 RemovedFirst:return a + b RemovedLast:}}]
  --- FAIL: TestADeleteWhoseExpectedRemovalDiffersWritesNothing (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.019s
  FAIL
  ```
- 2026-09-01 · 9d3e013* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:2522
- 2026-09-01 · 91d4268* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:2227
- 2026-09-01 · c8147c7* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:2084
- 2026-09-01 · c8147c7* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:1768
- 2026-09-01 · 9ea8a95* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:1672
- 2026-09-01 · 09d3ee5* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:1479
- 2026-09-01 · bfcd8d6* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:1484
- 2026-09-01 · 4a92d82* · exit 0 · `set -o pipefail …` · acceptance-sha256:997fd705940f19b0bd74a6af8f7e71ee26916c58c65e4c07661541b0e0fd4d7b · ms:1373
