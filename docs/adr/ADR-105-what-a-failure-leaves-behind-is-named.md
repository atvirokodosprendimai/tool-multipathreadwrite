# ADR-105: What a failure leaves behind is named

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — approved the plan "resolve the seven findings of the 2026-09-30 Codex design review" (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`), whose ADR-105 is this record's scope, and set the goal "deliver open tasks end to end". The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-066, ADR-075, ADR-086, ADR-102
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/state/write.go`, `internal/seen/seen.go`, `internal/authoring/authoring.go`, `internal/iter/iter.go`, `internal/mcp/ack.go`, `internal/mcp/wirepath.go`, `internal/mcp/schema.go`, `cmd/mrw/main.go`, `README.md`, `scripts/contract.sh`
**Enforced-by:** None — the test is written in T1 (follow-up below)
**Invalidates:** none — checked. ADR-004 promises nothing is left in the tree by a failed run except the named aside; this record reports the case where a removal mrw attempted fails, which ADR-004 did not cover. ADR-066's undo text names an aside it keeps; that text stays, and the path also appears in `left_behind`.
**Served-path change:** (1) A write receipt carries `left_behind`, root-relative paths mrw made in the tree and could not remove, or kept on purpose as a recovery file; absent when there are none. The CLI prints `left behind: <path>` for each, on success and on failure, and `mrw_write` spells them with `/`. (2) mrw's state files are replaced by rename rather than rewritten in place, so a reader never sees half of one; the ledger and the MCP acknowledgement store are synced before the rename. Exit codes keep their meanings: a leftover does not turn an applied write into a failure.
**Enforced-by:** `internal/apply/left105_test.go::TestEveryFailedCleanupIsNamedInLeftBehind`
## Context

**What was observed** (2026-09-30, the Codex design review of `f1d5996`, finding 5, confirmed in source at `1bd8690`):

1. `apply.go` `discard` (`:659-669`) removes staged temp files and the directories staging made with `_ = os.Remove`,
   so a removal that fails leaves a `.mrw-*` file or a directory in the tree while the receipt says nothing was
   written. `stageFile`'s four error paths (`:1761-1780`) do the same with their own temp file.
2. `pathop.go` removes the unlink placeholder on a failed close (`:207`) and every aside after a successful commit
   (`:283-285`) with `_ = os.Remove`; a placeholder whose removal is refused (`:210`) aborts the commit and stays.
   `probeName` (`apply.go:1703-1713`) names a probe it could not remove in its error, and no receipt field carries it.
3. `seen.save` (`seen.go:440`) and every other state write use `os.WriteFile`, which truncates and rewrites in
   place: a reader that runs in between, or a process killed in the middle, sees a short file. For the ledger that
   is a lost licence; for the tally, a lost count.

BACKLOG carries three entries this closes: "A `.mrw-aside-*` left behind" and "A probe left behind" (From ADR-066),
and "The cleanup errors mrw ignores" (From ADR-102).

**Audit of the class.** Two classes.
*A removal of something mrw made in the tree, whose failure is dropped or only named in prose.* Enumerated
2026-09-30 by `mrw read --grep 'os\.Remove' --exclude '*_test.go' internal/apply/` and reading each site: `discard`'s
temp and directory removals, `stageFile`'s four, `probeName`'s, the placeholder's two in `unlinkOne`, the final
aside removal, and the two undo outcomes that keep an aside — **12** sites, all in scope. `os.Remove(target)` in
`probeName` is the probe itself; its failure is one of the 12.
*A write of a file under mrw's state directory.* Enumerated by `mrw read --grep 'os\.WriteFile' --exclude '*_test.go'
internal/ cmd/mrw/`: `seen.go:440`, `authoring.go:202`, `:288`, `:405`, `iter.go:135`, `mcp/ack.go:450`,
`state.go:76` (the root marker), `state.go:135` (legacy migration) — **8**, all in scope.
**Left out:** `internal/curve`'s four writes, which a measurement tool makes into its own output tree, not mrw's
state; `state/lock.go`, which opens a lock file and writes nothing into it.

## Existing Primitives Audit

- **`DirsCreated`** (`apply.go:165-168`) and its three consumers (`report`, `slashResult`, the schema) — the model
  for a new root-relative list on the receipt.
- **`stageFileFn`, `commitRenameFn`, `probeNameFn`** (`apply.go:1688-1720`) — the seam pattern; `removeFn` joins them.
- **ADR-075's writer lock** (`state/lock.go`) — writers already take turns, so an atomic replace is about readers
  and about a process that dies mid-write, not about two writers.

## Decision

1. **`left_behind`.** Every cleanup removal in `internal/apply` goes through one helper over a `removeFn` seam. A
   path that is still there afterwards is appended to `Result.LeftBehind`, root-relative. A path already gone is
   not left behind. A directory still holding something is not named: whatever is in it is either named itself
   (a temp mrw could not remove) or not mrw's, and os.Remove's refusal of a non-empty directory is the guard
   ADR-004's `discard` relies on. `stageFile`'s error paths return their temp file to the caller, whose `discard`
   removes and reports it, so one place does both. An aside that the undo keeps, and a probe that could not be
   removed, are named too: the field answers "what is in my tree that I did not ask for".
2. **Atomic state writes.** `state.Write(name, data, perm)` writes a temp file in the same directory and renames it
   over `name`. If the rename is refused (Windows refuses to replace a file another process has open), the temp is
   removed and the old in-place write is used, so no platform is worse off than before. A state file its owner made
   read-only is written in place too, so it stays refused: a rename would replace it whatever its mode, and the
   ledger-failure fixtures of ADR-072 and ADR-102 (a `0444` ledger) rely on that refusal. All 8 sites use it.
3. **fsync, by measurement against bars registered before measuring.** The bars were registered in the approved
   plan (2026-09-30, `~/.claude/plans/ok-create-a-plan-whimsical-newell.md`, decision 5 and its critique) and are
   restated here unchanged: a state file's sync is kept if it costs **≤10 ms** median per write; a staged tree
   file's sync is kept if it adds **≤10 %** wall time to a 500-file, 5,000-hunk write (medians of 5 alternating
   runs each, measured with the load average below the core count). Measured 2026-09-30 on the owner's Mac (APFS,
   `File.Sync` is `F_FULLFSYNC`): a 100 KB state file's write, sync and rename took a **4.0 ms** median (40 runs,
   twice, in the state directory) against 0.09 ms unsynced — under the bar. `state.WriteSynced` syncs before the
   rename and is used for the two files that carry licences, the ledger and the acknowledgement store; the tally,
   ring, pricing, working set and marker are measurement or convenience, and use `state.Write`. The tree-file
   measurement is T3's first step, and its result decides whether staging syncs.
4. **A failure matrix** in the README: for each stage (validation, staging, commit, undo, ledger, check, process
   death, power loss), what is on disk, what the receipt says, the exit code, and how to recover.

No transaction journal: each file already lands by rename, and a journal is a subsystem with its own recovery bugs.

## Alternatives Considered

- **A transaction journal with crash recovery** — rejected: every file lands by an atomic rename already; a journal
  adds replay, its own format and its own failure modes to protect a window between renames that the receipt and
  the undo already report.
- **Fail the write when a cleanup fails** — rejected: the plan landed; a leftover is hygiene, and exit 1 or 3 would
  say the write did not happen or was unverified, which is false.
- **Only name leftovers in the error text** — rejected: the error text is prose; an MCP or `--json` caller needs a
  field, and a successful write has no error text at all.
- **Sync every state file** — rejected: a write call saves up to five state files, so syncing all of them costs
  about 20 ms per call to protect counts that are measurement.

## Component / Boundary Impact

Engine packages owned: `internal/apply` (T1; T3 if the tree measurement passes), `internal/state` (T2, T3),
`internal/seen` (T2, T3), `internal/iter` (T2). `internal/authoring`, `internal/mcp` and `cmd/mrw` are not engine
packages. `internal/plan`, `internal/lines`, `internal/rooted`, `internal/read`, `internal/check`, `internal/links`
stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `apply.Result.LeftBehind` (`left_behind`) | new receipt field | T1 | CLI human and `--json` receipts, `mrw_write` |
| `mrw_write` output schema | `left_behind` property | T1 | MCP hosts |
| state files | replaced by rename; ledger and ack store synced | T2, T3 | every mrw run |
| `README.md` | the failure matrix | T4 | callers |
| `scripts/contract.sh` | §201 (T2) | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `state.Write` | T2 | T3 | no |
| `Result.LeftBehind` | T1 | T4 | no |
| the fsync outcome | T3 | T4 | no |

## Implementation

See `tasks/README.md`: T1 (left_behind), T2 (atomic state writes), T3 (fsync by measurement), T4 (the failure matrix).

## Consequences

- **Positive:** a file mrw leaves in the tree is named on the receipt; a state file is never seen half-written; a
  power loss cannot tear the ledger on a filesystem that honours sync.
- **Negative:** each mrw call that saves the ledger pays about 4 ms on the measured machine.
- **Neutral:** receipts without leftovers are unchanged; exit codes keep their meanings.

## Out of Scope

- A contract row for `left_behind` (permanent: boundary: no fixture fails one removal between two steps of a single call from outside the process; T1's unit tests drive every site through `removeFn`, and the receipt tests drive each renderer)
- A transaction journal (permanent: boundary: rejected above; each file lands by rename)
- `internal/curve`'s writes (permanent: boundary: a measurement tool's own output tree, not mrw state)
- Anchoring file operations to a handle (deferred: ADR-106, `docs/adr/BACKLOG.md` "Anchoring file operations to a handle")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a rename over a state file another process holds open fails on Windows | Med | Low | `state.Write` falls back to the in-place write; the Windows CI jobs run T2's test |
| a temp state file survives a killed process | Low | Low | it sits in the state directory, outside the tree (ADR-004), named `.<file>.tmp-*` |
| sync cost differs on another machine | Med | Low | the bar and the measurement are dated and name the machine; only two files sync |

## Rollback

Revert T1–T4. The receipt field is additive; state files keep their format, so an older binary reads them.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/apply/left105_test.go::TestEveryFailedCleanupIsNamedInLeftBehind` in T1's commit.
