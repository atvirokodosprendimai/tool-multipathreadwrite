# ADR-079: mrw's bookkeeping survives racing processes and says what happened

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), whose ADR-079 is this record
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-038, ADR-043, ADR-055, ADR-056, ADR-072, ADR-075, docs/adr/BACKLOG.md
**Governs:** `internal/state/lock.go`, `internal/state/lock_unix.go`, `internal/state/lock_windows.go`, `internal/state/lock079_test.go`, `internal/state/state.go`, `internal/seen/seen.go`, `internal/seen/lock_unix.go`, `internal/seen/lock_windows.go`, `internal/iter/iter.go`, `internal/iter/update079_test.go`, `internal/authoring/authoring.go`, `internal/authoring/lock079_test.go`, `cmd/mrw/main.go`, `cmd/mrw/bookkeeping079_test.go`, `internal/mcp/tools.go`, `internal/mcp/dryrun079_test.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/authoring/lock079_test.go::TestConcurrentRecordsCountEveryOutcome`
**Invalidates:** ADR-043's *"Load unlocked"* for `mrw seen`, the working set and the tally; ADR-038 Decision 2's *"Load stays unlocked for readers that only report"*
**Served-path change:** `mrw iter` reads, changes and writes the working set under its lock, and every reader of it takes the lock; the tally, the recent-window ring and the pricing counters are read and written under one lock, a writer of them that cannot take it writes nothing, and `mrw stats --reset` reports what it discarded under the same hold; a first run's legacy migration takes each destination's lock; `mrw seen` reads the ledger under its lock; a clean `--dry-run` records nothing in `mrw stats` on either surface, and a refused one records one refusal.

## Context

**What was observed.** The klientams peer's leads from the v1.25.1 round (BACKLOG, "Unlocked state
beside the ledger") recorded that `internal/iter` and `internal/authoring` rewrite their files with no
lock and that `mrw seen` loads the ledger unlocked, and guessed the cost at "a working-set entry or a
tally count". It is worse: every one of those files is rewritten by truncate-and-write, and every
loader treats an empty file as empty rather than as an error — so a process racing another could read
the file mid-rewrite, rebuild from nothing and save, wiping the whole tally or working set. ADR-043
measured the same tear on the ledger (6 empty loads in 1,189), and the review of #233 saw it in 5 of
12 runs.

**And the dry run** (the review of #229): a clean `--dry-run` was tallied as `applied` on the CLI and
counted among the landed writes though nothing landed; over MCP every dry run was `refused_apply`.

**And the reviews of #240** (Codex and in-process) found the first cut incomplete: a writer that
could not take the tally's lock wrote anyway — a lock file mrw cannot open says nothing about
whether the tally beside it is writable — and `state.Migrate`, run at startup, checked for a
destination and copied the legacy file with no lock, so an `iter add` landing between the two was
overwritten. The tally half of §160 went red in 1 run of 18 with the lock removed, and `stats
--reset` read the count it reported before the hold that discarded it.

## Existing Primitives Audit

- `seen.hold` took an exclusive flock (LockFileEx on Windows) on a named file in the state directory,
  released by the kernel if the process dies. It was private to `seen`.
- `seen.Snapshot` already reads the ledger under its lock (ADR-075).

## Decision

1. `hold` moves to `internal/state` as `state.Hold(root, name)`, with its platform files; `seen`
   uses it unchanged.
2. `iter.Update(root, fn)` reads, changes and writes the working set under `iteration.lock`; `Load`
   takes the same lock; `mrw iter` changes the set only through `Update`.
3. The tally, the ring and the pricing counters share `authoring.lock`: every exported reader and
   writer takes it, over unexported bodies that never take it again. A writer that cannot take it
   writes nothing — the tally never fails a write, and loses that count; a reader reads past it;
   `Reset` reports it, and returns what it discarded, read under the same hold.
4. `mrw seen` reads through `seen.Snapshot`.
5. A clean dry run records nothing; a refused one is one refusal, on both surfaces.
6. `state.Migrate` holds each destination's lock (`<name>.lock`, the one `seen` and `iter` take)
   across its check and its copy, and skips a file whose lock it cannot take.

Rename-over saves stay out (ADR-043 asks for a measurement first): the lock removes the tear for every
mrw reader, which is the evidence available.

## Alternatives Considered

- **A non-blocking lock with a bound, falling back unlocked.** Rejected: it returns the race it
  exists to remove whenever the machine is slow; a holder that dies releases the lock anyway.
- **Rename-over saves.** Deferred, per ADR-043.

## Component / Boundary Impact

`internal/state` gains `Hold`; `internal/seen` loses its private copy; `internal/iter`,
`internal/authoring`, `cmd/mrw` and `internal/mcp` use them. Lock order is unchanged from ADR-075: the
write lock, then the ledger's; the working set's and the tally's are taken after a write releases
both, and never inside one another.

## Wiring & Contract Changes

No new refusal; the tally no longer counts a clean dry run. Contract §160, §161.

## Inter-task Contracts

T1 produces `state.Hold`, which T2 does not need.

## Implementation

See `tasks/`.

## Consequences

- A process stopped (not killed) while holding a state lock makes the next one wait, as ADR-038's
  ledger lock already does.
- `mrw stats` stops counting dry runs as landings; a caller who read `landed writes` gets the truth.
- `mrw stats` reads the tally, the ring and the counters in three holds: each file whole, not the
  three at one instant.

## Out of Scope

- Rename-over saves for the state files (deferred: ADR-043 asks for a measurement first; docs/adr/BACKLOG.md "Torn `Load`")
- A lock holder stopped with SIGSTOP (permanent: fact: flock is released on death, not on stop, the same as the ledger's lock since ADR-038)
- The MCP pending-ack store and `seen.IsStale`, which read or rewrite without a state lock (deferred: found in the review of #240, not measured; docs/adr/BACKLOG.md "Unlocked state beside the ledger")
- A plan refused by a filesystem error before validation, which the CLI does not tally and MCP counts as `refused_apply`, with or without `--dry-run` (deferred: found in the review of #240, older than this record; docs/adr/BACKLOG.md "Filesystem-error refusals are tallied on one surface")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller of `iter.Save` outside `Update` writes unlocked | Low | Low | `mrw` itself writes only through `Update`, and `Migrate` under the same lock; the doc comment says so |
| A slow check holds nothing, so no lock is held across it | — | — | tally writes happen after `writer.Apply` returns |

## Rollback

Revert the two tasks. The lock files are state, not data, and an older mrw ignores them.

## Follow-ups

- [x] Release with ADR-076 to ADR-078 and ADR-080 as v1.27.0 — tagged at `7058767` (#241), Status #242.
