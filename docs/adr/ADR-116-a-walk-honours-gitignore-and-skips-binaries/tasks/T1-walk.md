# Task ADR-116-T1: the walk honours .gitignore, skips binaries, and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `read.WalkOptions.NoIgnore`, `WalkSkipped`; CLI `--no-ignore`; MCP `no_ignore` and `skipped`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the walk honours .gitignore, skips binaries, and says so`

## Goal

`read.Walk` applies a checkout's `.gitignore` rules and `.git/info/exclude` with a native matcher, skips discovered binary files, counts every skip, and both surfaces say the counts and take a flag that walks as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/ignore.go` | add | the matcher |
| `internal/read/walk.go` | edit | the checkout found, rules loaded as the walk descends, pruning, binary skip, counts |
| `internal/read/ignore116_test.go`, `ignore116_unix_test.go`, `ignorefuzz116_test.go` | add | the tests |
| `cmd/mrw/main.go` | edit | `--no-ignore`, the `-- skipped:` line |
| `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema_test.go` | edit | `no_ignore`, receipt `skipped`, descriptions |
| `internal/guide/guide.go`, `internal/guide/guide_test.go` | edit | the walk sentence |
| `docs/receipts.txt`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | say so |
| `scripts/contract.sh` | edit | a row |

## Ordered Steps

1. [S1] Write `TestIgnoreRules` (a table per gitignore(5) rule), `TestAWalkSkipsWhatGitIgnores` (inside a checkout an ignored directory, an ignored file and a binary are skipped and counted; a named ignored file is served; `NoIgnore` serves all; outside a checkout `.gitignore` is not applied and a binary is still skipped), and `TestTheIgnoreMatcherAgreesWithGit` (skipped without git). Confirm RED. [proof: mutation]
2. [S2] The matcher and the walk. Mutants: negation ignored; anchoring ignored; the binary skip dropped; the checkout test dropped (rules applied outside a checkout). [proof: mutation]
3. [S3] Both surfaces, docs, receipts, the contract row (CLI `--grep` skips an ignored directory and counts it; the pair: `--no-ignore` serves it). [proof: acceptance]
4. [S4] What breaking it found, each fixed with its test: a `.gitignore` that is a FIFO hung the walk and one linked to `/dev/zero` read without end (bounded, non-blocking read); on a case-folding filesystem git ignores `A.LOG` for `*.log` and mrw served it (fold case as `git init` probes it); a worktree's or submodule's `info/exclude` was never read (follow the gitdir a `.git` file names, and its `commondir`); an ignored directory the caller named was pruned (the named start does not prune below it). Mutants: plain `os.Open`; `foldsCase` false; the gitdir file ignored; the named-start exemption dropped. [proof: mutation]
5. [S5] What the review of #315 found, each fixed with its test: a `.gitignore` that is a link was followed out of the root (not read, as git); the MCP INDEX answer carried no `skipped` and no note; the CLI note came first, not last; counts doubled for paths two walks met and counted what a named path served; a nested repository took the outer rules; bracket classes were read as regexp classes (`[[:alpha:]]` and `[z-a]` dropped the rule, a negated class matched `/`). Mutants: the link check dropped; the index's `skipped` dropped; the count's served-path test dropped; the nested checkout not recorded; a negated class matching `/`. [proof: mutation]
6. [S6] What the re-review of the fixes found: counting pruned directories was quadratic (served paths × pruned directories: 6.7 s user against 1.1 s on 20,000 packages each with an ignored `__pycache__/`; now one set, 0.8–1.1 s); a POSIX class name ran to the first `:]` anywhere, where git ends it at the first `]` and needs `:` before it; an unclosed class fell back to a literal `[`, where git never matches the rule; and matching went by character, where git goes by byte (`a?` does not match `aé`). The matcher fuzz gains those tokens and non-ASCII names, and reads git NUL-separated, since git quotes a non-ASCII path. Mutants: the `:` test dropped; an unclosed class read as a literal; matching by character. [proof: mutation]
7. [S7] What the second re-review found: under `core.ignorecase` the matcher folded with Go's `(?i)`, which also folds the Latin-1 runes bytes became, so `П*` skipped `😀.txt` and `é*` skipped `㩀.txt`. It now folds as git does — the subject's ASCII letters lowered, a literal lowered, a range and `[:upper:]` taking either case, no byte past 0x7f folded — each measured against git 2.56 with `core.ignorecase=true`; a capital class member and an escaped capital then match nothing with no case of their own, as in git (a mutant keeping a capital member survived, which is how the special case was found redundant and removed). And `(?s)`, so `**` crosses a newline in a name. Mutants: the subject not lowered; the range's lowered half dropped; `[:upper:]` not widened. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestIgnoreRules|TestAWalkSkipsWhatGitIgnores|TestTheIgnoreMatcherAgreesWithGit|TestTheIgnoreMatcherFoldsCaseWhereGitDoes|TestAnIgnoredDirectoryYouNameIsWalked|TestAWorktreeReadsTheCommonInfoExclude|TestAnIgnoreFileThatIsNotRegularIsNotWaitedOn|TestBracketClassesFollowGit|TestASkipIsCountedOncePerPath|TestANestedCheckoutHasItsOwnRules|TestALinkedGitignoreIsNotFollowed|TestClassesFoldAsGitDoes' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestIgnoreRules \(' "$out" \
  && grep -qE '^--- PASS: TestClassesFoldAsGitDoes \(' "$out" \
  && grep -qE '^--- PASS: TestBracketClassesFollowGit \(' "$out" \
  && grep -qE '^--- PASS: TestASkipIsCountedOncePerPath \(' "$out" \
  && grep -qE '^--- PASS: TestANestedCheckoutHasItsOwnRules \(' "$out" \
  && grep -qE '^--- PASS: TestAWalkSkipsWhatGitIgnores \(' "$out" \
  && grep -qE '^--- PASS: TestAnIgnoredDirectoryYouNameIsWalked \(' "$out" \
  && grep -qE '^--- PASS: TestAWorktreeReadsTheCommonInfoExclude \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ ./internal/guide/ -count=1 -timeout 600s \
  && grep -q 'no-ignore' scripts/contract.sh \
  && grep -q '^mcp_read skipped$' docs/receipts.txt \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestIgnoreRules` | `internal/read/ignore116_test.go` | each gitignore(5) rule: comments, escapes, trailing space, negation, directory-only, anchoring, `*` `?` `[...]` `**`, last match wins, no re-include under an ignored directory | none | S1, S2 |
| `TestAWalkSkipsWhatGitIgnores` | `internal/read/ignore116_test.go` | the walk skips and counts ignored directories, ignored files and binaries in a checkout; serves a named ignored file; `NoIgnore` serves all; outside a checkout only binaries are skipped | none | S1, S2 |
| `TestTheIgnoreMatcherAgreesWithGit` | `internal/read/ignore116_test.go` | over a fixture tree, the matcher's verdict equals `git check-ignore`'s for every path (skipped without git) | none | S1, S2 |
| `TestTheIgnoreMatcherAgreesWithGitOnRandomRules` | `internal/read/ignorefuzz116_test.go` | random rule files and paths, the matcher against `git check-ignore` (skipped without git; `MRW_IGNORE_FUZZ` sets the count) | none | S2 |
| `TestAnIgnoreFileThatIsNotRegularIsNotWaitedOn` | `internal/read/ignore116_unix_test.go` | a FIFO or a link to `/dev/zero` as `.gitignore` and `info/exclude` gives no rules, at once | none | S4 |
| `TestTheIgnoreMatcherFoldsCaseWhereGitDoes` | `internal/read/ignore116_test.go` | `foldsCase` gives the `core.ignorecase` `git init` set, and the walk then skips by case as git does | none | S4 |
| `TestAnIgnoredDirectoryYouNameIsWalked` | `internal/read/ignore116_test.go` | a named ignored directory is walked, rules still apply inside it, and a root walk still prunes it | none | S4 |
| `TestAWorktreeReadsTheCommonInfoExclude` | `internal/read/ignore116_test.go` | a worktree's `.git` file leads to the common `info/exclude`; a submodule's to its own | none | S4 |
| `TestBracketClassesFollowGit` | `internal/read/ignore116_test.go` | each class rule as git 2.56's check-ignore answered it: POSIX classes, reversed and chained ranges, escapes, an unknown class, never `/` | none | S5 |
| `TestClassesFoldAsGitDoes` | `internal/read/ignore116_test.go` | under core.ignorecase, each rule as git 2.56 answered it: literals fold, a class capital matches nothing, ranges and [:upper:] take either case, no non-ASCII byte folds | none | S7 |
| `TestASkipIsCountedOncePerPath` | `internal/read/ignore116_test.go` | a path two walks meet counts once; a file a named path served, and a directory a named path entered, are not counted | none | S5 |
| `TestANestedCheckoutHasItsOwnRules` | `internal/read/ignore116_test.go` | a nested repository's rules apply inside it and the outer ones do not, from the root, a named nested repo, or a directory within it; a root that is no checkout applies each repo's rules | none | S5 |
| `TestALinkedGitignoreIsNotFollowed` | `internal/read/ignore116_unix_test.go` | a `.gitignore` linked out of the root is not read: the file it names is served and nothing is counted | none | S5 |
| `TestAnIndexSaysWhatTheWalkSkipped` | `internal/mcp/skip116_test.go` | an INDEX answer carries `skipped` and the note naming `no_ignore` | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/read/ignore.go` |
| 2 — something selects it | every `--grep` walk and MCP `grep` |
| 3 — the caller can discover it | the `-- skipped:` line and `skipped` key name the counts and the flag |
| 4 — it is used | the gap list of 2026-10-01, item 4 |

## Mutation Log
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S2: negation ignored · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S2: anchoring ignored · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the binary skip dropped · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S2: the checkout test dropped, rules applied outside a checkout · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S4: a blocking, unbounded open of an ignore file · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S4: foldsCase always false · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S4: the gitdir a .git file names is ignored · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/read/ignore.go` · S4: the named-start exemption dropped · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d
- 2026-10-02 · 5b6072c* · mutant inconclusive · exit 1 · `internal/read/ignore.go` · S5: a linked .gitignore followed · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-02 · 5b6072c* · mutant killed · exit 1 · `internal/mcp/tools.go` · S5: the INDEX answer's skipped dropped · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · 5b6072c* · mutant killed · exit 1 · `internal/read/walk.go` · S5: a file a named path served still counted · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · 5b6072c* · mutant killed · exit 1 · `internal/read/walk.go` · S5: a nested checkout not recorded · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · 5b6072c* · mutant survived · exit 0 · `internal/read/ignore.go` · S5: a negated class matches / · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-02 · 5b6072c* · mutant killed · exit 1 · `internal/read/ignore.go` · S5: a negated class matches / · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · 5b6072c* · mutant killed · exit 1 · `internal/read/ignore.go` · S5: a linked .gitignore followed · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · f3ceff9* · mutant killed · exit 1 · `internal/read/ignore.go` · S6: a POSIX class name ends at the first :] anywhere · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · f3ceff9* · mutant killed · exit 1 · `internal/read/ignore.go` · S6: an unclosed class read as a literal [ · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · f3ceff9* · mutant killed · exit 1 · `internal/read/ignore.go` · S6: matching by character, not byte · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af
- 2026-10-02 · 5788397* · mutant killed · exit 1 · `internal/read/ignore.go` · S7: the subject not lowered · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418
- 2026-10-02 · 5788397* · mutant survived · exit 0 · `internal/read/ignore.go` · S7: a capital class member kept under fold · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-02 · 5788397* · mutant killed · exit 1 · `internal/read/ignore.go` · S7: the range's lowered half dropped · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418
- 2026-10-02 · 5788397* · mutant killed · exit 1 · `internal/read/ignore.go` · S7: [:upper:] not widened under fold · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418

## Invariants

- A named path is served as before; `--exclude` behaves as before; `NoIgnore` walks as ADR-007 did.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change beyond the walk sentence in `guide_test.go`.

## Out of Scope

- ast-grep's own walker (deferred: `docs/adr/BACKLOG.md` "From ADR-116")

## Verification Log
- 2026-10-02 · e1c7646* · exit 1 · `set -o pipefail …` · acceptance-sha256:92bcbfb0820a7c2d6f5b3ad5c6a02f6123dc14f7f64fa7b807c505563e418c33 · ms:393 · test-lock-sha256:dffd3cca5f871c297b64687c87117dc6f023a9e65761dd619ba92df8d095e11d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlOb0lnbm9yZSB3YWxrcyBldmVyeSByZWd1bGFyIGZpbGUJZjZiY2QzNDVhODVkOGJjZGI4NDE5NWI4OWY4YzlhNWY0ZTQ4NDY5MjU0NjVkMTdkM2JiMjQyZTY3NTQ0ZDhkNApib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdEFXYWxrU2tpcHNXaGF0R2l0SWdub3Jlcwk1ZWIyNjVmMTNjNzVkMjM5YTQ1ZmFmODEwMmZmN2U5YmI0ZTg3ZDBmOWQxODU3MDI5MzM4ODlkMzk5OTRjNzZhCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0SWdub3JlUnVsZXMJZWVkMGJiNjljMzE4NTkyOTkzM2M4YzRmMDRlZDc1YzA0ZGYwMDI3ZTY0OGYxMjQyN2FkOWZhODEzNTUxYWUxYwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJBZ3JlZXNXaXRoR2l0CWRiZTgwNDE5OGVlYzdhMTM2NDQ0ZGY4Y2EyYThkOWNjYzczY2RmZWI0MGViY2MyOTdhZjM4OTE5MDA2NTFmNTYKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCWEgbmFtZWQgaWdub3JlZCBmaWxlIGlzIHNlcnZlZAljZGY2NGQxZmU0N2U3OWY1OGVjZWZmYTI4ZTFiNDE4NDJkMzJlZDVkZjkwZTdmYmZiMzM4NTgyY2U4MDYwZGM0CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlpbnNpZGUgYSBjaGVja291dAk1YjAwN2YyOTE1MmMzNzk2NzkyZTMzOTk3ZTFhNjUyOTMzOGIwZDZlNzkxOTg4ZjcyNGIzZjYxY2NkMTdhZDZmCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlvdXRzaWRlIGEgY2hlY2tvdXQgLmdpdGlnbm9yZSBpcyBub3QgYXBwbGllZAk1ZjFjNTE3YmU1Y2YyMWVhNjk3MzY3NzE5ZGEwNmM4ZDYwMGU4ODc0OTg5MDE0MzM1NGZiZmRlZTRhNzM2ZTVk
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/read/ignore116_test.go:100:69: unknown field Skipped in struct literal of type WalkOptions
  internal/read/ignore116_test.go:120:10: undefined: WalkSkipped
  internal/read/ignore116_test.go:121:63: unknown field NoIgnore in struct literal of type WalkOptions
  internal/read/ignore116_test.go:121:79: unknown field Skipped in struct literal of type WalkOptions
  internal/read/ignore116_test.go:122:32: undefined: WalkSkipped
  internal/read/ignore116_test.go:128:10: undefined: WalkSkipped
  internal/read/ignore116_test.go:129:63: unknown field Skipped in struct literal of type WalkOptions
  internal/read/ignore116_test.go:129:63: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:22929
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:25909
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:23352
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:20752
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:22902
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:20655
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:20957
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:21651
- 2026-10-02 · e1c7646* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:beea74fd834dd17224e5aac57dff2ccd903d0cb010bdedb0ce45d8e46165785d · ms:0 · test-lock-sha256:0be1144a12a890c78a0b527a2a48812148076ef2f3ba2e22920c226f491e9844 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlOb0lnbm9yZSB3YWxrcyBldmVyeSByZWd1bGFyIGZpbGUJZjZiY2QzNDVhODVkOGJjZGI4NDE5NWI4OWY4YzlhNWY0ZTQ4NDY5MjU0NjVkMTdkM2JiMjQyZTY3NTQ0ZDhkNApib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdEFXYWxrU2tpcHNXaGF0R2l0SWdub3Jlcwk1ZWIyNjVmMTNjNzVkMjM5YTQ1ZmFmODEwMmZmN2U5YmI0ZTg3ZDBmOWQxODU3MDI5MzM4ODlkMzk5OTRjNzZhCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVdvcmt0cmVlUmVhZHNUaGVDb21tb25JbmZvRXhjbHVkZQllMmEwMmU2NWQxY2M2OWJmNzViYzEzZGJiYWU4MDU5MjdlODlkMTJhMzFkYWI4MzFhMDM1ZWNhNzY1NmMyNjIyCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QW5JZ25vcmVkRGlyZWN0b3J5WW91TmFtZUlzV2Fsa2VkCTA1ODBmZjY1ZjEzYmI0NjVjODg2MjEyNzg0M2Y5NmM0Y2E2NmU1YTRiODFkYjBlOGU3M2JkZTNjYzA4MjI3ZDQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RJZ25vcmVSdWxlcwk1YTk0ODM5ZjQxYjI3ZjFhNDI2NWQxNmY5YzYzNmY3NWVjNWNiNDJhOTg5MDMxNTM1M2M5MzM3YjdhNmYxOTE1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXQJMmQ3MjFlMzZiM2Y2Nzc1MTVjYjU0ODhkYjRhMzAzNDliMTJlMzVlN2ZiOTBhZmEyNDg3YzI4MjgwNzQ5NmZiYwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJGb2xkc0Nhc2VXaGVyZUdpdERvZXMJYWRiNWY3YTEyZWY2ZDZlZTA3N2YwYmEzZDk4OWYwYzdkNDhmNDRkMGIyZTk0YmQ2MzkyYTIwNjBhOWM3Y2Q5Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JYSBuYW1lZCBpZ25vcmVkIGZpbGUgaXMgc2VydmVkCWNkZjY0ZDFmZTQ3ZTc5ZjU4ZWNlZmZhMjhlMWI0MTg0MmQzMmVkNWRmOTBlN2ZiZmIzMzg1ODJjZTgwNjBkYzQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCWluc2lkZSBhIGNoZWNrb3V0CTViMDA3ZjI5MTUyYzM3OTY3OTJlMzM5OTdlMWE2NTI5MzM4YjBkNmU3OTE5ODhmNzI0YjNmNjFjY2QxN2FkNmYKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCW91dHNpZGUgYSBjaGVja291dCAuZ2l0aWdub3JlIGlzIG5vdCBhcHBsaWVkCTVmMWM1MTdiZTVjZjIxZWE2OTczNjc3MTlkYTA2YzhkNjAwZTg4NzQ5ODkwMTQzMzU0ZmJmZGVlNGE3MzZlNWQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl91bml4X3Rlc3QuZ28JVGVzdEFuSWdub3JlRmlsZVRoYXRJc05vdFJlZ3VsYXJJc05vdFdhaXRlZE9uCTk4MzY2OWYyYjMwMTlmMDIxNmJmNGZlMjg1MjA4Y2RiZjc3YzQ2M2MxYjI0Y2QzNjk3MzFhOWQwZDA1NzJlZjkKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZWZ1enoxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXRPblJhbmRvbVJ1bGVzCWI3ZWIxN2ZhZTRiMzUzOTgwZmQzNDY0N2Q1MTVhMDg5MzMyYzZhM2JhZGRiNzQ2YWNhMDAzYzEzNTUxMzljZmE · test-lock-kind:replace
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:24864
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:24355
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:24231
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:23976
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:24613
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:21973
- 2026-10-02 · 5b6072c* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:22727
- 2026-10-02 · 5b6072c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:0 · test-lock-sha256:4ab838a23c2e45e9f0d763e0513e7867125a375c8f26a81f98044414d5c22c76 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3NraXAxMTZfdGVzdC5nbwlUZXN0QW5JbmRleFNheXNXaGF0VGhlV2Fsa1NraXBwZWQJNTNjNzBhODQ4MGZlYjY3NzAxYWQwZTk0ODM0NzI2OGJmN2Q3ZDgxYTA1MDVjOGViMGYxNDQyZjcwZDBkMTA2Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JTm9JZ25vcmUgd2Fsa3MgZXZlcnkgcmVndWxhciBmaWxlCWY2YmNkMzQ1YTg1ZDhiY2RiODQxOTViODlmOGM5YTVmNGU0ODQ2OTI1NDY1ZDE3ZDNiYjI0MmU2NzU0NGQ4ZDQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBTmVzdGVkQ2hlY2tvdXRIYXNJdHNPd25SdWxlcwliY2IwNTdiMTIyOTUyZTA1OTEyZDFiMTY1NzA5MmMwZDczYjljODFkZmM2N2E5NmYyMTFkMTQwNWFlODg2ZWQ0CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVNraXBJc0NvdW50ZWRPbmNlUGVyUGF0aAlkODQxNGY5NmIyYzFhMDBlNWVmMzQ3Yjk1YmQ1ZWIwYjNlYTc1YmY3YWVlYjMyZTlhZmFmNGQxY2UwNDE3YzA1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVdhbGtTa2lwc1doYXRHaXRJZ25vcmVzCTVlYjI2NWYxM2M3NWQyMzlhNDVmYWY4MTAyZmY3ZTliYjRlODdkMGY5ZDE4NTcwMjkzMzg4OWQzOTk5NGM3NmEKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBV29ya3RyZWVSZWFkc1RoZUNvbW1vbkluZm9FeGNsdWRlCWUyYTAyZTY1ZDFjYzY5YmY3NWJjMTNkYmJhZTgwNTkyN2U4OWQxMmEzMWRhYjgzMWEwMzVlY2E3NjU2YzI2MjIKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBbklnbm9yZWREaXJlY3RvcnlZb3VOYW1lSXNXYWxrZWQJMDU4MGZmNjVmMTNiYjQ2NWM4ODYyMTI3ODQzZjk2YzRjYTY2ZTVhNGI4MWRiMGU4ZTczYmRlM2NjMDgyMjdkNApib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdEJyYWNrZXRDbGFzc2VzRm9sbG93R2l0CTkzYzczZjc2MjRjYjhjNGZlYjg1ZjUzMWFiODRjNTVlMzI5MmM1ZWVlYWE2Mjc2ZWE2NmYwYjUwYWY5NjQxYjMKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RJZ25vcmVSdWxlcwk1YTk0ODM5ZjQxYjI3ZjFhNDI2NWQxNmY5YzYzNmY3NWVjNWNiNDJhOTg5MDMxNTM1M2M5MzM3YjdhNmYxOTE1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXQJMmQ3MjFlMzZiM2Y2Nzc1MTVjYjU0ODhkYjRhMzAzNDliMTJlMzVlN2ZiOTBhZmEyNDg3YzI4MjgwNzQ5NmZiYwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJGb2xkc0Nhc2VXaGVyZUdpdERvZXMJYWRiNWY3YTEyZWY2ZDZlZTA3N2YwYmEzZDk4OWYwYzdkNDhmNDRkMGIyZTk0YmQ2MzkyYTIwNjBhOWM3Y2Q5Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JYSBuYW1lZCBpZ25vcmVkIGZpbGUgaXMgc2VydmVkCWNkZjY0ZDFmZTQ3ZTc5ZjU4ZWNlZmZhMjhlMWI0MTg0MmQzMmVkNWRmOTBlN2ZiZmIzMzg1ODJjZTgwNjBkYzQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCWluc2lkZSBhIGNoZWNrb3V0CTViMDA3ZjI5MTUyYzM3OTY3OTJlMzM5OTdlMWE2NTI5MzM4YjBkNmU3OTE5ODhmNzI0YjNmNjFjY2QxN2FkNmYKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCW91dHNpZGUgYSBjaGVja291dCAuZ2l0aWdub3JlIGlzIG5vdCBhcHBsaWVkCTVmMWM1MTdiZTVjZjIxZWE2OTczNjc3MTlkYTA2YzhkNjAwZTg4NzQ5ODkwMTQzMzU0ZmJmZGVlNGE3MzZlNWQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl91bml4X3Rlc3QuZ28JVGVzdEFMaW5rZWRHaXRpZ25vcmVJc05vdEZvbGxvd2VkCTAzZjk2ZWRlNzBhYTUwM2NlMDkwM2NiN2VkMjM1Y2I2ZWVhNTljZjNlNGI1OGI2ZjBjYmZlOTI2Mzk2NWZiNGUKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl91bml4X3Rlc3QuZ28JVGVzdEFuSWdub3JlRmlsZVRoYXRJc05vdFJlZ3VsYXJJc05vdFdhaXRlZE9uCTk4MzY2OWYyYjMwMTlmMDIxNmJmNGZlMjg1MjA4Y2RiZjc3YzQ2M2MxYjI0Y2QzNjk3MzFhOWQwZDA1NzJlZjkKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZWZ1enoxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXRPblJhbmRvbVJ1bGVzCThjNTI0ODA5ZjA2MjRiZDhiZGRjYjEzOTY4ZDljMGJhYTg0NjE4N2E1YTQ0YzU1MDZmYWVjOTdmYjFhNzllNGY · test-lock-kind:replace
- 2026-10-02 · f3ceff9* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:22368
- 2026-10-02 · f3ceff9* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:22270
- 2026-10-02 · f3ceff9* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:22857
- 2026-10-02 · f3ceff9* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fb27d1c140183dfaadea9a924029b8eeef7989b3fbad7368ad4a007cec8637af · ms:0 · test-lock-sha256:f90e41bc34ef5143bba6a831a79405d32fc96e5158c231a1167bb29b336ad3cc · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3NraXAxMTZfdGVzdC5nbwlUZXN0QW5JbmRleFNheXNXaGF0VGhlV2Fsa1NraXBwZWQJNTNjNzBhODQ4MGZlYjY3NzAxYWQwZTk0ODM0NzI2OGJmN2Q3ZDgxYTA1MDVjOGViMGYxNDQyZjcwZDBkMTA2Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JTm9JZ25vcmUgd2Fsa3MgZXZlcnkgcmVndWxhciBmaWxlCWY2YmNkMzQ1YTg1ZDhiY2RiODQxOTViODlmOGM5YTVmNGU0ODQ2OTI1NDY1ZDE3ZDNiYjI0MmU2NzU0NGQ4ZDQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBTmVzdGVkQ2hlY2tvdXRIYXNJdHNPd25SdWxlcwliY2IwNTdiMTIyOTUyZTA1OTEyZDFiMTY1NzA5MmMwZDczYjljODFkZmM2N2E5NmYyMTFkMTQwNWFlODg2ZWQ0CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVNraXBJc0NvdW50ZWRPbmNlUGVyUGF0aAlkODQxNGY5NmIyYzFhMDBlNWVmMzQ3Yjk1YmQ1ZWIwYjNlYTc1YmY3YWVlYjMyZTlhZmFmNGQxY2UwNDE3YzA1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVdhbGtTa2lwc1doYXRHaXRJZ25vcmVzCTVlYjI2NWYxM2M3NWQyMzlhNDVmYWY4MTAyZmY3ZTliYjRlODdkMGY5ZDE4NTcwMjkzMzg4OWQzOTk5NGM3NmEKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBV29ya3RyZWVSZWFkc1RoZUNvbW1vbkluZm9FeGNsdWRlCWUyYTAyZTY1ZDFjYzY5YmY3NWJjMTNkYmJhZTgwNTkyN2U4OWQxMmEzMWRhYjgzMWEwMzVlY2E3NjU2YzI2MjIKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBbklnbm9yZWREaXJlY3RvcnlZb3VOYW1lSXNXYWxrZWQJMDU4MGZmNjVmMTNiYjQ2NWM4ODYyMTI3ODQzZjk2YzRjYTY2ZTVhNGI4MWRiMGU4ZTczYmRlM2NjMDgyMjdkNApib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdEJyYWNrZXRDbGFzc2VzRm9sbG93R2l0CTIwMjNkY2UyMTIxYTI4ZjEzODZiN2ZkYmMwNTMwNzQyMTQzMTlkN2E0MWNhMjhlOTQzNDI4ZjE5OTE3YWJjN2YKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RJZ25vcmVSdWxlcwk1YTk0ODM5ZjQxYjI3ZjFhNDI2NWQxNmY5YzYzNmY3NWVjNWNiNDJhOTg5MDMxNTM1M2M5MzM3YjdhNmYxOTE1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXQJMmQ3MjFlMzZiM2Y2Nzc1MTVjYjU0ODhkYjRhMzAzNDliMTJlMzVlN2ZiOTBhZmEyNDg3YzI4MjgwNzQ5NmZiYwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJGb2xkc0Nhc2VXaGVyZUdpdERvZXMJYWRiNWY3YTEyZWY2ZDZlZTA3N2YwYmEzZDk4OWYwYzdkNDhmNDRkMGIyZTk0YmQ2MzkyYTIwNjBhOWM3Y2Q5Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JYSBuYW1lZCBpZ25vcmVkIGZpbGUgaXMgc2VydmVkCWNkZjY0ZDFmZTQ3ZTc5ZjU4ZWNlZmZhMjhlMWI0MTg0MmQzMmVkNWRmOTBlN2ZiZmIzMzg1ODJjZTgwNjBkYzQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCWluc2lkZSBhIGNoZWNrb3V0CTViMDA3ZjI5MTUyYzM3OTY3OTJlMzM5OTdlMWE2NTI5MzM4YjBkNmU3OTE5ODhmNzI0YjNmNjFjY2QxN2FkNmYKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCW91dHNpZGUgYSBjaGVja291dCAuZ2l0aWdub3JlIGlzIG5vdCBhcHBsaWVkCTVmMWM1MTdiZTVjZjIxZWE2OTczNjc3MTlkYTA2YzhkNjAwZTg4NzQ5ODkwMTQzMzU0ZmJmZGVlNGE3MzZlNWQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl91bml4X3Rlc3QuZ28JVGVzdEFMaW5rZWRHaXRpZ25vcmVJc05vdEZvbGxvd2VkCTAzZjk2ZWRlNzBhYTUwM2NlMDkwM2NiN2VkMjM1Y2I2ZWVhNTljZjNlNGI1OGI2ZjBjYmZlOTI2Mzk2NWZiNGUKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl91bml4X3Rlc3QuZ28JVGVzdEFuSWdub3JlRmlsZVRoYXRJc05vdFJlZ3VsYXJJc05vdFdhaXRlZE9uCTk4MzY2OWYyYjMwMTlmMDIxNmJmNGZlMjg1MjA4Y2RiZjc3YzQ2M2MxYjI0Y2QzNjk3MzFhOWQwZDA1NzJlZjkKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZWZ1enoxMTZfdGVzdC5nbwlUZXN0VGhlSWdub3JlTWF0Y2hlckFncmVlc1dpdGhHaXRPblJhbmRvbVJ1bGVzCTMxZmQ3NGIzYjkxODI0MWMzNWJkMTFlZjM0NTcxOGMyZWZkNzNiNjdhM2VkMzE1NWNkYTM1MTUwMWU4NTRiZTA · test-lock-kind:replace
- 2026-10-02 · 5788397* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418 · ms:24789
- 2026-10-02 · 5788397* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418 · ms:32311
- 2026-10-02 · 5788397* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418 · ms:28971
- 2026-10-02 · 5788397* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418 · ms:33609
- 2026-10-02 · 5788397* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d3cff562135c93c62b93d45396d8fd90b90faa4fe14b3c008535a352bd3b4418 · ms:0 · test-lock-sha256:a8c234963789487ebdff6072ce1cf0feedf3847ca658084148109ef8791f8aa6 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3NraXAxMTZfdGVzdC5nbwlUZXN0QW5JbmRleFNheXNXaGF0VGhlV2Fsa1NraXBwZWQJNTNjNzBhODQ4MGZlYjY3NzAxYWQwZTk0ODM0NzI2OGJmN2Q3ZDgxYTA1MDVjOGViMGYxNDQyZjcwZDBkMTA2Mwpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JTm9JZ25vcmUgd2Fsa3MgZXZlcnkgcmVndWxhciBmaWxlCWY2YmNkMzQ1YTg1ZDhiY2RiODQxOTViODlmOGM5YTVmNGU0ODQ2OTI1NDY1ZDE3ZDNiYjI0MmU2NzU0NGQ4ZDQKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBTmVzdGVkQ2hlY2tvdXRIYXNJdHNPd25SdWxlcwliY2IwNTdiMTIyOTUyZTA1OTEyZDFiMTY1NzA5MmMwZDczYjljODFkZmM2N2E5NmYyMTFkMTQwNWFlODg2ZWQ0CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVNraXBJc0NvdW50ZWRPbmNlUGVyUGF0aAlkODQxNGY5NmIyYzFhMDBlNWVmMzQ3Yjk1YmQ1ZWIwYjNlYTc1YmY3YWVlYjMyZTlhZmFmNGQxY2UwNDE3YzA1CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0QVdhbGtTa2lwc1doYXRHaXRJZ25vcmVzCTVlYjI2NWYxM2M3NWQyMzlhNDVmYWY4MTAyZmY3ZTliYjRlODdkMGY5ZDE4NTcwMjkzMzg4OWQzOTk5NGM3NmEKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBV29ya3RyZWVSZWFkc1RoZUNvbW1vbkluZm9FeGNsdWRlCWUyYTAyZTY1ZDFjYzY5YmY3NWJjMTNkYmJhZTgwNTkyN2U4OWQxMmEzMWRhYjgzMWEwMzVlY2E3NjU2YzI2MjIKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RBbklnbm9yZWREaXJlY3RvcnlZb3VOYW1lSXNXYWxrZWQJMDU4MGZmNjVmMTNiYjQ2NWM4ODYyMTI3ODQzZjk2YzRjYTY2ZTVhNGI4MWRiMGU4ZTczYmRlM2NjMDgyMjdkNApib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdEJyYWNrZXRDbGFzc2VzRm9sbG93R2l0CTIwMjNkY2UyMTIxYTI4ZjEzODZiN2ZkYmMwNTMwNzQyMTQzMTlkN2E0MWNhMjhlOTQzNDI4ZjE5OTE3YWJjN2YKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RDbGFzc2VzRm9sZEFzR2l0RG9lcwk0MzhjNzg0N2I3YmRhZTA5M2VhMjkxNTc3MGI2ZGYyOTA3YjVhNmY3MWE5MTI1Y2Y2Mzk2ODQxYmI3MGE2MjMwCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlUZXN0SWdub3JlUnVsZXMJNWE5NDgzOWY0MWIyN2YxYTQyNjVkMTZmOWM2MzZmNzVlYzVjYjQyYTk4OTAzMTUzNTNjOTMzN2I3YTZmMTkxNQpib2R5CWludGVybmFsL3JlYWQvaWdub3JlMTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJBZ3JlZXNXaXRoR2l0CTJkNzIxZTM2YjNmNjc3NTE1Y2I1NDg4ZGI0YTMwMzQ5YjEyZTM1ZTdmYjkwYWZhMjQ4N2MyODI4MDc0OTZmYmMKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCVRlc3RUaGVJZ25vcmVNYXRjaGVyRm9sZHNDYXNlV2hlcmVHaXREb2VzCWFkYjVmN2ExMmVmNmQ2ZWUwNzdmMGJhM2Q5ODlmMGM3ZDQ4ZjQ0ZDBiMmU5NGJkNjM5MmEyMDYwYTljN2NkOTMKYm9keQlpbnRlcm5hbC9yZWFkL2lnbm9yZTExNl90ZXN0LmdvCWEgbmFtZWQgaWdub3JlZCBmaWxlIGlzIHNlcnZlZAljZGY2NGQxZmU0N2U3OWY1OGVjZWZmYTI4ZTFiNDE4NDJkMzJlZDVkZjkwZTdmYmZiMzM4NTgyY2U4MDYwZGM0CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlpbnNpZGUgYSBjaGVja291dAk1YjAwN2YyOTE1MmMzNzk2NzkyZTMzOTk3ZTFhNjUyOTMzOGIwZDZlNzkxOTg4ZjcyNGIzZjYxY2NkMTdhZDZmCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdGVzdC5nbwlvdXRzaWRlIGEgY2hlY2tvdXQgLmdpdGlnbm9yZSBpcyBub3QgYXBwbGllZAk1ZjFjNTE3YmU1Y2YyMWVhNjk3MzY3NzE5ZGEwNmM4ZDYwMGU4ODc0OTg5MDE0MzM1NGZiZmRlZTRhNzM2ZTVkCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdW5peF90ZXN0LmdvCVRlc3RBTGlua2VkR2l0aWdub3JlSXNOb3RGb2xsb3dlZAkwM2Y5NmVkZTcwYWE1MDNjZTA5MDNjYjdlZDIzNWNiNmVlYTU5Y2YzZTRiNThiNmYwY2JmZTkyNjM5NjVmYjRlCmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmUxMTZfdW5peF90ZXN0LmdvCVRlc3RBbklnbm9yZUZpbGVUaGF0SXNOb3RSZWd1bGFySXNOb3RXYWl0ZWRPbgk5ODM2NjlmMmIzMDE5ZjAyMTZiZjRmZTI4NTIwOGNkYmY3N2M0NjNjMWIyNGNkMzY5NzMxYTlkMGQwNTcyZWY5CmJvZHkJaW50ZXJuYWwvcmVhZC9pZ25vcmVmdXp6MTE2X3Rlc3QuZ28JVGVzdFRoZUlnbm9yZU1hdGNoZXJBZ3JlZXNXaXRoR2l0T25SYW5kb21SdWxlcwkyOTg2MDAxYjJhMDVhOTgxZGE4YzFlNTgxYzM1ZTI1MDNmMjIyMzc3ZDIyNTBkZDJhOTE5ZTE5ODZiNGUyMjA0 · test-lock-kind:replace
