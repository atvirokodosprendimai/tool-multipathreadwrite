# ADR-048: mrw models no target syntax

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-035, ADR-037, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/apply/apply.go`, `internal/guide/guide.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** Every request that mrw parse or refuse target-language structure. Enumerated 2026-09-12 with

```
git ls-files internal/apply/apply.go internal/guide/guide.go
```

Field reports wanted wrap-tail as an engine. Shared() already teaches it. Line-oriented is ADR-001 / ADR-035.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record the refusal. Do **not** write a syntax-aware parser this turn. mrw puts the lines you gave where you said. Wrap-tail stays teaching.

## Alternatives Considered

- **A Blade/HTML/YAML checker — rejected: one checker per language ends the property that one editor takes every hunk.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-048-mrw-models-no-target-syntax/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- A parser this turn (permanent: boundary: record only; line-oriented is ADR-001)
- A padded write echo (deferred: docs/adr/BACKLOG.md)
- Playtrix (external: wing_playtrix: wing_playtrix/inbox)
- Implementing an engine dream in the same turn as this record (permanent: boundary: M said record only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reads Accepted as "implement now" | Med | High | Decision says record only; T1 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Delete the record. Nothing in the engine depends on it.

## Follow-ups

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said execute all till 050; T1 receipts; no target-syntax parser.
