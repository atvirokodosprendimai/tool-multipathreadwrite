# Task ADR-069-T4: apply_patch and search_replace keep a path's trailing space; contract §131

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** exact paths out of the apply_patch and search_replace compilers
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `apply_patch keeps the path`, `search_replace keeps the path`, `the binary edits x-space through apply_patch`, `no engine file changes`

## Goal

`internal/ingest/applypatch.go:76`, `:91`, `:103`, `:110`, `:187` and `searchreplace.go:40`, `:83`
TrimSpace the path after the format's marker. The grammar needs one separator space after the
colon or marker, not a full trim. Strip exactly one leading space with `TrimPrefix(rest, " ")`, and
keep a search/replace filename line as written (a blank line is still skipped). Decide whether a path
is present on a trimmed copy, so a marker followed only by whitespace is still "no path": today a
`<<<<<<< SEARCH` with trailing blanks falls back to the previous filename, and a whitespace-only
`*** Delete File:` is refused as empty. The compiled plan already quotes a spaced path (`quotePlanPath`).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | one separator space, not TrimSpace |
| `internal/ingest/searchreplace.go` | edit | likewise; filename line kept |
| `internal/ingest/space_test.go` | new | `TestApplyPatchAndSearchReplaceKeepATrailingSpace` |
| `scripts/contract.sh` | edit | §131 |

## Ordered Steps

1. [S1] Write the test; confirm RED on an assertion. [proof: mutation]
   - apply_patch `*** Update File: x ` compiles to a hunk on `x ` (quoted), and so do Add, Delete
     and `*** Move to: d `.
   - search_replace with `x ` after `<<<<<<< SEARCH`, and as a filename line, compiles to `x `.
   - `*** Update File: my file.go` (the existing `applypatch_space_test.go` case) is unchanged.
   - `<<<<<<< SEARCH` followed by two spaces still falls back to the previous filename line, and
     `*** Delete File:` followed by only spaces is still refused as empty.
2. [S2] Change the sites; GREEN; every `internal/ingest` test stays green. [proof: mutation]
   Mutant: the Update File site goes back to TrimSpace.
3. [S3] §131 through the binary: after `read -- 'x '`, an apply_patch update of `x ` applies to
   `x ` and leaves `x` unchanged. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 131\. ' scripts/contract.sh \
  && go test ./internal/ingest/ -count=1 -v -run 'TestApplyPatchAndSearchReplaceKeepATrailingSpace' 2>&1 | tee /tmp/adr069-T4.out \
  && grep -q '^--- PASS: TestApplyPatchAndSearchReplaceKeepATrailingSpace ' /tmp/adr069-T4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T4.out \
  && ./scripts/contract.sh > /tmp/adr069-T4-contract.out 2>&1 \
  && grep -q '^  PASS  an apply_patch path keeps its trailing space' /tmp/adr069-T4-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw internal/iter internal/apply internal/ingest)" ] \
  && go vet ./cmd/mrw/ ./internal/iter/ ./internal/apply/ ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestApplyPatchAndSearchReplaceKeepATrailingSpace` | `internal/ingest/space_test.go` | both compilers emit `x `, not `x` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §131 |
| 2 — something selects it | the CLI / compiler path every caller of that surface takes |
| 3 — the caller can discover it | the header, receipt or refusal names the exact path |
| 4 — it is used | §131 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:29504
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:32016
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:219 · test-lock-sha256:d1a0b7dbeaa9ba25fe016c413459f27c7d7b4aa93274ceae432d30e1f7c2ee16 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9zcGFjZV90ZXN0LmdvCVRlc3RBcHBseVBhdGNoQW5kU2VhcmNoUmVwbGFjZUtlZXBBVHJhaWxpbmdTcGFjZQk2ZDAxMmU0MmYxODViYmMwZTI0MGQ2MGQ5ZDE5MjBlOTE2ZDExYWRhNzNlMDVkOTk1NjNhYzk3Yjc2YWE1Zjcx
  ```
  --- last 10 line(s) of stdout (of 26 after folding 26 raw)
      space_test.go:58: search_replace on the SEARCH line compiled to "x", want "x "
      space_test.go:63: search_replace filename line compiled to "x", want "x "
  --- FAIL: TestApplyPatchAndSearchReplaceKeepATrailingSpace (0.00s)
      --- FAIL: TestApplyPatchAndSearchReplaceKeepATrailingSpace/update (0.00s)
      --- FAIL: TestApplyPatchAndSearchReplaceKeepATrailingSpace/add (0.00s)
      --- FAIL: TestApplyPatchAndSearchReplaceKeepATrailingSpace/delete (0.00s)
      --- FAIL: TestApplyPatchAndSearchReplaceKeepATrailingSpace/move (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	0.006s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:29893
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:29706
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · ms:30309

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · pathAfter goes back to TrimSpace: the apply_patch and search_replace cases and §131 must go red · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · covers:apply_patch keeps the path
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · the search_replace filename line is trimmed again: its case goes red · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · covers:search_replace keeps the path
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · pathAfter goes back to TrimSpace: §131 goes red · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · covers:the binary edits x-space through apply_patch
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:caa4175e44fc7b30caf6b74a3e2f64212eb6195b7289566e18f88030e6d7e4b2 · covers:no engine file changes

## Invariants

- A path with no edge whitespace compiles exactly as before.
- `*** Update File:x` (no separator space) still compiles to `x`.

## Risks

- A patch generator that pads paths with extra spaces now names padded files; the write is refused as unread, naming them.
- Windows cannot hold `x` and `x ` apart; the space fixtures skip there, visibly.

## Out of Scope

- A path with two or more leading spaces after the marker keeps all but one (permanent: boundary: one separator space is the grammar)

## Stop Condition

Stop if keeping the path exactly needs a format change or an exit-code change.
