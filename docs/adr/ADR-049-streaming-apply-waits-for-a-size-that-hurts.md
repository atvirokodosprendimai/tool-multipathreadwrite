# ADR-049: Streaming apply waits for a size that hurts

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/apply/apply.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** `internal/apply` reading a whole file into memory. Enumerated 2026-09-12 with

```
git ls-files internal/apply/apply.go
```

Measured 2026-08-31 on a 14 MB / 200,000-line file: five hunks in 0.15 s. Fine at that size, unbounded in principle.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record only. Do **not** write a streamer this turn. Promote when a real file makes it hurt — record that size. No decision until then.

## Alternatives Considered

- **Stream tonight — rejected: no size that hurts.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-049-streaming-apply-waits-for-a-size-that-hurts/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- A streamer this turn (permanent: boundary: record only)
- A `cmd N` registry (deferred: docs/adr/BACKLOG.md)
- Implementing an engine dream in the same turn as this record (permanent: boundary: M said record only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reads Accepted as "implement now" | Med | High | Decision says record only; T1 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Delete the record. Nothing in the engine depends on it.

## Follow-ups

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said execute all till 050; T1 receipts; whole-file apply stays; streamer waits.
