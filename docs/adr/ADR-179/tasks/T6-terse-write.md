# Task ADR-179-T6: `mrw write --terse` prints one line per file and every failure and advisory

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw write --terse`; contract §310
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestTerseWritePrintsOneLinePerFile passed`, `TestTerseKeepsEveryFailedHunkAndAdvisory passed`, `TestTerseKeepsTheCheckReportAndItsExit passed`, `TestTerseLeavesTheJSONReceiptUnchanged passed`, `the touched packages pass whole`, `the contract run`, `section 310 exists`, `the row: mrw write --terse prints one line per file`, `the row: mrw write --terse still prints a failed hunk`, `the row: mrw write --terse keeps the exit 3 of a failing check`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (f): `--terse` prints one line per file (edit, unlink, rename and `--dry-run` forms) and the summary line, no preview; every failed hunk, skip row and `balance`, `closer` and `hint:` row still prints; the check report, step verdicts and their exit codes are unchanged; `--json` is byte for byte the same; `--terse --quiet` exits 2.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/terse179_test.go` | new | the red tests |
| `cmd/mrw/main.go` | edit | the flag on `write` (`:1692-1720`) and the terse branch of `report` (`:2981`) |
| `scripts/contract.sh` | edit | §310 |

## Ordered Steps

1. [S1] Write `TestTerseWritePrintsOneLinePerFile`, `TestTerseKeepsEveryFailedHunkAndAdvisory`, `TestTerseKeepsTheCheckReportAndItsExit`, `TestTerseLeavesTheJSONReceiptUnchanged` and see them RED.
2. [S2] The file lines. Mutant: the per-hunk `ok` rows printed under `--terse` (a plan file has no preview: `Options.Preview` is set only for a plan maker's dry run, `cmd/mrw/main.go:2188`). [proof: mutation]
3. [S3] The advisories. Mutant: `--terse` drops the `closer` row as `--quiet` does. [proof: mutation]
4. [S4] `--json`. Mutant: `--terse` drops `hunks` from the JSON receipt. [proof: mutation]
5. [S5] Contract §310: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestTerseWritePrintsOneLinePerFile|TestTerseKeepsEveryFailedHunkAndAdvisory|TestTerseKeepsTheCheckReportAndItsExit|TestTerseLeavesTheJSONReceiptUnchanged)$' 2>&1) \
  && grep -qE '^--- PASS: TestTerseWritePrintsOneLinePerFile \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTerseKeepsEveryFailedHunkAndAdvisory \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTerseKeepsTheCheckReportAndItsExit \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTerseLeavesTheJSONReceiptUnchanged \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 310\.' scripts/contract.sh \
  && grep -qF 'PASS  mrw write --terse prints one line per file' <<<"$c" \
  && grep -qF 'PASS  mrw write --terse still prints a failed hunk' <<<"$c" \
  && grep -qF 'PASS  mrw write --terse keeps the exit 3 of a failing check' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTerseWritePrintsOneLinePerFile` | `cmd/mrw/terse179_test.go` | three hunks in two files: two file lines and the summary, no preview line; the unlink, rename (with an edit: tally and sha), symlink (`target`) and `--dry-run` forms | none | S1, S2 |
| `TestTerseKeepsEveryFailedHunkAndAdvisory` | `cmd/mrw/terse179_test.go` | a failed plan prints its FAIL and skip rows; a balance and a closer row print; a delete and a `from=` copy keep their rows | none | S1, S3 |
| `TestTerseKeepsTheCheckReportAndItsExit` | `cmd/mrw/terse179_test.go` | a failing check exits 3 with its report; a `--then-sh` step prints its verdict | none | S1 |
| `TestTerseLeavesTheJSONReceiptUnchanged` | `cmd/mrw/terse179_test.go` | `--json` with and without `--terse` are equal; `--terse --quiet` exits 2 | none | S1, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §310 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · the per-hunk ok rows printed under --terse (S2): a plan file has no preview, the rows are what the flag removes · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops the closer row as --quiet does (S3) · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops hunks from the JSON receipt (S4) · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · the per-hunk ok rows printed under --terse (S2): a plan file has no preview, the rows are what the flag removes · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseWritePrintsOneLinePerFile passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops the closer row as --quiet does (S3) · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseKeepsEveryFailedHunkAndAdvisory passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops hunks from the JSON receipt (S4) · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseLeavesTheJSONReceiptUnchanged passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops the row of a delete, the only account of what it removed · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseKeepsEveryFailedHunkAndAdvisory passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --terse drops the copied-from line of a from= copy · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseKeepsEveryFailedHunkAndAdvisory passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/terse179.go` · --terse drops the symlink target · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseWritePrintsOneLinePerFile passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/terse179.go` · --terse drops the edit tally and sha of an edit beside a rename · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · covers:TestTerseWritePrintsOneLinePerFile passed

## Invariants

- A line not shown whole is not licensed.
- Without the flag, the read or write prints what it printed before this task, plus T1's one hint line.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-27 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:361 · test-lock-sha256:9e5ecefd07e5415809acb8f285abba7d722328e1603a51614f568db1a6eec5e7 · test-lock-file:docs/adr/locks/9e5ecefd07e5415809acb8f285abba7d722328e1603a51614f568db1a6eec5e7.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:0 · test-lock-sha256:a4b408ce3e67b7fad35e7fde3f26cf0161583506a8826be1b953b8707b22a409 · test-lock-file:docs/adr/locks/a4b408ce3e67b7fad35e7fde3f26cf0161583506a8826be1b953b8707b22a409.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134202
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134857
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134521
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:130099
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:135084
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134890
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:135206
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:0 · test-lock-sha256:4bccd58df3742a819f5ca6722efb9f0b49574392b276e3f60ddba0315272679a · test-lock-file:docs/adr/locks/4bccd58df3742a819f5ca6722efb9f0b49574392b276e3f60ddba0315272679a.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134539
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:135403
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:135171
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:136046
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:130275
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:131393
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:132860
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:134060
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:133921
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:133879
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:169286
- 2026-10-29 · 33aa28b7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:58e01603c9fbd6e6574565dc45c481af327ac17661b9f3a9f8ff09e928c8e336 · ms:0 · test-lock-sha256:1d434fa9459cd7ed75a7c95ea889694ce28e5d07eaf7b43fbfa96f409adc297b · test-lock-file:docs/adr/locks/1d434fa9459cd7ed75a7c95ea889694ce28e5d07eaf7b43fbfa96f409adc297b.lock · test-lock-kind:replace
