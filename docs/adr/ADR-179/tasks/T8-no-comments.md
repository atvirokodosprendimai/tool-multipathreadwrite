# Task ADR-179-T8: `--no-comments` elides Go comment blocks, leaving them unlicensed

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw read --no-comments`; the comment counter in `docs/bench/output-study/measure_output.py`; contract §312
**Consumes:** none
**Data dependency:** needs the Claude Code transcripts on this Mac for S2 (`~/.claude/projects`); every later step is hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestNoCommentsElidesGoCommentBlocks passed`, `TestNoCommentsKeepsDirectivesAndTheCgoPreamble passed`, `TestAnElidedCommentLineIsNotLicensed passed`, `TestNoCommentsNamesTheFilesItServedWholeOnce passed`, `the touched packages pass whole`, `the contract run`, `section 312 exists`, `the row: --no-comments elides a Go comment block`, `the row: an elided comment line is refused a write`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (h): in a `.go` file, scanned whole with `go/scanner`, a run of two or more whole-line comment lines (`//` or `/* */`) is served as `-- … N comment line(s), A-B` and left unlicensed; trailing comments, `//go:` and `//line` directives and the cgo preamble stay; other files and `-` are served whole, named in one stderr line per invocation.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/output-study/measure_output.py` | edit | S2: whole-line comment runs (`//` and `/* */`, string lines excluded as far as text allows) in `.go` blocks of `mrw read` output |
| `internal/read/comments179_test.go` | new | the red unit tests |
| `cmd/mrw/comments179_test.go` | new | the red licensing test through the binary |
| `internal/read/comments.go` | new | the Go comment-run finder |
| `internal/read/read.go` | edit | the elision in the serving loop, elided lines left out of `served` |
| `cmd/mrw/main.go` | edit | the flag and its conflicts |
| `internal/guide/guide.go` | edit | the flag named in `CLI()` (`:77`), which `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) requires the moment the flag exists |
| `scripts/contract.sh` | edit | §312 |

## Ordered Steps

1. [S1] Write `TestNoCommentsElidesGoCommentBlocks`, `TestNoCommentsKeepsDirectivesAndTheCgoPreamble`, `TestAnElidedCommentLineIsNotLicensed`, `TestNoCommentsNamesTheFilesItServedWholeOnce` and see them RED.
2. [S2] Before any implementation: Extend `docs/bench/output-study/measure_output.py` with the characters of whole-line comment runs of two or more lines (`//` and `/* */`) inside `.go` blocks of `mrw read` output, run it over `--since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, the registered window, and compare against the bar in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-177, ADR-178 and ADR-179": at least 1.0% of `mrw read` characters, numerator and denominator printed. Below the bar, stop: the task goes to Zy with the numbers. [proof: human: the person compares the script's printed share with the BACKLOG bar]. **Result (2026-10-28, `python3 -I docs/bench/output-study/measure_output.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, 156 transcript files, 129 sessions with a result in the window, 107 with an mrw call):** whole-line comment runs of two or more lines in `.go` blocks of `mrw read` output: 2,530,272 characters in 33,900 lines (`comment_run_chars`) over 57,276,199 `mrw read` characters (`comment_run_denominator_mrw_read_chars`) = **4.418%** (`comment_run_share_pct`), at or past the 1.0% bar, so the task proceeds. Counted from text alone: a `//` line inside a raw string counts, so the figure is an upper bound. **Corrected counter, same window:** the first counter took a code-bearing line such as `/* c */ var x` for a comment line and kept block state across the gaps between served ranges; it now counts only lines holding nothing but comment, resets block state at every gap and file block, and does not count a range that starts inside a block (conservative). Re-run: 2,530,272 characters in 33,900 lines over 57,276,199 = **4.418%**, unchanged to the digit, so the bar (1.0%) is still passed.
3. [S3] The finder. Mutant: a line holding `//` anywhere counted as a comment line. [proof: mutation]
4. [S4] The runs. Mutant: a single comment line elided. [proof: mutation]
5. [S5] The licence. Mutant: elided lines added to `served`. [proof: mutation]
6. [S6] Contract §312: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/read/ -count=1 -v -run '^(TestNoCommentsElidesGoCommentBlocks|TestNoCommentsKeepsDirectivesAndTheCgoPreamble|TestAnElidedCommentLineIsNotLicensed|TestNoCommentsNamesTheFilesItServedWholeOnce)$' 2>&1) \
  && grep -qE '^--- PASS: TestNoCommentsElidesGoCommentBlocks \(' <<<"$out" \
  && grep -qE '^--- PASS: TestNoCommentsKeepsDirectivesAndTheCgoPreamble \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAnElidedCommentLineIsNotLicensed \(' <<<"$out" \
  && grep -qE '^--- PASS: TestNoCommentsNamesTheFilesItServedWholeOnce \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/read/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 312\.' scripts/contract.sh \
  && grep -qF 'PASS  --no-comments elides a Go comment block' <<<"$c" \
  && grep -qF 'PASS  an elided comment line is refused a write' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoCommentsElidesGoCommentBlocks` | `internal/read/comments179_test.go` | a doc block, a `/* */` block, a range starting inside a block (judged by a whole-file scan, `//line` directives not moving line numbers), a `//` inside a string kept, a trailing comment kept, a single comment line kept | none | S1, S3, S4 |
| `TestNoCommentsKeepsDirectivesAndTheCgoPreamble` | `internal/read/comments179_test.go` | `//go:build`, `//go:generate`, `//line` and the comment above `import "C"` are served | none | S1, S3 |
| `TestAnElidedCommentLineIsNotLicensed` | `cmd/mrw/comments179_test.go` | a write to an elided line is refused, a shown line written; refused (exit 2) beside `--quiet`, `--stat`, `--count`, `--sub`, `--max-matches`, `--fold`, `--outline`, `--json-path` | none | S1, S5 |
| `TestNoCommentsNamesTheFilesItServedWholeOnce` | `cmd/mrw/comments179_test.go` | two `.py` files and `-`: served whole, one stderr line | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §312 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/comments.go` · a line holding a comment anywhere counted as a comment line (S3) · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsElidesGoCommentBlocks passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a single comment line elided (S4) · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsElidesGoCommentBlocks passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · elided lines added to served (S5) · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestAnElidedCommentLineIsNotLicensed passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the line after a range counts toward a run · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestAnElidedCommentLineIsNotLicensed passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a line a grep matched can be elided or folded · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsElidesGoCommentBlocks passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/comments.go` · //export is elided · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsKeepsDirectivesAndTheCgoPreamble passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/comments.go` · a scan error is ignored · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsNamesTheFilesItServedWholeOnce passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a comment note that does not fit is lost unnamed · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsElidesGoCommentBlocks passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a .go file that does not scan is served whole unnamed · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsNamesTheFilesItServedWholeOnce passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/comments.go` · a grouped import C spec loses its preamble · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · covers:TestNoCommentsKeepsDirectivesAndTheCgoPreamble passed

## Invariants

- A line not shown whole is not licensed.
- Without the flag, the read or write prints what it printed before this task, plus T1's one hint line.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's; S2 reads real transcripts, so its numbers name the window they were taken over.

## Stop Condition

S2's share is below its BACKLOG bar: stop before any code and take the numbers to Zy. Otherwise the fence exits 0.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:325 · test-lock-sha256:4c83c93163610c811e315661bf8a552dee59866aedb8e4fee04a19ef7fdcc253 · test-lock-file:docs/adr/locks/4c83c93163610c811e315661bf8a552dee59866aedb8e4fee04a19ef7fdcc253.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:0 · test-lock-sha256:4c83c93163610c811e315661bf8a552dee59866aedb8e4fee04a19ef7fdcc253 · test-lock-file:docs/adr/locks/4c83c93163610c811e315661bf8a552dee59866aedb8e4fee04a19ef7fdcc253.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:137886
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:138326
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:137690
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:132243
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:145131
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:143448
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:144315
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:144657
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:143989
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:140818
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:138591
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:133974
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:135319
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:133582
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:238981
- 2026-11-01 · 632e1948* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e38866d8057f7fa80f4f75c0349f84e4f133790ee2c58dd4284aaff14a4e0c40 · ms:0 · test-lock-sha256:d78a209ad8c058b058e1a9641253b0370ab299ca50c2613d50d3c5136d760f63 · test-lock-file:docs/adr/locks/d78a209ad8c058b058e1a9641253b0370ab299ca50c2613d50d3c5136d760f63.lock · test-lock-kind:replace
