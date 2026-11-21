# Task ADR-215-T1: a rooted plan path is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the refusal in `internal/apply` and its single sentence; the header splitter's first token; the tests below
**Consumes:** `rooted.IsRooted`, `rooted.Contains`, `apply.resolve`, the header splitter
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `every op refuses a rooted path and a path inside the root`, `the refusal names the root-relative spelling`, `a refused plan writes nothing and its siblings skip`, `a header whose first token begins with a slash is read as a path`, `the Windows spellings are rooted`, `the engine packages this record does not own are untouched`, `every named test passed and none skipped`

## Goal

ADR-215 Decisions 1 to 5.

- **`TestEveryPlanOpRefusesARootedPath`** (`internal/apply`): for replace, insert-before, insert-after, delete, create, unlink, rename (source and destination), a hunk whose path is `/dd/zz.txt` (a file `dd/zz.txt` exists), the absolute path of a file inside the root and the absolute path of one outside it, is refused as a failed hunk: the reason holds "is absolute, and every path in a plan is relative to the root" and the spelling to write (`dd/zz.txt`; the relative path for the inside file); a sibling hunk in the same plan reports `skip`; the tree is byte-identical and no directory was made. The Windows spellings `\dd\zz.txt`, `C:x` and `C:\x` are driven through `rooted.IsRooted`'s seam where one exists, or asserted rooted by the predicate, and refused with the same sentence.
- **`TestAHeaderThatBeginsWithASlashIsOnePath`** (`internal/plan`): `@@ /zz.txt 1 replace`, `@@ /dd 1 replace` and `@@ /zz.txt 1 replace anchor="x/y"` parse to a hunk whose path is that token (and no error); `@@ dd/zz.txt /z1/ replace` and `@@ dd/zz.txt /a/,/b/ replace` still read their second token as a pattern; every header form in the existing parser tests gives the result it gave.
- **`TestMrwWriteRefusesARootedPathLikeTheCLI`** (`internal/mcp`): `mrw_write` of the same plan gives the same sentence and `isError`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the refusal, at `resolve` or the loop that calls it |
| `internal/apply/pathop.go` | edit | the rename destination and the not-exist wording use the one sentence |
| `internal/plan/plan.go` | edit | the header splitter's first token |
| `internal/apply/rooted215_test.go`, `internal/plan/rooted215_test.go`, `internal/mcp/rooted215_test.go` | new | the tests |

## Ordered Steps

1. [S1] The three tests. RED on `main`: `/dd/zz.txt` applies, the header form errors. [proof: acceptance]
2. [S2] The refusal and the splitter. First list every caller of `resolve` and say which paths reach it (the record's audit). Mutants, each killed by a named failure: the refusal skips create; the refusal skips the inside-the-root form; the sentence names no spelling; a sibling is `ok` instead of `skip`; the splitter reads the first token as a pattern again. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env go test -count=1 -v ./internal/apply -run '^TestEveryPlanOpRefusesARootedPath$' > "$d/apply" \
  && env go test -count=1 -v ./internal/plan -run '^TestAHeaderThatBeginsWithASlashIsOnePath$' > "$d/plan" \
  && env go test -count=1 -v ./internal/mcp -run '^TestMrwWriteRefusesARootedPathLikeTheCLI$' > "$d/mcp" \
  && grep -q '^--- PASS: TestEveryPlanOpRefusesARootedPath (' "$d/apply" \
  && grep -q '^--- PASS: TestAHeaderThatBeginsWithASlashIsOnePath (' "$d/plan" \
  && grep -q '^--- PASS: TestMrwWriteRefusesARootedPathLikeTheCLI (' "$d/mcp" \
  && ! grep -q -- '--- SKIP' "$d/apply" "$d/plan" "$d/mcp" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest)" \
  && [ -z "$st" ]
```

What each clause holds. The three named tests exist, pass and none is skipped. The packages the record does not own (`internal/rooted`, `internal/ingest` and the read side among them) have no committed non-test change since the merge base and no uncommitted one.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryPlanOpRefusesARootedPath` | `internal/apply/rooted215_test.go` | every op, inside and outside the root, the Windows spellings, siblings, an untouched tree | none | S1, S2 |
| `TestAHeaderThatBeginsWithASlashIsOnePath` | `internal/plan/rooted215_test.go` | the first token is a path; the second still a pattern | none | S1, S2 |
| `TestMrwWriteRefusesARootedPathLikeTheCLI` | `internal/mcp/rooted215_test.go` | MCP parity | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | every plan hunk |
| 3 — the caller can discover it | the refusal names what to write; T2's words |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #648 and the counts in the record |

## Mutation Log

## Verification Log

## Invariants

- No receipt key is added; a refused plan writes nothing; no exit code of the table changes.

## Risks

- Editing a test file that another record locks moves its lock; S2 lints and relocks what it names.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI; every gate in `AGENTS.md` passes; the engine diff names ADR-215 as owner and has a `go-reviewer` pass; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- The words and the contract row (deferred: T2)
