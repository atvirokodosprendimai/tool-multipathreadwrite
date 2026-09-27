# ADR-085: The MCP pending store is changed under its lock

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — approved the backlog plan that names this record, having said "we can take our own decisison besides M, since we own this now"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-039, ADR-075, ADR-079, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/ack.go`, `internal/mcp/tools.go`
**Enforced-by:** `internal/mcp/pending085_test.go::TestConcurrentHoldsLoseNoPendingSpan`
**Served-path change:** two `mrw mcp` servers on one checkout no longer lose each other's pending acknowledgements: an `mrw_read` page's checkpoint ids are held in `pending.json` under a lock, so a later `ack` of an id that was served always finds it. Receipts, exit codes and error texts are unchanged.

## Context

**What was observed** (the review of #240, filed in BACKLOG under "Bookkeeping survives racing
processes", left open): `hold` and `promote` (`internal/mcp/ack.go`) load `pending.json`, change it
and save it with a plain `os.WriteFile`, guarded only by the server's in-process `gate` mutex. Two
servers on one checkout — two MCP hosts, or a host that runs one per window — can each load the store,
add their spans and save, and the later save drops the earlier's. A page that was served then has
checkpoint ids that match nothing, so its acknowledgement licenses nothing and the caller's next
write is refused "has not been read". ADR-079 closed the same shape for the working set and the tally.

**The class**, enumerated 2026-09-27 with `mrw read --grep 'loadPending\(|savePending\(' internal/mcp`:
`hold` and `promote` load and save; `nameTheAck` only loads, to decide whether a refusal can name the
remedy. No other code touches the store.

**Left out:** `seen.IsStale`, which BACKLOG named beside it. `save` truncates the ledger and writes the
header and every line in one `os.WriteFile`, so a racing `IsStale` sees an empty file, which it reads
as not stale, or the whole header; a false stale notice needs a partial first line, which that write
does not produce. It would also put the change in `internal/seen`, an engine package, for no
observable effect.

## Existing Primitives Audit

- `state.Hold(root, name)` (`internal/state/lock.go`) is the file lock ADR-079 uses for the working
  set and the tally; one more lock file name, no new mechanism.
- `iter.Load`'s "a lock that cannot be taken is read past" is the precedent for the read-only
  `nameTheAck`.

## Decision

1. `hold` and `promote` load, change and save `pending.json` inside `state.Hold(root,
   "pending.json.lock")`.
2. `nameTheAck` reads the store under the same lock, and reads past it if the lock cannot be taken:
   it only chooses the wording of a refusal.
3. The order is the pending lock, then the ledger's: `promote` records into the ledger while it holds
   the pending lock, and nothing takes the pending lock while holding a ledger lock (`promote` runs
   before `writer.Apply` takes the write lock, and `nameTheAck` after it releases it).

## Alternatives Considered

- **Keep the in-process gate and document one server per checkout.** Rejected: nothing enforces it,
  and ADR-075 already made writers take turns across processes on the same checkout.
- **Make `seen.IsStale` read under the ledger lock too.** Rejected for now: see Context, "Left out".

## Component / Boundary Impact

`internal/mcp` only.

## Wiring & Contract Changes

A new lock file in the checkout's state directory. Contract §167 drives eight `mrw mcp` servers.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- An `mrw_read` beside another server's read waits for its turn at the store, for the time of one
  small JSON load and save.

## Out of Scope

- `seen.IsStale` under the ledger lock (permanent: boundary: a single-write save leaves no torn header to misread; the Context names the reasoning)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A lock that cannot be taken fails a read | Low — the same state directory holds the store | Low | the read reports the error, as a failed save did before |

## Rollback

Revert the task.

## Follow-ups

None.
