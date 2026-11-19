# ADR-204: mrw can undo its own last write

**Status:** Proposed
**Owner's decision so far:** on 2026-11-19 Zy asked whether an undo should work "only in parent repo" or on "whatever the agent touched", was given the scope in Decision 1 to 3 below, and answered *"ok, re proposal on mrw - agreed"*, then *"good, go on with mrw undo"*. That agrees the scope and that it is built. The storage, limits, exit codes and receipt in Decisions 4 to 9 are the drafter's and wait for Zy with this record.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-005, ADR-066, ADR-075, ADR-077, ADR-111, ADR-143, ADR-161, ADR-203
**Supersedes:** none. It sets aside E6's STOP verdict (BACKLOG.md, "Pre-registered for an undo of mrw's own write") on the owner's word.
**Governs:** `internal/undo/**`, `internal/writer/flow.go`, `internal/apply/apply.go`, `cmd/mrw/main.go`, `cmd/mrw/undo204.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/agents.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/undo204_test.go::TestUndoRestoresTheLastWriteWholeOrNotAtAll`
**Served-path change:** a new subcommand, `mrw undo`, puts back what the last landed mrw write of this checkout changed, when every file it changed still holds the bytes that write left. Every landed write keeps what it replaced, in mrw's state directory, for the newest writes.

## Context

**The gap.** Claude Code's `/rewind` restores files its own edit tools changed. It does not track a file only a shell command changed, and every mrw write is a shell command (`docs/agents.md` §7). A session that wants an mrw write back does it by hand: `git checkout`, `git stash`, or a second plan typed from memory.

**The measurement, and that it said no.** E6 (2026-11-19, `docs/bench/effect-study/`) counted how often a session reverts with git a file mrw just wrote: 9 near reverts, 22 path reverts and 95 tree reverts in 9,686 landed writes outside this repository, against a bar of 25 and 0.5%. Verdict STOP. It could not see a revert made by writing the old lines back. The owner decided to build it anyway; this record does not claim the count supports it.

**What mrw knows, and only that.** mrw knows each file a write of its own changed, the bytes it validated against and the bytes it left. It knows nothing of what an editor, a formatter, `sed` or `git` did. An undo that reached for those would restore a file to a state the caller no longer has, which is the silent wrong edit this tool exists to prevent.

**Audit of the class**: *the ways an mrw call changes the tree.* `git grep -n 'writer.Land\|writer.Prepare' -- cmd internal` at `c7b9c917`: every write reaches the tree through `writer.Land` (`internal/writer/flow.go`): `mrw write` with a native plan, `--create`, `--grep --sub`, `--ast-grep --rewrite`, the two foreign formats, and `mrw_write` over MCP. `mrw skill --dir` writes through the same engine. `mrw check`, `--then` steps and `mrw run` run commands the project chose and can change anything; mrw does not know what. `mrw hook read`, `mrw read` and `mrw seen` change no file of the tree.

## Existing Primitives Audit

- `writer.Land` is the one place a landing is known, under the write lock (ADR-075), with apply's per-file result: path, `sha_before`, `sha_after`, created, removed, `renamed_to`, `target`.
- Apply stages each new file beside its target and commits by rename, with an undo of its own when a commit step fails (ADR-066). It reads every original it validates.
- mrw's state for a checkout lives outside the tree (ADR-004), keyed by the checkout's top (ADR-161 Decision 14); `mrw seen --prune` removes the state of a checkout that is gone. `mrw-refused` and `mrw-runs` are kept beside it with their own retention.
- A write to a path with a `.git` component is refused (ADR-143), so no before-image of one exists.

## Decision

1. **`mrw undo` undoes what mrw wrote, and nothing else.** The unit is one landed write call: the files in its receipt. `mrw undo` puts back the newest one not yet undone; run again, it walks back to the one before.
2. **The scope is the ledger's.** The checkout's top when there is one, otherwise the root mrw was run with. It does not use git and works in a directory that is not a repository.
3. **Whole or not at all, and only over the bytes mrw left.** Before anything is restored, every file of that write must be as the write left it: an edited or created file holds exactly `sha_after`; an unlinked path is still absent; a rename's destination holds `sha_after` and its source is still absent. If any differs, nothing is changed, exit 1, and each differing file is named with what was expected. The restore is committed as a write is (staged, renamed, ADR-066's reporting when a step fails).
4. **What a write keeps.** For each landed write, under the write lock: the bytes and mode of every file it changed or removed, as they were before, and the record of what it did. They are kept in the checkout's state directory, in `undo/`, mode 0700 with 0600 files, never in the tree, and are removed with the checkout's state by `mrw seen --prune`.
5. **Limits.** The newest 16 landed writes are kept. A write whose before-images total more than 64 MiB keeps none: its receipt says `undo: not kept (N bytes of before-images, over the 64 MiB limit)`, and it ends the walk back: an undo stops there and says why. A write that landed only in part (ADR-066's PARTIALLY APPLIED) keeps none and ends the walk the same way. A dry run keeps nothing.
6. **What it restores.** An edit: the earlier bytes and mode. A create: the file is removed, and directories that write made are removed when empty. An unlink: the file comes back. A rename: the destination goes and the source comes back, with the edits of the same plan undone. A write through an in-root link: the file the link led to.
7. **After an undo.** The restored files are recorded in the ledger as files mrw wrote, at the restored bytes, so ADR-203 judges the next edit as after any write. The undone write's entry is removed; there is no redo. No check and no step is run: `mrw check` is the next command if one is wanted.
8. **The command.** `mrw undo` (the newest), `mrw undo --dry-run` (what it would restore, nothing changed), `mrw undo --list` (the kept writes, newest first: when, how many files, the first paths), `mrw undo --forget` (empties the kept writes of this checkout). `--json` prints one document, receipt type `undo`, listed in `docs/receipts.txt`. Exit `0` restored (or listed, or forgotten); `1` refused because a file differs or there is nothing to undo, nothing changed; `2` usage or a filesystem failure, with ADR-066's words when files were touched.
9. **Not over MCP, and not configurable.** `mrw_write` writes are kept and can be undone from the CLI; no MCP tool is added. There is no switch to turn the keeping off.
10. **What would make this decision wrong.** Nobody using it: BACKLOG.md ("Pre-registered for ADR-204") counts `mrw undo` calls thirty days after the release; none, and it goes back to Zy with the choice of removing it. And the copies costing more than they give: the same reading reports the size of `undo/` per checkout.

## Alternatives Considered

- **Undo whatever changed since a point in time.** Rejected: mrw does not know what other tools wrote, and restoring over their work is a silent wrong edit.
- **Lean on git (`git stash`, `git checkout`).** Rejected: it needs a repository, loses uncommitted work the write did not touch, and a file never committed has nothing to return to.
- **Keep a reverse plan instead of bytes.** Rejected: a plan is line-oriented and a file's mode, its final newline and its line endings are not all in it; bytes are exact.
- **Do not build it.** What E6's verdict says. The owner chose otherwise.

## Component / Boundary Impact

A new package, `internal/undo` (the store, the verification, the restore). `internal/writer/flow.go` calls it from `Land`. `internal/apply` hands over the bytes it validated against; it is an engine package owned here for that change. The trust boundary moves in one way: copies of file contents the caller wrote over now exist outside the tree, in a directory only the user can read, for at most 16 writes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw undo` and its flags | new subcommand, exit codes as Decision 8 | T2 | every caller with a shell |
| a write receipt | a line when no undo was kept | T1 | the caller |
| the `undo` receipt | new receipt type in `docs/receipts.txt` | T2 | `--json` callers |
| `<state>/undo/` | new persistent state per checkout | T1 | `mrw undo`, `mrw seen --prune` |
| `mrw instructions`, `AGENTS.md`, `README.md`, `docs/agents.md` §7 | what undo covers and does not | T2 | every caller |
| `scripts/contract.sh` §350, §351 | the keeping; the undo and its refusals | T1, T2 | the contract gate |

## Inter-task Contracts

| Producer | Consumer | Contract | Verification |
|----------|----------|----------|--------------|
| T1 | T2 | the entry a landed write leaves in `undo/`: its files, their before bytes and modes, `sha_before`, `sha_after`, what each did | T2's tests restore from entries T1's code wrote, not from fixtures |

## Implementation

`tasks/README.md`: T1 (a write keeps what it replaced), T2 (`mrw undo`).

## Consequences

- **Positive:** a wrong mrw write is one command to take back, exactly, without git.
- **Negative:** every landed write copies what it replaced; contents the caller overwrote persist outside the tree until 16 newer writes or `--forget`; a new subcommand, a new store and four flags to keep.
- **Neutral:** no existing exit code or receipt key changes.

## Out of Scope

- A redo (deferred: docs/adr/BACKLOG.md)
- An undo over MCP (deferred: docs/adr/BACKLOG.md)
- Undoing what a check, a step or another tool wrote (permanent: boundary: mrw does not know what they changed)
- Undoing one file of a write and not the rest (permanent: boundary: a plan applies whole or not at all, ADR-001, and so does its undo)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The measurement said this is rarely needed | High | Low | the owner decided; Decision 10's reading can remove it |
| Before-images hold secrets the caller deleted on purpose | Med | Med | 0700/0600, bounded to 16 writes, `mrw undo --forget`, named in the instructions |
| A restore over a file another tool changed | Low | High | Decision 3: every file is verified by sha before anything moves; mutants on each arm |
| Large writes double their I/O | Med | Low | the 64 MiB limit; the receipt says when nothing was kept |
| The restore fails half way | Low | High | it commits as a write does and reports as ADR-066 does |

## Rollback

Before a release, revert T2 then T1. After one, `mrw undo` is a public command: it would be refused with a sentence, and the store left to `--prune`.

## Follow-ups

- [ ] The reading of BACKLOG.md, "Pre-registered for ADR-204".
