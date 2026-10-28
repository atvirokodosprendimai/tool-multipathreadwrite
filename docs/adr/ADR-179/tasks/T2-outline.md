# Task ADR-179-T2: `--outline` prints a Go or Markdown file's structure and licenses nothing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw read --outline`; the outline counter in `docs/bench/output-study/measure_output.py`; contract §306
**Consumes:** none
**Data dependency:** needs the Claude Code transcripts on this Mac for S2 (`~/.claude/projects`); every later step is hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestOutlineListsGoDeclarationsWithTheirLines passed`, `TestOutlineListsMarkdownHeadingsOutsideFences passed`, `TestOutlineNamesTheFilesItCannotOutlineOnce passed`, `TestOutlineLicensesNoLine passed`, `the touched packages pass whole`, `the contract run`, `section 306 exists`, `the row: --outline lists the declarations of a Go file with their lines`, `the row: --outline licenses no line`, `the row: --outline names the files it cannot outline once and exits 1`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (b): `--outline` lists `.go` top-level declarations (`go/parser`) and `.md` headings outside fences, each with its line range; files it cannot outline are named in one stderr line per invocation and the read exits 1; a parse error is incomplete; the file is hashed with no spans; the conflicts, `-` among them, are refused by name.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/output-study/measure_output.py` | edit | S2: whole-file reads over 150 lines broken down by extension |
| `internal/read/outline179_test.go` | new | the red unit tests |
| `cmd/mrw/outline179_test.go` | new | the red tests through the binary: licensing, the one notice, refusals |
| `internal/read/outline.go` | new | the Go and Markdown outliners |
| `internal/read/read.go` | edit | the outline branch in `Run` (`:654`, branch `:815-836`), recorded as `Count` records (`:157-160`) |
| `cmd/mrw/main.go` | edit | the flag (`:918`) and its conflicts (`:1076-1217`) |
| `internal/guide/guide.go` | edit | the flag named in `CLI()` (`:77`), which `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) requires the moment the flag exists |
| `scripts/contract.sh` | edit | §306 |

## Ordered Steps

1. [S1] Write `TestOutlineListsGoDeclarationsWithTheirLines`, `TestOutlineListsMarkdownHeadingsOutsideFences`, `TestOutlineNamesTheFilesItCannotOutlineOnce`, `TestOutlineLicensesNoLine` and see them RED.
2. [S2] Before any implementation: Extend `docs/bench/output-study/measure_output.py` with the extension of each whole-file read over 150 lines (from the Read tool's `file_path` and the mrw header), run it over `--since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, the registered window, and compare against the bar in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-177, ADR-178 and ADR-179": `.go` and `.md` together at least 25% of that bucket's characters. Below the bar, stop: the task goes to Zy with the numbers. [proof: human: the person compares the script's printed shares with the BACKLOG bar]. **Result (2026-10-28, `python3 -I docs/bench/output-study/measure_output.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, 156 transcript files, 129 sessions with a result in the window, 107 with an mrw call):** the bucket is 2,727,761 characters (Read 1,566,745 + `mrw read` 1,161,016, equal to the registered baseline); `.go` 131,601 + `.md` 707,362 = 838,963 = **30.76%** (`outline_go_md_share_pct`), at or past the 25% bar, so the task proceeds.
3. [S3] The outliners. Mutant: the Markdown scan does not track fences. [proof: mutation]
4. [S4] The Go ranges. Mutant: a declaration's end taken from its first line. [proof: mutation]
5. [S5] The licence. Mutant: the outline records the file whole. [proof: mutation]
6. [S6] Contract §306: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/read/ -count=1 -v -run '^(TestOutlineListsGoDeclarationsWithTheirLines|TestOutlineListsMarkdownHeadingsOutsideFences|TestOutlineNamesTheFilesItCannotOutlineOnce|TestOutlineLicensesNoLine)$' 2>&1) \
  && grep -qE '^--- PASS: TestOutlineListsGoDeclarationsWithTheirLines \(' <<<"$out" \
  && grep -qE '^--- PASS: TestOutlineListsMarkdownHeadingsOutsideFences \(' <<<"$out" \
  && grep -qE '^--- PASS: TestOutlineNamesTheFilesItCannotOutlineOnce \(' <<<"$out" \
  && grep -qE '^--- PASS: TestOutlineLicensesNoLine \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/read/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 306\.' scripts/contract.sh \
  && grep -qF 'PASS  --outline lists the declarations of a Go file with their lines' <<<"$c" \
  && grep -qF 'PASS  --outline licenses no line' <<<"$c" \
  && grep -qF 'PASS  --outline names the files it cannot outline once and exits 1' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOutlineListsGoDeclarationsWithTheirLines` | `internal/read/outline179_test.go` | func with receiver, type, var block, const; ranges from the file set; a parse error is incomplete | none | S1, S4 |
| `TestOutlineListsMarkdownHeadingsOutsideFences` | `internal/read/outline179_test.go` | a `#` inside a fence is not a heading; a range ends before the next heading of its level | none | S1, S3 |
| `TestOutlineNamesTheFilesItCannotOutlineOnce` | `cmd/mrw/outline179_test.go` | two `.py` files and a `.go`: one stderr line naming both, exit 1 | none | S1 |
| `TestOutlineLicensesNoLine` | `cmd/mrw/outline179_test.go` | after an outline a write is refused; `--outline` beside `--grep`, `--ast-grep`, `--sub`, `--count`, `--stat`, `--quiet`, `-C`, `--fold`, `--max-matches`, `--json-path`, `--if-changed`, on `-` and with an address exits 2 | none | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §306 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · the Markdown scan does not track fences (S3) · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsMarkdownHeadingsOutsideFences passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a declaration's end taken from its first line (S4) · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsGoDeclarationsWithTheirLines passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the outline records the file whole (S5) · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineLicensesNoLine passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --outline accepts --max-lines and --max-cols · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineLicensesNoLine passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a withheld outline is not a problem · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineLicensesNoLine passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a fence closes on a shorter run · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsMarkdownHeadingsOutsideFences passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a fence closes on the other character · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsMarkdownHeadingsOutsideFences passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a closing sequence needs no whitespace · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsMarkdownHeadingsOutsideFences passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · an empty heading is missed · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsMarkdownHeadingsOutsideFences passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a decl cut by a parse error prints 3-0 · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsGoDeclarationsWithTheirLines passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/outline.go` · a //line directive moves the outline start · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · covers:TestOutlineListsGoDeclarationsWithTheirLines passed

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
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:806 · test-lock-sha256:e97d9700828056ab71068d77dd654f7784e8fb47ba0ab3eb5ea0e9d9f13abff7 · test-lock-file:docs/adr/locks/e97d9700828056ab71068d77dd654f7784e8fb47ba0ab3eb5ea0e9d9f13abff7.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:0 · test-lock-sha256:549d96881e1342acc0663e5e80fc1e84aa34ab8202d9d099f9ed252ed0ed4fd7 · test-lock-file:docs/adr/locks/549d96881e1342acc0663e5e80fc1e84aa34ab8202d9d099f9ed252ed0ed4fd7.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:137768
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:137216
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:137954
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:132577
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:132955
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:134316
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:138429
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:140066
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:137813
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:138822
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:141595
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:138694
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:138129
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:138450
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:133463
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:134457
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:133035
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:ee0994c6e5693c37867003ff3e530d9f339f31dd3999ce45c62823ebf3eaa7ab · ms:141719
