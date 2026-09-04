# Task ADR-021-T1: Refuse the second spelling of one file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one check in one loop, one test, one contract row)
**Owner:** unassigned
**Produces:** the identity check and its refusal text
**Consumes:** `resolve` (ADR-006, unchanged), `os.SameFile` as `sameFileEntry` already uses it (issue #47)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `two spellings of one file are refused`, `two real files under near-identical names still apply`, `the refusal names both spellings`, `the built binary refuses it too`

## Goal

A plan that names one existing file under two spellings is refused with both spellings named and
nothing written, on every filesystem, instead of applying both hunks and keeping only the last.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the identity check in the grouping loop, reported through the second spelling's first hunk |
| `internal/apply/apply_test.go` | edit | **the ADR's Enforced-by** — a symlink alias (one inode everywhere) and, where the filesystem folds case, two spellings; plus two real files that must still apply |
| `scripts/contract.sh` | edit | §56 — the Context's plan through the built binary |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): a plan naming `real.txt` and `link.txt` (a symlink
   to it) is refused, nothing written, both names in the reason; on a filesystem that folds case,
   `Same.txt` and `same.txt` likewise; two genuinely different files `a.txt` and `a.TXT` on a
   case-sensitive filesystem still apply. [proof: acceptance]
2. [S2] In the grouping loop, resolve each NEW path and ask `os.SameFile` against every file already
   grouped. Ask the filesystem; fold nothing. [proof: mutation]
3. [S3] Report the match as a failed hunk — the second spelling's first — with a reason naming the
3. [S3] Report the match as a failed hunk — the second spelling's first, its siblings skipped — with a
   reason naming the earlier spelling and its plan line, and let ADR-001's existing abort do the rest:
4. [S4] Only for paths that exist. A `create` has no inode yet; that collision is deferred with a
   BACKLOG entry, not claimed. [proof: acceptance]
5. [S5] Add contract §56: the Context's plan through the built binary — exit 1, the file unchanged,
   both spellings in the output — paired with the same two hunks under ONE spelling applying at
   exit 0. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/apply/ -v 2>&1 | tee /tmp/adr021-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr021-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAPlanThatNamesOneFileTwiceIsRefusedWhicheverTheSpelling)\b' /tmp/adr021-t1.out)" = "1" ] \
  && grep -q '^# 56\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/mcp)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name,
`# 56.`, and the Go identifier `sameFileAs`. **§56**: 53 is the highest on `main`; 54 and 55 are held
by #85 and #88, both open — reserved by reading the open PRs, not by listing the file.

**`internal/apply` is deliberately NOT in the go/no-go clauses** — this record changes it. Every
other engine directory and `internal/mcp` are, and must stay byte-identical.

⚠ **THE UNIT TEST ALONE CANNOT PROVE THIS.** The check can be correct in `Apply` and the CLI can still
reach it — it does, because the CLI calls `Apply` — but §56 is what binds the promise to the BINARY
a caller runs, with the exact plan from the Context.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanThatNamesOneFileTwiceIsRefusedWhicheverTheSpelling` | `internal/apply/apply_test.go` | **the ADR's Enforced-by** — a symlink alias is refused everywhere, two spellings where the filesystem folds case, one refusal per file with siblings skipped, and two real files still apply | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `Apply` is the only write path; both the CLI and the MCP tool call it. §56 proves it through the CLI binary |
| 3 — the caller can discover it | the refusal names both spellings and says "one file, one spelling per plan" |
| 4 — it is used | nothing counts refusals by reason and nothing will (ADR-009). The evidence for building it is the break campaign of 2026-09-04: one real defect in 45 probes, and this was it |

## Mutation Log
<!-- filled during execution -->
- 2026-09-04 · 70b9f05 · mutant killed · exit 1 · `internal/apply/apply.go` · S2: make the identity question always answer no — `if false` in place of `os.SameFile`. TestAPlanThatNamesOneFileTwiceIsRefusedWhicheverTheSpelling fails on the symlink half: real.txt and link.txt applied as two files, the measured defect · acceptance-sha256:7dbc2dde4ea5fae35610ce64bf14437a21dfce8a3b23976132f8dd030b5b8002
- 2026-09-04 · 70b9f05 · mutant killed · exit 1 · `internal/apply/apply.go` · S3: drop the first spelling from the refusal. The same test fails on "the refusal must name both spellings so the plan can be fixed in one edit" · acceptance-sha256:7dbc2dde4ea5fae35610ce64bf14437a21dfce8a3b23976132f8dd030b5b8002

## Invariants

- A plan naming one existing file under two path strings is refused; nothing is written.
- The refusal names both spellings and the first spelling's plan line, rides the second spelling's first hunk, and its siblings are skipped.
- Two genuinely different files with near-identical names still apply.
- Nothing folds case; `os.SameFile` decides.
- Every engine directory except `internal/apply`, and `internal/mcp`, are byte-identical; `go.mod` declares exactly one requirement.

## Risks

- The test passes on Linux CI because case never folds there. Mitigated: the symlink half is one inode on every filesystem and is not skippable.
- The check refuses `a.go` and `./a.go`. Mitigated: both clean to one key before the loop and never reach the check; the existing test for that spelling stays green.

## Stop Condition

Stop if the check needs to know what filesystem it is on, or needs to fold case anywhere. That is the
belief-about-the-filesystem rule #47 rejected, and wanting it means `os.SameFile` was not enough — which
would be a different record.

## Out of Scope

- Two `create` ops that would collide (deferred: docs/adr/BACKLOG.md — the create-collision entry)
- Folding case (permanent: boundary: parent ADR, Decision 5)
- Locking against a concurrent writer (permanent: boundary: parent ADR, Out of Scope)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 88c8139* · exit 0 · `set -o pipefail …` · acceptance-sha256:7dbc2dde4ea5fae35610ce64bf14437a21dfce8a3b23976132f8dd030b5b8002 · ms:27948
- 2026-09-04 · 7a1c206* · exit 0 · `set -o pipefail …` · acceptance-sha256:7dbc2dde4ea5fae35610ce64bf14437a21dfce8a3b23976132f8dd030b5b8002 · ms:30252
- 2026-09-04 · 6d00059* · exit 0 · `set -o pipefail …` · acceptance-sha256:7dbc2dde4ea5fae35610ce64bf14437a21dfce8a3b23976132f8dd030b5b8002 · ms:55064
