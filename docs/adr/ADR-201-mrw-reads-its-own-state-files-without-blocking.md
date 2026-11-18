# ADR-201: mrw reads its own state files without blocking

**Status:** Accepted
**Accepted:** 2026-11-18 by Zy. Asked "Should I open a record to make those four state-file reads non-blocking as well?", Zy answered *"YEs of course"*. Told then that the draft (#665) covers six reads, supersedes one clause of ADR-109 and takes ownership of #663's change, and that it needed acceptance before any code, Zy answered *"go on, do not stop"* and set the session to continue without pausing. The amendments below were made after that answer and were not shown to Zy before execution: this line is the session's reading of the two answers, stated so it can be challenged.
**Date:** 2026-11-18
**Amended:** 2026-11-18, before any code, from an adr-challenger pass and a Codex xhigh challenge (gpt-6.2-sol), each finding checked against the code: a seventh read (`mrw seen --prune`) joined the class; the pending store reads a FIFO as an empty store and not as an error; the write counter is not saved over a file it could not read; the lock files' outcome is measured and their error names its path; the check log is described where it is; the audit's commands are re-runnable and carry their counts; #663's chronology is corrected.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-009, ADR-034, ADR-056, ADR-075, ADR-077, ADR-085, ADR-104, ADR-105, ADR-108, ADR-109, ADR-110, ADR-127, ADR-164, ADR-166, ADR-175, ADR-177, ADR-200
**Supersedes:** ADR-109, in one Out of Scope clause: "`check.lastLines` and `authoring`'s loaders (permanent: boundary: they read mrw's own files under its state directory, which ADR-077 serves to nobody; the legacy files in the checkout are T5's)". Every other decision of ADR-109 stands.
**Governs:** `internal/regular/regular.go`, `internal/state/state.go`, `internal/state/lock.go`, `internal/state/prune.go`, `internal/writer/writes.go`, `internal/mcp/ack.go`, `internal/check/check.go`, `internal/authoring/authoring.go`, `cmd/mrw/run177.go`, `scripts/contract.sh`
**Enforced-by:** `internal/state/fifo201_unix_test.go::TestAFifoAtAStateFileDoesNotHoldMrw`
**Served-path change:** a FIFO, socket or device where mrw expects one of its own state files no longer holds a command. The root marker is replaced; a lock's holder is unknown; the write counter reads zero; the pending acknowledgements read as an empty store; another directory's marker makes `mrw seen --prune` keep that directory; a check's or a run's log reads as empty. A lock that cannot be taken for a reason other than its holder says which file.

## Context

**The failure, reproduced** on mrw v1.89.0 (03d5e57), 2026-11-18, in scratch roots, each command under a five-second limit and killed at it:
- a FIFO at `root`, the marker in a root's state directory: `mrw read f.txt`. `state.Dir` reads the marker with `os.ReadFile` on every command.
- a FIFO at the `root` of another state directory: `mrw seen --prune --dry-run`.
- a FIFO at `pending-v3.json`: `mrw seen --forget`.

Opening a FIFO for reading waits for a writer that never comes.

**The same failure had already been fixed for four files, against a standing decision.** ADR-200 T1 (#662, `7b42560b`) read the new numbering counters through `regular.Open`, after its review found that a FIFO there held a write whose files had reached disk. #663 (`03d5e577`, v1.89.0) did the same for the tally, the recent ring and the pricing counters, which had the hang since before v1.88.0, and gave all four a short non-blocking retry for a file under a lease, with no save over a file that stays busy. ADR-109 had put exactly `authoring`'s loaders out of scope, permanently. Neither pull request named that clause. This record names it, supersedes it, and owns those changes.

**Why the boundary was the wrong one.** ADR-077 says mrw does not serve or edit its own state through `read` and `write`. It says nothing about what else writes there: the directory is the user's, and any process of that user can put a file in it. The case is not an attacker. It is a command that hangs with no line on standard error, and a write that lands and then says nothing, which is the failure this tool exists to prevent.

**Audit of the class**: *a content open, by the served binary, of a file mrw itself keeps.* Lock waits and filesystem stalls are not in it. Counted at `bdb8904c`, 2026-11-18:
- `grep -rnE 'os\.(Open|ReadFile|OpenFile)\(' --include='*.go' internal cmd | grep -v _test.go | grep -v -E '^(internal/bench|internal/curve|cmd/bench|cmd/curve|cmd/nativebench)/'` → 13 lines.
- `grep -rnE '[A-Za-z_]\.(Open|OpenFile|ReadFile)\(' --include='*.go' internal/state internal/apply | grep -v _test.go`, for opens through an `*os.Root`, which the first form cannot see.
- `grep -rn 'regular\.Open\b' --include='*.go' internal cmd | grep -v _test` → 25 lines.

Already non-blocking:
- the read ledger (`internal/seen/seen.go`) and the working set (`internal/iter/iter.go`): ADR-109.
- the tally, the recent ring, the pricing and numbering counters (`internal/authoring/authoring.go`, through its `openRegular`): #662 and #663, owned here.

Blocking today, decided here:
1. `internal/state/state.go`, the `root` marker, `os.ReadFile`, on every command that needs state.
2. `internal/state/prune.go` `readAll`, another directory's `root` marker, an open through its `*os.Root`, in `mrw seen --prune` and its dry run.
3. `internal/state/lock.go` `holderOf`, a lock's `.holder` file, `os.ReadFile`, read after a bounded wait has run out (ADR-110): a FIFO there defeats the timeout that just expired.
4. `internal/writer/writes.go`, the write counter, `os.ReadFile`, on every landing and after a check (ADR-127).
5. `internal/mcp/ack.go` `loadPending`, the pending acknowledgements, `os.ReadFile`, on `mrw_read`, `mrw_write` and `mrw seen --forget`.
6. `internal/check/check.go` `lastLines`, `os.Open`, on a log mrw has just created with `O_EXCL`: in the system temp directory for a check or a step, under `<state home>/mrw-runs` for `mrw run`. A FIFO gets there only by a replacement after the create.
7. `cmd/mrw/run177.go` `scanLog`, `os.Open`, on the same run log after the command ends.

Not in the class:
- the lock files themselves, opened `O_RDWR|O_CREATE`: the open of a FIFO that way does not wait. Measured on macOS, v1.89.0: `flock` on it fails, and the command exits 2 with `mrw: operation not supported`, naming no file. On Linux `flock` on a FIFO is expected to succeed; T1 states what it finds. Decision 4 gives the error its path.
- reads of files in the checkout (`internal/mcp/ack.go` `currentSHA`, `internal/writer/drift.go`, `internal/state/state.go` `readLegacy`): ADR-108 and ADR-109.
- the plan file and the `--files-from` list: a pipe is a legitimate input there (ADR-109).
- the retention walks under `mrw-refused` and `mrw-runs`: names and metadata only.

## Existing Primitives Audit

- `regular.Open` (ADR-109) opens without blocking and asks the descriptor what it opened. ADR-109 Decision 1 describes the package as one function; this record adds two beside it and leaves `Open` as it is, directories included.
- `internal/apply/tree.go` already opens through an `*os.Root` without blocking and checks the descriptor; the prune read needs the same.
- Each site has a path today for "the file cannot be read". `describe` keeps a directory whose marker it cannot read (ADR-034). `loadPending` reads a store that will not parse as empty, which licenses nothing.
- `state.Write` replaces a file by renaming a sibling over it and refuses a target with no owner-write bit (ADR-105), so a writable FIFO is replaced without being opened.
- `statehint.About` recognises a state failure by the path in an `*fs.PathError` (ADR-166's serve-without-checkpoints path depends on it).

## Decision

1. **Every read in the "blocking today" list opens without blocking.** `regular.ReadFile(path)` reads a regular file whole; `regular.OpenIn(root, name)` opens through an `*os.Root`. Both refuse a FIFO, a socket, a device and, for `ReadFile`, a directory, with an `*fs.PathError` that names the path and wraps `regular.ErrNotRegular` or the system's error.
2. **What each site does with a file that is not regular.**
   - the marker: `state.Dir` rewrites it, as it does today for any marker it cannot read or that names another root, and returns the directory.
   - another directory's marker: `describe` keeps the directory as unidentified.
   - a lock's holder: unknown; the refusal keeps its "another process" wording.
   - the write counter: zero, and the bump writes 1 over it, as for a missing counter.
   - the pending store: an empty store, as for one that will not parse. It licenses nothing, the next save replaces it, and `mrw seen --forget` goes on to empty the ledger (ADR-175). An error here would refuse every `mrw_read` until someone deleted the file.
   - a check's or a run's log: no lines. A run's receipt then says `0 lines · 0 bytes` for output it cannot see, with the log path and the command's exit code unchanged.
3. **A read that failed for another reason is not saved over.** The write counter and the pending store are read and then saved. When the read fails for a reason other than "missing" or "not a regular file" (a file under another process's lease, a permission), the counter is not written and the landing reports no generation, which ADR-127 allows as best effort and which today resets a valid counter to 1; the pending store returns its error, as today. A counter whose content does not parse still reads as zero.
4. **A lock that cannot be taken says which file.** `Hold` and `HoldFileWithin` wrap an error from opening or locking the lock file in an `*fs.PathError` with the lock's path. Waiting for a holder is unchanged (ADR-110).
5. **Nothing falls back to a blocking open, and ADR-109's clause is superseded.** A read of mrw's own state is a loader like any other.
6. **What would make this decision wrong.** A site where failing at once leads somewhere worse than the wait did: Decision 2 states each outcome and T1 tests it. A platform where the non-blocking open of a regular state file fails where the blocking one worked: Go ignores `O_NONBLOCK` on Windows, and T1 adds controls for a regular file, a missing file and a directory that run there.

## Alternatives Considered

- **Leave it, as ADR-109 decided.** Rejected: three hangs are reproduced on the released binary with one `mkfifo` each, and #662 and #663 already crossed the line for four files.
- **Check the file type with `Stat` before a blocking open.** Rejected by ADR-109 for every other loader: the swap between the check and the open is the race it closed.
- **Refuse the pending store that is not a regular file.** Rejected: the refusal names no cause the caller can act on from an MCP host, does not heal, and stops `--forget`.
- **Refuse to run when the state directory holds anything that is not a regular file or a directory.** Rejected: a scan on every command, where treating the file as unreadable is enough.

## Component / Boundary Impact

Two engine packages, each owned here for this change: `internal/state` (the marker, the prune read, the holder, the lock error) and `internal/check` (`lastLines`). Also `internal/regular` (two functions), `internal/writer` (the counter), `internal/mcp` (the pending store), `cmd/mrw` (the run log). `internal/seen`, `internal/plan`, `internal/read` and `internal/apply` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `regular.ReadFile`, `regular.OpenIn` | new | T1 | state, writer, mcp |
| seven read sites | open without blocking | T1 | every command |
| a lock error | carries the lock's path | T1 | the CLI's state hint |
| `scripts/contract.sh` §344 | a FIFO at the root marker, and at another directory's, does not hold the binary | T1 | the contract gate |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** no file mrw keeps can hold a command or swallow the receipt of a write that landed, and a lock that fails names itself.
- **Negative:** on Linux a state file under another process's lease is read as unreadable for that call where a blocking open would have waited; for the write counter that is one landing without a generation.
- **Neutral:** no receipt key or exit code is added. One error text gains a path.

## Out of Scope

- The measurement tools' reads (`internal/bench`, `internal/curve`, `cmd/bench`, `cmd/curve`, `cmd/nativebench`) (permanent: boundary: they are not the served binary and read files their own run wrote)
- A plan file or a `--files-from` list that is a pipe (permanent: boundary: ADR-109, a pipe is a legitimate input there)
- A state directory that is itself replaced or made unreadable (permanent: boundary: an unusable state directory is already named as mrw's, with how to set `XDG_STATE_HOME`, by ADR-164; that is a different failure from a file inside a usable one)
- Waiting on a lock another process holds (permanent: boundary: ADR-110 bounds the write lock's wait; a wait is not a content read)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A site's outcome for a FIFO is worse than the hang | Med | Med | Decision 2 states each; T1 tests each; the go-reviewer pass reads each path |
| `flock` on a FIFO differs by platform | Med | Low | T1's lock case accepts a lock taken or an error that names the lock file, and says which it saw |
| A lost update on a state file under a lease | Low | Low | Decision 3: nothing is saved over a file that could not be read |

## Rollback

Revert T1. No receipt key or state format is added, so nothing is left behind.

## Follow-ups

- [ ] None beyond T1.
