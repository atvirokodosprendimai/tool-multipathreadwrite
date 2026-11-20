# ADR-209: A content commit that fails partway undoes the files it already renamed

**Status:** Proposed
**Drafted for:** Zy, 2026-11-20, from an independent critique of the same day (finding S1). Zy's instruction on the critique's findings, as relayed to the drafter by the session that commissioned these records, was *"address all, no choosing"*, so this record carries one decision. Challenged before acceptance (Codex at xhigh, 2026-11-20, on `2ded135b`: AMEND); every amendment is applied below or answered in History. It is not accepted, and nothing here merges before Zy reads it.
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

ADR-066 Decisions 2 and 3 (path ops undone newest first; the receipt reports what reached disk) stand. ADR-132's exit classes are unchanged.
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
- ADR-106 already loads a file's ID when it is stat'ed, because on Windows `os.SameFile` otherwise loads it lazily by path and answers false when that fails (`apply.go:2647-2652`).

## Decision

1. **An aside for every existing content target, before the first rename.** After the checks that already run before the first rename (`apply.go:1167-1196`), each content target that exists is hard-linked under a fresh `.mrw-aside-*` name in its own directory, through the root. A name already taken is another name.
2. **A link that cannot be made does not stop the write.** That file commits as today, with no aside, whatever the reason the link was refused; ADR-132's exit classes are unchanged. If the commit later fails, that file cannot be put back: the receipt reports it `written`, `PARTIALLY APPLIED`, and `UNDO INCOMPLETE` names it with the link's error. A write that lands reports nothing about it.
3. **What the undo checks before it removes anything.** Before each content rename, mrw stats the staged temp, with its file ID loaded then, and keeps that record. The undo removes a path only when it is a regular file that matches that record by identity, size and modification time (`changedSince`'s comparison), so a file another process put there or edited in place since the rename is left alone. It removes it with a file-only removal (never a directory, which `tree.remove` would also take, `tree.go:145-152`), at the resolved path the forward rename used (`staged[i].target`, ADR-106).
4. **The undo, newest first, after the path ops' own undo.** For each content file renamed in, newest first: check it (Decision 3); remove it; for a file that existed, link its aside back to the path and remove the aside. A link never replaces, so a path something else holds by then keeps that thing, and the aside stays as a recovery file. A created file is removed and the directories staging made for it are removed as today. Whatever could not be put back is named under `UNDO INCOMPLETE` and in `left_behind`.
5. **What is promised about other processes, and what is not.** mrw's write lock orders mrw writers (ADR-075); nothing locks out other processes. During an undo: a change another process makes to a path between the check and the removal is lost; between the removal and the link back, the path holds nothing; a process holding the old file open, or another name for it, can change the aside's bytes, and what is restored is that file as it is then — what the path would hold had mrw not run. The forward commit has the same check-then-act window before each rename (ADR-106).
6. **The receipt and the ledger describe the final tree.** `ALREADY WRITTEN` and every error text are built after both undos, not before. A file's `written` is true when its path no longer holds its original: restored files are not written and their records drop out; a file whose undo was refused (another process's file there, or the restore link failed) stays `written`. The ledger records a file only when its path holds the bytes mrw wrote, so a file left absent or held by another process is not recorded (an internal mark on the file's result, not a receipt key). Hunk verdicts follow ADR-066 Decision 3 and ADR-114 (an edit and a rename of one file keep their own verdicts). `mrw stats` counts `refused_apply` when nothing is `written`, `partially_applied` otherwise (ADR-102).
7. **After a commit that landed, every aside is removed**; one that cannot be is in `left_behind`.
8. **The words.** `README.md`'s failure table (the commit, undo and process-death rows) and the MCP `left_behind` sentence (`internal/mcp/tools.go:1685`) say: a commit failure puts the content files back; `PARTIALLY APPLIED` names what could not be; a `.mrw-aside-*` beside a content file is its previous version, and after a killed undo the path can be missing with its content only in the aside.

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

## Follow-ups

- [ ] None beyond T1.
