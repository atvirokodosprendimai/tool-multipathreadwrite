# Task ADR-164-T1: a refusal says sha= does not stand in for anchor=

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the refusal clause in `internal/apply/apply.go`; one sentence in `internal/guide/guide.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the clause appears with sha=`, `the clause is absent without sha=`, `the full guide says it`, `the core does not`, `the binary prints it`

## Goal

Decision 1: a multi-line `replace` refused for having no `anchor=` says, when it carries `sha=`, that `sha=` does not stand in; the full guide says the same; the core is untouched.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the `carries no anchor=` refusal gains the clause when `h.SHA != ""` |
| `internal/apply/anchor164_test.go` | add | the engine test |
| `internal/guide/guide.go` | edit | the full contract's anchor sentence gains `sha= does not stand in for it.` |
| `internal/guide/sha164_test.go` | add | the guide test |
| `scripts/contract.sh` | edit | §273 |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestAShaGuardDoesNotStandInForAnAnchor (a three-line `replace` with a matching `sha=` and no `anchor=` is refused, its reason holds `carries no anchor=` and `sha= checks that the whole file is unchanged`; the same hunk without `sha=` is refused without that clause) and TestTheFullGuideSaysShaDoesNotStandInForAnAnchor (`guide.CLI()` holds `sha= does not stand in for it`; `guide.Core()` does not). [proof: acceptance]
2. [S2] The clause in the refusal. [proof: mutation]
3. [S3] The guide sentence. [proof: mutation]
4. [S4] Contract §273 drives the built binary: a `sha=`-only multi-line replace exits 1 with the clause, nothing written; the pair: the same plan with `anchor=` applies. [proof: human: the executor runs ./scripts/contract.sh once and reads the §273 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAShaGuardDoesNotStandInForAnAnchor$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAShaGuardDoesNotStandInForAnAnchor \(' "$out" \
  && go test ./internal/apply/ ./internal/guide/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheFullGuideSaysShaDoesNotStandInForAnAnchor|TestAMultiLineReplaceWithoutAnAnchorIsRefused|TestTheCoreNamesTheSubWrite|TestInstructionsCoreFlagPrintsTheCore' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheFullGuideSaysShaDoesNotStandInForAnAnchor \(' "$out" \
  && grep -qE '^--- PASS: TestAMultiLineReplaceWithoutAnAnchorIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesTheSubWrite \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 273\. ' scripts/contract.sh > /tmp/adr164-t1-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMultiLineReplaceWithoutAnAnchorIsRefused` | `internal/apply/apply_test.go` | the refusal's existing words still hold (ADR-035) | none | — |
| `TestTheCoreNamesTheSubWrite` | `cmd/mrw/core141_test.go` | the core stays under its word ceiling (ADR-141) | none | — |

Planned, named without backticks until they exist: TestAShaGuardDoesNotStandInForAnAnchor in `internal/apply/anchor164_test.go` (S1, S2) and TestTheFullGuideSaysShaDoesNotStandInForAnAnchor in `internal/guide/sha164_test.go` (S1, S3).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the clause, the sentence |
| 2 — something selects it | `planFile`'s anchor check, which every CLI write and `mrw_write` run; contract §273 drives the built binary |
| 3 — the caller can discover it | the refusal itself; `mrw instructions` |
| 4 — it is used | nothing measures this; built from one field report (2026-10-20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The refusal's existing words, exit code and (absent) kind are unchanged (§73, ADR-035, ADR-087).
- `guide.Core()` is unchanged (ADR-141).

## Risks

- None beyond the record's.

## Out of Scope

- The core (permanent: boundary: the record's Decision 1)

## Stop Condition

Stop and bring it to Zy if the sentence cannot be added to the full guide without breaking a guide test that pins the full text's shape.
