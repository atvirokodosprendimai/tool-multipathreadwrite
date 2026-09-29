# Task ADR-098-T1: An ast_grep index pages with `after`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `after` with `ast_grep` (`afterCursor` in `internal/mcp/tools.go`); contract §189
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the index pages to the end`, `after without a finder is refused`, `the index names its finder`, `a contract row drives the binary`, `the tree is gofmt-clean`, `no engine package changes`, `go.mod declares one requirement`

## Goal

An oversized `ast_grep` read over MCP returns an INDEX that `after` pages to exhaustion, with no file lost
or repeated, and the index prose names `ast_grep` and says to repeat until `next_index` is empty; `after`
with plain specs is still refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the guard at `:296` refuses `after` only when neither `grep` nor `ast_grep` is set, naming both; `afterCursor(specs, after)` (new) holds the skip now inline in `grepSpecs` (`:1440-1450`); `astGrepSpecs` takes `after` and applies it; the call site `:311` passes `a.After`; the handler computes the finder's name ONCE (`grep` or `ast_grep`) and passes that one value to `matchIndex` at all three of its callers — raw-content overflow (`:427`), encoded-receipt overflow (`:505`) and checkpoint overflow (`:528`) — whose prose (`:1543`, `:1548`) names it and says "until next_index is empty" |
| `internal/mcp/astgrepindex098_test.go` | add | the tests below |
| `scripts/contract.sh` | edit | §189, after §188 |

**What selects it:** the `ast_grep` branch of the mrw_read handler (`tools.go:309-316`) calls
`astGrepSpecs`; the guard at `:296` is the only gate. Deleting the `a.After` argument at `:311` returns page
one forever, which the paging test catches by its page bound. The finder's name is one variable in the
handler, so no single caller of `matchIndex` can name a different finder from the others.

## Ordered Steps

1. [S1] Write the tests below in `internal/mcp/astgrepindex098_test.go` and confirm
   `TestAnAstGrepIndexPagesToTheEnd` RED on `db39d42` (its `after` call is refused).
   `TestAfterWithoutGrepIsRefused` (`tools_test.go:1050`) is unchanged and must stay green. [proof: mutation]
2. [S2] Factor `afterCursor` out of `grepSpecs`, give `astGrepSpecs` the `after` parameter, pass it at the
   call site, and widen the guard. Doc comment on `afterCursor` says why the cursor is a path. [proof: mutation]
   Mutants to kill: the call site passes `""`; the guard left as `a.Grep == ""`; `afterCursor` uses `<`
   instead of `<=` (the cursor's own file is served again, caught by the no-duplicate check); the ast_grep
   branch skips `afterCursor`.
3. [S3] `matchIndex` takes the finder's name: "Send the SAME ast_grep again with after=… and repeat until
   next_index is empty", and "WITH the same ast_grep" for an ast_grep index; grep's wording otherwise
   unchanged except "empty" for "absent". [proof: mutation] Mutant: the handler's finder variable fixed to `grep`.
4. [S4] Contract §189, driving `$MRW mcp` under `bounded` with a fake `ast-grep` on PATH that reports one hit
   in each of enough files to overflow a small `--max-result-chars`: page with `after` until `next_index` is
   empty, at most 50 pages, and require every file exactly once; paired, `after` with plain specs is refused
   and names both finders.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §189's rows printed — the whole contract takes minutes and a fence runs at least three times per task, so the fence greps the section and this step runs it]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAnAstGrepIndexPagesToTheEnd|TestAfterWithoutAFinderIsRefusedNamingBoth|TestTheIndexNamesTheFinderThatMadeIt|TestAnIndexTooLargeToServePagesByFile|TestAfterWithoutGrepIsRefused|TestAnAstGrepIndexNamesACROnlyFileItRefused' -v 2>&1 | tee "$out" \
  && missing=$(for t in TestAnAstGrepIndexPagesToTheEnd TestAfterWithoutAFinderIsRefusedNamingBoth TestTheIndexNamesTheFinderThatMadeIt TestAnIndexTooLargeToServePagesByFile TestAfterWithoutGrepIsRefused TestAnAstGrepIndexNamesACROnlyFileItRefused; do grep -qE "^--- PASS: $t \(" "$out" || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 189\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && base="$(git merge-base HEAD origin/main)" \
  && eng="$(git diff -U0 "$base" -- internal/apply/apply.go internal/read/read.go)" \
  && [ -z "$(printf '%s\n' "$eng" | grep -E '^[-+][^-+]' | grep -vE '^[-+][[:space:]]*//')" ] \
  && names="$(git diff --name-only "$base" -- internal/apply internal/read)" \
  && [ -z "$(printf '%s\n' "$names" | grep -v '^$' | grep -vxE 'internal/apply/apply.go|internal/read/read.go')" ] \
  && git diff --quiet "$base" -- internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc | grep '^??')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

The paging test carries the verdict: it is red on `db39d42`, and the grep regressions beside it cannot
satisfy it. The PASS loop names every test, so a renamed one fails the fence instead of being skipped.
The engine clauses are T2's, so this fence stays green on the finished branch: a working-tree diff against
the merge-base may touch only COMMENT lines, only in `internal/apply/apply.go` and `internal/read/read.go`
(the two T2 owns), and no untracked file may appear in an engine package. Each `git diff` is its own
`&&` step, so a git failure fails the fence rather than reading as an empty diff.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAstGrepIndexPagesToTheEnd` | `internal/mcp/astgrepindex098_test.go` | with `installFakeAstGrep` reporting one hit per file for enough files (hits emitted in REVERSE path order) that the answer is an INDEX at the default ceiling: `next_index` non-empty; following `after` until it is empty, bounded at 20 pages, yields every file exactly once, each page either another index or a served read | — | S1, S2 |
| `TestAfterWithoutAFinderIsRefusedNamingBoth` | `internal/mcp/astgrepindex098_test.go` | `after` with plain specs is `isError` and its text names `grep` and `ast_grep`; `after` with `ast_grep` is not refused | — | S1, S2 |
| `TestTheIndexNamesTheFinderThatMadeIt` | `internal/mcp/astgrepindex098_test.go` | through the handler, an ast_grep index's prose says `SAME ast_grep` and never `SAME grep`; a grep index's says `SAME grep`; both say `next_index is empty` and not `next_index is absent` | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `afterCursor`, the widened guard, the finder-named prose |
| 2 — something selects it | the mrw_read handler's ast_grep branch passes `a.After`; the tests drive `call` through `callTool`, §189 through the built binary |
| 3 — the caller can discover it | the index itself says how to continue; T2 corrects the `after` description; T3 documents it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence for building it is the 2026-09-29 audit row H8 |

## Mutation Log
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · the ast_grep call site drops the cursor: page one forever · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:the index pages to the end
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · the cursor keeps its own file: served on two pages · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:the index pages to the end
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · the guard still refuses after with ast_grep · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:after without a finder is refused
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · the refusal names grep only · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:after without a finder is refused
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · the finder variable fixed to grep: an ast_grep index says resend the grep · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:the index names its finder
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/mcp/tools.go` · internal/mcp not gofmt-clean · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:the tree is gofmt-clean
- 2026-09-29 · db39d42* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package (internal/lines) changed against the merge-base · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · covers:no engine package changes

## Invariants

- `after` with neither finder is refused; `TestAfterWithoutGrepIsRefused` unchanged and green.
- The grep index's paging is unchanged (`TestAnIndexTooLargeToServePagesByFile`).
- No engine package changes in this task; `go.mod` one requirement.

## Risks

- A real ast-grep's hit order: `read.AstGrep` sorts (`astgrep.go:200`); the fake emits reverse order so a
  cursor that trusted input order would lose files and the test would say so.

## Stop Condition

Stop and ask if `TestAnAstGrepIndexPagesToTheEnd` is not red on `db39d42`, or if an existing test must
change to pass.

## Out of Scope

- The served descriptions other than the index prose — T2.
- README and AGENTS — T3.

## Verification Log
- 2026-09-29 · db39d42* · exit 1 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6487 · test-lock-sha256:79b69449c2f352188ca5a0c21cd1a8384c3c2c20b2fe70d203df193c85778517 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hc3RncmVwaW5kZXgwOThfdGVzdC5nbwlUZXN0QWZ0ZXJXaXRob3V0QUZpbmRlcklzUmVmdXNlZE5hbWluZ0JvdGgJOTUwNTUzZWNjNmRjNDE5ZjI3ZmY5NDhiN2MxM2M4ZmMxMTg0MzNlNjQ0NDY5YWE2ZTM3MWRjNmEwOWU4MjJmMwpib2R5CWludGVybmFsL21jcC9hc3RncmVwaW5kZXgwOThfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSW5kZXhQYWdlc1RvVGhlRW5kCTE4ZGY4MzQxNmZiMDkzYjI4YzNhNWIzMjU4MDczNTcxMzhjZjkwMTFkMjdkZTFkMjIyNDgxODYxOGQwZjk2YTUKYm9keQlpbnRlcm5hbC9tY3AvYXN0Z3JlcGluZGV4MDk4X3Rlc3QuZ28JVGVzdFRoZUluZGV4TmFtZXNUaGVGaW5kZXJUaGF0TWFkZUl0CThlMzY4ZWRkNzUyZjA1OTcxNDExZGRmZTg0NmY3MGU2OTc0ZWQxMWNkYWM4ZmNlNjMyOTBmMjNkOWYzNzU2N2Q
  ```
  --- last 10 line(s) of stdout (of 35 after folding 35 raw)
  --- FAIL: TestTheIndexNamesTheFinderThatMadeIt (0.82s)
  === RUN   TestAnAstGrepIndexNamesACROnlyFileItRefused
  --- PASS: TestAnAstGrepIndexNamesACROnlyFileItRefused (0.31s)
  === RUN   TestAnIndexTooLargeToServePagesByFile
  --- PASS: TestAnIndexTooLargeToServePagesByFile (4.35s)
  === RUN   TestAfterWithoutGrepIsRefused
  --- PASS: TestAfterWithoutGrepIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	6.231s
  FAIL
  ```
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6876
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6519
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6639
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6868
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6887
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6765
- 2026-09-29 · db39d42* · exit 0 · `set -o pipefail …` · acceptance-sha256:c87d4d96f4f23a231009206ad46ba3866f1ae028ad5e19e8313dc001f253b818 · ms:6787
- 2026-09-29 · human-observed · S4 observed 2026-09-29: ./scripts/contract.sh run unpiped on this branch (T1 applied), exit 0 'contract holds', with §189 printed: PASS an ast_grep index pages to the end over the built binary, and after with plain specs is refused naming both finders
