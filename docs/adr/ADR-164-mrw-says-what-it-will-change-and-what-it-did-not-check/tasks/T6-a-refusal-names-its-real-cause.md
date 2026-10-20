# Task ADR-164-T6: a refusal names its real cause

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** three refusal texts in `internal/plan/plan.go` and `internal/apply/apply.go`
**Consumes:** `internal/apply/apply.go` as T1 leaves it (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the pattern refusal names the slash`, `a pattern refusal without a later slash is unchanged`, `the not-read refusal names a cut line`, `the create refusal names both spellings`, `kinds and exit codes unchanged`, `the binary prints them`

## Goal

Decision 6: the three refusals the peer's report met name their cause, with their kind, exit code and existing words kept.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `parseHeader`'s unknown-op refusal names an address that ended at an unescaped `/`; the `create`-address refusals say `use "-" or "0"` |
| `internal/apply/apply.go` | edit | the not-read refusal's clause; the `create`-address refusals' words |
| `internal/plan/refusal164_test.go` | add | the parser test |
| `internal/apply/refusal164_test.go` | add | the apply test |
| `scripts/contract.sh` | edit | §278 |

## Ordered Steps

1. [S1] Enumerate the tests and contract rows that pin these three refusals' words, and record the command and count here: `mrw read --grep 'unknown op|has not been read|create takes no address' --exclude '*.md' internal cmd scripts`. Then write the failing tests and see them red: TestAPatternEndedByASlashSaysSo (`@@ t.md /^\| `scripts/release.mjs` \| edit/ replace` is refused naming the early end and `\/`; `@@ t.md /x/ frob`, `@@ t.md /x/ frob anchor="a/b"` and `@@ t.md /x/ frob body=@dir/file` keep today's words only; `/^\| `scripts\/release.mjs` \| edit/` parses), TestANotReadRefusalNamesACutLine (the refusal holds `served cut is not read` and its old words; its kind is still `refusal.NotRead`), TestACreateAddressNamesBothSpellings (`@@ n 1 create` says `use "-" or "0"` from the parser and from apply; `0 create` and `- create` apply). [proof: acceptance]
2. [S2] The pattern clause in `parseHeader`. [proof: mutation]
3. [S3] The not-read clause. [proof: mutation]
4. [S4] The `create`-address words, both packages. [proof: mutation]
5. [S5] Contract §278 through the built binary: each of the three refusals carries its clause, exit 2 for the parser's and 1 for apply's, nothing written; the pair: the escaped pattern and `0 create` apply. [proof: human: the executor runs ./scripts/contract.sh once and reads the §278 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/plan/ ./internal/apply/ -count=1 -timeout 300s -run 'TestAPatternEndedByASlashSaysSo$|TestANotReadRefusalNamesACutLine$|TestACreateAddressNamesBothSpellings$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPatternEndedByASlashSaysSo \(' "$out" \
  && grep -qE '^--- PASS: TestANotReadRefusalNamesACutLine \(' "$out" \
  && grep -qE '^--- PASS: TestACreateAddressNamesBothSpellings \(' "$out" \
  && go test ./internal/plan/ ./internal/apply/ ./cmd/mrw/ ./internal/adversarial/ -count=1 -timeout 600s -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheEngineAndTheParserRefuseInTheSameWords \(' "$out" \
  && grep -qE '^--- PASS: TestTheEngineAndTheParserRefuseWithOneKind \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 278\. ' scripts/contract.sh > /tmp/adr164-t6-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMultiLineReplaceWithoutAnAnchorIsRefused` | `internal/apply/apply_test.go` | an apply refusal's existing words are matched by `Contains`, the shape S1's enumeration expects | none | — |
| `TestTheEngineAndTheParserRefuseInTheSameWords` | `internal/adversarial/planformat_test.go` | the parser's and the engine's create-address words change together | none | S4 |
| `TestTheEngineAndTheParserRefuseWithOneKind` | `internal/adversarial/kinds087_test.go` | each refusal keeps its kind (ADR-087) | none | — |

Planned, named without backticks until they exist: TestAPatternEndedByASlashSaysSo in `internal/plan/refusal164_test.go` (S1, S2), TestANotReadRefusalNamesACutLine and TestACreateAddressNamesBothSpellings in `internal/apply/refusal164_test.go` (S1, S3, S4).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three texts |
| 2 — something selects it | `plan.Parse` and `apply`, which every CLI write and `mrw_write` run; contract §278 drives the binary |
| 3 — the caller can discover it | the refusal itself |
| 4 — it is used | nothing measures this; met by the peer's report (2026-10-19/20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- Each refusal's kind (ADR-087), exit code and existing words are unchanged; a test that matches the old words with `Contains` still passes, and S1 lists any that match exactly.

## Risks

- A test or contract row pins one of these refusals by exact equality; S1 enumerates them first.

## Out of Scope

- A ledger record of cut lines (permanent: boundary: the record's Out of Scope)

## Stop Condition

Stop and bring it to Zy if S1 finds a pin by exact equality in a test another record owns, or if a header whose op is unknown can hold a later `/` for a reason other than an early-ended pattern that the clause would misname.
