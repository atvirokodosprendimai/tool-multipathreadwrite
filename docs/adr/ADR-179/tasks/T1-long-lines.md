# Task ADR-179-T1: a long line served whole says so once; `MRW_MAX_COLS` sets the default

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the long-line hint; `MRW_MAX_COLS`; contract §305
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestAReadOfAnOverlongLineHintsOnceOnStderr passed`, `TestMRWMaxColsAppliesWhenTheFlagIsAbsent passed`, `TestMRWMaxColsDoesNotApplyToAQuietRead passed`, `the touched packages pass whole`, `the contract run`, `section 305 exists`, `the row: a read of a line over 400 columns hints once`, `the row: MRW_MAX_COLS cuts and the cut line is refused a write`, `the row: MRW_MAX_COLS leaves a --quiet read whole`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (a): a content read — of files or of `-` — that serves a line over 400 columns whole adds one stderr hint per invocation; `MRW_MAX_COLS=N`, read with `os.Getenv`, applies `--max-cols N` where the flag is absent and lines are served, never under `--quiet`, `--sub`, `--rewrite`, `--count`, `--stat`, `--outline` or `--json-path`, and never trips a refusal; a bad value is named and ignored.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/longline179_test.go` | new | the red tests through the binary |
| `internal/read/read.go` | edit | count lines over 400 served whole, beside the `--max-cols` cut (`:1022-1031`), for the caller |
| `cmd/mrw/main.go` | edit | the hint after the answer (beside `:1535-1546`); `MRW_MAX_COLS` where `MaxCols` is set (`:1506`) |
| `cmd/mrw/stdin.go` | edit | the same at standard input's `Options` (`:103-117`) |
| `scripts/contract.sh` | edit | §305 |

## Ordered Steps

1. [S1] Write `TestAReadOfAnOverlongLineHintsOnceOnStderr`, `TestMRWMaxColsAppliesWhenTheFlagIsAbsent`, `TestMRWMaxColsDoesNotApplyToAQuietRead` and see them RED.
2. [S2] The hint. Mutant: the hint printed once per file instead of once per invocation. [proof: mutation]
3. [S3] The variable. Mutant: `MRW_MAX_COLS` read even when `--max-cols` is set. [proof: mutation]
4. [S4] The exceptions. Mutant: the variable applied under `--quiet`. [proof: mutation]
5. [S5] Contract §305: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestAReadOfAnOverlongLineHintsOnceOnStderr|TestMRWMaxColsAppliesWhenTheFlagIsAbsent|TestMRWMaxColsDoesNotApplyToAQuietRead)$' 2>&1) \
  && grep -qE '^--- PASS: TestAReadOfAnOverlongLineHintsOnceOnStderr \(' <<<"$out" \
  && grep -qE '^--- PASS: TestMRWMaxColsAppliesWhenTheFlagIsAbsent \(' <<<"$out" \
  && grep -qE '^--- PASS: TestMRWMaxColsDoesNotApplyToAQuietRead \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 305\.' scripts/contract.sh \
  && grep -qF 'PASS  a read of a line over 400 columns hints once' <<<"$c" \
  && grep -qF 'PASS  MRW_MAX_COLS cuts and the cut line is refused a write' <<<"$c" \
  && grep -qF 'PASS  MRW_MAX_COLS leaves a --quiet read whole' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadOfAnOverlongLineHintsOnceOnStderr` | `cmd/mrw/longline179_test.go` | two files with long lines: one hint naming the count and the longest; none with `--max-cols`; none when no line is over 400; the same on `-`; none when stdout is the null device (§305) | none | S1, S2 |
| `TestMRWMaxColsAppliesWhenTheFlagIsAbsent` | `cmd/mrw/longline179_test.go` | the variable cuts a file and `-`, the cut line is refused a write, `--max-cols 0` overrides, a bad value is named | none | S1, S3 |
| `TestMRWMaxColsDoesNotApplyToAQuietRead` | `cmd/mrw/longline179_test.go` | `--quiet` licenses the whole line under the variable; `--sub` and `--count` reads are not refused because of it | none | S1, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §305 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the long-line counter reset for each file: the hint says 1 line(s) instead of 2, so the exact-text assertion must go red (S2) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/longline179.go` · MRW_MAX_COLS read even when --max-cols is given, so --max-cols 0 no longer turns it off (S3) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/longline179.go` · the variable applied under --quiet: a quiet read would license less than it is asked to (S4) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the long-line counter reset for each file: the hint says 1 line(s) instead of 2, so the exact-text assertion must go red (S2) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:TestAReadOfAnOverlongLineHintsOnceOnStderr passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/longline179.go` · MRW_MAX_COLS read even when --max-cols is given, so --max-cols 0 no longer turns it off (S3) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:TestMRWMaxColsAppliesWhenTheFlagIsAbsent passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/longline179.go` · the variable applied under --quiet: a quiet read would license less than it is asked to (S4) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:TestMRWMaxColsDoesNotApplyToAQuietRead passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · the long-line hint printed when stdout is the null device (main.go) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:the row: a read of a line over 400 columns hints once
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · the long-line hint printed when stdout is the null device (stdin.go) · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:the row: a read of a line over 400 columns hints once
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/longline179.go` · --quiet=false disables the variable · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · covers:TestMRWMaxColsAppliesWhenTheFlagIsAbsent passed

## Invariants

- A line not shown whole is not licensed.
- Without the flag, the read or write prints what it printed before this task.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-27 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:842 · test-lock-sha256:d33cb6d015deac91d888b7f023151aea5406c1118c19a9280716700968518c8f · test-lock-file:docs/adr/locks/d33cb6d015deac91d888b7f023151aea5406c1118c19a9280716700968518c8f.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:0 · test-lock-sha256:1a4cdedcb59cb4c27da825668056c2725c9a4f4c710605d6532649924ac10254 · test-lock-file:docs/adr/locks/1a4cdedcb59cb4c27da825668056c2725c9a4f4c710605d6532649924ac10254.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:135721
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:137900
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:135081
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:130006
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:137779
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:134825
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:136355
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:0 · test-lock-sha256:1a4cdedcb59cb4c27da825668056c2725c9a4f4c710605d6532649924ac10254 · test-lock-file:docs/adr/locks/1a4cdedcb59cb4c27da825668056c2725c9a4f4c710605d6532649924ac10254.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:136095
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:135407
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:130754
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:0 · test-lock-sha256:dd90d54398b3390aa7d41104ecb9679c6274061e1f0b3cd1281fddd5a2480b81 · test-lock-file:docs/adr/locks/dd90d54398b3390aa7d41104ecb9679c6274061e1f0b3cd1281fddd5a2480b81.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:135075
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:129971
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:133010
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:133928
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:134274
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:133851
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:85d5c151acc066ccad6f519c973f01f41637a88295f61a0f12bacf54add6e5f5 · ms:148402
