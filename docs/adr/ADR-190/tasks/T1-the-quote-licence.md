# Task ADR-190-T1: an exact unique SEARCH block licenses the lines it quotes

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the quoted mark on compiled hunks; apply's quote check; `TestAnExactSearchBlockLicensesTheLinesItQuotes`; contract §328
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an exact unique quote licenses its lines`, `nothing outside the quote is licensed by it`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/quote190_test.go` | new | the red test |
| `internal/ingest/searchreplace.go` | edit | write `quote=MAC` on each compiled hunk |
| `internal/plan/plan.go` | edit | parse `quote=` as a guard |
| `internal/apply/apply.go` | edit | verify the MAC; re-match under the lock; no `covered()` bypass |
| `internal/mcp/mcp.go` | edit | the format's description says an exact block needs no read |
| `internal/ingest/searchreplace_test.go`, `internal/mcp/writeformat*_test.go`, `scripts/contract.sh` §84 | edit | they pin today's refusal of an unread exact SEARCH block; they move to the new rule, and keep a refused case for a non-unique or changed block |
| `internal/guide/guide.go` | edit | the full instructions name SEARCH/REPLACE as the no-read form |
| `scripts/contract.sh` | edit | §328 |

## Ordered Steps

1. [S1] Write the failing test `TestAnExactSearchBlockLicensesTheLinesItQuotes` and see it RED. Cases, each against an empty ledger:
   - a SEARCH block quoting lines 3-4 exactly, matching once: applies;
   - the same block when line 4 changed after the compile: refused as changed;
   - a block that matches twice: refused;
   - a native `@@ f 3-4 replace` of the same lines: refused as unread;
   - a multi-line block ending in a kept line: applies; the same block without a kept line, on a never-read file: refused for the line after;
   - a quoted hunk beside an unquoted sibling in the same file: the sibling is refused as unread;
   - a block whose compiled range would extend past the quote: refused;
   - an identical line inserted before the quoted one between compile and apply: refused, as the quote no longer matches once at its range;
   - a native plan carrying a typed `quote=`: refused.
2. [S2] The compiler hands apply quoted hunks. Mutants, each killed:
   - no quote carried, so the first case goes red;
   - the hunk's range widened one line past the quote, so the leakage case goes red;
   - the licence applied to every hunk in the file, so an unquoted sibling case goes red.
   [proof: mutation]
3. [S3] Apply compares the quote under the write lock, through the writer path. Mutants, each killed:
   - skip the comparison, so the changed-line case goes red;
   - compare before the lock, so a test that changes the file between them goes red;
   - check the bytes at the range but not uniqueness, so the inserted-duplicate case goes red;
   - accept any `quote=` without its MAC, so the typed-quote case goes red.
   [proof: mutation]
4. [S4] No plan text can set the quote: a native plan with any option is licensed by the ledger alone. Contract §84's unread cases move to the new rule, and its non-unique and changed cases stay refused. [proof: acceptance]
5. [S5] The guide and contract §328: through the built binary, an empty ledger, and `mrw write --format=search_replace` applies, exit 0. The pair: a native plan of the same lines is refused, exit 1. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/ingest/ ./internal/apply/ ./internal/plan/ -count=1 -v -run '^TestAnExactSearchBlockLicensesTheLinesItQuotes$' 2>&1) \
  && grep -qE '^--- PASS: TestAnExactSearchBlockLicensesTheLinesItQuotes \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 328\. ' scripts/contract.sh \
  && grep -qF 'PASS  an exact SEARCH block applies with no read before it' <<<"$c" \
  && grep -qF 'PASS  the pair: a native plan of the same lines is refused as unread' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnExactSearchBlockLicensesTheLinesItQuotes` | `internal/ingest/quote190_test.go` | the quote licenses its lines and nothing else; a changed, repeated or native line is refused | none | S1, S2, S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw write --format=search_replace`, MCP `format: search_replace` |
| 3 — the caller can discover it | `mrw instructions` |
| 4 — it is used | no telemetry (ADR-009); the paid run measures it |

## Mutation Log
- 2026-11-02 · 2849a1d0 · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · the compiler carries no quote mark so the exact block is an unquoted hunk and the no-read case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:an exact unique quote licenses its lines
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · the hunk range is widened one line past the quote, so the compiled range is not the quote and the kept-closer case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:nothing outside the quote is licensed by it
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/apply/apply.go` · the licence is lifted for the whole file as soon as any hunk is quoted, so an unquoted sibling must go unrefused and the sibling case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:nothing outside the quote is licensed by it
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/apply/apply.go` · apply skips the comparison of the quote with the file it holds, so a line changed after the compile stays licensed and the changed-line case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:an exact unique quote licenses its lines
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/quote/quote.go` · the quote is compared at the range but not for uniqueness, so a duplicate of the quoted lines elsewhere stays licensed and the duplicate case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:an exact unique quote licenses its lines
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/quote/quote.go` · any well formed quote mark is accepted without its MAC, so a quote typed into a native plan licenses its lines and the typed-quote case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:an exact unique quote licenses its lines
- 2026-11-02 · 2849a1d0* · mutant killed · exit 1 · `internal/apply/apply.go` · apply accepts a quote mark on an op other than replace, so a compiled mark moved onto a delete licenses it and the moved-op case must go red · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · covers:nothing outside the quote is licensed by it

## Verification Log
- 2026-11-02 · 84267ed8* · exit 1 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:1430 · test-lock-sha256:7bf0ee472c41643df19fc13f3fb7547cfd5e5d15d633e02957a212fec6fa93e7 · test-lock-file:docs/adr/locks/7bf0ee472c41643df19fc13f3fb7547cfd5e5d15d633e02957a212fec6fa93e7.lock
  ```
  ```
- 2026-11-02 · 2849a1d0 · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:102044
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:114448
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:113734
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:104819
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:80528
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:81310
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:81257
- 2026-11-02 · 2849a1d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6c3d61bb9c3ae30dd5478f50453bdc843d00a68141de27714e1134f58e7b25aa · ms:95034

## Invariants

- A native plan's licence is exactly the ledger's.
- A quote is compared with the file under the write lock.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0.

## Out of Scope

- apply_patch (deferred: docs/adr/BACKLOG.md)
