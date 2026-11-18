# ADR-201: mrw reads its own state files without blocking

**Status:** Proposed
**Drafted for:** Zy, 2026-11-18. Asked "Should I open a record to make those four state-file reads non-blocking as well?", Zy answered *"YEs of course"*. That approves the draft. Its content, written after the answer, awaits Zy's reading.
**Date:** 2026-11-18
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-009, ADR-056, ADR-075, ADR-077, ADR-104, ADR-108, ADR-109, ADR-127, ADR-177, ADR-200
**Supersedes:** ADR-109, in one Out of Scope clause: "`check.lastLines` and `authoring`'s loaders (permanent: boundary: they read mrw's own files under its state directory, which ADR-077 serves to nobody)". Every other decision of ADR-109 stands.
**Governs:** `internal/regular/regular.go`, `internal/state/state.go`, `internal/state/lock.go`, `internal/writer/writes.go`, `internal/mcp/ack.go`, `internal/check/check.go`, `internal/authoring/authoring.go`, `cmd/mrw/run177.go`, `scripts/contract.sh`
**Enforced-by:** `internal/state/fifo201_unix_test.go::TestAFifoAtAStateFileDoesNotHoldMrw`
**Served-path change:** a FIFO, socket or device at one of mrw's own state files no longer holds a command. Each read treats it as it treats a file it cannot read: the root marker is rewritten, a lock's holder is unknown, the write counter reads zero, the pending acknowledgements are refused as not a regular file, a check's or a run's log reads as empty.

## Context

**The failure, reproduced** on mrw v1.89.0 (03d5e57), 2026-11-18, in a scratch root: a FIFO put at `root`, the marker in the checkout's state directory, and then `mrw read f.txt` under a five-second limit: no output, killed at the limit. `state.Dir` reads the marker with `os.ReadFile` on every command, and opening a FIFO for reading waits for a writer that never comes.

**The same failure had already been found and fixed for four files, against a standing decision.** ADR-200 T1's review found that a FIFO at the `numbering` counters held a write after its files had reached disk: no receipt, no exit. #663 (v1.89.0) fixed it there and for the tally, the recent ring and the `pricing` counters, which had it since before v1.88.0. ADR-109 had put exactly those loaders out of scope, permanently, on the ground that ADR-077 serves mrw's state to nobody. #663 did not name that clause. This record names it, supersedes it, and owns #663's change.

**Why the boundary was the wrong one.** ADR-077 says mrw does not serve or edit its own state through `read` and `write`. It says nothing about what else writes there: the directory is the user's, mode 0700, and any process of that user can put a file in it. The case is not an attacker. It is a write that lands and then says nothing, which is the failure this tool exists to prevent, and a command that hangs with no line on standard error.

**Audit of the class**: *a read of a file under mrw's state home by a served command.* Found with `grep -rn 'os\.Open(\|os\.ReadFile(\|os\.OpenFile(' --include='*.go' internal cmd | grep -v _test.go`, leaving out `internal/bench`, `internal/curve`, `cmd/bench`, `cmd/curve` and `cmd/nativebench`, which are measurement tools and not the served binary; and with `grep -rn 'regular\.Open(' --include='*.go' internal cmd | grep -v _test`, 2026-11-18 at `f4d09b8f`.

Already non-blocking:
- the read ledger and its legacy forms (`internal/seen/seen.go`, five opens), the working set (`internal/iter/iter.go`), the pending store's legacy read (`internal/mcp/ack.go`), one read in `internal/state/state.go`, the drift check (`internal/writer/drift.go`): ADR-109.
- the tally, the recent ring, the pricing and numbering counters (`internal/authoring/authoring.go`): #663.

Blocking today, decided here:
- `internal/state/state.go`, the `root` marker, `os.ReadFile`, on every command that needs state. Reproduced above.
- `internal/state/lock.go`, a lock's `.holder` file, `os.ReadFile`, read when a writer waits or is refused (ADR-110).
- `internal/writer/writes.go`, the write counter, `os.ReadFile`, read on every landing and by a check's drift line (ADR-127).
- `internal/mcp/ack.go`, the pending acknowledgements, `os.ReadFile`, on every `mrw_read` and `mrw_write`.
- `internal/check/check.go` `lastLines`, the check's output log under the state home, `os.Open`, after every check.
- `cmd/mrw/run177.go` `scanLog`, a run's log under `mrw-runs`, `os.Open`, after every `mrw run`.

Not in the class:
- `internal/state/lock.go`, the lock files themselves, opened `O_RDWR|O_CREATE`: opening a FIFO read-write does not wait for a peer. T1 tests it and the record is amended if it does.
- `cmd/mrw/main.go`, the plan file and the `--files-from` list: a pipe is a legitimate input there (ADR-109).
- `cmd/mrw/refused176.go` and `internal/check/runlog177.go`: they create with `O_EXCL` and write.

## Existing Primitives Audit

- `regular.Open` (ADR-109) opens without blocking and asks the descriptor what it opened. Every site above can use it; four of them read a whole small file, so one helper beside it, `regular.ReadFile`, reads a regular file whole and refuses anything else.
- Each site already has a path for "the file cannot be read". Nothing new is decided about what a command does then.
- `internal/authoring`'s busy handling (#663): a regular file under another process's lease answers `EWOULDBLOCK` to a non-blocking open on Linux. Where a read is followed by a save of the same file, reading that as empty would overwrite it.

## Decision

1. **Every read in the audit's "blocking today" list opens without blocking**, through `regular.Open` or `regular.ReadFile`. A FIFO, a socket or a device there is `regular.ErrNotRegular`, and the site does what it does today for a file it cannot read.
2. **A read that is followed by a save of the same file does not save over a file that was only busy.** The write counter is the one such site here: `bumpWrites` reads, adds one and writes. When the read reports the file busy (`EWOULDBLOCK` after the short retry #663 uses), the counter is not written, and the landing reports no generation, as it does today when the counter cannot be written. The marker is rewritten only when its content is wrong or it is not a regular file, never because it was busy.
3. **Nothing falls back to a blocking open.**
4. **ADR-109's clause is superseded, and #663 is owned here.** A read of mrw's own state is a loader like any other.
5. **What would make this decision wrong.** A site where "cannot be read" and "is not a regular file" must differ for a caller: T1's tests state the outcome at each site, and a site that needs its own answer gets it named in the task before the code. A platform where the non-blocking open of a regular state file fails where the blocking one worked: the Windows jobs run every existing state test, since Go ignores `O_NONBLOCK` there.

## Alternatives Considered

- **Leave it, as ADR-109 decided.** Rejected: the hang is reproduced on the released binary by one `mkfifo`, and #663 already crossed the line for four files.
- **Check the file type with `Stat` before a blocking open.** Rejected by ADR-109 for every other loader: the swap between the check and the open is the race it closed.
- **Refuse to run when the state directory holds anything that is not a regular file or a directory.** Rejected: a scan on every command, and a refusal where today's outcome (treat the file as unreadable) is enough.

## Component / Boundary Impact

Two engine packages, each owned here for this change: `internal/state` (the marker and the holder) and `internal/check` (`lastLines`). Also `internal/regular` (the helper), `internal/writer` (the counter), `internal/mcp` (the pending store), `cmd/mrw` (the run log). `internal/seen`, `internal/plan`, `internal/read` and `internal/apply` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `regular.ReadFile` | new: a regular file read whole, anything else refused | T1 | state, writer, mcp |
| six read sites | open without blocking | T1 | every command |
| `scripts/contract.sh` §344 | a FIFO at the root marker does not hold `mrw read`; paired with the same read on a clean state | T1 | the contract gate |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** no file in mrw's state can hold a command or swallow the receipt of a write that landed.
- **Negative:** on Linux a state file under another process's lease is read as unreadable for that call where a blocking open would have waited; for the write counter that is one landing without a generation.
- **Neutral:** no receipt key, exit code or refusal text is added.

## Out of Scope

- The measurement tools' reads (`internal/bench`, `internal/curve`, `cmd/bench`, `cmd/curve`, `cmd/nativebench`) (permanent: boundary: they are not the served binary and read files their own run wrote)
- A plan file or a `--files-from` list that is a pipe (permanent: boundary: ADR-109, a pipe is a legitimate input there)
- A state directory that is itself replaced or made unreadable (permanent: boundary: an unusable state directory is already named as mrw's, with how to set `XDG_STATE_HOME`, by ADR-164; that is a different failure from a file inside a usable one)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A site's "unreadable" path does something worse for a FIFO than hanging (a marker rewrite that follows a symlink, a refusal with a confusing reason) | Med | Med | T1's test states the outcome at each site, and the go-reviewer pass reads each path |
| The lock files do block on a FIFO on one platform | Low | Med | T1 tests it on Linux CI and macOS; the record is amended if so |
| `EWOULDBLOCK` on a regular state file under a lease loses an update | Low | Low | Decision 2: nothing is saved over a busy file |

## Rollback

Revert T1. No receipt key or state format is added, so nothing is left behind.

## Follow-ups

- [ ] None beyond T1.
