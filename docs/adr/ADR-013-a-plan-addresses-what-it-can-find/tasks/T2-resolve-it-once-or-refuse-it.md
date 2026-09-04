# Task ADR-013-T2: Resolve it exactly once, or refuse it

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (one package, and the safety argument)
**Owner:** unassigned
**Produces:** pattern resolution and the ambiguity refusal in `apply`
**Consumes:** `plan.Addr` carrying a compiled pattern (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `exactly-once resolution`, `the ambiguity refusal`, `the ledger still binds a resolved pattern`

## Goal

A pattern address resolves against the ORIGINAL file to exactly one line, or the hunk fails saying
why — and a pattern is never a way to edit a file the caller has not read.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | resolve a pattern where `$` is resolved, and fail on 0 or ≥2 matches |
| `internal/apply/apply_test.go` | edit | **the ADR's Enforced-by**, plus the ledger-ordering test |
| `scripts/contract.sh` | edit | §45 — drive the real binary through both refusals |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a pattern matching one line resolves to it; a
   pattern matching nothing fails naming the pattern; a pattern matching twice fails naming the
   pattern **and both line numbers**; and a pattern resolving onto a line the ledger never served is
   refused as unread. [proof: acceptance]
2. [S2] Resolve inside the existing loop, against `orig` — the file as it was before any hunk
   applied. ADR-001's rule is the reason patterns and line numbers in one plan cannot disagree about
   what they address, and resolving against anything else silently reintroduces the offset
   arithmetic the format exists to remove. [proof: mutation]
3. [S3] Refuse ambiguity. Zero matches and two-or-more matches are both hunk failures, never a
   choice. The ≥2 message carries the matched line numbers because that is what lets the caller act
   — narrow the pattern, or address by number. [proof: mutation]
4. [S4] **Resolve BEFORE `covered()`, and pass it the resolved span.** This is the whole safety
   argument and it is an ordering, not a check: `covered()` already takes `(from, to)` and is called
   inside the resolution loop (`apply.go:499-511`), so a resolved pattern meets the same per-line
   ledger test a typed number meets. Getting this backwards would make a pattern a ledger bypass,
   which is the one way this feature is worse than not having it. [proof: mutation]
5. [S5] Add contract §45: drive the built binary through a real tree — a pattern that resolves and
   applies, a pattern that matches twice and is refused with both line numbers named, and a pattern
   that resolves onto an unread line and is refused as unread. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/apply/ -v 2>&1 | tee /tmp/adr013-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr013-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestAnAmbiguousRegexAddressIsRefused|TestARegexAddressResolvesAgainstTheOriginalFile|TestARegexAddressIsStillSubjectToTheLedger|TestAPatternMatchingNothingIsRefused)\b' /tmp/adr013-t2.out)" = "4" ] \
  && grep -q '^# 45\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the four test names
and `# 45.`. §45 was confirmed free — the highest section in `contract.sh` is 44, from ADR-012-T2.

`TestAnAmbiguousRegexAddressIsRefused` is the ADR's `Enforced-by`, and **its fixture is authored
independently of the plan that addresses it.** ADR-012 shipped a mutant that survived precisely
because a fixture built from the plan cannot falsify the plan's guard; a two-match test whose file
was generated from the pattern would be the same defect wearing a different name.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAmbiguousRegexAddressIsRefused` | `internal/apply/apply_test.go` | **the ADR's Enforced-by** — two matches is a refusal naming both lines, never a choice | — | S1, S3 |
| `TestAPatternMatchingNothingIsRefused` | `internal/apply/apply_test.go` | S3 — the other half of ambiguity, and the one a moved line produces | — | S1, S3 |
| `TestARegexAddressResolvesAgainstTheOriginalFile` | `internal/apply/apply_test.go` | S2 — a two-hunk plan where an earlier hunk changes the line count still resolves the later pattern against the original | — | S1, S2 |
| `TestARegexAddressIsStillSubjectToTheLedger` | `internal/apply/apply_test.go` | S4 — the safety argument: a pattern resolving onto an unread line is refused as unread | — | S1, S4 |
| `TestTheEndPatternIsTheFirstMatchAtOrAfterTheStart` | `internal/apply/apply_test.go` | S3 amended — the end is a delimiter, so `/^}/` works even though it closes every function | — | S1, S3 |
| `TestAnEndPatternOnlyAboveTheStartIsRefused` | `internal/apply/apply_test.go` | S3 amended — relaxing exactly-once for the end must not let it delimit nothing | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | resolution is on the only path from a parsed hunk to a written file; §45 drives it through the real binary |
| 3 — the caller can discover it | not yet — T3 teaches it. The refusal messages are discoverable the hard way, which is why their wording is tested |
| 4 — it is used | nothing counts pattern addresses; ADR-012's Context records why the available tally cannot attribute them |

## Mutation Log

- 2026-09-04 · 2e8d959 · mutant killed · exit 1 · `internal/apply/apply.go` · S3: resolve ambiguity by taking the first match instead of refusing — the silent wrong edit this record exists to make unrepresentable · acceptance-sha256:8f4c7dda0c16e849b62de76ed7b1be31255b184d39cd71867dfdc2dcd67ba98f
- 2026-09-04 · 2e8d959 · mutant killed · exit 1 · `internal/apply/apply.go` · S4: let `covered()` return true for a patterned hunk — the ledger bypass, and the one way this feature is worse than not having it · acceptance-sha256:8f4c7dda0c16e849b62de76ed7b1be31255b184d39cd71867dfdc2dcd67ba98f
- 2026-09-04 · eeeb615 · mutant killed · exit 1 · `internal/apply/apply.go` · S3 amended: apply exactly-once to the END pattern as well as the start — the defect that shipped in the first cut and made this record's own headline example fail, because `^}` closes every function in the file · acceptance-sha256:8f4c7dda0c16e849b62de76ed7b1be31255b184d39cd71867dfdc2dcd67ba98f

## Invariants

- A pattern endpoint resolves to exactly one line or the hunk fails.
- Resolution reads the ORIGINAL file, never a partially-applied one.
- A resolved pattern passes through `covered()` exactly as a typed line number does.
- An ambiguity refusal names the matched line numbers.
- A failed pattern hunk writes nothing, and its siblings report `skipped` — ADR-001, unchanged.

## Risks

- The ordering in S4 is easy to get wrong in a later refactor and silent when wrong. Mitigated by
  `TestARegexAddressIsStillSubjectToTheLedger`, and by §45 asserting it against the real binary.
- A test fixture built from the plan cannot falsify the guard. Mitigated by authoring the two-match
  file independently — see Acceptance.

## Stop Condition

Stop if exactly-once resolution turns out to be unusable in practice — if the common Go idiom this
is meant to serve (`/^func (s \*Store) Get/`) routinely matches more than once in real files. That
would mean the ambiguity rule is right and the FEATURE is wrong, and the record should be withdrawn
rather than softened into first-match.

## Out of Scope

- Teaching the form on any surface (T3)
- `occurrence=N` (deferred: docs/adr/BACKLOG.md — stated in the parent ADR)
- Changing `read`'s looser rule (permanent: boundary: serving two matches is useful, editing two is a bug)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 2e8d959* · exit 0 · `set -o pipefail …` · acceptance-sha256:8f4c7dda0c16e849b62de76ed7b1be31255b184d39cd71867dfdc2dcd67ba98f · ms:22045
- 2026-09-04 · eeeb615* · exit 0 · `set -o pipefail …` · acceptance-sha256:8f4c7dda0c16e849b62de76ed7b1be31255b184d39cd71867dfdc2dcd67ba98f · ms:11942
