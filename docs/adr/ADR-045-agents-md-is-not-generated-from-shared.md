# ADR-045: AGENTS.md is not generated from Shared()

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-037, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/guide/guide.go`, `AGENTS.md`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** The two documents that must contain `guide.Shared()` verbatim. Enumerated 2026-09-12 with

```
git ls-files internal/guide/guide.go AGENTS.md
```

ADR-037 refused a two-way sync tax. `Contains(Shared())` is the gate, not a generator.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record the refusal again. Do **not** generate `AGENTS.md` from `Shared()` this turn. Revisit only if `Contains(Shared())` starts failing because AGENTS.md reworded a sentence the binary still has.

## Alternatives Considered

- **Generate AGENTS.md tonight — rejected: process tax ADR-037 refused.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-045-agents-md-is-not-generated-from-shared/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- A generator this turn (permanent: boundary: record only)
- Putting 040 quoting sentences in `Shared()` (permanent: boundary: 4096 / ADR-037)
- Raising `maxInstructionsChars` (permanent: boundary: ADR-037's go/no-go)
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
