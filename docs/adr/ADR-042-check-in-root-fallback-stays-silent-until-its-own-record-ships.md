# ADR-042: Check in-root fallback stays silent until its own record ships

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/check/check.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** The path where `mrw check` cannot place an in-root file and falls back to the whole-project command. Enumerated 2026-09-12 with

```
git ls-files internal/check/check.go
```

Parked since PR #15. An in-root typo can PASS the root's check. Inbox: `mrw check` never says whether the scope you asked for was honoured.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record the defect. Do not change the fallback this turn. The next execute must make a miss honoured-or-refused, not a silent whole-project PASS. ADR-003 rule 2 still owns the verdict.

## Alternatives Considered

- **Refuse every unplaceable in-root path tonight — rejected: needs its own execute, not a drive-by in 040.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- Changing `internal/check` this turn (permanent: boundary: record only)
- Scope derivation for languages other than Go (deferred: docs/adr/BACKLOG.md)
- Playtrix or another wing (external: wing_playtrix: wing_playtrix/inbox)
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
