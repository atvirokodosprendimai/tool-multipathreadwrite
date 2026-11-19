# ADR-204: mrw can undo its own last write

**Status:** Proposed
**Owner's decisions so far (Zy, 2026-11-19):** the scope, on being asked whether an undo should work "only in parent repo" or on "whatever the agent touched": *"ok, re proposal on mrw - agreed"*, then *"good, go on with mrw undo"* (only what mrw itself wrote; one write call; per checkout; no dependence on git; refused unless every file still holds what that write left; before-images kept in mrw's state, outside the tree; checks, steps and other tools not undone), and that it goes to a record; Zy had been told of E6's STOP first. And the depth, after the design review below: *"option 1, keep only the last write"*. Everything else here (where the copy is kept and when it is refused, the size limit, links and metadata, the ledger after an undo, the exit codes, the receipt, the reading) is the drafter's and waits for Zy with this record.
**Date:** 2026-11-19
**Rewritten:** 2026-11-19, before any code, after a design review (Codex gpt-6.2-astra at xhigh, read-only on `1b889800`: REJECT of a first draft that kept sixteen writes). Its findings were checked against the code and are built into the Decision; History, below, says what the first draft got wrong.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-005, ADR-007, ADR-034, ADR-066, ADR-071, ADR-075, ADR-076, ADR-077, ADR-111, ADR-125, ADR-129, ADR-132, ADR-134, ADR-143, ADR-161, ADR-190, ADR-203
**Supersedes:** none. It sets aside E6's STOP verdict (BACKLOG.md, "Pre-registered for an undo of mrw's own write") on the owner's word.
**Governs:** `internal/undo/**`, `internal/writer/writer.go`, `internal/mcp/tools.go`, `internal/apply/apply.go`, `internal/apply/pathop.go`, `cmd/mrw/main.go`, `cmd/mrw/undo204.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/agents.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/undo204_test.go::TestUndoPutsBackTheLastWriteOrRefusesBeforeAnythingMoves`
**Served-path change:** a new subcommand, `mrw undo`, puts back what the last landed mrw write of this checkout changed, when every directory entry it changed is still as that write left it. A landed write keeps what it replaced, for that one write only, in mrw's state directory.

## Context

**The gap.** Claude Code's `/rewind` restores files its own edit tools changed. It does not track a file only a shell command changed, and every mrw write is one (`docs/agents.md` §7). A session that wants an mrw write back uses `git checkout`, `git stash`, or a second plan typed from memory.

**The measurement, and that it said no.** E6 (2026-11-19, `docs/bench/effect-study/`): 9 near reverts, 22 path reverts and 95 tree reverts by git in 9,686 landed writes outside this repository, against a bar of 25 and 0.5%, which is 49 at that count (the reading at scanner commit `a5baadfd`). Verdict STOP. It could not see a revert made by writing the old lines back. The owner decided to build it anyway; this record does not claim the count supports it.

**What mrw knows, and only that.** For a write of its own: each directory entry it changed, what was there before and what it left. Nothing of what an editor, a formatter, `sed` or `git` did. An undo that reached for those would put a file into a state the caller no longer has.

**Audit of the class**: *the ways an mrw call changes the tree.* At `1b889800`, `git grep -nE 'writer\.Prepare\(|apply\.Apply\(' -- cmd internal ':!*_test.go'` gives five lines: `cmd/mrw/main.go` (`mrw write`: a native plan, `--create`, `--grep --sub`, `--ast-grep --rewrite`, the two foreign formats), `cmd/mrw/skill168.go` (`mrw skill --dir`, whose root is the directory it installs into), `internal/mcp/tools.go` (`mrw_write`), `internal/writer/writer.go` (`applyCounted`, which all three reach and which takes the checkout's write lock, applies, and releases it, ADR-075), and `cmd/curve`, a benchmark that works on a scratch copy and is not mrw. Each of the first three keeps a copy and can be undone, `mrw skill --dir` included: `mrw --root DIR undo` takes an installed skill back. `mrw undo` is itself a new path that changes the tree; Decision 9 says which of a landing's duties it carries. `mrw check`, `--then` and `--then-sh` steps and `mrw run` run commands the project chose; mrw does not know what they change. `mrw read`, `mrw hook read`, `mrw seen`, `mrw iter`, the MCP acknowledgement store, `mrw-refused` and the one-time copy of a legacy `./.mrw` write only mrw's own state, which is inside the tree only when the caller put it there (ADR-077).

## Existing Primitives Audit

- `applyCounted` is the critical section of a write. `Land` calls it and does not hold the lock itself.
- Apply's validation read (`readLines`, `internal/apply/apply.go`) has each regular file's exact bytes and, from the same open file, its metadata. No second read is needed to keep a copy, and none would be guaranteed to see the same bytes.
- Apply commits by staged rename, file by file. When a step fails after others succeeded it reports what reached disk (`PARTIALLY APPLIED`, or `NOTHING WRITTEN` when its own undo put path operations back, ADR-066). Content already replaced is not rolled back. It refuses a target held without delete sharing before any rename (ADR-125, ADR-132).
- A path is judged by `rooted.Resolve` (ADR-007, ADR-071): inside the root, no `.git` component (ADR-143), not mrw's own state (ADR-077, ADR-134), no name Windows will not keep.
- The ledger and the lock are the checkout's (ADR-161 Decision 14); a path is still judged against the root the call was given.
- `mrw-refused` and `mrw-runs` live beside the state directory because the caller reads or edits them. Nothing the caller should open lives inside it.

## Decision

1. **`mrw undo` undoes the last landed mrw write of the checkout, and nothing else.** One write call, the directory entries in its receipt. There is one kept write, so there is no walk back and no redo.
2. **A write discards the previous copy before it touches the tree.** Inside the write lock: the kept entry, if any, is removed first; then the plan is applied; then, only when every file landed, the new entry is published complete, by rename. If the old entry cannot be removed, the write is refused, exit 2, nothing written, saying that the undo copy could not be cleared and how to clear it (`mrw undo --forget`); this is the one way the store can stop a write, and it is what makes "the kept write is the last write" true. If the new entry cannot be stored (a full disk, an unwritable or unverified store, the size limit), the write still lands, nothing is kept, and the caller is told why: a line on standard error, `mrw: undo: not kept: REASON`, printed under `--quiet` and `--terse` too, and the key `undo_not_kept` in the `--json` and `mrw_write` receipts, present only then. A partial landing keeps nothing. A refused plan and a dry run remove nothing and keep nothing.
3. **What is kept.** For each directory entry the write changed, as it was before: its kind (regular file, symbolic link, absent), a regular file's bytes and permission bits, a link's literal text; and what the write left there: kind and sha, or absent. For a write through an in-root link, the entry is the file the link led to. The directories the write created. The checkout and the invocation root the paths are relative to. Bytes are taken from apply's own validation read. Before-images over 64 MiB in total keep nothing, without a second read or a copy of that size; 64 MiB is this record's choice, the limit ADR-104 puts on a request line.
4. **Where.** One flat file in the checkout's state directory (the directory the ledger is in, so a write from a subdirectory keeps it at the checkout's top), holding the record and the bytes together. Flat, because `mrw seen --prune` counts a state directory's files and not its subdirectories (ADR-034); inside the state directory and not beside it as `mrw-refused` and `mrw-runs` are (ADR-176, ADR-177), because those are for the caller to open and this is not. It is written only when that directory is one mrw created or verified as the user's alone (not group or world accessible on unix; on Windows inside the user's profile, where the inherited ACL is the user's) and not inside the checkout. Otherwise nothing is kept and the receipt says so. It stays until the next write of the checkout, `mrw undo --forget`, or `mrw seen --prune` of a checkout that is gone; nothing retires it by age.
5. **`mrw undo` has three outcomes, and says which.**
   - *Refused before anything moves*, exit 1: some directory entry is not as the write left it. Every such entry is named with what was expected and what was found. The entry is kept, so the caller can repair and retry. Also exit 1: there is nothing to undo.
   - *Restored*, exit 0: every entry is as it was before the write.
   - *Failed during the restore*, exit 2: reported as a write's commit failure is, `PARTIALLY APPLIED` naming what was restored and what was not, or `NOTHING WRITTEN`. The entry is kept when nothing was restored and removed when anything was, since it no longer describes the tree.
   Each entry is checked once before the first move and again at its own move.
6. **What "as the write left it" means.** One expected state per directory entry, not per operation: a regular file with the recorded sha and the permission bits the write left; a link with the recorded text; or absent. A plan that unlinked `c` and renamed `b` onto `c` expects `c` present and `b` absent. On a case-folding filesystem the spelling in the parent's listing is compared, as ADR-129 does.
7. **What is restored, and what is not undone.** A regular file's bytes and permission bits; a link's text; an entry's absence; a directory the write created, when it is empty and still that directory. Not restored and said so: ownership, extended attributes, ACLs and Windows attributes other than read-only. A write that changed one name of a hard-linked file is refused an undo with a sentence saying why: rewriting one name cannot put the link back. A read-only target is refused as a write refuses it (ADR-076).
8. **The entry is untrusted when it is read back.** Before anything is staged: its format version, that it names this checkout, every path through `rooted.Resolve` under the root this undo was given, kinds and modes from the allowed set, lengths and digests of every blob. A path outside the invocation root refuses the whole undo, exit 1, naming the root to run it from; it never undoes part. A damaged or unknown entry is refused, exit 2, and left in place.
9. **The ledger and the counters after an undo.** An undo that changed the tree is counted as a write for the drift a running check reports (ADR-127); it is not a landed write in `mrw stats` and keeps no entry of its own. A restored regular file is recorded as a file mrw wrote, with nothing shown, so ADR-203 judges the next edit as after any write. A file that returns from an unlink, and a file the undone write had only a partial licence on (ADR-190), are recorded as not read: the next edit needs a read. A path that is gone again is dropped.
10. **The command.** `mrw undo`; `mrw undo --dry-run` (what it would restore and whether it would be refused, nothing changed); `mrw undo --forget` (removes the kept entry and any abandoned temporaries, says what it removed, exit 0 when there was nothing). `--json` prints one document, receipt type `undo`, its keys listed in `docs/receipts.txt`, for every outcome including a refusal. `--check` and `--then` are not flags of `undo`; it starts no command, so it runs at any `MRW_STEP_DEPTH`. Not over MCP: a write made by `mrw_write` is kept and undone from the CLI.
11. **What would make this decision wrong.** Nobody using it: BACKLOG.md ("Pre-registered for ADR-204") counts successful restores thirty days after the release; fewer than one per 1,000 landed writes, and it goes back to Zy with the choice of removing it.

## Alternatives Considered

- **Several kept writes.** The first draft. Rejected by the owner after the review: it needs an ordering protocol and barriers so that a write whose copy failed is never skipped, and keeps more overwritten content.
- **Undo whatever changed since a point in time.** Rejected: mrw does not know what other tools wrote.
- **Lean on git.** Rejected: it needs a repository, touches uncommitted work the write did not, and a file never committed has nothing to return to.
- **A reverse plan instead of bytes.** Rejected: a plan is line-oriented; a mode, a final newline, mixed line endings and a link are not all in it.
- **Never let the store stop a write.** Rejected for the one case in Decision 2: if a stale entry cannot be removed and the write lands anyway, the next `mrw undo` would revert an older write over a newer one.
- **Do not build it.** What E6's verdict says. The owner chose otherwise.

## Component / Boundary Impact

A new package, `internal/undo` (the entry, its validation, the restore). `internal/writer/writer.go` calls it inside `applyCounted`'s critical section. `internal/apply` hands over what its validation read holds and the kinds of the entries it changes; it is an engine package owned here for that. The trust boundary moves twice: copies of content the caller wrote over exist outside the tree for one write, and an entry read from mrw's state can cause writes into the tree, which is why Decision 8 treats it as input.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw undo`, `--dry-run`, `--forget`, `--json` | new subcommand, exit codes as Decision 5 | T2 | every caller with a shell |
| a write receipt | a line, and a `--json`/MCP key, when no undo copy was kept, with the reason | T1 | the caller |
| a write | refused, exit 2, when a stale undo copy cannot be removed | T1 | the caller |
| the `undo` receipt | new receipt type | T2 | `--json` callers |
| the kept entry in the checkout's state | new persistent state | T1 | `mrw undo`, `mrw seen --prune` |
| `mrw instructions`, `AGENTS.md`, `README.md`, `docs/agents.md` §7 | what undo covers and does not | T2 | every caller |
| `scripts/contract.sh` §350, §351 | the keeping; the undo and its refusals | T1, T2 | the contract gate |

## Inter-task Contracts

| Producer | Consumer | Contract | Verification |
|----------|----------|----------|--------------|
| T1 | T2 | the entry a landed write publishes: per directory entry its state before and after, the created directories, the checkout and root | T2's tests restore from entries T1's code wrote, never from hand-made fixtures, except the tampered ones Decision 8 needs |

## Implementation

`tasks/README.md`: T1 (a landed write keeps what it replaced), T2 (`mrw undo`), in one pull request: T1 alone would copy content with no command to forget it, and `deadcode` would flag what only T2 calls.

## Consequences

- **Positive:** the last mrw write is one command to take back, exactly as far as Decision 7 goes, without git.
- **Negative:** every landed write copies what it replaced; up to 64 MiB of content the caller overwrote persists outside the tree until the next write of that checkout or `--forget`, with no limit of age; one new way a write can be refused (a stale copy that cannot be removed); a new subcommand and a new store to keep.
- **Neutral:** no existing exit code or receipt key changes meaning; one key is added to the write receipts.

## Out of Scope

- More than one kept write, and a redo (deferred: docs/adr/BACKLOG.md)
- An undo over MCP (deferred: docs/adr/BACKLOG.md)
- Undoing what a check, a step or another tool wrote (permanent: boundary: mrw does not know what they changed)
- Undoing part of a write (permanent: boundary: a plan applies whole or not at all, ADR-001, and its undo is asked for whole)
- Restoring ownership, ACLs and extended attributes (permanent: boundary: a staged replacement is a new file, and mrw's writes do not carry them either)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The measurement said this is rarely needed | High | Low | the owner decided; Decision 11's reading can remove it |
| The copy holds a secret the caller deleted on purpose | Med | Med | one write only; a verified private location or nothing kept; `mrw undo --forget`; said in the instructions |
| A restore over a file another tool changed | Low | High | Decisions 5 and 6: checked before the first move and at each move; mutants on every arm |
| A tampered entry writes where it should not | Low | High | Decision 8; tests with hand-damaged entries for each field |
| In a checkout several sessions write to (ADR-075), the last write is another session's | Med | Med | the undo receipt and `--dry-run` name the files and when the write was made; the instructions say to look first in a shared checkout; the restore holds the write lock from the choice of entry to its removal |
| A write is refused because a stale copy cannot be removed | Low | Med | the refusal names `mrw undo --forget` and the path; a test makes it happen |
| An external process changes a file between the last check and its rename | Low | Med | not preventable without a different premise; said in the instructions as the limit of the guarantee |

## Rollback

Before a release, revert T2 then T1. After one, `mrw undo` is a public command: it would be refused with a sentence, writes would stop keeping a copy, and `mrw undo --forget` would keep working, since `--prune` never touches a live checkout.

## History

The first draft (`1b889800`) kept the newest sixteen writes and walked back through them. The design review rejected it: its restore claimed an atomicity the commit does not have; a write whose copy failed could be skipped over by an undo of an older one; the entry was trusted when read back; its per-operation checks refused valid plans; bytes and mode did not capture links; it named the wrong lock boundary and capture point; an undo under a sub-root could reach a sibling; its privacy claims outran its placement; a byte match could hide a permission change; and restoring an unlinked file could widen a licence. Its header credited the owner with choices the owner had not made. None of it is built.

## Follow-ups

- [ ] The reading of BACKLOG.md, "Pre-registered for ADR-204".
