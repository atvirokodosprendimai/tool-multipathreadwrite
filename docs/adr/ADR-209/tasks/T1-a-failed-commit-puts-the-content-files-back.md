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
| `edited_in_place_after_rename` | replace in each; the third rename refused; the seam appends to `a.txt` before the undo reaches it | `a.txt` left with the append; its aside kept, holding the original bytes, named in `left_behind` and `UNDO INCOMPLETE`; `a.txt` `written`; `b.txt` restored; `PARTIALLY APPLIED`; the ledger has no record of `a.txt` at the planned sha |
| `path_taken_before_restore` | as above, but the seam creates `b.txt` after the undo removes it | the restore link fails; `b.txt` keeps the seam's file; its aside kept, holding the original bytes, and named |
| `no_aside_for_one_file` | replace in each; the link seam refuses `b.txt`; the third rename refused | `a.txt` restored; `b.txt` changed and `written`; `UNDO INCOMPLETE` names `b.txt` and the link's error; `PARTIALLY APPLIED` |
| `edit_and_rename_move_back_refused` | replace in `a.txt` and rename it to `e/a.txt` (ADR-114), then an unlink refused; the seam refuses the rename back | `e/a.txt` holds the edit and stays `written`; the aside of `a.txt` kept and named; each hunk keeps its own verdict |
| `landed` | replace in each, no seam failing | applied; no `.mrw-aside-*` in the tree |
| `aside_removal_refused` | replace in each; the removal seam refuses one aside after landing | applied; that aside in `left_behind` |

**`TestTheCLIPutsTheContentFilesBackWhenACommitFails`** (`cmd/mrw`, the built binary): `a.txt` and `d/x.txt` read; `a.txt` replaced and `d/x.txt` unlinked through a seam-free failure where the runner can make one (a read-only directory; skipped as root), else the subtest says why: exit 2, `NOTHING WRITTEN`, `a.txt` restored, `mrw stats --json` counts `refused_apply`, and the next write to `a.txt` lands without a new read.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the asides, the staged temp's record, the content undo, the error built after it, the cleanup |
| `internal/apply/pathop.go` | edit | its error text built after the content undo; hand-off to it |
| `internal/apply/tree.go` | edit | `link` and a file-only `removeFile` through the root, behind seams |
| `internal/writer/writer.go` | edit | records only files whose path holds what mrw wrote |
| `internal/apply/undo209_test.go` | new | the first table |
| `cmd/mrw/undo209_test.go` | new | the CLI test |
| Go tests that assert a partial commit after a content rename | edit | their expected outcome is now the undone one. `git grep -ln -e 'ALREADY WRITTEN' -e 'PARTIALLY APPLIED' -- '*_test.go'` names 19 files at `2ded135b` across `cmd/mrw`, `internal/apply`, `internal/mcp`, `internal/guide` and `internal/bench`, among them the CLI `--json`, MCP, stats and refused-stdin tests; some assert only the words. Each is read, and those whose fixture renames a content file before the failure are updated; `TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped` (`internal/apply/pathop_commit_test.go:271`), which §119 names, becomes the undone case |
| `internal/mcp/tools.go` | edit | the `left_behind` sentence |
| `scripts/contract.sh` | edit | §360; §119's pointer; §141; §196 |
| `README.md` | edit | the commit, undo and process-death rows |

## Ordered Steps

1. [S1] The two tests; RED on every subtest but `landed` and `aside_removal_refused`'s first half (an applied write), and on the CLI test. [proof: acceptance]
2. [S2] The asides and the undo. Mutants, each killed: no aside made (`third_rename_refused`); the content undo skipped after a path op failed (`path_op_refused`); the undo oldest first (the recorded call order in `third_rename_refused`); created directories kept (`nested_create`); identity alone, without size and time (`edited_in_place_after_rename`); the restore by rename in place of link (`path_taken_before_restore`); a refused link stops the write (`no_aside_for_one_file`); the error text built before the undo (`third_rename_refused`); the ledger mark ignored (`edited_in_place_after_rename`); the asides not removed after landing (`landed`); the file ID taken after the rename (Windows CI, `third_rename_refused`). [proof: mutation]
3. [S3] Contract §360 drives `$MRW`: the row in the record's Wiring table, with `ls -i` before and after for the edit's inode, paired with the writable directory applying all three; §196, §141 and §119 updated. The sentences. [proof: acceptance]
4. [S4] The cost: ADR-105 T3's protocol (a 500-file, 5,000-hunk write and a one-file write, 5 alternating runs each, the old build against the new, each on a fresh copy with its own `XDG_STATE_HOME`) on the owner's Mac and the Linux and Windows CI runners; the medians, the load and the filesystem go into the record's Consequences before it is accepted. [proof: human: the medians, the load, the filesystem and the machine of each run, recorded in the record]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./cmd/mrw -run '^(TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed|TestTheCLIPutsTheContentFilesBackWhenACommitFails)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 2 ] \
  && [ "$(grep -c '^    --- PASS: TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed/' "$out")" -eq 9 ] \
  && ! grep -q -- '--- SKIP: TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed' "$out" \
  && grep -q '^# 360\. ' scripts/contract.sh \
  && grep -q 'ls -i' <(sed -n '/^# 360\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) \
  && ! grep -q 'PARTIALLY APPLIED' <(sed -n '/^# 196\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) \
  && grep -q 'a commit failure puts the content files back' <(tr -s '\n ' '  ' < README.md) \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed` | `internal/apply/undo209_test.go` | the first table, by the bytes, the identity, the directory listing, the recorded call order, the receipt and the ledger | none | S1, S2 |
| `TestTheCLIPutsTheContentFilesBackWhenACommitFails` | `cmd/mrw/undo209_test.go` | exit 2, `NOTHING WRITTEN`, the file restored, the tally, the licence kept | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §360 |
| 2 — something selects it | every write with an existing content target, on both surfaces |
| 3 — the caller can discover it | the receipt (`NOTHING WRITTEN`, `UNDO INCOMPLETE`, `left_behind`); `README.md`'s failure table |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is §196, which reaches the half-plan from the built binary |

## Mutation Log

## Verification Log

## Invariants

- Each content file still lands by one rename over its target.
- No receipt key is added, removed or renamed.
- Nothing is restored over a path that holds something; nothing is removed that does not match the file mrw renamed there.

## Risks

- After §196 and §141 assert the undone outcome, no contract row drives `PARTIALLY APPLIED` or `partially_applied` from the built binary: both are then reachable only through a failed undo step or a refused link, which no fixture outside the process can force. The Go subtests `edited_in_place_after_rename`, `path_taken_before_restore`, `no_aside_for_one_file` and `edit_and_rename_move_back_refused` carry them, on Linux and Windows CI.
- §360 and the CLI test need a directory the runner cannot write; as root they skip, as §196 does. The fence does not accept that skip for the Go table, whose rows use seams.
- The fence proves §360's text and §196's change, not that they ran: they are proved by `scripts/contract.sh`, a gate run before the commit (`lifecycle.md` §6).
- The Windows CI job is the first place the link and the aside's removal meet a held file; a failure there is a finding for the record, not a test to loosen.

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 with §360's rows printed, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes on Linux and Windows CI, S4's numbers are in the record, and the record is Accepted by Zy.

## Out of Scope

- Locking other processes out (permanent: boundary: the record's Out of Scope; ADR-075 locks only mrw writers)
