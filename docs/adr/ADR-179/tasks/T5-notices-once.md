# Task ADR-179-T5: no notice is printed twice in one invocation

**Depends-on:** T1, T2, T4, T7, T8
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the once-per-invocation rule, its enumerated notice list and its class test; contract §309
**Consumes:** the hint line (T1), the outline notice (T2), the fold note (T4), the match-budget note (T7), the comment notice (T8)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestNoNoticeIsPrintedTwiceInOneInvocation passed`, `the touched packages pass whole`, `the contract run`, `section 309 exists`, `the row: a read of many files prints each notice once`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (e): one test drives a fixture over several specs and files that triggers every notice on an enumerated list — the sites `mrw read --grep '"mrw: |"-- [a-z]' --count cmd internal --exclude '*_test.go'` finds (37 in 17 files on 2026-10-27) less the per-range reports (`!!`, `WITHHELD`, `NOT FOUND`, the `--count` total), plus the notices T1, T2, T4, T7 and T8 add — and fails on any notice printed twice — however many files or specs it concerns, each notice is one line naming every path once — and on any listed notice the fixture did not trigger; each repeat it finds is fixed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/notices179_test.go` | new | the red class test and its list |
| `cmd/mrw/main.go` | edit | whatever repeats, found by S1 |
| `internal/read/read.go` | edit | whatever repeats, found by S1 |
| `scripts/contract.sh` | edit | §309 |

## Ordered Steps

1. [S1] Write `TestNoNoticeIsPrintedTwiceInOneInvocation` with the enumerated list and see it RED: until the fixture triggers every listed notice, its completeness check fails.
2. [S2] The fixes. Mutant: T1's hint printed per file. [proof: mutation]
3. [S3] Contract §309: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestNoNoticeIsPrintedTwiceInOneInvocation)$' 2>&1) \
  && grep -qE '^--- PASS: TestNoNoticeIsPrintedTwiceInOneInvocation \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 309\.' scripts/contract.sh \
  && grep -qF 'PASS  a read of many files prints each notice once' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoNoticeIsPrintedTwiceInOneInvocation` | `cmd/mrw/notices179_test.go` | every listed notice is triggered, and each appears once | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §309 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · the long-line hint printed twice instead of once (S2) · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · covers:TestNoNoticeIsPrintedTwiceInOneInvocation passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · a guard carried by two flags is named twice · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · covers:TestNoNoticeIsPrintedTwiceInOneInvocation passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a file served as bytes is named once per spec · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · covers:TestNoNoticeIsPrintedTwiceInOneInvocation passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · guard notice names only the first path · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · covers:TestNoNoticeIsPrintedTwiceInOneInvocation passed

## Invariants

- A line not shown whole is not licensed.
- Without the flag, the read or write prints what it printed before this task, plus T1's one hint line.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's.

## Stop Condition

The fence exits 0. Stop and ask if a repeated notice is printed by `internal/seen`, which this record does not own.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:393 · test-lock-sha256:d22d984a4e410d5d914a6e5780053de8c89d13144d5532c41eb736a3f83598f7 · test-lock-file:docs/adr/locks/d22d984a4e410d5d914a6e5780053de8c89d13144d5532c41eb736a3f83598f7.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:0 · test-lock-sha256:d22d984a4e410d5d914a6e5780053de8c89d13144d5532c41eb736a3f83598f7 · test-lock-file:docs/adr/locks/d22d984a4e410d5d914a6e5780053de8c89d13144d5532c41eb736a3f83598f7.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:61042
  ```
  ```
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:61013
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:139128
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:138365
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:133283
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:133236
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:0 · test-lock-sha256:9d57fe976756f39b8e82fc29cb1ed4dcdc76deeaa11eed8e648daa9cebcd9e92 · test-lock-file:docs/adr/locks/9d57fe976756f39b8e82fc29cb1ed4dcdc76deeaa11eed8e648daa9cebcd9e92.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:158871
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:141094
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:0369b4831c559c9585bddb2d264091c18ed8f853b9f734cbd665f6d1fdc4fec9 · ms:135602
