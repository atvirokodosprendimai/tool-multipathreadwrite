# Task ADR-013-T1: The grammar accepts a pattern, and changes nothing else

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one package)
**Owner:** unassigned
**Produces:** `plan.Addr` carrying a compiled pattern
**Consumes:** —
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the pattern forms parse`, `every existing address form is unchanged`

## Goal

`ParseAddr` accepts `/re/` and `/re/,/re/` and rejects a malformed one by name, while every address
this repository has ever written parses to exactly the hunks it parses to today.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `Addr` gains the pattern form; `ParseAddr` learns the two syntaxes |
| `internal/plan/plan_test.go` | edit | its tests, including the regression corpus |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): `/^func Apply/` and `/^func Apply/,/^}/` parse to an
   `Addr` carrying compiled patterns; an unterminated `/re` and an uncompilable `/re(/` are refused
   naming the pattern; and a corpus of every existing address form parses byte-identically to today.
   [proof: acceptance]
2. [S2] Extend `Addr` so a pattern endpoint is representable **without** making a line-number address
   representable two ways. A parsed address is either lines or patterns, never a mix of the two
   fields left half-set — the shape that invites a resolver to read the wrong one. [proof: mutation]
3. [S3] Compile at PARSE time, not at resolve time. A plan carrying an uncompilable pattern is a
   malformed document and should be refused before anything touches the tree, which is also what
   makes the refusal cheap and the failure a parse error rather than a hunk failure. [proof: mutation]
4. [S4] Leave resolution alone. `apply` does not learn anything in this task; a pattern that parses
   and resolves nowhere is T2's problem, and T1 shipping a resolver would make T2's tests pass
   against code no record had accepted yet. Proved negatively: the whole suite stays green with
   `internal/apply` untouched, which the fence's `go test ./...` and `./scripts/contract.sh` clauses
   are what assert. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/plan/ -v 2>&1 | tee /tmp/adr013-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr013-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAPatternAddressParses|TestAMalformedPatternIsRefusedAtParseTime|TestEveryExistingAddressFormIsUnchanged)\b' /tmp/adr013-t1.out)" = "3" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the three test names
and `AddrPattern`. `gofmt -l .` and `go vet ./...` lead the fence deliberately — ADR-012's fences
omitted both, CI's Format step runs `gofmt`, and a formatting failure reached CI twice in one day
because the fence a task calls "done" was not the gate CI applies.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPatternAddressParses` | `internal/plan/plan_test.go` | S1, S2 — both forms, and the compiled pattern is reachable | — | S1, S2 |
| `TestAMalformedPatternIsRefusedAtParseTime` | `internal/plan/plan_test.go` | S3 — an uncompilable pattern fails the document, not a hunk | — | S1, S3 |
| `TestEveryExistingAddressFormIsUnchanged` | `internal/plan/plan_test.go` | the go/no-go — `N`, `N-M`, `N-`, `$`, `$-1`, `0` parse exactly as before, and nothing in `apply` moved | — | S1, S2, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | `ParseAddr` is on the only path from a plan document to a hunk |
| 3 — the caller can discover it | not yet — T3 teaches the form. A grammar nobody is told about is T1 working as scoped |
| 4 — it is used | nothing counts pattern addresses, and the ADR-009 tally deliberately cannot attribute them; see ADR-012's Context for why that instrument is not borrowed |

## Mutation Log

- 2026-09-04 · 2e8d959 · mutant killed · exit 1 · `internal/plan/plan.go` · S3: accept the empty pattern `//`, which matches every line and so can never resolve to exactly one — a document that cannot possibly apply, accepted at parse time · acceptance-sha256:f133612dbaf014760e2ff641fc82f3af1f6f8a3ba92bfbdd00c8f7763288da06

## Invariants

- Every address form that parsed before parses to the same `Addr`.
- A pattern is compiled at parse time; an uncompilable one fails the document.
- An `Addr` is lines or patterns, never both.
- `go.mod` declares exactly one requirement.

## Risks

- The regression corpus is written from memory and misses a form. Mitigated by taking it from the
  addresses `contract.sh` and the ADR examples actually contain, not from the doc comment.

## Stop Condition

Stop if representing a pattern in `Addr` forces `internal/plan` to import `internal/apply` or to
learn how long a file is. Resolution belongs to whoever reads the file, which is T2.

## Out of Scope

- Resolving a pattern to a line (T2)
- Teaching the form on any surface (T3)
- Symbol / AST addressing (permanent: boundary: stated in the parent ADR)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 2e8d959* · exit 0 · `set -o pipefail …` · acceptance-sha256:f133612dbaf014760e2ff641fc82f3af1f6f8a3ba92bfbdd00c8f7763288da06 · ms:49500
- 2026-09-04 · eeeb615* · exit 0 · `set -o pipefail …` · acceptance-sha256:f133612dbaf014760e2ff641fc82f3af1f6f8a3ba92bfbdd00c8f7763288da06 · ms:11795
