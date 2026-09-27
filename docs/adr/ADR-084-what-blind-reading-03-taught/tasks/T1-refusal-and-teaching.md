# Task ADR-084-T1: the -M refusal names the form; the text teaches a write's exits and --exclude pruning

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `-M` refusal in `plan.ParseAddr`; two `CLI()` sentences; the `--exclude` help
**Consumes:** `guide.CLI`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write -M names 1-M`, `N- and - still parse`, `instructions teach a write's exits`, `instructions teach exclude pruning`, `a contract row drives the binary`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

Blind reading 03's agents met `bad line number ""` for `@@ f -2 delete`, and were not told a write's exit 1 and 2 or that `--exclude` prunes a directory.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/minus084_test.go` | new | `-2` names `1-2`; `-`, `2-`, `1-2`, `3` still parse; a plan document gets the same words |
| `internal/guide/teach084_test.go` | new | `CLI()` carries both sentences |
| `internal/plan/plan.go` | edit | the refusal in `ParseAddr` |
| `internal/guide/guide.go` | edit | the exit sentence and the pruning clause |
| `cmd/mrw/main.go` | edit | the `--exclude` flag help |
| `AGENTS.md` | edit | the pruning sentence |
| `scripts/contract.sh` | edit | §166 |
| `docs/adr/BACKLOG.md` | edit | the entry closed |

## Ordered Steps

1. [S1] Write both tests; they fail on v1.28.0. [proof: mutation]
2. [S2] Refuse `-M` by name in `ParseAddr`; add the text. [proof: mutation]
3. [S3] Contract §166 pairs `-2` with `1-2` on the built binary and reads `mrw instructions` and `read --help`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ ./internal/guide/ -count=1 -timeout 180s -run 'TestAWriteAddressThatStartsWithMinusNamesTheForm|TestCLITeachesAWritesExitCodesAndExcludePruning' -v 2>&1 | tee /tmp/adr084-T1.out \
  && missing=$(for t in TestAWriteAddressThatStartsWithMinusNamesTheForm TestCLITeachesAWritesExitCodesAndExcludePruning; do grep -qE "^--- PASS: $t \(" /tmp/adr084-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 166\. ' scripts/contract.sh \
  && grep -q 'prunes that whole subtree' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteAddressThatStartsWithMinusNamesTheForm` | `internal/plan/minus084_test.go` | the refusal names `1-2`; the siblings parse | — | S1, S2 |
| `TestCLITeachesAWritesExitCodesAndExcludePruning` | `internal/guide/teach084_test.go` | both sentences in `CLI()` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `ParseAddr`, `CLI()`, the flag help |
| 2 — something selects it | every plan address; `mrw instructions`; `read --help` |
| 3 — the caller can discover it | the refusal names the form; the instructions say the exits and the pruning |
| 4 — it is used | blind reading 03's agents hit all three; ADR-009 refuses telemetry, so later use is not observed |

## Verification Log
(empty until execute)
- 2026-09-27 · 64d4657* · exit 1 · `set -o pipefail …` · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · ms:571 · test-lock-sha256:9218b30a7a4b399abaa320510b8ffdad65bb1c4fd725dd9241adc7cd645b4823 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL3RlYWNoMDg0X3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNBV3JpdGVzRXhpdENvZGVzQW5kRXhjbHVkZVBydW5pbmcJY2VjNmQ2ZDVjZjhlMmU5NzdjNDFhOGU2OTgxMjE4ZDE0YjY4M2JjYzJjYzM5ZThiNjUwZTIwMjY2M2EwMzAyMgpib2R5CWludGVybmFsL3BsYW4vbWludXMwODRfdGVzdC5nbwlUZXN0QVdyaXRlQWRkcmVzc1RoYXRTdGFydHNXaXRoTWludXNOYW1lc1RoZUZvcm0JMzM1M2IxM2I1YWJlMjdkMzA1NmZmYjgzMGIyZjQyOTlkMTQ4ZjUzODc5M2ZkYmI2MGI0NjJjOWI2NjRiOWZjYg
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  --- FAIL: TestAWriteAddressThatStartsWithMinusNamesTheForm (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.187s
  === RUN   TestCLITeachesAWritesExitCodesAndExcludePruning
      teach084_test.go:18: CLI() does not teach "A write exits 1 when a hunk fails validation, and nothing is written; 2 on a usage or filesystem failure."
      teach084_test.go:18: CLI() does not teach "a bare directory name prunes that whole subtree"
  --- FAIL: TestCLITeachesAWritesExitCodesAndExcludePruning (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.186s
  FAIL
  ```
- 2026-09-27 · 64d4657* · exit 0 · `set -o pipefail …` · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · ms:939
- 2026-09-27 · 64d4657* · exit 0 · `set -o pipefail …` · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · ms:1279
- 2026-09-27 · 64d4657* · exit 0 · `set -o pipefail …` · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · ms:414
- 2026-09-27 · 64d4657* · exit 0 · `set -o pipefail …` · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · ms:450

## Mutation Log
(empty until execute)
- 2026-09-27 · 64d4657* · mutant killed · exit 1 · `internal/plan/plan.go` · a write -M falls through to the empty-start parse error · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · covers:a write -M names 1-M
- 2026-09-27 · 64d4657* · mutant killed · exit 1 · `internal/guide/guide.go` · the instructions stop teaching a write exit 1 and 2 · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · covers:instructions teach a write's exits
- 2026-09-27 · 64d4657* · mutant killed · exit 1 · `internal/guide/guide.go` · the instructions stop saying a bare directory prunes · acceptance-sha256:08df9587003338ea673a0febed4a31fa8b24451f6e5bbdd7e2012a9ae55a10b0 · covers:instructions teach exclude pruning

## Invariants

- Exit codes are unchanged; `-`, `N-`, `N-M`, `N` and patterns parse as before.
- `Shared()` is unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- The read side's `-M` (permanent: boundary: a read takes `-M` as "from the start", ADR-063)

## Stop Condition

The fence exits 0 and contract §166 passes.
