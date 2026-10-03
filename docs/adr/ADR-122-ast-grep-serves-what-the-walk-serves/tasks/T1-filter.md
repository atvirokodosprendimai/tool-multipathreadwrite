# Task ADR-122-T1: ast-grep's hits pass the walk's rules

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `read.AstGrepOptions`, `read.hitJudge`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `ast-grep's hits pass the walk's rules`

## Goal

`--ast-grep` and `ast_grep` serve the files `--grep` would, count what they drop the same way, and take `--no-ignore`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | edit | the flags, the judge, `AstGrepOptions` |
| `internal/read/walk.go` | edit | `hitJudge` over the walker |
| `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go` | edit | `--no-ignore` / `no_ignore` beside ast-grep; `skipped` |
| `internal/read/*_test.go` | edit | the callers take `AstGrepOptions{}` |
| `internal/read/astgrep122_test.go` | add | the tests |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | the `no_ignore` description |
| `AGENTS.md`, `docs/adr/BACKLOG.md`, `scripts/contract.sh` | edit | say so; close the entry; §221 |

## Ordered Steps

1. [S1] Write `TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted`, `TestAstGrepUnderNoIgnoreServesEveryHit`, `TestAstGrepIsAskedToIgnoreOnlyWhatMrwIgnores`, `TestANamedFileTheRulesIgnoreIsServedByAstGrep`. Confirm RED. [proof: mutation]
2. [S2] The flags, the judge, the wiring. Mutants: the judge skipped; a nested checkout's rules not applied; the binary test dropped; `vcs` turned off by default. [proof: mutation]
3. [S3] Docs, BACKLOG, contract §221 (an ignored directory's hit dropped and counted, the pair under `--no-ignore` served). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 600s -run 'TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted|TestAstGrepUnderNoIgnoreServesEveryHit|TestAstGrepIsAskedToIgnoreOnlyWhatMrwIgnores|TestANamedFileTheRulesIgnoreIsServedByAstGrep|TestAstGrepCountsExcludeAndNamedStartsAsTheWalkDoes' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted \(' "$out" \
  && grep -qE '^--- PASS: TestAstGrepUnderNoIgnoreServesEveryHit \(' "$out" \
  && grep -qE '^--- PASS: TestAstGrepIsAskedToIgnoreOnlyWhatMrwIgnores \(' "$out" \
  && grep -qE '^--- PASS: TestANamedFileTheRulesIgnoreIsServedByAstGrep \(' "$out" \
  && grep -qE '^--- PASS: TestAstGrepCountsExcludeAndNamedStartsAsTheWalkDoes \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 221\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted` | `internal/read/astgrep122_test.go` | ignored file, ignored directory, nested checkout's rule, binary and `.git` hits dropped and counted; a hidden file served | none | S1, S2 |
| `TestAstGrepUnderNoIgnoreServesEveryHit` | `internal/read/astgrep122_test.go` | under NoIgnore nothing is dropped | none | S1, S2 |
| `TestAstGrepIsAskedToIgnoreOnlyWhatMrwIgnores` | `internal/read/astgrep122_test.go` | the `--no-ignore` sources before `--json`, by mode | none | S1, S2 |
| `TestANamedFileTheRulesIgnoreIsServedByAstGrep` | `internal/read/astgrep122_test.go` | a named ignored file is served | none | S1, S2 |
| `TestAstGrepCountsExcludeAndNamedStartsAsTheWalkDoes` | `internal/read/astgrep122_test.go` | an excluded hit is not counted; a named start below an ignored directory is entered | none | S1, S2 |
| `TestAnEmptyAstGrepAnswerCarriesSkipped` | `internal/mcp/astgrep122_test.go` | an empty ast_grep answer reports skipped | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `hitJudge`, `astGrepIgnoresOff` |
| 2 — something selects it | `read.AstGrep` judges every hit |
| 3 — the caller can discover it | the `-- skipped:` line, `skipped`, AGENTS.md |
| 4 — it is used | BACKLOG "From ADR-116"; the plan's amendment |

## Mutation Log
- 2026-10-03 · 4b07a1b* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: every hit passes the walk judgement · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
- 2026-10-03 · 4b07a1b* · mutant survived · exit 0 · `internal/read/walk.go` · S2: a nested checkout applies its own rules below it · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-03 · 4b07a1b* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: a binary hit is dropped and counted · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
- 2026-10-03 · 4b07a1b* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: ast-grep keeps .gitignore by default, so it never opens what both skip · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
- 2026-10-03 · 4b07a1b* · mutant survived · exit 0 · `internal/read/walk.go` · S2: a nested checkout applies its own rules below it, not the outer ones (corrects the survivor above: its fixture had no outer rule reaching inside) · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-03 · 4b07a1b* · mutant killed · exit 1 · `internal/read/walk.go` · S2: a nested checkout applies its own rules below it, not the outer ones (corrects the survivors above: their fixture had no outer rule reaching inside) · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e
- 2026-10-03 · 940f271* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: --exclude before the ignore rules, so an excluded hit is not counted (the review of #329) · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f
- 2026-10-03 · 940f271* · mutant survived · exit 0 · `internal/read/astgrep.go` · S2: a named start below an ignored directory is entered, not counted (the review of #329) · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-03 · 940f271* · mutant survived · exit 0 · `internal/read/astgrep.go` · S2: a named start below an ignored directory is entered, not counted (corrects the survivor above: its only hit was under the start, so nothing marked gen/) · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-03 · 940f271* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: a named start below an ignored directory is entered, not counted, even when it serves nothing (corrects the two survivors above) · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f

## Invariants

- A named file is served whatever the rules say; `--grep`'s walk is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the judge cannot agree with the walk without changing `--grep`'s behaviour.

## Out of Scope

- A list-driven ast-grep run (permanent: boundary: Zy chose the filter)

## Verification Log
- 2026-10-03 · 4b07a1b* · exit 1 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:123 · test-lock-sha256:80bc7558dec67482c31b3eeb162d734ead94f8757e23f80700021ce1c582a639 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVUaGVSdWxlc0lnbm9yZUlzU2VydmVkQnlBc3RHcmVwCTllYjc2MDFhZmVhY2M3ODAzNGEyNjk5NzU3MjgzMzgzMzhiYjA0YWQ5ZDE2YjAyY2M5OTBhZjEwZWE0ODRjM2MKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSGl0VGhlV2Fsa1dvdWxkU2tpcElzRHJvcHBlZEFuZENvdW50ZWQJNzFjNmNhMmQzZTViNjQ4MGRhNzVlYjIzZGUzOWU3NTFjMjllN2YyNjQ0Y2RkNDcyMzM5MTUyNzJhYjlmMTdkNwpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwSXNBc2tlZFRvSWdub3JlT25seVdoYXRNcndJZ25vcmVzCWNlZjc1MWE1MTM1ODcyNmY2MWNjMmQ1ZWI4YTM5ZTIwZWMzOTJlNTQ2NDE5OTI0MzliNjc0MmIyN2E0NjY3NDIKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QXN0R3JlcFVuZGVyTm9JZ25vcmVTZXJ2ZXNFdmVyeUhpdAliYmRjMjNkMTQzOWY1NDkxMDQxZGNjYjVkZDI4MzU0MGRiZGU2M2ZmMmVhOGM1MTk2YmUzZWNlMmY0YWZmMzg5
  ```
  --- last 10 line(s) of stdout (of 24 after folding 24 raw)
  internal/read/astgrep096_test.go:238:114: too many arguments in call to AstGrep
  	have (string, []string, string, nil, unknown type)
  	want (string, []string, string, []string)
  internal/read/astgrep096_test.go:268:93: undefined: AstGrepOptions
  internal/read/astgrep096_test.go:268:93: too many arguments in call to AstGrep
  	have (string, []string, string, nil, unknown type)
  	want (string, []string, string, []string)
  internal/read/astgrep096_test.go:268:93: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:41215
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:44547
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:44019
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:44113
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:41252
- 2026-10-03 · 4b07a1b* · exit 0 · `set -o pipefail …` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:45380
- 2026-10-03 · 4b07a1b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:5e87b9fd10063c03ed4c437f0436e81ad1aac6c9da639885546b185cd6f2a31e · ms:0 · test-lock-sha256:0edbb01e89efd7a6686851b93578f8f43ea716b881808eab3024af609fc2db2f · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVUaGVSdWxlc0lnbm9yZUlzU2VydmVkQnlBc3RHcmVwCTllYjc2MDFhZmVhY2M3ODAzNGEyNjk5NzU3MjgzMzgzMzhiYjA0YWQ5ZDE2YjAyY2M5OTBhZjEwZWE0ODRjM2MKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSGl0VGhlV2Fsa1dvdWxkU2tpcElzRHJvcHBlZEFuZENvdW50ZWQJYjk0MDBkZGI5ZTc3ZjhmMmNhZmIwNjc0MmMzYjEzZWQ3NGI0MDM1NzE1NDE3ZmViMGUxZDE2ZDIyMzc2ZTM2Ngpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwSXNBc2tlZFRvSWdub3JlT25seVdoYXRNcndJZ25vcmVzCWNlZjc1MWE1MTM1ODcyNmY2MWNjMmQ1ZWI4YTM5ZTIwZWMzOTJlNTQ2NDE5OTI0MzliNjc0MmIyN2E0NjY3NDIKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QXN0R3JlcFVuZGVyTm9JZ25vcmVTZXJ2ZXNFdmVyeUhpdAliYmRjMjNkMTQzOWY1NDkxMDQxZGNjYjVkZDI4MzU0MGRiZGU2M2ZmMmVhOGM1MTk2YmUzZWNlMmY0YWZmMzg5 · test-lock-kind:replace
- 2026-10-03 · 940f271* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:0 · test-lock-sha256:cf4bfa8f5f2f38dbe1d1bfa22c818443d3a8c00392aa05dc5822049b94d71e00 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5FbXB0eUFzdEdyZXBBbnN3ZXJDYXJyaWVzU2tpcHBlZAkxNmE3YzI3MzgzNjhlODg0MTczNWUyODU4N2Q1ZDNhNGUwZWM5NmU3YjlkNGZmNjAzOWU0MDc5Njg1OGU2NDhkCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVUaGVSdWxlc0lnbm9yZUlzU2VydmVkQnlBc3RHcmVwCTllYjc2MDFhZmVhY2M3ODAzNGEyNjk5NzU3MjgzMzgzMzhiYjA0YWQ5ZDE2YjAyY2M5OTBhZjEwZWE0ODRjM2MKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSGl0VGhlV2Fsa1dvdWxkU2tpcElzRHJvcHBlZEFuZENvdW50ZWQJYjk0MDBkZGI5ZTc3ZjhmMmNhZmIwNjc0MmMzYjEzZWQ3NGI0MDM1NzE1NDE3ZmViMGUxZDE2ZDIyMzc2ZTM2Ngpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwQ291bnRzRXhjbHVkZUFuZE5hbWVkU3RhcnRzQXNUaGVXYWxrRG9lcwlhNTVlODNiZGE5NDA3NTE1MmMxYWI1Njk4OTVhYzRmZDcxMDQwYTY5MWFkNDI1ZWExZmJhYzU0M2UzZmE4NTFjCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFzdEdyZXBJc0Fza2VkVG9JZ25vcmVPbmx5V2hhdE1yd0lnbm9yZXMJY2VmNzUxYTUxMzU4NzI2ZjYxY2MyZDVlYjhhMzllMjBlYzM5MmU1NDY0MTk5MjQzOWI2NzQyYjI3YTQ2Njc0Mgpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwVW5kZXJOb0lnbm9yZVNlcnZlc0V2ZXJ5SGl0CWJiZGMyM2QxNDM5ZjU0OTEwNDFkY2NiNWRkMjgzNTQwZGJkZTYzZmYyZWE4YzUxOTZiZTNlY2UyZjRhZmYzODk · test-lock-kind:replace
- 2026-10-03 · 940f271* · exit 0 · `set -o pipefail …` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:40910
- 2026-10-03 · 940f271* · exit 0 · `set -o pipefail …` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:40348
- 2026-10-03 · 940f271* · exit 0 · `set -o pipefail …` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:40112
- 2026-10-03 · 940f271* · exit 0 · `set -o pipefail …` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:40525
- 2026-10-03 · 940f271* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3505a11333c8f5428b7deeb4b7af61b2bafef1825f0cdff26d38a06a5130328f · ms:0 · test-lock-sha256:b2af00db21b039e27f70986adb9c5860da51921f7301377ce5f711950a72f606 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5FbXB0eUFzdEdyZXBBbnN3ZXJDYXJyaWVzU2tpcHBlZAkxNmE3YzI3MzgzNjhlODg0MTczNWUyODU4N2Q1ZDNhNGUwZWM5NmU3YjlkNGZmNjAzOWU0MDc5Njg1OGU2NDhkCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFOYW1lZEZpbGVUaGVSdWxlc0lnbm9yZUlzU2VydmVkQnlBc3RHcmVwCTllYjc2MDFhZmVhY2M3ODAzNGEyNjk5NzU3MjgzMzgzMzhiYjA0YWQ5ZDE2YjAyY2M5OTBhZjEwZWE0ODRjM2MKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAxMjJfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSGl0VGhlV2Fsa1dvdWxkU2tpcElzRHJvcHBlZEFuZENvdW50ZWQJYjk0MDBkZGI5ZTc3ZjhmMmNhZmIwNjc0MmMzYjEzZWQ3NGI0MDM1NzE1NDE3ZmViMGUxZDE2ZDIyMzc2ZTM2Ngpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwQ291bnRzRXhjbHVkZUFuZE5hbWVkU3RhcnRzQXNUaGVXYWxrRG9lcwlhY2FhMjg0NjlmMGYxYWQ0NTQ0ZWNiMDM1ODdiNTY5ZTgyZmU2YjIxZjNlZTkxZjlkYTY1ZmU2YjRiZjA5OWZjCmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMTIyX3Rlc3QuZ28JVGVzdEFzdEdyZXBJc0Fza2VkVG9JZ25vcmVPbmx5V2hhdE1yd0lnbm9yZXMJY2VmNzUxYTUxMzU4NzI2ZjYxY2MyZDVlYjhhMzllMjBlYzM5MmU1NDY0MTk5MjQzOWI2NzQyYjI3YTQ2Njc0Mgpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDEyMl90ZXN0LmdvCVRlc3RBc3RHcmVwVW5kZXJOb0lnbm9yZVNlcnZlc0V2ZXJ5SGl0CWJiZGMyM2QxNDM5ZjU0OTEwNDFkY2NiNWRkMjgzNTQwZGJkZTYzZmYyZWE4YzUxOTZiZTNlY2UyZjRhZmYzODk · test-lock-kind:replace
