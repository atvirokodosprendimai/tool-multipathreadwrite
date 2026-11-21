# Task ADR-209-T1: a failed commit puts the content files back

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the content asides and their undo in `internal/apply`; a link and a file-only removal through the root in `tree.go`; the final-state receipt and the ledger mark; contract §360, and §119, §141 and §196 updated; Decision 8's words in `README.md` and `internal/mcp/tools.go`; the cost, measured
**Consumes:** `commitFailed`, `commitPathOps`' undo, `writtenSoFar`, `res.noteIfLeft`, `res.cleanUp`, `commitRenameFn`, `changedSince`'s comparison, `os.Root.Link`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a failed content rename puts the earlier files back`, `a failed path op puts the content files back`, `a nested create and its directories are removed`, `the undo runs newest first`, `a file edited in place since the rename is left and its aside kept`, `a restore never replaces`, `a kept aside holds the previous bytes`, `a file with no aside commits as today and is named`, `the receipt and ledger describe the final tree`, `a landed write leaves no aside`, `a cleanup that fails is left_behind`, `the file ID is taken before the rename`, `every subtest ran and none skipped`, `the CLI says NOTHING WRITTEN and restores the file`, `the contract rows exist with their trigger`, `the new README sentence is present`, `the engine packages are untouched, tracked and untracked`

## Goal

ADR-209 Decisions 1 to 8.

**`TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed`** (`internal/apply`), one subtest per row, over `a.txt`, `b.txt`, `c.txt` (one line each, read whole), with seams on the rename, the link and the removal that record each call in order. "Restored" means the bytes and the identity (`os.SameFile` against a `Stat` taken, its file ID loaded, before the write) are the originals, and no `.mrw-*` entry is left in the tree:

| Subtest | Plan and seam | Outcome |
|---------|---------------|---------|
| `third_rename_refused` | replace a line in each; the third content rename refused | `a.txt`, `b.txt` restored; the undo calls recorded for `b.txt` before `a.txt`; the third hunk `failed`, the rest `skipped`; no file `written`; the error carries no `ALREADY WRITTEN` |
| `path_op_refused` | replace in `a.txt`, unlink `d/x.txt`; the unlink refused | `a.txt` restored; `d/x.txt` in place; the error built after both undos |
| `nested_create` | create `n/m/k.txt` (new directories `n`, `n/m`), replace in `a.txt`; the `a.txt` rename refused after `k.txt` landed (asserted through the seam) | `n` absent; `a.txt` unchanged |
| `edited_in_place_after_rename` | replace in each; the third rename refused; the seam appends to `a.txt` before the undo reaches it | `a.txt` left with the append; its aside kept, holding the original bytes, named in `left_behind` and `UNDO INCOMPLETE`; `a.txt` `written`, its hunk `failed` with the undo's reason; `b.txt` restored, its hunk `skipped`; `PARTIALLY APPLIED`; `a.txt`'s result carries the mark that it does not hold mrw's bytes |
| `path_taken_before_restore` | as above, but the seam creates `b.txt` after the undo removes it | the restore link fails; `b.txt` keeps the seam's file, `written`, its hunk `failed`; its aside kept, holding the original bytes, and named |
| `no_aside_for_one_file` | replace in each; the link seam refuses `b.txt`; the third rename refused | the removal seam records nothing for `b.txt`; `a.txt` restored; `b.txt` holds mrw's bytes, `written`, its hunk `ok`; `UNDO INCOMPLETE` names `b.txt` and the link's error; `PARTIALLY APPLIED` |
| `edit_and_rename_move_back_refused` | replace in `a.txt` and rename it to `e/a.txt` (ADR-114), then an unlink refused; the seam refuses the rename back | `e/a.txt` holds the edit and stays `written`, both its hunks `ok`; the aside of `a.txt` kept and named |
| `landed` | replace in each, no seam failing | applied; no `.mrw-aside-*` in the tree |
| `aside_removal_refused` | replace in each; the removal seam refuses one aside after landing | applied; that aside in `left_behind` |

**`TestTheCLIPutsTheContentFilesBackWhenACommitFails`** (`cmd/mrw`, the built binary): `a.txt` and `d/x.txt` read; `a.txt` replaced and `d/x.txt` unlinked in a directory made read-only (skipped as root, saying why): exit 2, `NOTHING WRITTEN`, `a.txt` restored, `mrw stats --json` counts `refused_apply`, and the next write to `a.txt` lands without a new read.

**`TestTheLedgerSkipsAFileWhosePathDoesNotHoldWhatMrwWrote`** (`internal/writer`): the writer's ledger step, fed an `apply.Result` whose file carries the mark of Decision 6 beside one that does not, records the second and not the first; with every file restored, it records none.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the asides, the staged temp's record, the content undo, the error built after it, the cleanup |
| `internal/apply/pathop.go` | edit | its error text built after the content undo; hand-off to it |
| `internal/apply/tree.go` | edit | `link` and a file-only `removeFile` through the root, behind seams |
| `internal/writer/writer.go` | edit | records only files whose path holds what mrw wrote |
| `internal/writer/undo209_test.go` | new | the ledger test |
| `internal/apply/undo209_test.go` | new | the first table |
| `cmd/mrw/undo209_test.go` | new | the CLI test |
| Go tests that assert a partial commit after a content rename | edit | their expected outcome is now the undone one. `git grep -ln -e 'ALREADY WRITTEN' -e 'PARTIALLY APPLIED' -- '*_test.go'` names 19 files at `2ded135b` across `cmd/mrw`, `internal/apply`, `internal/mcp`, `internal/guide` and `internal/bench`, among them the CLI `--json`, MCP, stats and refused-stdin tests; some assert only the words. Each is read, and those whose fixture renames a content file before the failure are updated; `TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped` (`internal/apply/pathop_commit_test.go:271`), which §119 names, becomes the undone case |
| `internal/mcp/tools.go` | edit | the `left_behind` sentence |
| `scripts/contract.sh` | edit | §360; §119's pointer; §141; §196 |
| `README.md` | edit | the commit, undo and process-death rows |

## Ordered Steps

1. [S1] The two tests; RED on every subtest but `landed` and `aside_removal_refused`'s first half (an applied write), and on the CLI test. [proof: acceptance]
2. [S2] The asides and the undo. Mutants, each killed: no aside made (`third_rename_refused`); the content undo skipped after a path op failed (`path_op_refused`); the undo oldest first (the recorded call order in `third_rename_refused`); created directories kept (`nested_create`); identity alone, without size and time (`edited_in_place_after_rename`); the restore by rename in place of link (`path_taken_before_restore`); a refused link stops the write (`no_aside_for_one_file`); a file with no aside removed by the undo (`no_aside_for_one_file`); a `written` file's hunk left `ok` when its path holds another file (`edited_in_place_after_rename`); the error text built before the undo (`third_rename_refused`); the writer ignoring the mark (`TestTheLedgerSkipsAFileWhosePathDoesNotHoldWhatMrwWrote`); the asides not removed after landing (`landed`); the file ID taken after the rename (Windows CI, `third_rename_refused`). [proof: mutation]
3. [S3] Contract §360 drives `$MRW`: the row in the record's Wiring table, with `ls -i` before and after for the edit's inode, paired with the writable directory applying all three; §196, §141 and §119 updated. The sentences. [proof: acceptance]
4. [S4] The cost: ADR-105 T3's protocol (a 500-file, 5,000-hunk write and a one-file write, 5 alternating runs each, the old build against the new, each on a fresh copy with its own `XDG_STATE_HOME`) on the owner's Mac and the Linux and Windows CI runners; the medians, the load and the filesystem go into the record's Consequences before it is accepted. [proof: human: the medians, the load, the filesystem and the machine of each run, recorded in the record]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./internal/writer ./cmd/mrw -run '^(TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed|TestTheLedgerSkipsAFileWhosePathDoesNotHoldWhatMrwWrote|TestTheCLIPutsTheContentFilesBackWhenACommitFails)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && [ "$(grep -c '^    --- PASS: TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed/' "$out")" -eq 9 ] \
  && ! grep -q -- '--- SKIP: TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed' "$out" \
  && grep -q '^# 360\. ' scripts/contract.sh \
  && grep -q 'ls -i' <(sed -n '/^# 360\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) \
  && ! grep -q 'PARTIALLY APPLIED' <(sed -n '/^# 196\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) \
  && readme="$(tr -s '\n ' '  ' < README.md)" \
  && grep -qF 'a commit failure puts the content files back' <<<"$readme" \
  && grep -qF 'PARTIALLY APPLIED names what could not be put back' <<<"$readme" \
  && grep -qF 'after a killed undo the path can be missing, its content only in the aside' <<<"$readme" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed` | `internal/apply/undo209_test.go` | the first table, by the bytes, the identity, the directory listing, the recorded call order, the receipt and the ledger | none | S1, S2 |
| `TestTheCLIPutsTheContentFilesBackWhenACommitFails` | `cmd/mrw/undo209_test.go` | exit 2, `NOTHING WRITTEN`, the file restored, the tally, the licence kept | none | S1, S2 |
| `TestTheLedgerSkipsAFileWhosePathDoesNotHoldWhatMrwWrote` | `internal/writer/undo209_test.go` | the ledger records only files whose path holds what mrw wrote | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §360 |
| 2 — something selects it | every write with an existing content target, on both surfaces |
| 3 — the caller can discover it | the receipt (`NOTHING WRITTEN`, `UNDO INCOMPLETE`, `left_behind`); `README.md`'s failure table |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is §196, which reaches the half-plan from the built binary |

## Mutation Log
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · no aside is made · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a failed content rename puts the earlier files back
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · the content undo is skipped after a path op failed · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a failed path op puts the content files back
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · the undo runs oldest first · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:the undo runs newest first
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · the directories staging made are kept · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a nested create and its directories are removed
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · identity alone, without size and modification time · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a file edited in place since the rename is left and its aside kept
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · the restore is a rename, not a link · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a restore never replaces
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · a refused link stops the write · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a file with no aside commits as today and is named
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · a file with no aside is removed by the undo · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a file with no aside commits as today and is named
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · a written file's hunk is left ok when its path holds another file · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:the receipt and ledger describe the final tree
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · the error text is built before the undo · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:the receipt and ledger describe the final tree
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/writer/writer.go` · the writer ignores the mark · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:the receipt and ledger describe the final tree
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · the asides are not removed after landing · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a landed write leaves no aside
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/apply.go` · a failing aside removal is swallowed, not named · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a cleanup that fails is left_behind
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · the kept aside is not named in left_behind · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:a kept aside holds the previous bytes
- 2026-11-20 · a4040443* · mutant killed · exit 1 · `internal/apply/undo209.go` · the restored file's record is not dropped · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · covers:the receipt and ledger describe the final tree

## Verification Log
- 2026-11-20 · a4040443* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:846 · test-lock-sha256:937e571fc61cef853c88eb715060b8dd576c73696dab0c34408cfb18214c4d17 · test-lock-file:docs/adr/locks/937e571fc61cef853c88eb715060b8dd576c73696dab0c34408cfb18214c4d17.lock
  ```
  ```
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2902
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:5930
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2556
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:4821
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2887
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:3520
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2983
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2936
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:4261
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:3470
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:3650
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:3385
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2747
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:3714
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2978
- 2026-11-20 · a4040443* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2566
- 2026-11-20 · 677cf538 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:1039
- 2026-11-20 · 5cae6024 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:1332
- 2026-11-20 · human-observed · S4 on macOS only (the owner's Mac, APFS on the internal SSD, 10 cores), main `751edd9b` against this branch `36c12461`, both built with `go build ./cmd/mrw`; ADR-105 T3's fixture (500 files of 100 lines in 20 directories, 10 single-line replaces per file, every file read first, `mrw write --no-check` timed alone, each run on a fresh copy with its own `XDG_STATE_HOME`) and a one-file write; 21 alternating runs each. Medians (min, max): 500 files and 5,000 hunks 259.7 ms (219.5, 357.2) on main, 378.8 ms (341.1, 681.9) on the branch, +119 ms (+46%), 0.24 ms per file; one file 14.2 ms (11.4, 27.2) on main, 16.95 ms (11.4, 48.5) on the branch, +2.8 ms (+19%). Load average (1 minute) was under 8 when the runs began and 7.4 to 9.4 at their end, other sessions running: the maxima are noise. Linux and Windows are not measured yet: from CI runners or peers.
- 2026-11-20 · 43cd4d4e · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:2156
- 2026-11-21 · human-observed · the aside's name lookup on macOS only (the owner's Mac, APFS, 10 cores, 1-minute load 6.4 to 7.0): one-file write to a directory of 100,000 entries, hard-linked into a fresh copy each run, read first, `mrw write --no-check` timed alone, 7 alternating runs of three builds: main `751edd9b` 14.2 ms median (13.4 min), the branch with the whole directory listed and each entry stat'ed to find the stored spelling (`32b6114d` plus the file-only removal) 405.0 ms (332.8), the branch with the spelling taken from the open descriptor's path (F_GETPATH, one name only) and a listing only where that is not enough 16.7 ms (13.1). Linux needs no listing on a volume that keeps case apart (a case-swapped lookup finds nothing); Windows lists the directory and is not measured.
- 2026-11-21 · 43e31c5e · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:1482
- 2026-11-21 · 196dccef · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:954
- 2026-11-21 · 71c39da9 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:1007
- 2026-11-21 · human-observed · correction of the row above on the lookup (the owner's Mac, 1-minute load 3.7): the claim that Linux needs no listing on a volume that keeps case apart held only for a name with a letter to fold; the lookup now also returns without listing for a name with no letter and no non-ASCII byte, and for a non-ASCII name on a volume that keeps case apart (a lookup of the name with its case reversed finds nothing). A file with two names no longer lists the directory on darwin when the descriptor's path names the plan's spelling. Measured in a directory of 100,000 entries, 5 alternating runs, main against this branch: one name 12.6 ms and 22.2 ms, two names 12.2 ms and 21.6 ms (the reviewer's earlier figures: two names 18 to 19 ms on main and 189 to 195 ms on the branch before this change). Linux and Windows are not measured.
- 2026-11-21 · human-observed · the final table of one-file writes, which supersedes the two rows above on the lookup (the figures 14.2 to 16.7 ms and 12.6 to 22.2 ms): main `10b83cc2` against the branch `6584d9e1`, 9 rotating runs each and a second copy of main as a control, the owner's Mac, load 3.2 to 6.2: (a) 20 entries main 18.6 ms, branch 17.7 ms, control 17.8 ms; (b) 100,000 entries, one name 21.3, 21.1 and 20.5 ms; (c) two names 21.2, 21.0 and 20.7 ms. The +9.6 ms of the row before came from comparing a main built before ADR-210 with a branch that holds ADR-210's directory and temp syncs: ADR-210's cost, not this record's. The earlier 14.2 to 16.7 ms row compared two builds without ADR-210 and is consistent with this one inside noise. Linux and Windows are not measured.
- 2026-11-21 · cb028dcb · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:982
- 2026-11-21 · 6463edea · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:6a24744ed070c38f252d0bbaacded5e8df65ce9ee5137547e629796a8fe282c8 · ms:5007

## Invariants

- Each content file still lands by one rename over its target.
- No receipt key is added, removed or renamed.
- Nothing is restored over a path that holds something; nothing is removed that does not match the file mrw renamed there. A restore follows a parent directory swapped for a link inside the root after the commit, as the forward rename would (the record's Decision 5).

## Risks

- After §196 and §141 assert the undone outcome, no contract row drives `PARTIALLY APPLIED` or `partially_applied` from the built binary: both are then reachable only through a failed undo step or a refused link, which no fixture outside the process can force. The Go subtests `edited_in_place_after_rename`, `path_taken_before_restore`, `no_aside_for_one_file` and `edit_and_rename_move_back_refused` carry them, on Linux and Windows CI.
- The Verification Log row stamped `677cf538` is superseded by the `5cae6024` row: that commit is unreachable after the rebase onto main (it is `e2779ac4` now). The row is kept, never deleted.
- The aside's name lookup is measured on macOS only. On Linux a volume that keeps case apart is detected by one lookup of the name with its case reversed (also for a non-ASCII name), which avoids a listing, but a rooted `ReadDir` stats every entry it lists (Go 1.27.4, `os/file_unix.go`, `newUnixDirent`), so any directory that is listed pays for it: the Linux cost is not measured, and T1 stays `partial` until it is.
- S4 is measured on macOS only (the log row above): Linux and Windows come from CI runners or peers, and the task stays partial until they do.
- §360 and the CLI test need a directory the runner cannot write; as root they skip, as §196 does. The fence does not accept that skip for the Go table, whose rows use seams.
- The fence proves §360's text and §196's change, not that they ran: they are proved by `scripts/contract.sh`, a gate run before the commit (`lifecycle.md` §6).
- The Windows CI job is the first place the link and the aside's removal meet a held file; a failure there is a finding for the record, not a test to loosen.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI, as a user that is not root, with §360's failure case and its control both `PASS` and neither `SKIP`; `adr-debt` reports nothing broken or unreceipted; every gate in `AGENTS.md` passes on Linux and Windows CI; S4's numbers are in the record; and the record is Accepted by Zy.

## Out of Scope

- Locking other processes out (permanent: boundary: the record's Out of Scope; ADR-075 locks only mrw writers)
