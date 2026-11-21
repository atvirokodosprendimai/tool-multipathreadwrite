# ADR-209: A content commit that fails partway undoes the files it already renamed

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accepted the rest"*, to the question of accepting the open records ADR-207 to ADR-211 (#723, #720) as they stand.
**Drafted for:** Zy, 2026-11-20, from an independent critique of the same day (finding S1). Zy's instruction on the critique's findings, as relayed to the drafter by the session that commissioned these records, was *"address all, no choosing"*, so this record carries one decision. Challenged before acceptance (Codex at xhigh, 2026-11-20, on `2ded135b`: AMEND); every amendment is applied below or answered in History.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-057, ADR-066, ADR-071, ADR-075, ADR-102, ADR-105, ADR-106, ADR-114, ADR-125, ADR-132
**Supersedes:** in part:
- ADR-066's rejected alternative "Undo content renames too" and its Out of Scope item "Undoing content renames already committed (permanent: boundary: …)";
- ADR-001's amendment that "the partial tree ADR-001 accepts is now the content files already renamed into place", which becomes the files an undo step could not put back and the files no aside could be made for;
- ADR-106 Decision 3's "earlier files stay written … since content renames are never undone": after the second identity check stops the commit, the earlier files are put back;
- ADR-071 T2's commit-time fold outcome (contract §141: the first create's body kept, `PARTIALLY APPLIED`): the first create is removed and nothing is written;
- ADR-102's contract §196, whose read-only-directory trigger now writes nothing, counts `refused_apply`, and leaves `a.txt` licensed by the read made before it.
- ADR-114 Decision 3's "If a later path op fails and the undo reverses this rename, the source's record is restored to the edit that landed … so the receipt and the ledger say the source holds the edited content": the content undo then puts the source's original back and its record drops out; where it cannot, the record says what the path holds (Decision 6).

ADR-066 Decision 2 (path ops undone newest first) stands. ADR-066 Decision 3 stands but for one clause: "Hunks whose file reached disk are `ok`" becomes "hunks whose path holds what they wrote are `ok`", so a file whose path, after an undo, is missing or holds another process's file has its hunks `failed` though it stays `written` (Decision 6). ADR-132's exit classes are unchanged.
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/apply/tree.go`, `internal/writer/writer.go`, `internal/mcp/tools.go`, `scripts/contract.sh`, `README.md`
**Enforced-by:** `internal/apply/undo209_test.go::TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed`
**Served-path change:** a write whose commit fails after some content files were renamed into place puts those files back and reports `NOTHING WRITTEN`, exit 2, where it reported `PARTIALLY APPLIED`. `PARTIALLY APPLIED` (exit 2, with `UNDO INCOMPLETE`) remains for two cases only: an undo step that could not put a file back, and a file for which no aside could be made before the commit (a filesystem with no hard links, a link the system refuses); the receipt names each. A killed run can leave a `.mrw-aside-*` beside a content file: its previous version, by a second name.

## Context

**What happens today.** Staging writes every content file to a temp beside its target (`stageFile`, `internal/apply/apply.go:2505-2562`), and every check that can be made before the first rename is made there (ADR-001, ADR-106, ADR-125; `:1167-1196`). Then the commit loop renames each temp over its target, one at a time (`:1197-1226`), keeping nothing of the file it replaces. A failure at file *i* — a create whose name another rename folded into (`:1206`), a target another process changed (`changedSince`, `:1212`), a rename refused (`:1217`) — returns `commitFailed` with files 0 to *i*-1 changed on disk, named `ALREADY WRITTEN` (`writtenSoFar`, `:1244`): `PARTIALLY APPLIED`, exit 2. A path-op failure after the content loop (`commitPathOps`, `:1227-1233`) undoes the path ops (ADR-066) and leaves the content files changed; its error text, `ALREADY WRITTEN` included, is built before it returns (`internal/apply/pathop.go:447`). The receipt is honest; the tree is half a plan.

**What the path ops already do.** An unlink moves its file to a `.mrw-aside-*` name. Unlinks that vacate a rename's destination run before the renames, the other unlinks after them, all after the content loop (`pathop.go:452-459`). A failure undoes every rename newest first and then every aside (`pathop.go:291-331`); the rename back is a replacing rename (`commitRenameFn`, `:297`), so ADR-066's "no undo step overwrites a file" is a statement about the plan's own paths, not a no-replace operation. ADR-066 rejected the same undo for content files: "the originals are already replaced; keeping backups of every file changes ADR-001's staging design for a failure staging now makes rare" (`docs/adr/ADR-066-…md:105`).

**Reachable from the built binary.** Contract §196 (`scripts/contract.sh:8026-8046`) drives one on Linux with no seam: `a.txt` replaced and `d/x.txt` unlinked in a read-only directory exits 2 `PARTIALLY APPLIED` with `a.txt` changed. On a filesystem that folds `ß` and `ss` (APFS), §141 (`:6753-6766`) drives a second.

**How an aside could be made, measured** (scratch program, not committed; `go run . DIR` on the owner's Mac, APFS on Apple Fabric SSD, 10 cores, load average 6.5 before and 7.0 to 9.2 after, 2026-11-20; 9 alternating runs each, the publish step only, existing 4-byte targets replaced by ~4 KB files in one directory):

| n files | rename only (today) | + rename the target aside first | + hard-link the target aside first |
|---------|--------------------|--------------------------------|------------------------------------|
| 1 | 0.110 ms | 0.154 ms | 0.328 ms |
| 50 | 6.2 ms | 8.7 ms | 17.6 ms |
| 500 | 66 ms | 86 ms | 180 ms |

Moving the target aside (the path ops' way) is cheaper, but between the two renames the target's path holds nothing: a reader can find it missing, and a run killed there leaves it missing. `README.md:464` says "Each file lands by a rename, so no file is ever half-written". A hard link leaves the target in place, so the commit stays one rename per file and a killed run leaves every path holding its old or its new content. `changedSince` compares the leaf's kind, then identity, size and modification time (`apply.go:2631-2667`); a link changes the link count and the change time, neither of which it reads.

**Where a link can be refused.** A filesystem with no hard links (FAT, exFAT, some network and FUSE mounts); Linux `fs.protected_hardlinks` for a file the user does not own; on Windows, Go 1.27.4's `Root.Link` opens the source for `FILE_WRITE_ATTRIBUTES` (`internal/syscall/windows/at_windows.go`, `Linkat`), while ADR-125's probe asks for `DELETE` (`internal/apply/replace_windows.go`), so passing that probe does not show a link can be made. mrw cannot sort these causes reliably, so it does not try (Decision 2).

**Audit of the class**: *the ways a commit can stop after the first rename.* A content rename refused; a target changed after staging; a create's name taken; a path op refused (an unlink or a rename, ADR-066). The first three are in the content loop, the last after it. All four leave content files renamed today.

## Existing Primitives Audit

- `commitFailed` renders a receipt from `res.Files`; `commitPathOps`' undo drops the records of what it put back (`pathop.go:323-329`).
- `noteIfLeft` and `cleanUp` name what mrw made and could not remove (`left_behind`, ADR-105).
- `os.Root.Link` exists in the Go the module names (`go 1.27.4`), so the link goes through the same root as every other tree operation (`internal/apply/tree.go`).
- A hard link never replaces its destination, on POSIX and on Windows.
- ADR-106 already loads a file's ID when it is stat'ed, because on Windows `os.SameFile` otherwise loads it lazily by path and answers false when that fails (`apply.go:649`, beside the comparison in `changedSince`, `:2647-2652`).

## Decision

1. **An aside for every existing content target, before the first rename.** After the checks that already run before the first rename (`apply.go:1167-1196`), each content target that exists is hard-linked under a fresh `.mrw-aside-*` name in its own directory, through the root. A name already taken is another name.
   - 2026-11-21, review of #760: the aside also records the spelling the directory lists for the target, chosen by identity, and the restore links back under it, so a plan that names `a.txt` as `A.TXT` on a volume that folds case does not rename the file by failing. Where the directory lists the file under several names and none is the plan's, nothing is guessed: no aside is made, and the file is reported as one the undo cannot put back (Decision 2). The undo removes the renamed file by moving it to a private name and looking at it there, since `os.Root` has no unlink that cannot take an empty directory; a file that vanished before the removal is treated as removed.
2. **A link that cannot be made does not stop the write.** That file commits as today, with no aside, whatever the reason the link was refused; ADR-132's exit classes are unchanged. If the commit later fails, that file cannot be put back: the receipt reports it `written`, `PARTIALLY APPLIED`, and `UNDO INCOMPLETE` names it with the link's error. A write that lands reports nothing about it.
3. **What the undo checks before it removes anything.** Before each content rename, mrw stats the staged temp, with its file ID loaded then (as `apply.go:649` does for a target), and keeps that record. The undo removes a path only when it is a regular file that matches that record by identity, size and modification time (`changedSince`'s comparison), so a file another process put there or edited in place since the rename is left alone. It removes it with a file-only removal (never a directory, which `tree.remove` would also take, `tree.go:145-152`), at the resolved path the forward rename used (`staged[i].target`, ADR-106). That path is resolved, but `os.Root` still follows a relative link inside the root met on the way (`tree.go:20-29`): a parent directory swapped for such a link after the commit is followed by the undo, as it would be by the forward rename. The identity check means the undo removes nothing but the file it renamed in; the link back can then land where the swapped parent leads, still inside the root (Decision 5).
4. **The undo, newest first, after the path ops' own undo.** For each content file renamed in, newest first: a file with no aside (Decision 2) is left as written, and nothing is removed for it; any other is checked (Decision 3) and removed; for a file that existed, its aside is linked back to the path and then removed. A link never replaces, so a path something else holds by then keeps that thing, and the aside stays as a recovery file. A created file is removed and the directories staging made for it are removed as today. Whatever could not be put back is named under `UNDO INCOMPLETE` and in `left_behind`.
5. **What is promised about other processes, and what is not.** mrw's write lock orders mrw writers (ADR-075); nothing locks out other processes. During an undo: a change another process makes to the path between the check and the move is put back by the move when it is not the file mrw wrote, an entry put at mrw's own private name or at an aside's name between a check and the act that follows is not caught (a named limit), and nothing is promised for the instant between the move and the link back; between the removal and the link back, the path holds nothing; a parent directory another process swaps for a link inside the root is followed, so the restored file can land where that link leads; a process holding the old file open, or another name for it, can change the aside's bytes, and what is restored is that file as it is then — what the path would hold had mrw not run. The forward commit has the same check-then-act window before each rename (ADR-106), and the same parent-link exposure (ADR-071, `tree.go:20-29`).
6. **The receipt and the ledger describe the final tree.** `ALREADY WRITTEN` and every error text are built after both undos, not before. Two things are kept apart. A file's `written` (`files[].written`, unchanged in meaning) is true when its path no longer holds its original: restored files are not written and their records drop out; a file with no aside, a file another process changed after the rename, and a file whose restore link failed stay `written`. A hunk's verdict says whether its path holds what the hunk wrote: `ok` for a file with no aside, which still holds mrw's bytes; `failed`, with the undo's reason, for a file whose path is now missing or holds another process's file (today's builder leaves any `written` file's hunks `ok`, `apply.go:1022-1028`, which T1 changes); `skipped` for a restored file, as ADR-066 Decision 3 says for a file not written. The ledger records a file only when its path holds the bytes mrw wrote, so a file left absent or held by another process is not recorded (an internal mark on the file's result, not a receipt key, read by the writer, `internal/writer/writer.go:98-108`). ADR-114's edit and rename of one file keep their own verdicts by the same rule. `mrw stats` counts `refused_apply` when nothing is `written`, `partially_applied` otherwise (ADR-102).
   - 2026-11-20, in building T1: where a path op fails and the file also has an edit hunk (ADR-114's edit and rename of one file), the edit hunk now reports `skipped`, not `failed` and not `ok`. The commit marks the failing hunk by its path, which would also fail the edit of that file with the rename's reason; but the edit was put back with the rename, so it is the rename that failed and the edit that did not land, and the edit's reason would be false. `commitFailed` (`internal/apply/apply.go`) takes the edit hunks of the failing path as skipped. The verdict of the hunk that failed, the unlink or the rename itself, is unchanged.
7. **After a commit that landed, every aside is removed**; one that cannot be is in `left_behind`.
   - 2026-11-21, merged with ADR-210: the order of one write is the staged temps synced, the asides linked, the renames, the asides removed, and only then the changed directories synced. A directory an aside is made or removed in is its target's, which the changed directories recorded for ADR-210 already hold; syncing after the removal makes the removal as stable as the renames, so a crash cannot leave an aside the synced state does not know about. A `SyncError` after landing is a landed write whose asides are already gone. A commit that fails and is put back syncs no directory: nothing was written, and `NOTHING WRITTEN` is as true of the page cache as of the disk; an undo that could not finish is a partial write, which ADR-210 promises nothing of either.
   - 2026-11-21, the targeted final pass on #760: the undo no longer depends on deleting what it moved. The renamed file is moved into a private `.mrw-undo-*` directory made exclusively (so the move replaces nothing, and a directory swapped in is moved whole and looked at there), the previous version is linked back into the vacant path first, and the private entry is deleted last: a deletion that fails is a named leftover in `left_behind`, whatever the entry is and whatever it holds, and the undo is complete. An entry that is not the file is put back where it was; if it cannot be, it stays in the private directory, that directory is named, and the path is marked as not holding mrw's bytes. A README never tells a user to delete a `.mrw-undo-*` entry: it can hold the only copy of what another process put at the target. Named limit: an entry another process swaps in at mrw's own private name between its check and its deletion cannot be closed here, because `os.Root` offers no file-only unlink (`unlinkat` with flags 0), and its `Remove` falls back to removing an empty directory. `docs/adr/BACKLOG.md` carries the entry; ADR-213 (Proposed, `golang.org/x/sys`) would supply both the file-only unlink and a no-replace rename. The limit is stated, not closed. The aside is also checked under its own name against what the plan loaded before the first rename, so a file that stood at the path for the instant the link was made refuses the write, exit 1, nothing written.
   - 2026-11-21, the last round on #760: the aside's name is a window of the same kind. It sits in the target's directory, which other processes can write, for the whole commit. It is checked against what the plan loaded before the first rename, before it is linked back and before each removal, and an entry that is not that file is neither linked back nor removed: it is named in `left_behind`, and the file mrw wrote stays at the path (moved back from the private directory). The instant between a check and the act that follows it is not closed, for the same reason as the private name: it needs `unlinkat` flags 0 and a no-replace rename that `os.Root` does not offer (BACKLOG, ADR-213).
8. **The words.** `README.md`'s failure table and the MCP `left_behind` sentence (`internal/mcp/tools.go:1685`) say, in these words: the commit row, `a commit failure puts the content files back`; the undo row, `PARTIALLY APPLIED names what could not be put back`; the process-death row, `a .mrw-aside-* beside a content file is its previous version; after a killed undo the path can be missing, its content only in the aside`.

## Alternatives Considered

- **Rename the target aside, as path ops do.** Rejected: the cheapest (above), but the target's path is empty between two renames and a killed run can leave it so; the commit would stop being one rename per file.
- **Refuse a write when an aside cannot be made.** Rejected: it stops every write on a filesystem with no hard links, and the causes of a refused link cannot be sorted into ADR-132's classes reliably.
- **Restore from the bytes apply already holds in memory.** Rejected: the undo would need disk space and a new temp at the moment something has already failed, and the restored file would lose its mode, attributes and identity.
- **Copy every target before the commit.** Rejected: doubles the I/O for the rare failure; a link costs no data.
- **A no-replace rename (`renamex_np`, `renameat2`) for the restore.** Rejected: one call per platform outside `os.Root`, for what a link already gives.
- **Leave it.** What ADR-066 chose. Rejected now: §196 reaches the half-plan from the built binary with no seam.

## Component / Boundary Impact

Engine package owned here: `internal/apply` (the asides, the undo, the cleanup; `tree.go` gains a link and a file-only removal through the root). `internal/writer/writer.go` records only files whose path holds what mrw wrote. `internal/mcp/tools.go` changes one sentence. `internal/plan`, `internal/read`, `internal/seen`, `internal/check` and `internal/state` are unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a commit failure after a content rename | `NOTHING WRITTEN`, exit 2, files restored; `PARTIALLY APPLIED` with `UNDO INCOMPLETE` only for a file the undo could not restore or no aside covered | T1 | every caller |
| `mrw stats` | such a failure counts `refused_apply` when nothing stays written | T1 | its readers |
| the ledger | records only files whose path holds what mrw wrote | T1 | the next write |
| the tree during a commit | one `.mrw-aside-*` hard link per existing content target, removed after | T1 | anything watching the tree |
| `README.md` failure table, `mrw_write`'s `left_behind` sentence | Decision 8 | T1 | callers recovering a run |
| `scripts/contract.sh` §196 | ADR-102's row: the trigger now writes nothing, counts `refused_apply`, and `a.txt` is still licensed by the read before it | T1 | the contract gate |
| `scripts/contract.sh` §141 | the fold at commit: nothing written, neither name present | T1 | the contract gate |
| `scripts/contract.sh` §119 | its pointer to the Go test that drives a partial commit, renamed | T1 | readers of the contract |
| `scripts/contract.sh` §360 | a distinct promise: an edit and a create in a new nested directory, then an unlink refused at commit: the edit restored with its original inode, the create and its new directories gone, no `.mrw-*` left, exit 2 `NOTHING WRITTEN`; paired with the directory writable, which applies all three | T1 | the contract gate |

No receipt key is added or renamed (ADR-111): `files[].written` keeps its meaning (the path no longer holds its original), and `left_behind` and `error` carry the rest.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** a plan applies whole or not at all through the commit loop too, for every file an aside covers.
- **Negative — the cost:** one link and one unlink per existing content file on every write that lands: about 0.22 ms per file in the table above (one file: +0.22 ms on a write whose whole CLI run took a median 11.0 ms over 30 runs, same machine, same day). That is the publish step only; T1 measures the whole CLI run before and after. Directories gain a short-lived entry per file, which a file watcher sees. A killed run can leave a `.mrw-aside-*` that is the only copy of a file's previous version.
- **Measured on macOS (T1 S4, 2026-11-20; Linux and Windows not yet):** ADR-105 T3's 500-file, 5,000-hunk write took a median 259.7 ms on main and 378.8 ms with the asides over 21 alternating runs each (+119 ms, +46%, 0.24 ms per file, the table's 0.22 ms); a one-file write 14.2 ms and 16.95 ms (+2.8 ms, +19%), against the 11.0 ms median CLI run above. APFS, 10 cores, load average under 8 at the start and 7.4 to 9.4 at the end. The cost is the link and the removal per existing content file, as estimated.
- **The aside's name lookup (review of #760, 2026-11-21; macOS only):** finding the spelling a directory lists for a target by listing the directory and stat'ing its entries cost 405 ms for a one-file write in a directory of 100,000 entries (main: 14.2 ms). It is now one open and one F_GETPATH where a file has one name (16.7 ms), nothing on a volume that keeps case apart, and a listing of names without stat of the siblings otherwise. Windows lists the directory and is not measured.
- **Final table, macOS (T1 S4, 2026-11-21; supersedes the figures of the two lookup bullets above it and the reviewer's 18 to 21 ms and 189 to 195 ms):** one write with `--no-check`, the staged state fresh each run, main `10b83cc2` against the branch `6584d9e1`, both built with `go build -trimpath` from `git archive` trees, 9 runs of each in rotating order, plus a second copy of main as a control, the owner's Mac (APFS, 10 cores, 1-minute load 3.2 to 6.2 during the runs). Medians (min, max): (a) an ordinary directory of 20 entries: main 18.6 ms (17.3, 28.1), branch 17.7 (17.6, 24.4), control 17.8 (17.5, 28.3); (b) a directory of 100,000 entries, one name: main 21.3 (19.8, 22.1), branch 21.1 (19.4, 26.4), control 20.5 (19.4, 26.1); (c) the same with a second hard link: main 21.2 (20.3, 24.5), branch 21.0 (19.3, 22.9), control 20.7 (20.0, 23.2). The branch is within the control's noise in all three: the asides, the identity checks and the name lookup add nothing measurable to a one-file write, and the lookup lists no directory in any of them. The 500-file write of the first row below is where the aside's link and removal show (+119 ms, 0.24 ms per file), and it is unchanged. Linux and Windows are not measured.
- **Neutral:** where no link can be made the behaviour is today's, and the receipt says so when it matters.

## Out of Scope

- Locking other processes out of the paths a commit or its undo touches (permanent: boundary: ADR-002 and ADR-075 lock only mrw writers; Decision 5 states what another process can lose)
- Recovering a run killed mid-commit (permanent: boundary: a killed process runs no undo; the README process-death row says what is left and how to recover)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Windows refuses the link, or the aside's removal, on a target held open | Med | Med | Decision 2 commits without the aside; a removal that fails is `left_behind`; T1's Go tests run in the Windows CI job |
| The link cost on another filesystem or OS is larger | Med | Low | T1 measures the whole CLI run, old and new builds, on Linux, Windows and the owner's Mac |
| Tests asserting today's `PARTIALLY APPLIED` turn red | High | Low | §119, §141, §196 and the Go tests are named in T1 |

## Rollback

Revert T1. An aside left by a killed run of the new build is an ordinary file the revert does not touch.

## History

- 2026-11-20, Codex at xhigh on `2ded135b`, AMEND, four points, all applied after checking each against the code: the fallback is one rule (no aside, today's behaviour, named in the receipt) and ADR-132's classes are untouched, since Windows `Root.Link` asks for `FILE_WRITE_ATTRIBUTES` where ADR-125's probe asks for `DELETE`; the undo removes only a regular file matching the staged temp's identity, size and time, through a file-only removal, and Decision 5 states what other processes can lose; the receipt and ledger are built from the final tree, and ADR-106, ADR-071 T2 and ADR-102 §196 are amended; T1 proves the rollback (a nested create, the file ID loaded early, undo order, an aside's bytes, a cleanup failure, an edit with a rename). The Context's account of the path-op order is corrected.
- 2026-11-20, T1's first-red receipt (`adr-verify`, exit 1) was taken with the code already in place and failed on the missing contract row and README sentences, not on the Go tests. The tests had been seen failing first in the working tree: the nine `TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed` subtests were red against the commit loop as it was, before the asides existed.

## Follow-ups

- [ ] None beyond T1.
