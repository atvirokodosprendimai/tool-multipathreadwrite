# Task ADR-217-T1: explicit content edits in a disposable checkout

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** the benchmark executable and its contract exercise
**Consumes:** the rooted reader, checkout write lock and apply engine
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a shifted unique target and deletion land together`, `every bad match is reported before a write`, `a large batch preserves unrelated text`, `the prototype is reached as a command`

## Goal

Implement ADR-217 decisions 1–4. Verify file contents on success and failure, including the fresh match after another edit shifted a line.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/content-edit/main.go` | add | standalone comparison arm |
| `bench/content-edit/main_test.go` | add | verify final file contents |
| `scripts/contract.sh` | edit | exercise the executable |
| `docs/adr/BACKLOG.md` | edit | track the trial |
| `docs/architecture.md` | edit | register the benchmark package and command |

## Ordered Steps

1. [S1] Write the regression tests and observe red. [proof: acceptance]
2. [S2] Implement request validation, exact matching and a single guarded landing. Exercise a missing and duplicate match alongside a valid sibling. [proof: acceptance]
3. [S3] Drive the binary at contract §383 and run the final gates. [proof: acceptance]

## Acceptance

```bash
set -eu
go test -count=1 ./bench/content-edit
grep -q '^# 383\. ' scripts/contract.sh
base=$(git merge-base HEAD origin/main)
git diff --quiet "$base" -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest internal/apply internal/plan
st=$(git status --porcelain --untracked-files=all -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest internal/apply internal/plan)
[ -z "$st" ]
trial=$(mktemp -d)
trap 'rm -rf "$trial"' EXIT
go build -o "$trial/content-edit" ./bench/content-edit
mkdir "$trial/tree"
cd "$trial/tree"
printf 'target\nkeep\n' > a
printf 'keep\n' > "$trial/expected"
printf '%s\n' '[{"target":"a","old":["target"],"replacement":[]}]' | XDG_STATE_HOME="$trial/state" "$trial/content-edit"
cmp a "$trial/expected"
```

What the engine clause holds. The eleven paths have no committed change since the merge base and no uncommitted or untracked file (ignored files excepted). Until 2026-11-26 the clause compared them against `724d564e`, the commit the prototype was written on, which later engine work on `main` (#785, #787, #791) made false in every checkout. That the landing changed none of them is a fact about one commit, checked outside the fence because a comparison of two commits cannot fail: `git diff --quiet 724d564e ee8e8472` over the same paths exits 0 (`ee8e8472` is #784 on `main`; checked 2026-11-26). §383 is named by the `grep`; that it runs is the contract gate's, so it has no row in the Tests table, where only a test body can be locked.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestContentEditBatch` | `bench/content-edit/main_test.go` | scoped exact replacement and direct deletion | none | S1, S2 |
| `TestContentEditConflicts` | `bench/content-edit/main_test.go` | reports every unmatched edit and leaves the tree intact | none | S1, S2 |
| `TestContentEditInvalid` | `bench/content-edit/main_test.go` | malformed and overlapping requests leave the tree intact | none | S2 |
| `TestContentEditAfterShift` | `bench/content-edit/main_test.go` | a second edit finds its target after a line shift | none | S2 |
| `TestContentEditGeneratedBatch` | `bench/content-edit/main_test.go` | 500 edits produce the expected final tree | none | S2 |
| `TestContentEditComparisonBudget` | `bench/content-edit/main_test.go` | repeated blocks exhaust one batch-wide matching budget | none | S2 |
| `TestContentEditMixedEndings` | `bench/content-edit/main_test.go` | mixed line endings cause a refusal before landing | none | S2 |

## Verification Log
- 2026-11-22 · 724d564e* · exit 0 · `go test -count=1 ./bench/content-edit && grep -q '^# 383\. ' scripts/contract.sh && git diff --quiet 724d564e -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest internal/apply internal/plan` · acceptance-sha256:43d37e791348c9a963857caceacc07a793c1e232ff8fdda297fa3484c8de8624 · ms:474
- 2026-11-22 · 724d564e* · exit 0 · `go test -count=1 ./bench/content-edit && grep -q '^# 383\. ' scripts/contract.sh && git diff --quiet 724d564e -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest internal/apply internal/plan` · acceptance-sha256:43d37e791348c9a963857caceacc07a793c1e232ff8fdda297fa3484c8de8624 · ms:413
- 2026-11-22 · 724d564e* · exit 0 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:898
- 2026-11-22 · 724d564e* · exit 0 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:665
- 2026-11-22 · 724d564e* · exit 0 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:697
- 2026-11-22 · 724d564e* · exit 0 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:1224
- 2026-11-22 · 724d564e* · exit 1 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:757 · test-lock-sha256:13cd68078427b7237e2b92de1ccf480f19d1c706f108a1679851e4d63b218af7 · test-lock-file:locks/13cd68078427b7237e2b92de1ccf480f19d1c706f108a1679851e4d63b218af7.lock
  ```
  --- last 2 line(s) of stdout
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/bench/content-edit [build failed]
  FAIL
  --- last 9 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/bench/content-edit [github.com/atvirokodosprendimai/tool-multipathreadwrite/bench/content-edit.test]
  bench/content-edit/main_test.go:42:15: undefined: execute
  bench/content-edit/main_test.go:64:12: undefined: execute
  bench/content-edit/main_test.go:91:16: undefined: execute
  bench/content-edit/main_test.go:110:11: undefined: execute
  bench/content-edit/main_test.go:120:12: undefined: execute
  bench/content-edit/main_test.go:130:15: undefined: execute
  bench/content-edit/main_test.go:135:15: undefined: execute
  bench/content-edit/main_test.go:157:12: undefined: execute
  ```
- 2026-11-22 · 724d564e* · exit 0 · `set -eu …` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:1519
- 2026-11-25 · 1e55658d* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · ms:58 · test-lock-sha256:13cd68078427b7237e2b92de1ccf480f19d1c706f108a1679851e4d63b218af7 · test-lock-file:docs/adr/locks/13cd68078427b7237e2b92de1ccf480f19d1c706f108a1679851e4d63b218af7.lock · test-lock-kind:replace
- 2026-11-26 · 418388db* · exit 0 · `set -eu …` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:866
- 2026-11-26 · 418388db* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:11 · test-lock-sha256:fcaf27108ec17f6080984606c621bad1c8e894921541168d1729f1bb96b2e8f1 · test-lock-file:docs/adr/locks/fcaf27108ec17f6080984606c621bad1c8e894921541168d1729f1bb96b2e8f1.lock · test-lock-kind:replace
- 2026-11-26 · 418388db* · exit 0 · `set -eu …` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:1585
- 2026-11-26 · 418388db* · exit 0 · `set -eu …` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:1592
- 2026-11-26 · 418388db* · exit 0 · `set -eu …` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:1493
- 2026-11-26 · 418388db* · exit 0 · `set -eu …` · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · ms:1680
- 2026-11-26 · 89e8151f* · exit 0 · `set -eu …` · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · ms:1172
- 2026-11-26 · 89e8151f* · exit 0 · `set -eu …` · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · ms:2591
- 2026-11-26 · 89e8151f* · exit 0 · `set -eu …` · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · ms:1976
- 2026-11-26 · 89e8151f* · exit 0 · `set -eu …` · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · ms:2340
- 2026-11-26 · 89e8151f* · exit 0 · `set -eu …` · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · ms:2391

## Mutation Log
- 2026-11-22 · 724d564e* · mutant killed · exit 1 · `bench/content-edit/main.go` · Choosing the last repeated block instead of returning an ambiguous-match error must fail the batch-refusal test. · acceptance-sha256:43d37e791348c9a963857caceacc07a793c1e232ff8fdda297fa3484c8de8624
- 2026-11-22 · 724d564e* · mutant killed · exit 1 · `bench/content-edit/main.go` · Dropping nonempty replacements must fail the exact final text check for shifted lines. · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · covers:a shifted unique target and deletion land together
- 2026-11-22 · 724d564e* · mutant killed · exit 1 · `bench/content-edit/main.go` · Treating a repeated block as unique must remove the ambiguity report from the failed batch. · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · covers:every bad match is reported before a write
- 2026-11-22 · 724d564e* · mutant killed · exit 1 · `bench/content-edit/main.go` · Keeping only the last edit per target must fail the 500-edit final tree check. · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · covers:a large batch preserves unrelated text
- 2026-11-22 · 724d564e* · mutant killed · exit 1 · `bench/content-edit/main.go` · A command that claims success without calling the editor must leave the deletion unapplied. · acceptance-sha256:bb126c65793c5e229744964c199247a9180cb9fdba53c5460f73ae3e8a8376f8 · covers:the prototype is reached as a command
- 2026-11-26 · 418388db* · mutant killed · exit 1 · `bench/content-edit/main.go` · Dropping nonempty replacements must fail the exact final text check for shifted lines. · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · covers:a shifted unique target and deletion land together
- 2026-11-26 · 418388db* · mutant killed · exit 1 · `bench/content-edit/main.go` · Treating a repeated block as unique must remove the ambiguity report from the failed batch. · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · covers:every bad match is reported before a write
- 2026-11-26 · 418388db* · mutant killed · exit 1 · `bench/content-edit/main.go` · Keeping only the last edit per target must fail the 500-edit final tree check. · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · covers:a large batch preserves unrelated text
- 2026-11-26 · 418388db* · mutant killed · exit 1 · `bench/content-edit/main.go` · A command that claims success without calling the editor must leave the deletion unapplied. · acceptance-sha256:d838926738c9d26ba0468cfd28050640b856b2ec35ec0511f3d7f1ac4d4379f7 · covers:the prototype is reached as a command
- 2026-11-26 · 89e8151f* · mutant killed · exit 1 · `bench/content-edit/main.go` · Dropping nonempty replacements must fail the exact final text check for shifted lines. · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · covers:a shifted unique target and deletion land together
- 2026-11-26 · 89e8151f* · mutant killed · exit 1 · `bench/content-edit/main.go` · Treating a repeated block as unique must remove the ambiguity report from the failed batch. · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · covers:every bad match is reported before a write
- 2026-11-26 · 89e8151f* · mutant killed · exit 1 · `bench/content-edit/main.go` · Keeping only the last edit per target must fail the 500-edit final tree check. · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · covers:a large batch preserves unrelated text
- 2026-11-26 · 89e8151f* · mutant killed · exit 1 · `bench/content-edit/main.go` · A command that claims success without calling the editor must leave the deletion unapplied. · acceptance-sha256:531c375282b85eb7761c895195e28cd600ca532c803e90b7bce73e0dcdca22a9 · covers:the prototype is reached as a command

## Invariants

Validate the batch against one pre-edit version per target. Keep the baseline benchmark arms reproducible.

## Risks

A missing replacement could look like a deletion. Require an explicit array.

## Stop Condition

The acceptance fence passes and the built executable passes the end-to-end contract.

## Out of Scope

- Paid trials (deferred: docs/adr/BACKLOG.md)
