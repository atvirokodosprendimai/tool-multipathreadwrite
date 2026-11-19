# ADR-204: mrw can undo its own last write

**Status:** Withdrawn
**Withdrawn by:** Zy, 2026-11-19: *"stop undo"*, after the third text below was rejected too and the drafter recommended stopping. Nothing was built. `mrw undo` does not exist; the way back from an mrw write is git, or a second plan. This record is kept for what the three reviews established about what a safe undo would need, so the next attempt does not start from the first draft.
**Owner's decisions so far (Zy, 2026-11-19):** the scope, on being asked whether an undo should work "only in parent repo" or on "whatever the agent touched": *"ok, re proposal on mrw - agreed"*, then *"good, go on with mrw undo"* (only what mrw itself wrote; one write call; per checkout; no dependence on git; refused unless every file still holds what that write left; before-images kept in mrw's state, outside the tree; checks, steps and other tools not undone), and that it goes to a record; Zy had been told of E6's STOP first. The depth, after the first design review: *"option 1, keep only the last write"*. And the width, after the second rejection, offered A (a narrow first version: copies only for writes that edit or create regular files), B (the full design) or C (stop): *"A"*. Everything else here (where the copy is kept and when it is refused, the size limit, links and metadata, the ledger after an undo, the exit codes, the receipt, the reading) is the drafter's and waits for Zy with this record.
**Date:** 2026-11-19
**Rewritten twice:** 2026-11-19, before any code. A first draft kept sixteen writes (design review, Codex gpt-6.2-astra at xhigh on `1b889800`: REJECT; adr-challenger: REJECT). A second kept one write of any kind (Codex gpt-6.2-sol at xhigh on `6ef83207`: REJECT, fourteen findings). This third text is the narrow version the owner chose; History, below, lists what the two earlier ones got wrong and where each finding went.
**Third challenge, 2026-11-19, of this narrow text, before any code** (Codex gpt-6.2-sol at xhigh, read-only on `f7b70e7c`): **REJECT**, ten findings. The Decision below is not ready to build. Even for edits and creates of regular files on unix, a safe undo needs things this text does not yet have and the codebase does not yet offer:
- *A restore that cannot be redirected.* `os.Root` follows a symlink in a parent that stays inside the root, and its `Remove` removes an empty directory; parents have to be walked one handle at a time, the leaf rechecked after staging through the same parent handle, and a file-only unlink used.
- *A hook with one place.* Edit plans, path-only plans and mixed plans reach their first rename at different points in apply; the barrier has to sit at the first commit rename, with the cleanup of staged temporaries on its failure stated.
- *A ledger replacement that is one transaction.* `Drop` then `Record` release the ledger lock between them, a reader can slip in and keep its shown lines, aliases are missed, and a quote-only write's narrow licence (ADR-190) would be widened. That is a change to `internal/seen`, which this text said it would not touch.
- *Pending acknowledgements invalidated for the whole checkout before the first restored sha is visible*, not only for the invocation root and not at the end. That is a change to `internal/mcp`'s pending store.
- *A store whose privacy is shown*, which permission bits alone do not show on a filesystem with ACLs, macOS included, with every operation bound to the verified directory handle.
- *Named states for an interrupted undo* (eligible, consumed, publish-temporary), what a failed rearm reports, and what a later write does with a leftover.
- *Evidence from the handle apply validated* (link count, special bits, exact bytes), rechecked at commit, for every route into apply.
- *A wire contract for the receipt*, fixtures that reach the branches they name, and a reading whose numerator and denominator count the same population.

**Where this leaves the record: stopped.** Three texts, three rejections, each for hazards that are real. The narrowing did not make the work small: the hard parts are in the restore, the ledger and the pending acknowledgements, not in which operations are covered. Building it means owned changes in `internal/apply`, `internal/seen` and `internal/mcp` and a new store with its own protocol, against a measurement that found the need rare. That is the owner's call to make with this in front of them; nothing is built.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-005, ADR-007, ADR-034, ADR-066, ADR-071, ADR-075, ADR-076, ADR-077, ADR-111, ADR-125, ADR-129, ADR-132, ADR-134, ADR-143, ADR-161, ADR-190, ADR-203
**Supersedes:** none. It sets aside E6's STOP verdict (BACKLOG.md, "Pre-registered for an undo of mrw's own write") on the owner's word.
**Governs:** `internal/undo/**`, `internal/writer/writer.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go`, `internal/apply/apply.go`, `cmd/mrw/main.go`, `cmd/mrw/skill168.go`, `cmd/mrw/receipts111_test.go`, `cmd/mrw/undo204.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/agents.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/undo204_test.go::TestUndoPutsBackTheLastWriteOrRefusesBeforeAnythingMoves`
**Served-path change:** on unix, a landed write that only edits or creates regular files keeps what it replaced, for that one write, in mrw's state directory; a new subcommand, `mrw undo`, puts those files back when each is still as that write left it. Any other write, and every write on Windows, keeps nothing and says so.

## Context

**The gap.** Claude Code's `/rewind` restores files its own edit tools changed. It does not track a file only a shell command changed, and every mrw write is one (`docs/agents.md` §7). A session that wants an mrw write back uses `git checkout`, `git stash`, or a second plan typed from memory.

**The measurement, and that it said no.** E6 (2026-11-19, `docs/bench/effect-study/`): 9 near reverts, 22 path reverts and 95 tree reverts by git in 9,686 landed writes outside this repository, against a bar of 25 and 0.5%, which is 49 at that count (the reading at scanner commit `a5baadfd`). Verdict STOP. It could not see a revert made by writing the old lines back. The owner decided to build it anyway; this record does not claim the count supports it.

**What mrw knows, and only that.** For a write of its own: each directory entry it changed, what was there before and what it left. Nothing of what an editor, a formatter, `sed` or `git` did. An undo that reached for those would put a file into a state the caller no longer has.

**Audit of the class**: *the ways an mrw call changes the tree.* At `1b889800`, `git grep -nE 'writer\.Prepare\(|apply\.Apply\(' -- cmd internal ':!*_test.go'` gives five lines: `cmd/mrw/main.go` (`mrw write`: a native plan, `--create`, `--grep --sub`, `--ast-grep --rewrite`, the two foreign formats), `cmd/mrw/skill168.go` (`mrw skill --dir`, whose root is the directory it installs into), `internal/mcp/tools.go` (`mrw_write`), `internal/writer/writer.go` (`applyCounted`, which all three reach and which takes the checkout's write lock, applies, and releases it, ADR-075), and `cmd/curve`, a benchmark that works on a scratch copy and is not mrw. Each of the first three keeps a copy and can be undone, `mrw skill --dir` included: `mrw --root DIR undo` takes an installed skill back. `mrw undo` is itself a new path that changes the tree; Decision 9 says which of a landing's duties it carries. `mrw check`, `--then` and `--then-sh` steps and `mrw run` run commands the project chose; mrw does not know what they change. `mrw read`, `mrw hook read`, `mrw seen`, `mrw iter`, the MCP acknowledgement store, `mrw-refused` and the one-time copy of a legacy `./.mrw` write only mrw's own state, which is inside the tree only when the caller put it there (ADR-077).

## Existing Primitives Audit

- `applyCounted` (`internal/writer/writer.go`) is the critical section of a write: it takes the checkout's write lock, calls `apply.Apply` once, and releases it. `Land` does not hold the lock.
- `apply.Apply` returns before staging on a failed hunk or a dry run, and can still refuse after staging (name probes, identity checks, replaceability). Only inside it is the point known where the last refusal is past and the first rename has not happened.
- Apply's validation read has each regular file's bytes; today it returns split lines and drops the open file's metadata. Apply reports `Target` when a write went through an in-root link, and which paths it unlinked or renamed.
- Apply commits by staged rename, file by file; a step failing after others succeeded is reported `PARTIALLY APPLIED` (ADR-066). Content already replaced is not rolled back.
- `rooted.Resolve` judges a path string; apply mutates through an open `os.Root`. A check by name followed by a reopen by name can act on a different entry.
- `seen.Record` merges with a record of the same sha and can keep its shown lines; `seen.Drop` removes a record.
- `mrw seen --prune` counts a state directory's files and not its subdirectories.

## Decision

1. **`mrw undo` undoes the last landed mrw write of the checkout, when that write is one this version keeps a copy of.** There is one kept write: no walk back and no redo.
2. **What is kept: only the simplest writes.** A copy is kept when the landed write did nothing but replace the content of regular files and create regular files, each reached by its own name: no `unlink`, no `rename`, no path that led through a symbolic link, no file with more than one hard link, and none whose link count could not be read. For every other landed write nothing is kept, and the caller is told which rule ruled it out. On Windows nothing is kept in this version: a private store cannot be shown there by permission bits, and its attributes and held files need rules of their own (BACKLOG).
3. **A write discards the previous copy at the last moment it can be refused.** Apply gains a hook, called once per plan after its last pre-commit refusal and before its first rename. There the kept entry, if any, is removed; if it cannot be removed the write is refused, exit 2, nothing written, with the entry's path, the OS error and `mrw --root DIR undo --forget` as the remedy. A refused plan and a dry run never reach the hook and remove nothing. After apply returns, still inside the write lock, the new entry is published by rename only if every file landed and Decision 2 allows it. So: interrupted before the hook, the previous copy stays; after the hook and before publication, including a partial landing, there is no copy; after publication, only the new one. This holds for a process that is killed; it is not a guarantee across power loss, and the instructions say so.
4. **What "the last write" can mean.** The last write made in this checkout by an mrw that keeps copies, sharing this state home. An older mrw, or one run with another `XDG_STATE_HOME`, removes nothing: after it has written, the kept copy is of an earlier write, and `mrw undo` will still put that one back if its own files are untouched. The entry records when it was made and `mrw undo` prints it; the instructions name the limit.
5. **The entry.** One flat file in the directory the ledger is in (`seen.Dir`), so a write from a subdirectory keeps it at the checkout's top. It holds: a format version; the checkout and the invocation root; the time; and for each file its path relative to that root, whether it existed before, its bytes and permission bits before (the nine rwx bits; a file with setuid, setgid or sticky set rules the write out under Decision 2), and its sha and permission bits after. Bytes come from the handle apply validated, bounded: over 64 MiB in total, nothing is kept and no copy of that size is made. It is written 0600 through a temporary file in the same directory and renamed. It is kept only if that directory, opened without following links, is owned by the user with no group or other access, and is not inside the checkout; otherwise nothing is kept. Abandoned temporaries of its own name are removed by the next publish and by `--forget`.
6. **`mrw undo`, in order, under the write lock from first to last.** (a) Open the entry without following links and verify it as in Decision 5; none: exit 1, `nothing to undo`. (b) Read it once from that handle, bounded, and validate it as input: version; this checkout; the recorded root is the root this undo was given (else exit 1 naming the root to run it from); each path clean, relative, unique, with no `.git` component, not mrw's own state, no component that is a link; modes from the nine bits; every blob of its recorded length and digest. A damaged or foreign entry: exit 2, left in place. (c) Open the root as a handle and, through it, check every file: an edited file is a regular file with one link, the recorded sha and permission bits; a created file the same. Any that differs: exit 1, every difference named, nothing changed, the entry kept. (d) Move the entry out of the eligible name. (e) For each file, through the root handle: check it again, then stage and rename the earlier bytes with the earlier bits, or remove a created file. (f) Record the ledger and remove the entry.
7. **Its three outcomes.** *Refused before anything moves*, exit 1: (a) to (c). *Restored*, exit 0. *Failed during the restore*, exit 2: a failure at (d) changes nothing and leaves the entry eligible; a failure at (e) before any file moved puts the entry back; after a file moved it reports `PARTIALLY APPLIED`, which files were restored and which were not, and the entry is gone, since it no longer describes the tree. A process killed during (e) leaves no eligible entry.
8. **What is not put back, and said so.** Directories a create made are left. Ownership, ACLs and extended attributes are those of a new file. A file another process changes between its last check and its rename is not caught.
9. **The ledger and the counters after an undo.** For each restored file the record is replaced, not merged: dropped, then recorded as a file mrw wrote at the restored sha, with nothing shown, so ADR-203 judges the next edit as after any write. A removed created file is dropped. The checkout's pending MCP acknowledgements are emptied, as `mrw seen --forget` empties them, since an earlier sha is current again. An undo that changed the tree counts once as another write for the drift a running check reports (ADR-127), restores included in `drift_writers`' description; it is not a landed write in `mrw stats`. A ledger failure after the tree changed is exit 2 and says the files are restored and the next edit needs a read.
10. **The command and the receipts.** `mrw undo`; `mrw undo --dry-run` (steps (a) to (c), then what it would restore; exit 0 or 1; nothing changed); `mrw undo --forget` (removes the entry and its temporaries under the write lock, says what it removed, exit 0 when there was nothing; it does not touch the read ledger, as `mrw seen --forget` does not touch this). `--json`: one document of receipt type `undo` for every outcome reached after the flags are parsed, its keys listed in `docs/receipts.txt` and its type registered with the receipt test; a flag the parser rejects is plain, as for every command but `run`. `--check` and `--then` are not flags of `undo`; it starts no command and runs at any `MRW_STEP_DEPTH`. Not over MCP. On the write side: `mrw: undo: not kept: REASON` on standard error for every landed write that keeps nothing, a partial landing included, whatever the receipt style; and an optional string key `undo_not_kept` on the `write` and `mcp_write` receipts, with `mrw skill --dir` printing the same line.
11. **What would make this decision wrong.** Nobody using it: BACKLOG.md ("Pre-registered for ADR-204") reads it thirty days after the release; with at least 1,000 landed writes that kept a copy, fewer than one successful restore per 1,000 sends it back to Zy with the choice of removing it. And the narrow width leaving most writes without a copy: the same reading reports the share of landed writes that kept none, by reason.

## Alternatives Considered

- **Everything a plan can do** (links, renames, unlinks, hard-linked names, Windows). The second draft. Rejected twice at the record stage and set aside by the owner: each needs evidence mrw does not record today.
- **Several kept writes.** The first draft. Set aside by the owner.
- **Clear the previous copy before or after `apply.Apply`.** Rejected: before loses a good copy on a refused plan; after leaves a stale one if the process dies mid-commit.
- **Check paths by name.** Rejected: a name checked and then reopened can be another entry; the restore works through one open root.
- **Undo whatever changed since a point in time; lean on git; a reverse plan.** Rejected: mrw does not know what other tools wrote; git needs a repository and touches other work; a plan does not carry modes or exact bytes.
- **Do not build it.** What E6's verdict says. The owner chose otherwise.

## Component / Boundary Impact

A new package, `internal/undo` (the entry, its validation, the restore). `internal/writer/writer.go` passes apply the hook and publishes inside `applyCounted`'s critical section. `internal/apply` is the one engine package changed, owned here: the hook, and the hand-over of each file's validated bytes, permission bits and link count. `internal/seen`, `internal/state`, `internal/plan`, `internal/read` and `internal/check` are not changed. The trust boundary moves twice: a copy of content the caller wrote over exists outside the tree for one write; and a file in mrw's state can cause writes into the tree, which is why Decision 6 reads it as input, once, through a verified handle.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw undo`, `--dry-run`, `--forget`, `--json` | new subcommand, exit codes as Decision 7 | T1 | every caller with a shell |
| a write | a stderr line and the key `undo_not_kept` when nothing was kept; refused, exit 2, when a stale entry cannot be removed | T1 | the caller |
| `apply.Options` | a pre-commit hook; the kept material in the result | T1 | `internal/writer` |
| the `undo` receipt | new receipt type | T1 | `--json` callers |
| one file in the checkout's state | new persistent state | T1 | `mrw undo`, `mrw seen --prune` |
| `mrw instructions`, `AGENTS.md`, `README.md`, `docs/agents.md` §7 | what undo covers, what it does not, and its limits | T1 | every caller |
| `scripts/contract.sh` §350 | keep, undo, the refusals, the not-kept reasons | T1 | the contract gate |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1, one pull request.

## Consequences

- **Positive:** the commonest write, an edit or a create of ordinary files, is one command to take back, exactly, without git.
- **Negative:** such a write copies what it replaced; up to 64 MiB of content the caller overwrote persists outside the tree until the next write of that checkout or `--forget`, with no limit of age; one new way a write can be refused; writes that unlink, rename or touch a link, and every write on Windows, have no undo; a new subcommand and a new piece of state to keep.
- **Neutral:** no existing exit code or receipt key changes meaning; one key is added to the write receipts.

## Out of Scope

- Undo of a write that unlinks, renames, goes through a link or changes a hard-linked name (deferred: docs/adr/BACKLOG.md)
- Undo on Windows (deferred: docs/adr/BACKLOG.md)
- More than one kept write, and a redo (deferred: docs/adr/BACKLOG.md)
- An undo over MCP (deferred: docs/adr/BACKLOG.md)
- Undoing what a check, a step or another tool wrote (permanent: boundary: mrw does not know what they changed)
- Undoing a part the caller picks of a write (permanent: boundary: a plan applies whole or not at all, ADR-001, and its undo is asked for whole)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The measurement said this is rarely needed | High | Low | the owner decided; Decision 11's reading can remove it |
| Most real writes turn out to keep nothing | Med | Med | the reading reports the share by reason; the reasons are printed |
| The copy holds a secret the caller deleted on purpose | Med | Med | one write only; a verified private location or nothing; `mrw undo --forget`; said in the instructions |
| A restore over a file another tool changed | Low | High | Decision 6: checked before the first move and at each move, through one root handle; mutants on every arm |
| A tampered entry writes where it should not | Low | High | Decision 6 (b); a test damages each field |
| The copy is of an earlier write than the caller thinks (an older mrw, another state home, another session in a shared checkout) | Med | Med | Decision 4: the entry's time and files are printed by `mrw undo` and `--dry-run`; the instructions say to look first |
| A write is refused because a stale copy cannot be removed | Low | Med | the refusal names the path, the cause and the remedy; a test makes it happen |

## Rollback

Before a release, revert T1. After one, `mrw undo` is a public command: it would be refused with a sentence, writes would stop keeping a copy, and `mrw undo --forget` would keep working, since `--prune` never touches a live checkout.

## History

*First draft (`1b889800`), sixteen kept writes.* Rejected by the design review and the adr-challenger: its restore claimed an atomicity the commit does not have; a write whose copy failed could be skipped over; the entry was trusted when read back; per-operation checks refused valid plans; bytes and mode did not capture links; the lock boundary and capture point were wrong; an undo under a sub-root could reach a sibling; privacy claims outran the placement; a byte match could hide a permission change; restoring an unlinked file could widen a licence; the header credited the owner with choices not made. Where each went: one kept write (owner); Decisions 3, 6 and 7 for ordering, input validation and outcomes; Decision 2 rules out links, renames and unlinks; Decision 5 for placement and privacy; Decision 9 for the ledger.

*Second draft (`6ef83207`), one kept write of any kind.* Rejected again, fourteen findings: no confinement of the restore (now Decision 6, one root handle); an interrupted undo could run again (Decision 6 (d), Decision 7); the clear had no place `applyCounted` can see (Decision 3's hook); "the last write" exceeded what can be proved (Decision 4 says the limit); the privacy test proved nothing (Decision 5, and Windows left out); capture from a read that drops metadata (Decision 5, the hand-over is in apply); hard links, directories and permission bits without evidence (Decision 2 rules them out or Decision 5 records them; created directories are left); `seen.Record` merges (Decision 9 replaces); exit causes, receipts and the reading under-specified (Decisions 7, 10, 11 and the BACKLOG entry); and the claim that both first reviews were built in was too strong, which this section replaces with what went where.

None of either design is built.

## Follow-ups

- [ ] The reading of BACKLOG.md, "Pre-registered for ADR-204".
