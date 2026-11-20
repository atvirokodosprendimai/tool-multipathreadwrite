# Task ADR-208-T2: lines a search/replace write produced are not shown

**Status of the decision:** Proposed (ADR-208 Decision 1a).
**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the writer recording a `search_replace`-only write as written-shaped (`w|<lines>`, licensed and not shown); `needAfter` applying Decision 1's condition to every written-shaped record; T1's row `partial_written_licence` flipped; the sentences; contract §374
**Consumes:** T1's `needAfter`, `SameCallRewrite`, `aliasIndex`, `staleRemedySpan` and the `line-after` refusal kind (on `main` since `945654ef`); `seen.Observation`'s written-shaped partial record and its merge (unchanged); `apply.FileResult.QuotedOnly` and `Spans`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a search_replace-only write is recorded licensed and not shown`, `a written-shaped partial record needs the line after shown at the current sha`, `a read shows it`, `force after an external edit cannot reuse an old showing`, `quiet licences, plan licences, cut lines and a later write show nothing`, `a same-call rewrite and a kept tail still land`, `another spelling's record is judged by its own shape`, `a read-shaped record is judged as before`, `the legacy residual is pinned`, `mrw_write refuses until the ack`, `every named subtest passed in its package and none skipped`, `the contract row runs the repro and the alias pair`, `the unowned engine packages are untouched`

## Goal

ADR-208 Decision 1a.

**`TestASearchReplaceOnlyWriteIsRecordedLicensedNotShown`** (`internal/writer`), one subtest per row: `quote_only` (a write whose every hunk on the file is a verified quote records `Written`, `Spans` the lines it wrote, nothing shown); `mixed` (a write with a native hunk on the file records it whole and written, as today); `two_files` (a plan that quotes one file and edits another natively records each by its own hunks).

**`TestAMultiLineReplaceWhoseLineAfterASearchReplaceWroteIsRefusedUntilShown`** (`internal/apply`), one subtest per row. `f.go` is the record's repro after its `search_replace` write; the plan is `@@ f.go 6-8 replace anchor="func B"` with a body ending in `}`, unless the row says otherwise:

| Subtest | Ledger before the plan | Outcome |
|---------|------------------------|---------|
| `written_line_after` | `w|5-9`, nothing shown, current sha | failed, ADR-052's reason naming line 9 and ending with `mrw read --max-cols 0 f.go:6-9`; bytes unchanged |
| `shown_by_a_read` | the same, line 9 shown at the current sha | ok |
| `forced_after_external_edit` | line 9 shown at an older sha; the file changed since; `Force` | failed with the line-after reason (not the sha refusal, which `Force` skips) |
| `forced_nothing_shown` | `w|5-9`, nothing shown; `Force` | failed |
| `quiet_whole_merged` | a whole-file `--quiet` licence merged in (whole, nothing shown) | failed |
| `quiet_ranged_merged` | a ranged `--quiet` licence of 6-9 merged in | failed |
| `later_one_line_write` | line 9 shown at the first write's sha; then a second quoted write changed line 5, and the test asserts the record it left is `w|5-5` with nothing shown; then a ranged `--quiet` licence of 6-9 | failed |
| `plan_licence_later_call` | a `read --rewrite` licence of 6-9, the plan written by a later call | failed |
| `width_cut` | a read whose `--max-cols` cut line 9 | failed |
| `same_call_rewrite` | `w|5-9`; the plan marked as a same-call `--rewrite` | ok |
| `quoted_no_kept_tail` | `w|5-9`; a `search_replace` hunk over 6-8 whose REPLACE does not end with SEARCH's last line | failed |
| `quoted_kept_tail` | `w|5-9`; a block whose REPLACE ends with SEARCH's last line, compiled to a multi-line replace with the kept line quoted | ok |
| `alias_quiet_record` | a ranged `--quiet` licence of 6-9 recorded under a second spelling at the current sha; the plan under that spelling | failed with the line-after reason, not the unread one |
| `alias_read_record` | a read of 6-9 recorded under a second spelling at the current sha; the plan under that spelling | ok |
| `ends_on_last_line` | `w|5-9`; replacing 6-9 | ok |
| `read_shaped_record` | read-shaped, lines 1-9 read | ok (as today) |
| `forced_stale_read_shaped` | read-shaped, lines 1-9 read at an older sha; the file changed since; `Force` | ok (as today: a read-shaped record is judged by `Covers`, outside Decision 1a) |
| `occurrence_after_a_written_match` | `w|5-9`, nothing shown; a single-line replace by `/^\t}/ occurrence=2` whose first match is a written line | failed (ADR-118's unread earlier match) |
| `legacy_read_shaped_quote_record` | read-shaped 5-9, as an mrw before T2 recorded a quoted write | ok — the residual window Decision 1a states, pinned so a change to it is seen |
| `legacy_then_same_sha_t2_write` | that legacy record, then a T2 quoted write leaving the sha unchanged merged in | ok — the merge copies the legacy spans into the shown set (`seen.go:859`); the residual, pinned |

**`TestTheCLIRefusesAReplaceEndingAboveLinesASearchReplaceWrote`** (`cmd/mrw`, the built binary, runs on Windows CI), one subtest per row:

| Subtest | Steps | Outcome |
|---------|-------|---------|
| `refused_then_read` | `mrw write --create f.go`, the `search_replace` block, the plan | exit 1, the reason naming line 9 and the remedy `mrw read --max-cols 0 f.go:6-9`, bytes unchanged; after that read the same plan exits 0 |
| `seen_says_licensed` | the same, before the plan | `mrw seen` gives f.go's record as licensed, not shown |
| `alias_quiet_then_read` | a fresh copy; an in-root symlink `g.go` to it; `mrw read --quiet g.go:6-9`; the plan under `g.go` | exit 1 with the line-after reason and bytes unchanged; after `mrw read --max-cols 0 g.go:6-9`, exit 0 |
| `from_copy_needs_a_read` | a fresh copy; `@@ h.go 0 create from=f.go:6` | refused until `mrw read f.go:6` |
| `if_served_serves_written_lines` | a fresh copy; `mrw read --if-served f.go=<sha> f.go:5-9` | serves the lines, not `-- same as served` |
| `subroot_remedy_from_the_original_directory` | a checkout holding `sub/f.go`; the repro run as `mrw --root sub write`; then the remedy the refusal printed, `mrw --root sub read --max-cols 0 f.go:6-9`, run as printed from the original working directory; then the plan again | exit 1, the remedy naming the root; the remedy exits 0; the retry exits 0 |
| `saved_rewrite_plan` | `rewrite161_test.go`'s fixture: `mrw read --ast-grep P --rewrite R` saved as a plan and written by a later call | exit 1 naming the line after; after a read of it, exit 0; `mrw write --ast-grep P --rewrite R` on a fresh copy exits 0 |

**`TestMrwWriteRefusesAReplaceEndingAboveLinesASearchReplaceWrote`** (`internal/mcp`), one subtest per row: `refused` (`isError` with the failed hunk, its kind `line-after`, and the reason); `before_ack` (`mrw_read` of 6-9, then the plan before its `ack`: `isError`); `after_ack` (the `ack` sent, then the plan: lands).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/writer.go` | edit | a `search_replace`-only write records `Written: true`, its spans, nothing shown |
| `internal/apply/apply.go` | edit | `needAfter`: Decision 1's condition on every written-shaped record |
| `internal/apply/after208_test.go` | edit | T1's row `partial_written_licence` expects the refusal `needs a served line after 3` |
| `cmd/mrw/rewrite161_test.go` | edit | `TestARewriteReadAndWriteApplyAstGrepsResult`'s read-then-write leg (line 75): refusal, a read, then the retry; its same-call leg on a fresh fixture |
| `internal/writer/search208_test.go`, `internal/apply/after208t2_test.go`, `cmd/mrw/after208t2_test.go`, `internal/mcp/after208t2_test.go` | new | the four tests |
| tests that pin a `search_replace` write's read-shaped record, or a `from=`/`occurrence=`/`--if-served` answer over its lines | edit | `git grep -n -e QuotedOnly -e quotedOnly -- '*_test.go'` names the first; each is read and updated only where it pins what this task changes |
| `scripts/contract.sh` | edit | §374; §264's saved-plan leg becomes refusal, read, retry, with its PASS sentence unchanged, so ADR-161 T2's fence (which greps that sentence) still holds — `adr-verify docs/adr/ADR-161-instructions-lead-with-recipes/tasks/T2-rewrite.md` is run again after T2 lands |
| `AGENTS.md`, `README.md`, `internal/guide/guide.go` | edit | ADR-190's sentence on a `search_replace` write: its lines are licensed, not shown, so a later multi-line replace needs the line after read |

## Ordered Steps

1. [S1] The four tests, T1's row and the rewrite test; RED today: `quote_only`; in the second table `written_line_after`, `forced_after_external_edit`, `forced_nothing_shown`, `quiet_ranged_merged`, `later_one_line_write`, `plan_licence_later_call`, `width_cut`, `quoted_no_kept_tail`, `alias_quiet_record` and `occurrence_after_a_written_match`; every CLI row; `refused` and `before_ack`; T1's `partial_written_licence`; the rewrite test's read-then-write leg. The other rows hold today and stay. [proof: acceptance]
2. [S2] The record and the rule. Mutants, each killed: the writer records a quoted write as before (`quote_only`, the CLI's `refused_then_read`); `needAfter` keeps `obs.Whole()` (`written_line_after`, `quiet_ranged_merged`); the sha comparison dropped (`forced_after_external_edit`); `SameCallRewrite` ignored (`same_call_rewrite`, the CLI's `saved_rewrite_plan` same-call leg); the kept tail ignored (`quoted_kept_tail`); `Force` waives it (`forced_nothing_shown`); the rule applied to read-shaped records (`forced_stale_read_shaped`); other spellings not consulted (`alias_read_record`); the remedy loses the root (`subroot_remedy_from_the_original_directory`). [proof: mutation]
3. [S3] Contract §374 drives `$MRW` on `f374.go`: the repro exits 1 naming line 9 with the remedy `mrw read --max-cols 0 f374.go:6-9` and the file unchanged, paired with the same plan exiting 0 after that read; through an in-root symlink `g374.go`, a ranged `--quiet` read of 6-9 leaves the plan exit 1 with the same reason, paired with a read under `g374.go` landing it. §264's saved-plan leg. The sentences. [proof: acceptance]

## Acceptance

```bash
m=github.com/atvirokodosprendimai/tool-multipathreadwrite
tab="$(printf '\t')"
need() { f=$1; t=$2; shift 2; grep -q "^--- PASS: $t (" "$f" || return 1; for s in "$@"; do grep -q "^    --- PASS: $t/$s (" "$f" || return 1; done; }
d="$(mktemp -d)" \
  && go test -count=1 -v ./internal/writer -run '^TestASearchReplaceOnlyWriteIsRecordedLicensedNotShown$' > "$d/writer" \
  && go test -count=1 -v ./internal/apply -run '^(TestAMultiLineReplaceWhoseLineAfterASearchReplaceWroteIsRefusedUntilShown|TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown)$' > "$d/apply" \
  && go test -count=1 -v ./cmd/mrw -run '^(TestTheCLIRefusesAReplaceEndingAboveLinesASearchReplaceWrote|TestARewriteReadAndWriteApplyAstGrepsResult)$' > "$d/cli" \
  && go test -count=1 -v ./internal/mcp -run '^TestMrwWriteRefusesAReplaceEndingAboveLinesASearchReplaceWrote$' > "$d/mcp" \
  && grep -q "^ok  ${tab}$m/internal/writer${tab}" "$d/writer" && grep -q "^ok  ${tab}$m/internal/apply${tab}" "$d/apply" \
  && grep -q "^ok  ${tab}$m/cmd/mrw${tab}" "$d/cli" && grep -q "^ok  ${tab}$m/internal/mcp${tab}" "$d/mcp" \
  && need "$d/writer" TestASearchReplaceOnlyWriteIsRecordedLicensedNotShown quote_only mixed two_files \
  && need "$d/apply" TestAMultiLineReplaceWhoseLineAfterASearchReplaceWroteIsRefusedUntilShown written_line_after shown_by_a_read forced_after_external_edit forced_nothing_shown quiet_whole_merged quiet_ranged_merged later_one_line_write plan_licence_later_call width_cut same_call_rewrite quoted_no_kept_tail quoted_kept_tail alias_quiet_record alias_read_record ends_on_last_line read_shaped_record forced_stale_read_shaped occurrence_after_a_written_match legacy_read_shaped_quote_record legacy_then_same_sha_t2_write \
  && need "$d/apply" TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown partial_written_licence \
  && need "$d/cli" TestTheCLIRefusesAReplaceEndingAboveLinesASearchReplaceWrote refused_then_read seen_says_licensed alias_quiet_then_read from_copy_needs_a_read if_served_serves_written_lines subroot_remedy_from_the_original_directory saved_rewrite_plan \
  && grep -q '^--- PASS: TestARewriteReadAndWriteApplyAstGrepsResult (' "$d/cli" \
  && need "$d/mcp" TestMrwWriteRefusesAReplaceEndingAboveLinesASearchReplaceWrote refused before_ack after_ack \
  && ! grep -q -- '--- SKIP' "$d/writer" "$d/apply" "$d/cli" "$d/mcp" \
  && s374="$(awk '/^# [0-9]+\. /{on = /^# 374\. /} on' scripts/contract.sh)" \
  && grep -q '^# 374\. ' <<<"$s374" \
  && grep -qF 'mrw read --max-cols 0 f374.go:6-9' <<<"$s374" \
  && grep -qF 'g374.go' <<<"$s374" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" \
  && [ -z "$st" ]
```

What each clause holds. The four `go test` runs each pass in their own package; every named test and subtest passed there, none skipped. The tab in the `ok` lines is a literal one, built with `printf`. T1's row `partial_written_licence` passes only if its expectation is the refusal `needs a served line after 3` (T1's test matches `want` as a substring of the reason); that the row was changed is S1's red step, not a source grep. §374's clauses read its own body, up to and not including the next section's heading, and prove its text; that it runs is the contract gate's, required by the Stop Condition. The engine clauses hold the five unowned engine packages (`internal/seen` among them: Decision 1a changes no ledger code): the `git diff` against the merge base ignores committed test-file edits there, while the `git status` check rejects any uncommitted or untracked file there, test files included; files `.gitignore` ignores are outside both, and a failing `git status` fails the chain.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASearchReplaceOnlyWriteIsRecordedLicensedNotShown` | `internal/writer/search208_test.go` | the record a write leaves, by the loaded ledger | none | S1, S2 |
| `TestAMultiLineReplaceWhoseLineAfterASearchReplaceWroteIsRefusedUntilShown` | `internal/apply/after208t2_test.go` | the second table, by the verdict, the reason and the bytes | none | S1, S2 |
| `TestTheCLIRefusesAReplaceEndingAboveLinesASearchReplaceWrote` | `cmd/mrw/after208t2_test.go` | the CLI table, through the built binary and the writer | none | S1, S2 |
| `TestMrwWriteRefusesAReplaceEndingAboveLinesASearchReplaceWrote` | `internal/mcp/after208t2_test.go` | `isError` before the `ack`, landing after it | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §374 |
| 2 — something selects it | every multi-line replace on a written-shaped record, on both surfaces, under any spelling |
| 3 — the caller can discover it | the refusal names the read; `mrw seen` gives the record as licensed, not shown; `AGENTS.md`, `README.md`, `mrw instructions` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is go-reviewer's repro on #745, reproduced in the record |

## Mutation Log

## Verification Log

## Invariants

- The ledger's format and code are unchanged; an older mrw reads what this one writes.
- A read-shaped record is judged exactly as before; a whole written record is T1's.
- No receipt key is added, removed or renamed.

## Risks

- The residual window (records written before T2) is pinned by `legacy_read_shaped_quote_record`, not closed.
- The fence proves §374's text, not that it ran: §374 is proved by `scripts/contract.sh` on Linux CI (`lifecycle.md` §6). `contract.sh` does not run on Windows; the CLI and MCP tests carry the behaviour there.
- This task follows T1 as merged (`945654ef`, #745) and is rebased onto `main`; §374 goes after §359 and §371.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI with §374 passing; the four Go tests pass on Linux and Windows CI; `adr-debt` reports nothing broken or unreceipted; every gate in `AGENTS.md` passes; T1 is done; Zy has accepted Decision 1a.

## Out of Scope

- Records an mrw before T2 wrote for a `search_replace`-only write (permanent: boundary: the record's Out of Scope; closing them needs a format bump and a reset of v4 to v6 ledgers)
