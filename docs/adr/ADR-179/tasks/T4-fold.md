# Task ADR-179-T4: `--fold` collapses runs of identical and digit-only-different lines

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw read --fold`; contract §308
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines passed`, `TestAFoldedLineIsNotLicensed passed`, `the touched packages pass whole`, `the contract run`, `section 308 exists`, `the row: --fold collapses a run of lines that differ only in digits`, `the row: a folded line is refused a write`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (c): on a file or on `-`, a run of two or more non-blank lines identical, or identical but for digits, prints its first line and `-- ×N: lines A-B like A`; the first line is licensed, the folded ones are not; the conflicts are refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/fold179_test.go` | new | the red unit test |
| `cmd/mrw/fold179_test.go` | new | the red licensing test through the binary |
| `internal/read/read.go` | edit | the fold in the serving loop beside the `--max-cols` cut (`:990-1077`, fold branch `:1055-1077`), folded lines left out of `served` as cut lines are |
| `cmd/mrw/main.go` | edit | the flag and its conflicts |
| `cmd/mrw/stdin.go` | edit | the fold at standard input's `Options` (`:103-117`) |
| `internal/guide/guide.go` | edit | the flag named in `CLI()` (`:77`), which `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) requires the moment the flag exists |
| `scripts/contract.sh` | edit | §308 |

## Ordered Steps

1. [S1] Write `TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines`, `TestAFoldedLineIsNotLicensed` and see them RED.
2. [S2] The fold. Mutant: lines compared with their digits kept, so a digit run never folds. [proof: mutation]
3. [S3] The licence. Mutant: folded lines added to `served`. [proof: mutation]
4. [S4] Contract §308: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/read/ -count=1 -v -run '^(TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines|TestAFoldedLineIsNotLicensed)$' 2>&1) \
  && grep -qE '^--- PASS: TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAFoldedLineIsNotLicensed \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/read/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 308\.' scripts/contract.sh \
  && grep -qF 'PASS  --fold collapses a run of lines that differ only in digits' <<<"$c" \
  && grep -qF 'PASS  a folded line is refused a write' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines` | `internal/read/fold179_test.go` | identical run, digit run, a run broken by one other line, blank lines never fold | none | S1, S2 |
| `TestAFoldedLineIsNotLicensed` | `cmd/mrw/fold179_test.go` | a write to a folded line is refused, to the run's first line allowed; `-` folds too; refused beside `--quiet`, `--stat`, `--count`, `--sub`, `--max-matches` and `--json-path`, exit 2 | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §308 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · lines compared with their digits kept, so a digit run never folds (S2) · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · covers:TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · folded lines added to served (S3) · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · covers:TestAFoldedLineIsNotLicensed passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · lines compared by anything but their text between digit runs: a literal # folds as a digit · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · covers:TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines passed
- 2026-10-28 · 14bce654* · mutant survived · exit 0 · `internal/read/read.go` · a fold note that does not fit is lost unnamed · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · covers:TestAFoldedLineIsNotLicensed passed
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a fold note that does not fit is lost unnamed · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · covers:TestAFoldedLineIsNotLicensed passed

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
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:324 · test-lock-sha256:2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964 · test-lock-file:docs/adr/locks/2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:0 · test-lock-sha256:2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964 · test-lock-file:docs/adr/locks/2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:135171
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:135499
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:0 · test-lock-sha256:2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964 · test-lock-file:docs/adr/locks/2f06fe247651412d463a7cb28bdb18b9319bd0fc1f04b4a3a6e3d26104908964.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:132633
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:142650
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:143061
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:139512
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:133103
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:133924
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:133830
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:260c2f1228f6c6b4f9ec3f769114996603eb23985010cba7f670d597a484a899 · ms:136078
