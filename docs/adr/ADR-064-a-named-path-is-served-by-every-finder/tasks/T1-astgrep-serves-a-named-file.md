# Task ADR-064-T1: ast-grep applies ADR-007's exclusion rule in both halves; contract §116

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** both finders apply ADR-007's rule; §116; campaign probe
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a named regular file bypasses the glob`, `a walked excluded directory prunes its hits`, `a named directory is not pruned`, `the binary serves it`

## Goal

`read.AstGrep` builds the set of named regular files and the list of named directories through `astGrepRel`.
- A named file is served even when `exclude` matches it.
- For every other hit, the hit and each ancestor directory strictly below a start that contains it meet the glob; the hit is kept if any containing start admits it, in either naming order.
- A start (each named directory, `.` for the root, or the root when nothing is named) and anything above it are never tested; a hit no named path contains is tested from the root.

This lands the first tests of ADR-007:206-210 on both finders, and §116 drives both through the built binary.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | edit | replace the `pathExcluded` call at :98 with a named/ancestor-aware check; `os` import. |
| `internal/read/leftovers_stress_test.go` | edit | four `AstGrep` tests (below). |
| `cmd/mrw/planpath_test.go` | edit | `TestANamedFileIsServedThroughGrepDespiteExclude` (the first test of ADR-007:208 on `--grep`). |
| `cmd/mrw/astgrep_test.go` | edit | `TestANamedFileIsServedThroughAstGrepDespiteExclude`. |
| `scripts/contract.sh` | edit | §116, next free after §115. |
| `scripts/break-campaign.sh` | edit | one probe, tagged `[astgrep-named-excluded]`: fake ast-grep, `--exclude` + named file, exit 0. |
| `docs/adr/BACKLOG.md` | edit | ADR-064 row; the symlink-then-`..` receipt. |

## Ordered Steps

1. [S1] Write the eight Go tests and confirm that `TestAstGrepServesANamedFileTheGlobWouldExclude`, `TestAstGrepServesANamedFileGivenAsAnAbsolutePath`, `TestAstGrepPrunesAnExcludedDirectoryItWalked`, `TestAstGrepDoesNotPruneANamedDirectory` (its `sub/gen` case), `TestAstGrepTestsAnUncontainedHitFromTheRoot` and `TestANamedFileIsServedThroughAstGrepDespiteExclude` are RED on `main`'s `astgrep.go`, each failing on an assertion. `TestAstGrepAdmitsAHitThroughAnyContainingStart` and `TestANamedFileIsServedThroughGrepDespiteExclude` are green guards before and after. [proof: mutation]
2. [S2] Implement the check in `astgrep.go`, and confirm all six are GREEN. Mutants:
   - restore the bare `pathExcluded(rel, exclude)`, killing the named tests;
   - drop the ancestor loop, killing `…PrunesAnExcludedDirectoryItWalked`;
   - test ancestors at or above a named directory too, killing `…DoesNotPruneANamedDirectory`;
   - admit a hit through the first containing start only, killing `…AdmitsAHitThroughAnyContainingStart` in one of its two orders;
   - keep a hit no named path contains without testing it, killing `…TestsAnUncontainedHitFromTheRoot`;
   - build the named set from raw `p` instead of `astGrepRel`, killing `…GivenAsAnAbsolutePath`.
   [proof: mutation]
3. [S3] Write §116 with a shell-script fake ast-grep (§111's shape, one fake per case, each printing a fixed JSON hit), with every `$MRW` call under `perl -e 'alarm shift; exec @ARGV' 5`.
   - Good: `--ast-grep D --exclude b.go b.go` exits 0 with `==> b.go`, and the `--grep 'func D'` twin does the same.
   - Must-fail: the same with no path named exits 1 `no file matched` on both finders. A hit `vendor/v.go` with `--exclude vendor` and nothing named exits 1. The same hit with `vendor` named exits 0.
   - Confirm RED against v1.22.2 in a mini-harness (the prologue helpers and §116 alone, with `MRW=$(command -v mrw)`, never built over), then GREEN in the full `./scripts/contract.sh`.
   [proof: mutation]
4. [S4] Campaign probe; BACKLOG rows; scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 116\. ' scripts/contract.sh \
  && go test ./internal/read/ ./cmd/mrw/ -count=1 -v \
    -run 'TestAstGrepServesANamedFileTheGlobWouldExclude|TestAstGrepServesANamedFileGivenAsAnAbsolutePath|TestAstGrepPrunesAnExcludedDirectoryItWalked|TestAstGrepDoesNotPruneANamedDirectory|TestAstGrepAdmitsAHitThroughAnyContainingStart|TestAstGrepTestsAnUncontainedHitFromTheRoot|TestANamedFileIsServedThroughGrepDespiteExclude|TestANamedFileIsServedThroughAstGrepDespiteExclude|TestAstGrepExcludeDropsAHit|TestExcludeDropsAMatchingFileAndPrunesADirectory' 2>&1 | tee /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepServesANamedFileTheGlobWouldExclude ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepServesANamedFileGivenAsAnAbsolutePath ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepPrunesAnExcludedDirectoryItWalked ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepDoesNotPruneANamedDirectory ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepAdmitsAHitThroughAnyContainingStart ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepTestsAnUncontainedHitFromTheRoot ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestANamedFileIsServedThroughGrepDespiteExclude ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestANamedFileIsServedThroughAstGrepDespiteExclude ' /tmp/adr064-t1.out \
  && grep -q '^--- PASS: TestAstGrepExcludeDropsAHit ' /tmp/adr064-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr064-t1.out \
  && ./scripts/contract.sh > /tmp/adr064-t1-contract.out 2>&1 \
  && grep -q '^  PASS  ast-grep serves a named file the glob matches' /tmp/adr064-t1-contract.out \
  && grep -q '^  PASS  ast-grep prunes a walked directory the glob matches' /tmp/adr064-t1-contract.out \
  && grep -q '^  PASS  grep serves a named file the glob matches' /tmp/adr064-t1-contract.out \
  && go build -o /tmp/adr064-mrw ./cmd/mrw \
  && MRW=/tmp/adr064-mrw bash scripts/break-campaign.sh > /tmp/adr064-campaign.out 2>/dev/null \
  && grep -q '^\[astgrep-named-excluded\] exit=0' /tmp/adr064-campaign.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/mcp internal/guide internal/read/walk.go \
  && [ -z "$(gofmt -l internal/read cmd/mrw)" ] \
  && go vet ./internal/read/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAstGrepServesANamedFileTheGlobWouldExclude` | `internal/read/leftovers_stress_test.go` | a named file whose name the glob matches is served by `AstGrep` | — | S1, S2 |
| `TestAstGrepServesANamedFileGivenAsAnAbsolutePath` | `internal/read/leftovers_stress_test.go` | an absolute spelling of the named file is exempt too | — | S1, S2 |
| `TestAstGrepPrunesAnExcludedDirectoryItWalked` | `internal/read/leftovers_stress_test.go` | nothing named, `exclude vendor`, hit `vendor/v.go` is dropped | — | S1, S2 |
| `TestAstGrepDoesNotPruneANamedDirectory` | `internal/read/leftovers_stress_test.go` | `vendor` named and excluded, hit `vendor/v.go` is served; `sub` named, `exclude gen`, hit `sub/gen/x.go` is dropped | — | S1, S2 |
| `TestAstGrepAdmitsAHitThroughAnyContainingStart` | `internal/read/leftovers_stress_test.go` | paths `.` and `vendor/sub` in both orders, `exclude vendor`, hit `vendor/sub/x.go` is served | — | S1, S2 |
| `TestAstGrepTestsAnUncontainedHitFromTheRoot` | `internal/read/leftovers_stress_test.go` | paths `sub`, `exclude other`, a returned hit `other/y.go` no named path contains is dropped | — | S1, S2 |
| `TestANamedFileIsServedThroughGrepDespiteExclude` | `cmd/mrw/planpath_test.go` | `--grep` serves a named excluded file (ADR-007:208 pinned) | — | S1 |
| `TestANamedFileIsServedThroughAstGrepDespiteExclude` | `cmd/mrw/astgrep_test.go` | the CLI serves it with a fake ast-grep | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the six tests and §116 |
| 2 — something selects it | `cmd/mrw/main.go:715` and `internal/mcp/tools.go:1248` call `read.AstGrep` with the caller's paths; restoring the bare `pathExcluded` fails S1 and §116 |
| 3 — the caller can discover it | `mrw instructions` teaches `--exclude` as dropping files the walk finds, and that a `.git` you name is walked; AGENTS.md teaches the named `.git` case |
| 4 — it is used | ADR-009 refuses telemetry; found by exploration, not a field report |

## Mutation Log
(empty until execute)
- 2026-09-24 · 9874f44* · mutant inconclusive · exit 1 · `internal/read/astgrep.go` · bare pathExcluded restored: the named-file and walked-directory tests and §116 must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a named regular file bypasses the glob
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · ancestor loop removed: TestAstGrepPrunesAnExcludedDirectoryItWalked and §116 must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a walked excluded directory prunes its hits
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · ancestors at or above a named start tested too: TestAstGrepDoesNotPruneANamedDirectory must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a named directory is not pruned
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · only the first containing start admits: TestAstGrepAdmitsAHitThroughAnyContainingStart must go red in one order · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a named directory is not pruned
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · an uncontained hit is kept untested: TestAstGrepTestsAnUncontainedHitFromTheRoot must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a walked excluded directory prunes its hits
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · named set built from the raw spelling: TestAstGrepServesANamedFileGivenAsAnAbsolutePath must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a named regular file bypasses the glob
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI stops passing named paths: TestANamedFileIsServedThroughAstGrepDespiteExclude and §116 must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:the binary serves it
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/read/astgrep.go` · bare pathExcluded restored (named and starts kept referenced so it compiles): the named-file and walked-directory tests and §116 must go red · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · covers:a named regular file bypasses the glob

## Invariants

- `walk.go` and `internal/guide` are byte-identical.
- A hit with nothing named still meets the glob on its own path (`TestAstGrepExcludeDropsAHit`).
- Exit codes unchanged.
- No `§NN` row in the Tests table.

## Risks

- A named spelling through a symlinked directory and then `..` is not covered; it is recorded in BACKLOG, not promised.
- The fence runs the full `contract.sh` and the campaign (about 30 s here).

## Stop Condition

- If the only way to go green is to edit `walk.go`, `internal/guide`, `internal/mcp` or an engine package, stop.
- If the only way to go green changes `--grep`'s behaviour, stop.

## Out of Scope

- `--grep`'s behaviour (permanent: boundary: ADR-007:206-210)
- Real ast-grep in CI (permanent: boundary: the fake covers the mapping; ADR-058 shells out to what is present)

## Verification Log
(empty until execute)
- 2026-09-24 · 9874f44* · exit 1 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:1371 · test-lock-sha256:e22751ab9bbf226071056947b4102cf7572833fccd03c84e2bf0455ba33eed6b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF90ZXN0LmdvCVRlc3RBTmFtZWRGaWxlSXNTZXJ2ZWRUaHJvdWdoQXN0R3JlcERlc3BpdGVFeGNsdWRlCTZjYWM3ODY3MWRmYmE3MWY2MWZlN2Q4MGMyNjliZGM2Njg2ZmQ0MjliZTI1ZjU4NDc2NmE2NWNjODVlMTM5ZDAKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0QVByZXNlbnRBc3RHcmVwV2l0aFplcm9IaXRzSXNOb3RUaGVNaXNzaW5nQmluYXJ5UGF0aAlmM2FkM2NmNWZhMDU0MzAxMmY0YWI5ZDYwNTA4ZmQwMzgyZmQ3NDU0MjA1NzkyZmNhZTU3ZWE2NzVkNTQ0OWY5CmJvZHkJY21kL21ydy9hc3RncmVwX3Rlc3QuZ28JVGVzdEFzdEdyZXBPYnNlcnZlc09ubHlTZXJ2ZWRMaW5lcwlhNjVjYjNkNTJhOWY4MjZhNzYyZjZhZmZjNjgxYWUzZjU0MjQwMTQ5ZWEzYzdlYzVlMTRjMjNjZDdkY2ZlZWE5CmJvZHkJY21kL21ydy9hc3RncmVwX3Rlc3QuZ28JVGVzdEFzdEdyZXBTZXJ2ZXNSYW5nZXNUaHJvdWdoUmVhZAkzZTMzMDY3MjY0MDAyZDNhMmMxMjdjNjE5NGQ4MjQ1NjI2MzM1ODFkODcxZjZlMzg5ZDM4ZWEyYWU1ZDA5ZjE1CmJvZHkJY21kL21ydy9hc3RncmVwX3Rlc3QuZ28JVGVzdEdyZXBBbmRBc3RHcmVwVG9nZXRoZXJBcmVVc2FnZQk1NTIyNDFmYmJjMDc5YTg0NWFjMzc2Yjc0YTkyMjNiZmU4ZDFhMWRkMGVmMzQxNmE3NTE0YzMwYzkyMWVlZWM4CmJvZHkJY21kL21ydy9hc3RncmVwX3Rlc3QuZ28JVGVzdE1pc3NpbmdBc3RHcmVwSXNVc2FnZUFuZE5hbWVzVGhlQmluYXJ5CTNjNmU1MGY0N2JkNTcwZDExOGVjNTBmNTNmZmUzNGZhZTFjMGE1YzhlN2JjM2YzZmUzZWQ1MTZjNTMwMGEwMmEKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEFNaXNzaW5nUGxhbkZpbGVTYXlzV2hlcmVJdExvb2tlZAkxZjU2NzE3OThlOGQ5ZWRjZDA2MTFmY2YyMjc2NGY0OGQ2NTlhYTQ2YmIwNGQ5YzY5NDRkYjBiOTRjYmE1ZDdiCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RBTmFtZWRGaWxlSXNTZXJ2ZWRUaHJvdWdoR3JlcERlc3BpdGVFeGNsdWRlCTFlZTg1ZmJhZjJjODcyMWY1YzIzYzk1NzYxMmQyN2MyMDZkMjFlOTllMzc0MmQ1NzYzODY2MTk2NDhkZWQxMzUKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEFQbGFuSXNSZWFkUmVsYXRpdmVUb1RoZVdvcmtpbmdEaXJlY3RvcnkJY2NkODhlZWQwZDFmNjVmOWZjY2FmYmVhMzBiMTkwMmY3YTY2Y2FlYTM0MmMzZGE3MjJjNGUwNDVhOWJlOTI3MApib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0RXhjbHVkZURyb3BzQU1hdGNoaW5nRmlsZUFuZFBydW5lc0FEaXJlY3RvcnkJYzAwZDA1N2MwZmZjYWQwNTg1NTc1YjQ0N2U3MDk4NzhkNTkxMjY5YTMxZGUzZWQwZjI3NDkwMWM2MmQxMWRiNApib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0RmlsZXNGcm9tUmVhZHNTcGVjc0Zyb21TdGRpbgkwOTcxODhjZTg2NjZmZWQ5ZTZjNjdiOTJlNDY3MDYyYzRmOGQ4NGE2MzYxZmI1ZjkwYzMyNmQ3NGY1MGEwYTkyCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RHcmVwSG9ub3Vyc0FuQWJzb2x1dGVQYXRoSW5zaWRlVGhlUm9vdAljYzFjZTI4ODY3NTkwMWY5ODM5ZGIyNTQ1Zjk3OTMyNTdlYzkxMzVkYzUxY2ZiMzVhMmM0NjMyYzNmZDM0YTFkCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RHcmVwUmVwb3J0c0FQYXR0ZXJuVGhhdE1hdGNoZWROb0ZpbGUJZGE5NTU0N2RjYTU4YzFjNGM5NzBiZWZiM2ZkODMxYWU1MmQ0YzM1ODJlNzg4YTk0MGQ2ODU2NDA0OWFiYjg3MApib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0R3JlcFJlcG9ydHNBUmVmdXNlZFBhdGhBbmRTZXJ2ZXNUaGVSZXN0CWY5NGI4ZjUzNDcxNjlhYTZkYTlkNmYyZTk5MGI2NzBmN2Q3MmNlMWZkNDc4MjY5NzU0YTY5ZmNhNTI1ZTQ3YjcKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdEdyZXBSZXNvbHZlc1dvcmtpbmdTZXRQb2ludGVycwljMjNlZWRhMDY4ZTBiZGJjOWJkNDJlNWQzNDI4MjQxMmI1MGNmODI5YzFhNGYwNDg0MjMxNTE2NDFhMmZlMjhmCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RHcmVwV2l0aE5vUGF0aHNXYWxrc1RoZVJvb3QJZTEwNDlhMTkxMDQyMTY4YzU5NGMyMWZlY2IzYTlhMDg0NjNiZGI3ZjcxNTUxMzE2OThhY2JjMjU2ZGJjOTEwMQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0Tm9Bcmd1bWVudHNXaXRob3V0R3JlcFN0aWxsUmVhZHNUaGVXb3JraW5nU2V0CWYyM2MyYzE2N2MwNmM3ZTdmZmI0YzgxOTRlMzI5MmRiZmYyYTBmMTM3ZTliZjNiZTkyYjU5MDcwZTVmYmEwYzMKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFByZWNlZGVuY2VVc2VzRmxhZ1ByZXNlbmNlTm90RW1wdGluZXNzCTViNmU0YjcxYmQ1OTRkYTljMzlmNDBjMmU1Yzc3ZmI0ODhlYzg3Y2U3MTA0ZTU3N2YyZGQyMDVmMzQwNjYzYjEKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzQ291bnRzQVJlY29yZGVkV3JpdGUJN2FmYjA4OGUwYWU0NDEwNjVkNGE4MGI2YjZkMDM1Y2JlN2Y1NjkxYmU2NDRmZDk3YjI2ZWZhMGVhNzBkOWE0ZQpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNKU09OUGFyc2VzCTQ5OWQ3MGU2ZmM4MzdmYjgxYzJlMTYxMTliYjBhNDFjMDRmNGM5Njc4YzlmNjMxMWE5ODllZjU2MTMxNzgwNjIKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzTGFuZGVkTGluZVVzZXNBcHBsaWVkUGx1c0ZhaWxlZENoZWNrUGx1c0NoZWNrTm90UnVuCTFlMWUzNWYzYWNmNjdlZmMxNTNjZDQ4ODNkNWE4NWM1MjJlMzU5ZjgzNGFjN2FjMjMyNzQyYTBmZTJjY2MxZGUKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFN0YXRzT25BRnJlc2hDaGVja291dFNheXNOb3RoaW5nSXNSZWNvcmRlZFlldAkxM2IwM2UwNGQ2ZDZkNTVkZmM2ZWUyZmY4MDg3OTgyMGUyZGJjMzgzZTBlMjQyZjUzZWJlOWZlMjRmYmFmYjk3CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c1ByaWNlc1N0cmljdEJhbGFuY2UJNDZjODE3MDg2NTZjYzlhMmQ0MTgwMzJmMzRiOGQxNWM5ZTEwNThhMDZlM2JmNmY2YTMxOWU1MGFmZmUwMWRkNgpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0U3RhdHNQcmludHNGYWlsZWRDaGVja0V2ZW5XaGVuWmVybwk1OTFkMDdkYzExYjM0NDEzZDViMTEwNDdkYjlmYmZjM2QyM2U0MjllYTVjYjZhN2YzYjBjN2NkMjQ3OTZiZjU3CmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c1ByaW50c1RoZURlbm9taW5hdG9yQmVzaWRlVGhlUmF0ZQllOGIwZTI4NjIyZmRiNDhjMjIxZDFhNTFmZTExNmNkOTQyYjVlMGUzMWZiMjc1ZWU2YjBlYmYxMjJhM2NkNDNkCmJvZHkJY21kL21ydy9wbGFucGF0aF90ZXN0LmdvCVRlc3RTdGF0c1Jlc2V0RW1wdGllc0FuZFJlcG9ydHNXaGF0SXREaXNjYXJkZWQJZWE0OTA2MmExOWYyNjRlYzFhM2M1Y2JiNzhiYTM3M2E3NzA2YzZhNGQ2ZjZlMWFlZTJlYWNlZGFlNjVkMWZlNwpib2R5CWNtZC9tcncvcGxhbnBhdGhfdGVzdC5nbwlUZXN0VGhlRG9jdW1lbnRlZFVzYWdlRXJyb3JzQXJlRXJyb3JzCTQxY2YyYTQ1NjI1YmE0Zjc3NDU3YjQ4OGVmMTY5ZWFjOTkwNDM5NzcyMWMzYTIxNmM0ZWIxYTI5ZGFhYTJkYzYKYm9keQljbWQvbXJ3L3BsYW5wYXRoX3Rlc3QuZ28JVGVzdFRoZVBsYW5FcnJvcldyYXBzVGhlQ2F1c2UJZjY4MzJkMjdkYzFhOWY4N2MxZDE0NDgxNWJmMWMxMTE3ZDIzNmEzMzI1ZDY2NDMzYTczZjVmMGUzOTU4ZjM2Nwpib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCUZ1enpFbmdsaXNoV29yZFRva2VuCTM2ZTMwOWNhNzYwMDM1Yzc4MTMwMmY2ZTE0MWNlYzJhODNmZGEwOTc0N2FhZjBjOTQ4ZDM2YjNlNGU3ZTViNDUKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlGdXp6UGFyc2VBc3RHcmVwSlNPTglhMGJhYjI4MTk5NmVlNDI2ZDI3YzJhZjRkOTMxNzlmODU2YjQzOWVlMzg1YzAyNjliOTRiYWZkYzMwYTFjMTg5CmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFSZWFkYWJsZUVuZ2xpc2hOYW1lZEZpbGVEb2VzTm90SGludAlkNTEwYjU5ODM3NDhjMzJhN2YzNjMyYmJmYzkzOTI3ZDg2ZmMwMDZhODk0ODUyNjQ5OTcyOTBkZDQxZWZmYWRjCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFTbGFzaFBhdGhJc05vdEFuRW5nbGlzaFRva2VuCTk0NTA0OTQ4ZmUzNWNhNzkxZjVhMzI3ZTBhNTViODYwNjUyYmM0OWRjZjgzMzJjMGQ1OTk0ODZlNjcwMDgzNTkKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0QW5FeGlzdGluZ0ZpbGVEb2VzTm90UGFkVGhlRW5nbGlzaENvdW50CTE0YWM4N2MwZTExNDg2Njk3YWY0Njg4MGZmMTMzZDVlNDdiMGQzZWI4ZjA5YzQ0ODNlOTdhYTBkYmMxNmRiNTIKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0QXN0R3JlcEFkbWl0c0FIaXRUaHJvdWdoQW55Q29udGFpbmluZ1N0YXJ0CTllY2RkZGQyNDlmZTNmYTI0MTdiMjgxODFmMjVhZDc0ZGIwNTlmYjE2MTI2NGQzYzFmYWJhMjJkMDhmNWY1NGEKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0QXN0R3JlcERvZXNOb3RQcnVuZUFOYW1lZERpcmVjdG9yeQkyODU0ODZkZjU1OThhODJmNTIxYjc4MTI2MTY3ODVlNTZiYmNhMDJmN2VkZDdmZmI0MzFlYWI4NGRkMGQ3NjgxCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBFbXB0eUZpbGVGaWVsZElzTm90U2VydmVkCTU3N2JiMDI1YzgyNDQ2MmI2NjJmM2VhMDQyYjBhMzFkYTkzZjI4YjVmMTNmMTRkMjlmNWY1MmRmMjllNThhZjgKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0QXN0R3JlcEVuZEJlZm9yZVN0YXJ0Q29sbGFwc2VzVG9BUG9pbnQJZTBkNzY2YjdhODE4ZjVhMGUzODRkMDM4OGMxMzQ0MjQwOWFmNWJhMDNhM2U5OGU4YTQ3NzdiNTUxMWU2ODU0ZQpib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3RBc3RHcmVwRXhjbHVkZURyb3BzQUhpdAljY2U4MmUyZWUwYzMxMzBhOWNhMTg4YzNjNWZmMjhlOGE0ZTg2YmY5NjVhMjlmNDYwMWI5MmQ2OWRiMDhjNjk2CmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBIaXRPdXRzaWRlVGhlUm9vdElzQVByb2JsZW1Ob3RBU3BlYwk0ODU5ZDljY2Q2OWI4N2M1NWU2ZmI0M2EyMDMwZmY1NWQwMjZmNzgzMmMyYjdhZWVkY2UxNjA2ZjQ5NWU5OTI4CmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBKU09OT2JqZWN0SXNOb3RaZXJvSGl0cwllNjFjMmFiOTMzMzJlNWRiY2JhZGUwYTZjYTcyNzFmM2E5ODc4NGE1NjcwYThlNDVhYjgxOWYxYzlhZjM4NWZlCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBOZXZlclJlY29yZHMJN2VjNjQzMzk3ZjllNzI1YWM5OGZkYWI3NDUwZGM5ZDRhNjliYzc5Y2YyYWI2YmY0MjI4MGM0ZWUzODIwZjAwOQpib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3RBc3RHcmVwUGF0aEtleUlzQWNjZXB0ZWRMaWtlRmlsZQk1ZDhmY2Y4M2IxOTRhZDA4NWFmNGE4NTY5OGU0ZjZmMTY4ODg3YmQwZTQ4NDAwZjI0YTFkODgzNWEzZGY0MjZiCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBQcmVzZW50QmluYXJ5RXhpdE9uZVdpdGhFbXB0eUFycmF5SXNaZXJvSGl0cwk5YzczYzRkYWVjY2EzN2ExOTE2YzZlNTA0YWNlMGEyMjU3ZmIzMDU2YmFlNWU2OGQyZWVhMGVlMzk2OTE4MWIxCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBQcnVuZXNBbkV4Y2x1ZGVkRGlyZWN0b3J5SXRXYWxrZWQJODhmOWVkMTEwMDU3YWQzOTUxOTFhMzIxNzdmODAwNDAxNTVlYWQzNTAzNTFmZjI2MjMyN2RiZmJmMjA0MGE2MApib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3RBc3RHcmVwU2VydmVzQU5hbWVkRmlsZUdpdmVuQXNBbkFic29sdXRlUGF0aAkzZjVlNzk4ZDUwMTQzODU5MGNmNTQ5MmVhZTQwMjNlMWUxOGFmN2E3OWI4ZmRhNDM0MTQxZDMxNjgzZmE3MzZjCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBTZXJ2ZXNBTmFtZWRGaWxlVGhlR2xvYldvdWxkRXhjbHVkZQk5MDVmNjYzNGYwODU4YWUxZWQ3NmE1NWFlN2MyYzZiODI0YmJkODJlNWE2YmEwM2Y1Y2MzYWY2YzEyYzMwNGQ3CmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBUZXN0c0FuVW5jb250YWluZWRIaXRGcm9tVGhlUm9vdAk1ZmFhZDMwNzAzZTZmYTJlZDMyZDAyMmQwZGJlNzQyYzU0ODVmMDZlNDYxZTc0YzE0MWRlMTU4ZmYxNzBkOTlmCmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBaZXJvQmFzZWRMaW5lT25lU2VydmVzTGluZVR3bwkyOGExMWRhYWVmYzc4Y2FkNWQwOWI3ZjBlNDFlNmEwYTYyZjQ0NzRlY2I2MDBhNzk4OTU0ODk1Yzk1MjA2OGQ5CmJvZHkJaW50ZXJuYWwvcmVhZC9sZWZ0b3ZlcnNfc3RyZXNzX3Rlc3QuZ28JVGVzdEFzdEdyZXBaZXJvQmFzZWRMaW5lWmVyb1NlcnZlc0xpbmVPbmUJZDE5MjViNWE3ZTU4YmQ1ZWQ3ZDBiNzEwZGNmYzhkNWIyNmQyZjJlMzIxZDk0Y2E4MWE0NTFjNTkyNzlhZWI1Ygpib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3REaWdpdExlYWRpbmdUb2tlbnNTdGF5U2lsZW50CThlZjVkMzAzZTg4MWZjY2FlOWU4MzAwNzVkMTBhZjM2MzMwYWJkNDVjNzI3NzBjZTYyYjYyMjZkNTE2Y2JjZjEKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0RHVwbGljYXRlRW5nbGlzaFdvcmRTdGlsbENvdW50c1R3aWNlCTZhZWE2YTZhMDNkZGU5NTc5NDQzNzY3ODk1ZGEwMmY5N2Y2NzVkNzZjZTU2Mzc3YTBkYTYwNTg0MjM1N2Q2MjMKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0RW5nbGlzaFdvcmRUb2tlbk1hdGNoZXNUaGVTcGVjUmVnZXgJMzQwZGY2MjExNzY2ZTZjNTQ5YTI5NmEzM2U2YTc2ZjU0NzVlNzY4YzViNWI4MDEzZGMzNzQ0MzJkYzE4NWI5YQpib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3RFeGFjdGx5VHdvRW5nbGlzaFRva2Vuc0ZpcmVUaGVIaW50CTExOGQxZjlmODlmNjdiZWNjYWI5MTgzY2FmNjUxYzAxZTI2OTA1MTJkZDY1NDQyYTU3MWU1NzRhNWZmOWUzOWYKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0T25lRW5nbGlzaEFuZE9uZUdsb2JEb2VzTm90RmlyZVRoZUVuZ2xpc2hIaW50CTY0MmM0OTNiZDI2MWNjNWFiMzMzNzU1NWFiZjFhZmMyNWIwOGJhY2VhNjhiNzUxMjU3MWM3ZjVkYTZjOTA5YzYKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0UmFuZG9taXNlZEVuZ2xpc2hIaW50Rm9sbG93c1RoZVNwZWNPcmFjbGUJYTUxMDM2MjYzNGNjZjA3ZjFhN2UyZTE4MDhmZmFjMTczNDU5ZjMzM2FkMTBiZjllOWVjODRmZGQ0Y2ZmYWQ0NApib2R5CWludGVybmFsL3JlYWQvbGVmdG92ZXJzX3N0cmVzc190ZXN0LmdvCVRlc3RUd29NaXNzaW5nRG90dGVkRmlsZXNTdGF5U2lsZW50CTg3NWIyMjU1ZDcyNGY5M2RiNzBkZmMwYTRjYWFhYjc1OGI0NDg1ZTczMjYwMDQ0YmQzMGRkN2JhNDYxNTk2NWQKYm9keQlpbnRlcm5hbC9yZWFkL2xlZnRvdmVyc19zdHJlc3NfdGVzdC5nbwlUZXN0V2Fsa1NvdXJjZU5hbWVzTm9Bc3RHcmVwCTVhNTlmOWM3NDhkYjYwYjE5OTA0NWM5OTMxNzVjMzcwNjQ2NWI1NDMzN2RiNzNlOGQwNzNiODMzOTQyMmZhN2E
  ```
  --- last 10 line(s) of stdout (of 31 after folding 31 raw)
  === RUN   TestANamedFileIsServedThroughAstGrepDespiteExclude
      astgrep_test.go:137: a named excluded file must be served: no file matched /D/
  --- FAIL: TestANamedFileIsServedThroughAstGrepDespiteExclude (0.12s)
  === RUN   TestExcludeDropsAMatchingFileAndPrunesADirectory
  --- PASS: TestExcludeDropsAMatchingFileAndPrunesADirectory (0.00s)
  === RUN   TestANamedFileIsServedThroughGrepDespiteExclude
  --- PASS: TestANamedFileIsServedThroughGrepDespiteExclude (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.132s
  FAIL
  ```
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:29835
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:26937
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:34015
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:43132
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:38096
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:39185
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:50607
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:eea5b6228ba9d8c39437ca8094d0e2739374210a512314d3d7dd142621e5f961 · ms:27514
