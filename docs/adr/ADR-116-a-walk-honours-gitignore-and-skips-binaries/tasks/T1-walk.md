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

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestIgnoreRules|TestAWalkSkipsWhatGitIgnores|TestTheIgnoreMatcherAgreesWithGit|TestTheIgnoreMatcherFoldsCaseWhereGitDoes|TestAnIgnoredDirectoryYouNameIsWalked|TestAWorktreeReadsTheCommonInfoExclude|TestAnIgnoreFileThatIsNotRegularIsNotWaitedOn' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestIgnoreRules \(' "$out" \
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
