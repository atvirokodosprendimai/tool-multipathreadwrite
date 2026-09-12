# ADR-050: Windows state stays XDG until Windows is exercised

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-034, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/state/state.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** The directory mrw uses for per-root state on each OS. Enumerated 2026-09-12 with

```
git ls-files internal/state/state.go
```

The state path is XDG-shaped; Windows would want `%LOCALAPPDATA%`. Cross-compile in CI is not an exercise of the path.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record only. Do **not** move the state path this turn. XDG stays. Promote when a Windows contributor reports a real state-path miss, or when CI exercises more than the binary.

## Alternatives Considered

- **LOCALAPPDATA tonight — rejected: unexercised, not broken.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- Changing the state path this turn (permanent: boundary: record only)
- The rules hook on Windows (deferred: docs/adr/BACKLOG.md)
- Reopening ADR-019 B/C (permanent: fact: pick A stands; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
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
