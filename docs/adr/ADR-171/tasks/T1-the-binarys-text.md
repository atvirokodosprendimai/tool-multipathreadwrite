# Task ADR-171-T1: the binary teaches `body=` as optional, a plan in a file, and when to re-read

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the new core header, rule 1 and rule 4, recipe 5 and `body=` sentences in `internal/guide/guide.go`; the `body=` line of `mrw write --help`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `core rule 4 says body= is optional`, `the full text names every case that needs a count`, `the core header says when to re-read`, `recipe 5 sends the plan in a file`, `the write help says body= is optional`, `the core stays under 300 words and the full contract carries it`, `the binary builds`, `the contract row drives the binary`, `the engine packages are unchanged`

## Goal

ADR-171 Decisions 1-3 and 5 on the binary's text: `mrw instructions --core`, `mrw instructions`, `mrw write --help` and `mrw skill` stop teaching an agent to count every body, to re-read the rules after a compaction, and to send a plan through the shell.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `Core()` header, rule 1, rule 4 (`:125-129`); recipe 5 (`:52-53`); the `body=` sentences of `CLI()` (`:92-95`) |
| `internal/guide/cost171_test.go` | new | the tests below |
| `internal/guide/core141_test.go` | edit | rule 4's pinned phrases (`:24-25`) and the header prefix (`:44`) change with the text |
| `cmd/mrw/main.go` | edit | `mrw write --help`'s `body=` lines |
| `scripts/contract.sh` | edit | a new section driving `$MRW instructions --core`, `$MRW instructions` and `$MRW write --help`: the new guidance present, the old absent |

## Ordered Steps

1. [S1] Write `cost171_test.go` and see it RED against today's text. [proof: acceptance]
2. [S2] Core rule 4 and the full text's count sentences (Decision 1); `core141_test.go`'s pinned phrases follow. Mutant: rule 4 back to "body= is a line count and goes on the header". [proof: mutation]
3. [S3] The core header (Decision 2), within 300 words. Mutant: the header back to "(mrw instructions prints the whole contract)". [proof: mutation]
4. [S4] Recipe 5 and rule 1's `mrw write PLANFILE` (Decision 3). Mutant: recipe 5 back to the `printf … | mrw write -` form. [proof: mutation]
5. [S5] `mrw write --help`'s `body=` line and the contract row (Decision 5). Mutant: the help line back. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ ./cmd/mrw/ -count=1 -v -run '^(TestTheCoreTeachesTheBodyCountAsOptional|TestTheFullTextNamesEveryCaseThatNeedsACount|TestTheCoreSaysWhenToReRead|TestRecipeFiveSendsThePlanInAFile|TestTheWriteHelpSaysBodyIsOptional|TestTheCoreIsEightShortRulesTheFullContractStillCarries)$' 2>&1) \
  && for t in TestTheCoreTeachesTheBodyCountAsOptional TestTheFullTextNamesEveryCaseThatNeedsACount TestTheCoreSaysWhenToReRead TestRecipeFiveSendsThePlanInAFile TestTheWriteHelpSaysBodyIsOptional TestTheCoreIsEightShortRulesTheFullContractStillCarries; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go build -o bin/mrw ./cmd/mrw \
  && c=$(./bin/mrw instructions --core) && grep -qF 'omit body= unless' <<<"$c" && grep -qF 'after a compaction only if lost' <<<"$c" \
  && f=$(./bin/mrw instructions) && grep -qF 'mrw write "$TMPDIR/plan.txt"' <<<"$f" \
  && grep -qF "ADR-171" scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCoreTeachesTheBodyCountAsOptional` | `internal/guide/cost171_test.go` | rule 4 says to leave body= out and points at the reference; the old clause is gone | none | S1, S2 |
| `TestTheFullTextNamesEveryCaseThatNeedsACount` | `internal/guide/cost171_test.go` | the full text names body=0 create, any body line beginning `@@ ` (with raw=true for a valid header), a first line beginning body=, that from= and body=@path take none, and that a count cannot catch a hunk left out | none | S1, S2 |
| `TestTheCoreSaysWhenToReRead` | `internal/guide/cost171_test.go` | the header says to read the rules once a session and after a compaction only if lost | none | S1, S3 |
| `TestRecipeFiveSendsThePlanInAFile` | `internal/guide/cost171_test.go` | recipe 5's command writes a plan file and no recipe pipes "$plan" | none | S1, S4 |
| `TestTheWriteHelpSaysBodyIsOptional` | `cmd/mrw/cost171_test.go` | `mrw write --help` says body= is optional for an ordinary body | none | S1, S5 |
| `TestTheCoreIsEightShortRulesTheFullContractStillCarries` | `internal/guide/core141_test.go` | under 300 words, every rule carried by the full contract | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the text in `guide.go` and the write help |
| 2 — something selects it | `mrw instructions`, `--core`, `mrw skill`, `mrw write --help` print it |
| 3 — the caller can discover it | the standing trigger runs `--core`; the skill carries it |
| 4 — it is used | the Rubie runs are the evidence it was read; no telemetry (ADR-009) |

## Mutation Log
(empty until execute)
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S2 rule 4 back to the line-count clause · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S3 header back to the whole-contract pointer · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S4 recipe 5 back to the printf pipe form · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5 the write help line back · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S2 rule 4 back to the line-count clause · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:core rule 4 says body= is optional
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S2 the full text drops the empty-create case · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:the full text names every case that needs a count
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S3 header back to the whole-contract pointer · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:the core header says when to re-read
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S4 recipe 5 back to the printf pipe form · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:recipe 5 sends the plan in a file
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5 the write help line back · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:the write help says body= is optional
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S3 the core grows past 300 words · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · covers:the core stays under 300 words and the full contract carries it

## Invariants

- The engine packages do not change (the fence's last clause).
- Core stays eight rules, under 300 words (ADR-141).

## Risks

- None beyond ADR-171's.

## Stop Condition

The fence exits 0, and `./scripts/contract.sh` holds.

## Out of Scope

- The docs, the regenerated skill and the BACKLOG receipt (deferred: ADR-171 T2)
- The MCP instructions (permanent: boundary: ADR-171 Decision 5 leaves them unchanged)

## Verification Log
(empty until execute)
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:939 · test-lock-sha256:385729bffe53069117e1010743cb47ad7b5b9be1314b365c66c0a25f6e9b18cf · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3N0MTcxX3Rlc3QuZ28JVGVzdFRoZVdyaXRlSGVscFNheXNCb2R5SXNPcHRpb25hbAljZGUxNjQ0OGI4OWNmMDFmOGFmODRhNjM4NzI3MDEzOTVlMjM0YmJjYzcwNGJmNGQ5MjUzMGI3YThjNGRhZWRhCmJvZHkJaW50ZXJuYWwvZ3VpZGUvY29yZTE0MV90ZXN0LmdvCVRlc3RUaGVDb3JlSXNFaWdodFNob3J0UnVsZXNUaGVGdWxsQ29udHJhY3RTdGlsbENhcnJpZXMJZTNjZWM0Y2ZkNGYwNTIwZWIxMjZhODU0NDMxYTM4ZmM1NTQ3ZWY4ZDg3ZmQ0YzU1M2MyY2NhMzU1ZTM0ODM0Ywpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0UmVjaXBlRml2ZVNlbmRzVGhlUGxhbkluQUZpbGUJZTM2ZjFiYTJiMWVkNjA1MWQzYjU4M2NjOThmNjFmMTRiYjgyOGViYWI3ZjM2ZmM1ODU5YzI1YmI3MDQ2OGE2Ygpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0VGhlQ29yZVNheXNXaGVuVG9SZVJlYWQJZTRhYmJmOGMzMzI1OTExMzFmNzZmY2Y1NjU4MDk2YzU2Y2JhN2M0YTQxYmFmMTVlNDdiOTQxZTg4YjY1YzM3Ngpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0VGhlQ29yZVRlYWNoZXNUaGVCb2R5Q291bnRBc09wdGlvbmFsCWRmNWRhNTViZDE5ODlhNjUxYjMxMTczMmM1N2M5ZDk0ODdlMzMzM2EyYWIyZDAzYzdkOGYzYWRmMTIzMjc1MzAKYm9keQlpbnRlcm5hbC9ndWlkZS9jb3N0MTcxX3Rlc3QuZ28JVGVzdFRoZUZ1bGxUZXh0TmFtZXNFdmVyeUNhc2VUaGF0TmVlZHNBQ291bnQJYjA2MmE4MmZlYjEzYzBjNWIxYWNjZTJmMDkwMDg4NmM5ZDJkZmEzYmI1NjdiNGNmZmU1NWQ0NzIyNDdkNmZhOQ
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1039
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1838
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1890
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1689
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1915
- 2026-10-24 · 8278fcda* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:0 · test-lock-sha256:5b08992e009b136b266682298c4a48542bda1e7662eaf3bf24f53a90269be331 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3N0MTcxX3Rlc3QuZ28JVGVzdFRoZVdyaXRlSGVscFNheXNCb2R5SXNPcHRpb25hbAljZGUxNjQ0OGI4OWNmMDFmOGFmODRhNjM4NzI3MDEzOTVlMjM0YmJjYzcwNGJmNGQ5MjUzMGI3YThjNGRhZWRhCmJvZHkJaW50ZXJuYWwvZ3VpZGUvY29yZTE0MV90ZXN0LmdvCVRlc3RUaGVDb3JlSXNFaWdodFNob3J0UnVsZXNUaGVGdWxsQ29udHJhY3RTdGlsbENhcnJpZXMJN2JjNGIxOWE1Y2E2OTA1ZmIwNGViNzJhMTliMzNhYWUyMWViNTEzN2VlNTQ0YzM1ZDhlOGVmMDVlMzA4NjUwNQpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0UmVjaXBlRml2ZVNlbmRzVGhlUGxhbkluQUZpbGUJZTM2ZjFiYTJiMWVkNjA1MWQzYjU4M2NjOThmNjFmMTRiYjgyOGViYWI3ZjM2ZmM1ODU5YzI1YmI3MDQ2OGE2Ygpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0VGhlQ29yZVNheXNXaGVuVG9SZVJlYWQJZTRhYmJmOGMzMzI1OTExMzFmNzZmY2Y1NjU4MDk2YzU2Y2JhN2M0YTQxYmFmMTVlNDdiOTQxZTg4YjY1YzM3Ngpib2R5CWludGVybmFsL2d1aWRlL2Nvc3QxNzFfdGVzdC5nbwlUZXN0VGhlQ29yZVRlYWNoZXNUaGVCb2R5Q291bnRBc09wdGlvbmFsCWRmNWRhNTViZDE5ODlhNjUxYjMxMTczMmM1N2M5ZDk0ODdlMzMzM2EyYWIyZDAzYzdkOGYzYWRmMTIzMjc1MzAKYm9keQlpbnRlcm5hbC9ndWlkZS9jb3N0MTcxX3Rlc3QuZ28JVGVzdFRoZUZ1bGxUZXh0TmFtZXNFdmVyeUNhc2VUaGF0TmVlZHNBQ291bnQJYjA2MmE4MmZlYjEzYzBjNWIxYWNjZTJmMDkwMDg4NmM5ZDJkZmEzYmI1NjdiNGNmZmU1NWQ0NzIyNDdkNmZhOQ · test-lock-kind:replace
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1793
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1998
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1879
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1781
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1935
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:1984
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:38beaa97c56b41f1f0de32dce65d8f174c97f82e5bff0b39380a83d6137de400 · ms:749
