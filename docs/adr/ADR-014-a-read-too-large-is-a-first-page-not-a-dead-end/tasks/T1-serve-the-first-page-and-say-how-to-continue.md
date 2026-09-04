# Task ADR-014-T1: Serve the first page, and say how to continue

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (one package, and a ledger property)
**Owner:** unassigned
**Produces:** the first-page result and its continuation field
**Consumes:** `mcp.MaxResultChars` (ADR-011-T3, unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the continuation spec is correct`, `a page licenses only what it served`, `the result still reads as incomplete`

## Goal

An oversized `mrw_read` returns the lines that fit plus the exact spec to send next, and following
that spec to exhaustion reassembles the file byte for byte.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the oversized path serves a page and names the next spec instead of only refusing |
| `internal/mcp/schema.go` | edit | the read schema declares the continuation field, described — ADR-012's rule |
| `internal/mcp/tools_test.go` | edit | **the ADR's Enforced-by**, plus the ledger property |
| `scripts/contract.sh` | edit | §47 — page a real file to completion through the built binary |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): an oversized read returns content AND a next-spec;
   following the spec repeatedly terminates and the concatenated pages equal the whole file; a
   page-one read licenses a page-one write and REFUSES a page-two write; the result still carries
   `isError: true`. [proof: acceptance]
2. [S2] Serve the page. The line count that fits is already computed for today's suggestion message
   (`tools.go:372`) — the same arithmetic becomes the page span, so the advertised page and the
   served page cannot disagree. [proof: mutation]
3. [S3] Name the next spec as an ordinary mrw spec string, not an opaque token. A caller that can
   read it can narrow it; a cursor it cannot read is one it must trust. [proof: mutation]
4. [S4] Record ONLY the served span in the ledger. `seen.Record` already merges spans per sha, so
   this needs no new mechanism — but it needs a test, because a partial read that licensed the whole
   file would be the read-before-write bypass wearing a new costume, and a partial read that
   licensed nothing would make paging useless. [proof: mutation]
5. [S5] Keep `isError: true` and label the page in BOTH blocks — the prose and the structured
   content — with what it is and what remains. A page that reads like a whole file is the silent
   wrong answer ADR-011 refused; the only thing that makes paging different from truncation is that
   the caller must ask for the rest and can see that it must. [proof: mutation]
6. [S6] Add contract §47: drive the built binary, page a file larger than the limit to completion,
   assert the reassembly is byte-identical to the file and that the last page carries no next-spec.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr014-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr014-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAPagedReadReassemblesTheWholeFile|TestAPageLicensesOnlyWhatItServed|TestAnOversizedReadStillReadsAsIncomplete)\b' /tmp/adr014-t1.out)" = "3" ] \
  && grep -q '^# 47\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the three test names,
`# 47.`, and the Go identifier `nextRead`. **§47 rather than §45**: the highest section on `main` is
44, but ADR-013 takes 45 and 46 on an open branch, so taking them here would collide on merge rather
than on a gate. `gofmt -l .` and `go vet ./...` lead the fence, per ADR-012's follow-up.

`TestAPagedReadReassemblesTheWholeFile` is the ADR's `Enforced-by`. It is deliberately not a
presence check on the field: it FOLLOWS the spec to exhaustion and compares the reassembly to the
file, because a continuation that points at the wrong lines would pass every check that only asks
whether a continuation exists.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPagedReadReassemblesTheWholeFile` | `internal/mcp/tools_test.go` | **the ADR's Enforced-by** — following the continuation to exhaustion loses nothing and terminates | — | S1, S2, S3 |
| `TestAPageLicensesOnlyWhatItServed` | `internal/mcp/tools_test.go` | S4 — a page-one read licenses a page-one write and refuses a page-two write | — | S1, S4 |
| `TestAnOversizedReadStillReadsAsIncomplete` | `internal/mcp/tools_test.go` | S5 — `isError` stays true and both blocks say what remains | — | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | the oversized branch is on the only path a large read takes; §47 drives it through the real binary |
| 3 — the caller can discover it | partly — the field is in the declared `outputSchema` with a description, and T2 teaches it in prose |
| 4 — it is used | nothing counts how often a caller follows a continuation, and nothing will: that needs telemetry, refused by ADR-009, and the per-checkout tally could not attribute it anyway (ADR-012's Context) |

## Mutation Log

<!-- filled during execution -->

## Invariants

- Following the continuation to exhaustion reassembles the file byte for byte.
- The last page carries no continuation.
- A page records exactly the span it served, and licenses exactly that span.
- An oversized read is `isError: true` and says what remains, in both content blocks.
- No engine directory changes; `go.mod` declares exactly one requirement.

## Risks

- A continuation spec that is off by one loses or repeats a line, silently. Mitigated by the
  Enforced-by comparing a full reassembly rather than checking the field.
- Paging licenses more than it served. Mitigated by `TestAPageLicensesOnlyWhatItServed`, which
  attempts the page-two write and expects refusal.
- An infinite loop if the last page never clears the continuation. Mitigated by the Enforced-by
  bounding its own iterations and failing rather than hanging.

## Stop Condition

Stop if serving a page requires `internal/read` to learn anything new. The transport bounds the
transport (ADR-011's own reasoning); a page is a span, and spans already exist.

## Out of Scope

- Changing `MaxResultChars` (deferred: needs the quality curve — parent ADR, Decision 4)
- Teaching the continuation in prose (T2)
- Paging the write path (permanent: boundary: stated in the parent ADR)

## Verification Log
<!-- filled during execution -->
