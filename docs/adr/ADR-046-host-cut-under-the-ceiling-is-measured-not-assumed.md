# ADR-046: Host-cut under the ceiling is measured, not assumed

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-039, ADR-032, ADR-024, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/tools.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — measurement only
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** An MCP result that is under `MaxResultChars` and still gets cut by the host. Enumerated 2026-09-12 with

```
git ls-files internal/mcp/tools.go
```

Reading-18 measured a *paged* cut. ADR-039 licensed fitting reads by ack; it is not an under-ceiling host-cut measurement.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

This record is a **measure**. How we would observe: drive a fitting `mrw_read` whose encoded size is under the advertised ceiling, capture what the *host* delivered (not what mrw sent), and compare spans. A cut that drops a middle span under the ceiling is the datum. Do **not** change the engine until that measurement exists. ADR-039's ack path is not this evidence.

## Alternatives Considered

- **Treat ADR-039 as the measurement — rejected: it licensed fitting reads; it did not measure a host cut under the ceiling.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-046-host-cut-under-the-ceiling-is-measured-not-assumed/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- Engine change this turn (permanent: boundary: measure first)
- Paged cuts already measured in reading-18 (permanent: fact: paged, not under-ceiling; citation: file `docs/curve/reading-18-result.md:1`)
- Raising `MaxResultChars` (permanent: boundary: ADR-032 owns the knob)
- Implementing an engine dream in the same turn as this record (permanent: boundary: M said record only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reads Accepted as "implement now" | Med | High | Decision says record only; T1 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Delete the record. Nothing in the engine depends on it.

## Follow-ups

- [ ] Execute only on a later quote that names this number.
