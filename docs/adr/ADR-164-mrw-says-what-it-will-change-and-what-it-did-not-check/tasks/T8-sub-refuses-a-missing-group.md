# Task ADR-164-T8: a --sub replacement may not name a missing group

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the undefined-group refusal in `internal/read/plansource.go` (`BuildSub`)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a number past the groups is refused`, `an unknown name is refused`, `a defined reference expands`, `a literal $$ passes`, `both surfaces refuse`

## Goal

Decision 8: a replacement naming a group the pattern lacks is a usage error, never a silent empty expansion.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/plansource.go` | edit | `BuildSub` checks the replacement's references against `re.NumSubexp()` and `re.SubexpNames()` before walking |
| `internal/read/sub164_test.go` | add | the test |
| `scripts/contract.sh` | edit | §280 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestASubNamingAMissingGroupIsRefused (`BuildSub` with pattern `head -1` and replacement `^0?$1 x` is a usage `BuildError` naming `$1` and `$$`; with one group, `$1x` is refused, because Go reads the name `1x`, while `${1}x` plans; `${2}` with one group and `${name}` with no such name are refused; `$0`, `${word}` with `(?P<word>…)`, a literal `$$1`, a lone `$` at the end and `$-` plan as before, because Go keeps a `$` it cannot parse as a name). [proof: acceptance]
2. [S2] The check in `BuildSub`, by the same scan Go's `regexp.Expand` makes (`extract`: after `$`, either `{name}` or the longest run of letters, digits and `_`; an all-digit name is a group number; `$$` a literal dollar; a `$` that starts no name is kept), so it refuses exactly the references Go would expand to nothing. [proof: mutation]
3. [S3] Contract §280: `mrw write --grep 'head -1' --sub '$1 x' .` exits 2, nothing written; through `mrw mcp`, `mrw_write` with `grep`+`sub` answers an `isError` tool result and the server answers the next request; the pair: `--grep '(head) -1' --sub '$1 x'` applies. [proof: human: the executor runs ./scripts/contract.sh once and reads the §280 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestASubNamingAMissingGroupIsRefused$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestASubNamingAMissingGroupIsRefused \(' "$out" \
  && go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestASubReadPrintsAPlanAndLicensesItsLines|TestASubPlanParsesBackToTheSamePathsAndBodies|TestASubWriteAppliesThePlanInOneCommand|TestAnMCPSubWriteAppliesThePlanInOneCall' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestASubWriteAppliesThePlanInOneCommand \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 280\. ' scripts/contract.sh > /tmp/adr164-t8-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASubReadPrintsAPlanAndLicensesItsLines` | `internal/read/sub161_test.go` | a valid replacement plans as before (ADR-161) | none | — |
| `TestASubPlanParsesBackToTheSamePathsAndBodies` | `internal/read/sub161_test.go` | the plan text unchanged | none | — |

Planned, named without backticks until it exists: TestASubNamingAMissingGroupIsRefused in `internal/read/sub164_test.go` (S1, S2).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the check in `BuildSub` |
| 2 — something selects it | `BuildSub`, which `write --grep --sub`, `read --grep --sub` and `mrw_write`/`mrw_read` with `sub` all call; contract §280 drives the binary |
| 3 — the caller can discover it | the refusal itself |
| 4 — it is used | nothing measures this; met by a peer session's game-day script (2026-10-20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- A replacement whose references the pattern defines plans exactly as before; `--rewrite` (ast-grep's own metavariables) is untouched.

## Risks

- None beyond the record's.

## Out of Scope

- A `--literal-sub` flag (permanent: boundary: `$$` already writes a dollar, and the refusal names it)

## Stop Condition

Stop and bring it to Zy if Go's template rules (`regexp.Expand`) accept a reference form the check does not parse.
