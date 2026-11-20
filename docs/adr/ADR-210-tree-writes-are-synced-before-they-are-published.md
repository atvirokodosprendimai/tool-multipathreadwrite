# ADR-210: Tree writes are synced before they are published

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accepted the rest"*, to the question of accepting the open records ADR-207 to ADR-211 (#723, #720) as they stand. Decision 7 as proposed: the end-to-end cost T1 measures replaces ADR-105's 10 % bar, and Zy judges the record on those numbers.
**Drafted for:** Zy, 2026-11-20, from an independent critique of the same day (finding S4). Zy's instruction on the critique's findings, as relayed to the drafter by the session that commissioned these records, was *"address all, no choosing"*, so this record carries one decision. It sets aside a bar ADR-105 registered and a decision ADR-105 made on it, which is Zy's call and not the drafter's: the measurement below says what the change costs, and the criterion that replaces the bar is Zy's to state (Decision 7). Challenged before acceptance (Codex at xhigh, 2026-11-20, on `2ded135b`: AMEND); every amendment is applied below or answered in History.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-066, ADR-092, ADR-102, ADR-105, ADR-111, ADR-113, ADR-132, ADR-209
**Supersedes:** in part:
- ADR-105 Decision 3's "staging stays unsynced" and, for tree files, the ≤10 % bar it was judged by;
- ADR-105's Out of Scope item "Syncing the directory after a rename (permanent: boundary: …)", for the tree's directories only; for mrw's state directory it stands;
- the README failure table's power-loss row;
- `AGENTS.md`'s list of what a `2` after `applied` can mean, which gains a sync that failed.
**Governs:** `internal/apply/apply.go`, `internal/apply/tree.go`, `internal/writer/writer.go`, `internal/writer/flow.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`
**Enforced-by:** `internal/apply/sync210_test.go::TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns`
**Served-path change:** a write that reports `applied` with no `error` has had every sync below succeed before mrw records it in the ledger and returns: the bytes it staged, and outside Windows the directory entries it changed. A write takes longer: on the owner's Mac about 6.7 ms per write call plus about 0.05 ms per file in the publish step (measured below). A sync that fails before the first rename writes nothing, exit 2. One that fails after the renames is a landed write with `error`, exit 2 on the CLI and `isError` with the landed receipt on `mrw_write`, and the check and `then` steps do not run, as after a ledger failure.

## Context

**What happens today.** `stageFile` writes the temp, sets its mode and attributes and closes it, with no sync (`internal/apply/apply.go:2546-2560`). The commit renames it over the target (`apply.go:1217`, `(*tree).rename`, `internal/apply/tree.go:133-142`), and nothing syncs the target's directory. The ledger, written after `Apply` returns (`internal/writer/writer.go:108-122`), is synced before its own rename (`state.WriteSynced`, `internal/state/write.go:45-53`, ADR-105). The README's failure table says so: "staged tree files are not synced (+1335 % measured, ADR-105), so a write that returned can be lost".

**What a power loss can do.** POSIX does not order a rename after the data it names. A filesystem that persists the rename first can come back with the target's name on an empty or short file: the new content is lost and so is the previous version, which the rename replaced. The ledger, synced, can then name the sha of content that is not on disk. That record licenses nothing, because a write is judged against the sha of the file as it is (`apply.go:1444`, ADR-002), so the harm is the lost file, not a false licence.

**What ADR-105 decided, and on what.** A pre-registered bar: keep the tree sync if it adds at most 10 % wall time to a 500-file, 5,000-hunk write. Measured 2026-09-30 with one `File.Sync` per staged file: 0.138 s against 1.98 s, +1335 %, with the load average at 12 to 14 on 10 cores, above the limit its own protocol set (ADR-105 T3's sign-off). So staging stayed unsynced.

**Measured again, 2026-11-20**, by a scratch program that publishes N files the way `stageFile` and the commit loop do (`os.CreateTemp` beside an existing 4-byte target, ~4 KB written, chmod, close, rename), under four policies, 9 alternating runs each; `GOTOOLCHAIN=local go run . DIR` on the owner's Mac (APFS on an Apple Fabric SSD, 10 cores, load average 5.99 before and 5.35 after). The program and its raw samples are not committed. Medians, publish step only, every file in one directory:

| n files | none (today) | `File.Sync` per file | + directory sync per rename | `fsync` per file, one drain before the renames and one after |
|---------|--------------|----------------------|-----------------------------|----------------------------------------------------------------|
| 1 | 0.127 ms | 3.56 ms | 7.68 ms | 6.79 ms |
| 50 | 6.58 ms | 208 ms | 413 ms | 16.1 ms |
| 500 | 66.9 ms | 2105 ms | 4211 ms | 99.1 ms |

On macOS Go's `File.Sync` is `fcntl(F_FULLFSYNC)` (`internal/poll/fd_fsync_darwin.go:22` in Go 1.27.4), which drains the device's queue every time: the 4 ms per file. `man fcntl` on this Mac: F_FULLFSYNC "acts as a barrier, data that had been fsync'd on the same device before is guaranteed to be persisted when this call returns". So a plain `fsync` per file and one F_FULLFSYNC per device give every file that guarantee for one drain: the last column. Over today's column it costs about 6.7 ms per call (two drains) plus about 0.05 to 0.06 ms per file (`(16.1 − 6.58 − 6.7) / 50`, `(99.1 − 66.9 − 6.7) / 500`). Where a filesystem answers F_FULLFSYNC with `ENOTSUP`, Go falls back to a plain `fsync` (`fd_fsync_darwin.go:27-28`), and the drain is not a drain there. A separate probe the same day: a plain `fsync` on a directory descriptor on this Mac returns no error in 0.009 ms, and an F_FULLFSYNC on it in 3.9 ms. A one-file CLI write today takes a median 11.0 ms end to end (30 runs, same machine, same day).

Limits of the measurement: one machine; one directory (a plan over twenty directories syncs twenty); the publish step only, not a whole `mrw write`; Linux and Windows not measured. It does not show that the change breaks ADR-105's bar end to end at every size; T1 measures that.

**Audit of the class**: *what a write changes in the tree that a power loss could tear.* A content file's temp and its rename (staging, the commit); the directories staging makes; a path op's rename and unlink, and its aside (ADR-066); ADR-209's `.mrw-aside-*` links, if accepted. Bytes a rename-only plan moves were not written by mrw and are not synced by it. mrw's state files are outside the tree (ADR-004) and keep ADR-105's rule.

## Existing Primitives Audit

- `state.WriteSynced` and its `syncFn` seam are the model for a sync behind a seam a test can watch (ADR-105 T3).
- `(*tree)` reaches every tree path through one `os.Root`; `Root.Open` of a directory gives the handle a directory sync needs.
- `writer.LedgerError` is the existing class for "the write landed and something after it failed": exit 2, `applied` with `error`, no check, steps reported not run (ADR-092 Decision 5, ADR-102).

## Decision

1. **Every staged temp's data is synced before the first rename.** On macOS: a plain `fsync` on each temp as it is staged, then one F_FULLFSYNC on one synced temp of each device the temps are on, before the commit loop (`man fcntl` promises the drain for the same device only, and a root can hold a mount). Elsewhere: `File.Sync` on each temp (`fsync` on Linux and the BSDs, `FlushFileBuffers` on Windows).
2. **Every directory whose entries the write changed is synced once, after the last change and before `Apply` returns**: each content target's directory, each directory staging created and its parent, each path op's source and destination directory, and, if ADR-209 is accepted, after its asides are removed. On macOS a plain `fsync` on each, then one F_FULLFSYNC on one synced directory of each device; on Linux and the BSDs `fsync` on each.
3. **On Windows no directory is synced.** `FlushFileBuffers` needs a handle opened for writing, which `os.Open` of a directory does not give (not run on Windows for this record; T1's Windows job confirms it, and the record is amended if it does not hold). What is promised there is the staged bytes, flushed by Decision 1; whether a new name survives a power loss is not promised.
4. **What `applied` with no `error` promises**: every sync in Decisions 1 and 2 returned success, so the bytes mrw staged and, outside Windows, the directory entries it changed are on stable storage as far as the storage honours a sync. On Windows the promise is the staged bytes only; no directory entry is synced (Decision 3). A write that failed, or landed partly, promises nothing about durability.
5. **A sync that fails.** Before the first rename: the write stops, every temp is removed, `NOTHING WRITTEN`, exit 2 (ADR-132's environment class). After the renames — a filesystem that refuses a directory sync (`EINVAL`, `ENOTSUP`) included — the write has landed: a new internal class beside `LedgerError` (`writer.SyncError`) carries it, the ledger is still recorded, and both errors are kept when the ledger fails too. CLI: `applied` with `error` naming the path that could not be synced and saying the files are written but not known to be on stable storage, exit 2. `mrw_write`: `isError` with the landed receipt; the bounded fallback for an oversized receipt (`appliedButUnreportable`, `internal/mcp/tools.go:1658`) carries the same sentence. The check and every `then` step are not run, and are reported not run, as after a ledger failure (`internal/writer/flow.go:160-175`); `mrw stats` counts it as that path does, `check_not_run` when a check was due and `applied` otherwise.
6. **The words.** The README's power-loss row says `synced before the ledger` and `on Windows only the file contents are synced, not the directory entries`; a new row for a sync that failed after landing gives the recovery: `the files are written but not known to be on stable storage: do not send the plan again`. `AGENTS.md`'s sentence on a `2` after `applied` (`AGENTS.md:109-111`) adds `the files could not be synced`; the `error` key's description (`internal/mcp/schema.go:228`) adds the same case. No receipt key is added (ADR-111).
7. **The criterion that replaces ADR-105's bar is the owner's.** This record proposes: the end-to-end cost T1 measures (ADR-105's 500-file, 5,000-hunk workload and a one-file write, on macOS, Linux and Windows) is recorded here, and Zy accepts or rejects the record on those numbers. ADR-105's 10 % bar is not met by the publish-step numbers above at any size.

## Alternatives Considered

- **`File.Sync` per file and a directory sync per rename**, the critique's form. Rejected: 413 ms for 50 files on macOS against 16 ms for the same guarantee.
- **Keep ADR-105's decision; the README already states the risk.** Rejected: the previous version of a file can be lost with the new one, and an agent that saw `applied` has no way to know.
- **Rely on the ledger's own F_FULLFSYNC as the second drain.** Rejected: it holds only when mrw's state directory is on the tree's device.
- **Skip a directory sync the filesystem refuses.** Rejected: `applied` would then mean two things; the refusal is reported, and the files stay written.
- **Sync only what the plan created.** Rejected: an edited file's rename is the one that can leave the target empty.
- **An opt-in flag.** Rejected: durability that must be asked for is not a property of `applied`.

## Component / Boundary Impact

Engine package owned here: `internal/apply` (staging, the commit, the directory sync; `tree.go` opens a directory through the root). `internal/writer` gains `SyncError` and its classification; `internal/mcp` and `cmd/mrw` render it. `internal/state` is unchanged (ADR-105's rule for state files stands). `internal/plan`, `internal/read`, `internal/seen` and `internal/check` are unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| every landed write | staged bytes synced before the first rename, changed directories after the last change, then the ledger | T1 | every caller |
| a sync that fails | before the renames: `NOTHING WRITTEN`, exit 2; after: `applied` with `error`, exit 2 / `isError` with the landed receipt, check and steps not run | T1 | every caller |
| `writer.SyncError` | new internal class beside `LedgerError` | T1 | `cmd/mrw`, `internal/mcp` |
| `README.md`, `AGENTS.md`, the `error` key's description | Decision 6 | T1 | callers recovering after a crash or a refusal |
| `scripts/contract.sh` §361 | under `strace` (a proof job that has it; skipped elsewhere and saying so): every temp's successful `fsync` before the first tree rename, a successful `fsync` of each changed directory after the last tree change, and the ledger's rename after those; paired with a `--dry-run` on a fresh fixture, which must exit 0 and show no `fsync` on a tree path | T1 | the contract gate |

No receipt key is added or renamed (ADR-111): `applied`, `error` and `files[].written` carry the after-landing failure.

## Inter-task Contracts

None — one task. If ADR-209 is accepted first, its aside removals come before Decision 2's directory syncs; if this one lands first, ADR-209's task adds them.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** a write mrw reported `applied` with no `error` survives a power loss with its staged bytes and, outside Windows, its names, on storage that honours a sync; the previous version is not replaced by an empty file.
- **Negative — the cost:** on the owner's Mac about 6.7 ms per write call (two drains) plus about 0.05 ms per file in the publish step: a one-file write goes from a median 11.0 ms to about 18 ms; 50 files gain about 10 ms; 500 files about 32 ms. A plan over many directories adds one `fsync` per directory. Linux and Windows unmeasured until T1. On a mount that refuses a directory sync every write ends exit 2, its files written.
- **Neutral:** mrw's state files keep ADR-105's rule.

## Out of Scope

- Syncing mrw's state directory after the ledger's rename (permanent: boundary: ADR-105's Out of Scope stands for it; a ledger that comes back older only refuses edits until the files are read again)
- A durable rename on Windows (permanent: boundary: `os.Root.Rename` goes through Go's handle-relative `windows.Renameat` (`os/root_windows.go:387` in Go 1.27.4), whose flags mrw does not choose, and a directory cannot be flushed through `os.Open`'s handle)
- Bytes a rename-only plan moves (permanent: boundary: mrw did not write them; Decision 2 syncs the directory entries the rename changed)
- Storage that answers a sync before the data is stable (permanent: boundary: mrw cannot tell such an answer from a true one; Decision 4 promises only as far as the storage honours a sync)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The cost on Linux CI or a slow disk is much larger | Med | Med | T1 measures it end to end before the record is accepted (Decision 7) |
| A host whose tests write thousands of files through mrw slows down | Low | Med | the per-file part is a plain `fsync`; the drains are per call and per device |
| A mount refuses a directory sync, and every write there exits 2 | Low | Med | Decision 5 keeps the files written and says so; the case is named in the README row |

## Rollback

Revert T1; no state is involved.

## History

- 2026-11-20, Codex at xhigh on `2ded135b`, AMEND, three points, applied after checking each against the code: the promise is tied to syncs that succeeded and to what mrw wrote, Windows is limited to the staged bytes, and `os.Root.Rename`'s Windows path is named (it is `windows.Renameat`, not `MoveFileExW`); a sync failure after landing is a class of its own beside `LedgerError`, with the check, steps, tally and MCP fallback specified (`internal/writer/flow.go:160-175`, `internal/mcp/tools.go:1658`), and a mount that refuses a directory sync is decided; ADR-105's directory-sync exclusion is amended for the tree; the per-file cost is corrected from 0.2 ms (the policy's own slope, which includes today's publish cost) to the 0.05 to 0.06 ms it adds; §361 asks for every temp sync before the first rename. The replacement for ADR-105's bar is left to Zy (Decision 7).

## Follow-ups

- [ ] The end-to-end cost on macOS, Linux and Windows, in T1.
