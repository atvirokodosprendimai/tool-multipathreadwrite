# Task ADR-130-T1: a walk and an ast-grep judge prune a nested repository inside a checkout, and count it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `nestedRepo` and the `Nested` count in `internal/read/walk.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a walk and an ast-grep judge prune a nested repository inside a checkout, and count it`

## Goal

Decisions 1–5 of the record, each with a test that fails before it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/walk.go` | edit | prune and count a nested repository; `WalkSkipped.Nested`; `SkipNote` |
| `internal/read/astgrep.go` | edit | the judge drops a hit in one |
| `internal/read/nested130_test.go` | add | the tests |
| `internal/read/*_test.go`, `cmd/mrw/*_test.go`, `internal/mcp/*_test.go` | edit | tests that asserted ADR-116's walk into a nested repository, each named in the PR |
| `internal/mcp/mcp.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `internal/mcp/era_test.go`, `docs/receipts.txt` | edit | the `no_ignore` description and its golden; `skipped.nested` |
| `internal/mcp/schema_test.go` | edit | `readSchema`, the test-side declaration of `mrw_read`'s receipt keys, gains `skipped.nested` (ADR-111's test reads it) |
| `scripts/contract.sh` | edit | §233 |
| `AGENTS.md`, `README.md` | edit | the walk paragraph |

## Ordered Steps

1. [S1] Write `TestANestedRepositoryIsNotEntered` (inside a checkout: a match in `nested/` — a `.git` directory, and one with a gitfile — is not served and is counted; the same file named is served; `NoIgnore` serves it; a root that is no checkout walks each repository, and a repository nested in one of those is not entered) and `TestAnAstGrepHitInANestedRepositoryIsDropped`. Confirm RED. [proof: mutation]
2. [S2] The walk prunes and counts; the judge drops and counts; `SkipNote` names it. Mutants: the walk's prune dropped (a nested file is served); the judge's drop dropped; the "below a checkout" test answering true (a root that is no checkout prunes its top-level repositories). [proof: mutation]
3. [S3] `skipped.nested` in `docs/receipts.txt` and the schema; contract §233 — a grep in a checkout holding `nested/.git` does not serve `nested/` and says so, and the pair, `--no-ignore`, serves it; AGENTS.md and README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestANestedRepositoryIsNotEntered|TestAnAstGrepHitInANestedRepositoryIsDropped' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestANestedRepositoryIsNotEntered \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestAnAstGrepHitInANestedRepositoryIsDropped \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ ./internal/adversarial/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 233\. ' scripts/contract.sh \
  && grep -q '^mcp_read skipped.nested$' docs/receipts.txt \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANestedRepositoryIsNotEntered` | `internal/read/nested130_test.go` | a nested repository inside a checkout is pruned and counted; named, it is walked; `NoIgnore` walks it; a root that is no checkout walks each repository it holds | none | S1, S2 |
| `TestAnAstGrepHitInANestedRepositoryIsDropped` | `internal/read/nested130_test.go` | the ast-grep judge drops and counts a hit in one | none | S1, S2 |
| `TestANestedRepositoryNamedInAnotherCaseIsNotCounted` | `internal/read/nested130_test.go` | a start spelled in another case enters the repository, which is then not counted (the Codex review of #348) | none | S2 |
| `TestADirectoryBothIgnoredAndNestedCountsAlikeOnBothFinders` | `internal/read/nested130_test.go` | a directory both ignored and nested counts as ignored on the walk and the ast-grep judge alike (the Codex review of #348) | none | S2 |
| `TestTheFoldKeyMatchesEqualFold` | `internal/read/nested130_test.go` | the identity index keys names as strings.EqualFold folds them, σ and ς included (the Codex re-review of #348) | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the prune, the judge, the count |
| 2 — something selects it | `read.Walk` for every grep walk; `hitJudge.skip` for every ast-grep hit |
| 3 — the caller can discover it | the `-- skipped:` line; `skipped.nested`; AGENTS.md; README |
| 4 — it is used | BACKLOG "From the Windows peers"; no telemetry (ADR-009) |

## Invariants

- A path the caller names is walked.
- Outside a checkout, nothing changes.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a test asserting ADR-116's walk into a nested repository rests on something other than that clause.

## Out of Scope

- `core.ignorecase` (permanent: boundary: the record's)

## Mutation Log
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the walk prune dropped — a nested repository is entered · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the judge drop dropped — an ast-grep hit in a nested repository is served · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `internal/read/walk.go` · S2: below a checkout answers true — a root that is no checkout prunes its repositories · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · bef321e* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the judge prunes a nested repository named as the start (the in-process review of #348) · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · bef321e* · mutant killed · exit 1 · `internal/read/walk.go` · S2: a start spelled in another case is compared as a string — the repository it walked stays counted · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · bef321e* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the judge skips the ignored check — a directory both ignored and nested counts as nested · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da
- 2026-10-06 · 130cbbf* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the fold key keeps each rune as written — names EqualFold calls equal land in two buckets (the Codex re-review of #348) · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da

## Verification Log
- 2026-10-06 · d493df7* · exit 1 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:255 · test-lock-sha256:3ba305b676720bb39d9cfba545b403f48ebd10ad9cce55837d82868767bc7756 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QU5lc3RlZFJlcG9zaXRvcnlJc05vdEVudGVyZWQJMzQ5MGU3M2RjYTc4ODVjYWQzYTI2NTQzMWM4ODljMjczODA4NmEwNjlkMWJjNzZjMmEwYTRiN2UxNjJjZTM0YQpib2R5CWludGVybmFsL3JlYWQvbmVzdGVkMTMwX3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEhpdEluQU5lc3RlZFJlcG9zaXRvcnlJc0Ryb3BwZWQJZTJmMDA3Nzk0NjIxMmU0ZjVmODJhODJhNWZkZjkxNTgwMGU2MzhhZjQ5ZTFjNWUwZGMxZDA4MjYwNjc3YmNjNg
  ```
  --- last 7 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read.test]
  internal/read/nested130_test.go:26:72: sk.Nested undefined (type WalkSkipped has no field or method Nested)
  internal/read/nested130_test.go:34:77: sk.Nested undefined (type WalkSkipped has no field or method Nested)
  internal/read/nested130_test.go:52:72: sk.Nested undefined (type WalkSkipped has no field or method Nested)
  internal/read/nested130_test.go:73:71: sk.Nested undefined (type WalkSkipped has no field or method Nested)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:39438
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:39134
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:42337
- 2026-10-06 · bef321e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:0 · test-lock-sha256:0909cc09dc3b366a20c52cf7cf1ba3a9a753837c8f8c0031231952bc38f8650d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QU5lc3RlZFJlcG9zaXRvcnlJc05vdEVudGVyZWQJMzQ5MGU3M2RjYTc4ODVjYWQzYTI2NTQzMWM4ODljMjczODA4NmEwNjlkMWJjNzZjMmEwYTRiN2UxNjJjZTM0YQpib2R5CWludGVybmFsL3JlYWQvbmVzdGVkMTMwX3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEhpdEluQU5lc3RlZFJlcG9zaXRvcnlJc0Ryb3BwZWQJY2NmNzEyN2IzYzIxN2Q0ODRmMjE4MzM3NTFhMDhkMGJkYWJjOTZlZGY4OTA4M2RmZDJmZmFhZTI4YTI5YTE2Ng · test-lock-kind:replace
- 2026-10-06 · bef321e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:44291
- 2026-10-06 · bef321e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:0 · test-lock-sha256:49ea47547742696150a941d4155f7fdab80322ef9e2416ffe6973d692b3a2e45 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QURpcmVjdG9yeUJvdGhJZ25vcmVkQW5kTmVzdGVkQ291bnRzQWxpa2VPbkJvdGhGaW5kZXJzCThjNWVjNGFhZjlkNDcwNmMxNTQ0NWE1ZmJhYTlmNzZjODk1MTBjYmVhMjkwYmM0NDk4YzM5Y2UzMjM4M2ZmODkKYm9keQlpbnRlcm5hbC9yZWFkL25lc3RlZDEzMF90ZXN0LmdvCVRlc3RBTmVzdGVkUmVwb3NpdG9yeUlzTm90RW50ZXJlZAkzNDkwZTczZGNhNzg4NWNhZDNhMjY1NDMxYzg4OWMyNzM4MDg2YTA2OWQxYmM3NmMyYTBhNGI3ZTE2MmNlMzRhCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QU5lc3RlZFJlcG9zaXRvcnlOYW1lZEluQW5vdGhlckNhc2VJc05vdENvdW50ZWQJNWQxNzk0YjhlYjEzNzNkZDI0NjdjMTM0ODI2NjRiZGFiNTg0YzYxODdjZmZmODMzOTBhNjUwNjgxMjkwMTY0NApib2R5CWludGVybmFsL3JlYWQvbmVzdGVkMTMwX3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEhpdEluQU5lc3RlZFJlcG9zaXRvcnlJc0Ryb3BwZWQJY2NmNzEyN2IzYzIxN2Q0ODRmMjE4MzM3NTFhMDhkMGJkYWJjOTZlZGY4OTA4M2RmZDJmZmFhZTI4YTI5YTE2Ng · test-lock-kind:replace
- 2026-10-06 · bef321e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:46135
- 2026-10-06 · bef321e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:44687
- 2026-10-06 · 130cbbf* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:0 · test-lock-sha256:de1dbae0dfa1b4e0bf294607df2cf484fb564b27b4e52736b1afa730ca889e42 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QURpcmVjdG9yeUJvdGhJZ25vcmVkQW5kTmVzdGVkQ291bnRzQWxpa2VPbkJvdGhGaW5kZXJzCThjNWVjNGFhZjlkNDcwNmMxNTQ0NWE1ZmJhYTlmNzZjODk1MTBjYmVhMjkwYmM0NDk4YzM5Y2UzMjM4M2ZmODkKYm9keQlpbnRlcm5hbC9yZWFkL25lc3RlZDEzMF90ZXN0LmdvCVRlc3RBTmVzdGVkUmVwb3NpdG9yeUlzTm90RW50ZXJlZAkzNDkwZTczZGNhNzg4NWNhZDNhMjY1NDMxYzg4OWMyNzM4MDg2YTA2OWQxYmM3NmMyYTBhNGI3ZTE2MmNlMzRhCmJvZHkJaW50ZXJuYWwvcmVhZC9uZXN0ZWQxMzBfdGVzdC5nbwlUZXN0QU5lc3RlZFJlcG9zaXRvcnlOYW1lZEluQW5vdGhlckNhc2VJc05vdENvdW50ZWQJNWQxNzk0YjhlYjEzNzNkZDI0NjdjMTM0ODI2NjRiZGFiNTg0YzYxODdjZmZmODMzOTBhNjUwNjgxMjkwMTY0NApib2R5CWludGVybmFsL3JlYWQvbmVzdGVkMTMwX3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEhpdEluQU5lc3RlZFJlcG9zaXRvcnlJc0Ryb3BwZWQJY2NmNzEyN2IzYzIxN2Q0ODRmMjE4MzM3NTFhMDhkMGJkYWJjOTZlZGY4OTA4M2RmZDJmZmFhZTI4YTI5YTE2Ngpib2R5CWludGVybmFsL3JlYWQvbmVzdGVkMTMwX3Rlc3QuZ28JVGVzdFRoZUZvbGRLZXlNYXRjaGVzRXF1YWxGb2xkCWQ0ZTNjOWZlNDNlNGE5ZjQyZmQ4OWIxMTU1MmMyMzg3NjZiMjBiYWUzZWZjZGFlNWUyYTM0ZDEzY2I3ZDhkZWM · test-lock-kind:replace
- 2026-10-06 · 130cbbf* · exit 0 · `set -o pipefail …` · acceptance-sha256:6b3a8852b5bca816f0c503d6380e62764924cd2c0c74b3a55a90fd681b9187da · ms:40995
