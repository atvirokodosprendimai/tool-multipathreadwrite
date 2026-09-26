# Task ADR-040-T2: Unquoted `anchor=` consumes until the next `key=`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** unquoted `anchor=` consume-to-next-key (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `unquoted spaced anchor parsing`, `quoted anchor still working`, `other keys unchanged`

## Goal

If Accept quotes the parse fork: `anchor=func openTestStore` is a successful parse whose Anchor
is `func openTestStore`, and a following `body=1` / `lines=1` still binds.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `splitHeader` — the lexer. The selector. |
| `internal/plan/plan_test.go` | edit | `TestAnUnquotedSpacedAnchorParsesUntilTheNextKey` — the field-report fixture as a success. |
| `scripts/contract.sh` | edit | §79 or a sibling assertion: the built binary accepts the unquoted form and still refuses a single-quoted one, unless Accept also quoted single quotes. |

## Ordered Steps

1. [S1] Write `TestAnUnquotedSpacedAnchorParsesUntilTheNextKey` and confirm it is RED: today's parse is exit-2 `option "openTestStore" is not key=value`. [proof: mutation]
2. [S2] Implement consume-to-next-key for `anchor=` only. Quoted `anchor="func openTestStore"` stays valid. `sha=`, `lines=`, `body=`, `raw=` do not gain the rule unless Accept named them. [proof: mutation]
3. [S3] Drive the built binary: unquoted spaced `anchor=` applies (or dry-runs) ; single quotes still fail unless Accept said otherwise. [proof: mutation]
4. [S4] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -count=1 -v \
  -run 'TestAnUnquotedSpacedAnchorParsesUntilTheNextKey' 2>&1 | tee /tmp/adr040-t2.out \
  && grep -q '^--- PASS: TestAnUnquotedSpacedAnchorParsesUntilTheNextKey' /tmp/adr040-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr040-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnUnquotedSpacedAnchorParsesUntilTheNextKey` | `internal/plan/plan_test.go` | `anchor=func openTestStore body=1` parses; Anchor is `func openTestStore`; `body=` still binds | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAnUnquotedSpacedAnchorParsesUntilTheNextKey` |
| 2 — something selects it | `splitHeader`; reverting consume-to-next-key fails S1 |
| 3 — the caller can discover it | `write --help` still names double quotes (T1); the new form is extra, not a replacement |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 4b70dd0* · mutant inconclusive · exit 1 · `internal/plan/plan.go` · without joinUnquotedAnchorOptions the field-report header must fail T2 · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `internal/plan/plan.go` · without joinUnquotedAnchorOptions the field-report header must fail T2 · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `internal/plan/plan.go` · without joinUnquotedAnchorOptions the field-report header must fail T2 · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · covers:unquoted spaced anchor parsing

## Invariants

- Double-quoted `anchor=` still works.
- A multi-line replace still requires `anchor=` (ADR-035).
- Today's successful writes stay successful.
- Accept quoted the parse fork (*"good, accepted all"*).

## Risks

- Consume-to-next-key eats a following `body=1` into the anchor. Mitigation: S1 fixture includes a following key.
- T1's refusal test and this success test contradict. Mitigation: rewrite T1's fixture to a non-`anchor=` trailing token, or to single quotes, in the same change.

## Stop Condition

Stop if the implementation wants to apply the rule to every key without Accept naming that.

Stop if a previously successful quoted plan starts failing.

## Out of Scope

- Teach (that's T1; may already be done)
- `mrw version` (that's T3)
- Single quotes, unless Accept named them

## Verification Log
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · ms:359
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · ms:398
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · ms:354
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · ms:432
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:987c0afbceb21e20c8a2cef072206103b086b1d5d0452177cfbda561e019f1e8 · ms:411
