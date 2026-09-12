# ADR-047: Python str is taught as a line count

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-040, ADR-027, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `internal/guide/guide.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** `write --help` and `guide.CLI()` already name that `body=` is a line count and that Python `str` splits characters — this record owns that leftover, not a new parser.
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** The help a PATH caller reads to learn what `body=` counts. Enumerated 2026-09-12 with

```
git ls-files cmd/mrw/main.go internal/guide/guide.go
```

A Python caller who uses `len(body)` as `body=` character-splits. That is the caller. ADR-027 already uses `body=0` as a line count.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Teach it on the 040 help surface. Do **not** write a Python-aware parser. `body=` stays a line count. This record exists so the leftover is not re-derived as an engine dream.

## Alternatives Considered

- **Parse Python source — rejected: caller builds the plan.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-047-python-str-is-taught-as-a-line-count/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- A Python parser (permanent: boundary: the caller builds the plan)
- Changing `body=` to characters (permanent: fact: ADR-027's body=0 is a line count; citation: file `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md:45`)
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
