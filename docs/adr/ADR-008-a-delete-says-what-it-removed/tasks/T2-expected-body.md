# Task ADR-008-T2: A delete may say which lines it expects to remove

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `delete` with a body, meaning the expected removal
**Consumes:** `Hunk.Body` legal for `OpDelete` (T1)
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
go test ./internal/plan/ ./internal/apply/ ./internal/adversarial/ -run 'DeleteBody|ExpectedRemoval' -v 2>&1 | tee /tmp/adr008-t2.out \
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
| `TestABodylessDeleteIsUnchanged` | `internal/apply/apply_test.go` | the opt-in stays opt-in | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `planFile`'s delete branch; removing the comparison leaves `TestADeleteWhoseExpectedRemovalDiffersWritesNothing` red |
| 3 — the caller can discover it | the README guard table and two contract rows, asserted by the fence |
| 4 — it is used | nothing measures this yet |

## Mutation Log

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
