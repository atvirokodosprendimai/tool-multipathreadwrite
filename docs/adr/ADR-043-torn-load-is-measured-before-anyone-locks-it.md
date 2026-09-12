# ADR-043: Torn Load is measured before anyone locks it

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-038, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/seen/seen.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — measurement only
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** Every `seen.Load` that reads a ledger `os.WriteFile` may be rewriting. Enumerated 2026-09-12 with

```
git ls-files internal/seen/seen.go
```

ADR-038 locks `Record` and leaves `Load` unlocked and `save` as `os.WriteFile`. A reader that opens the ledger mid-write could see a torn file. Unmeasured.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

This record is a **measure**. How we would observe a tear: two processes, one `Record` looping, one `Load` looping, on a ledger large enough that `WriteFile` is not atomic on the host; a torn parse that is not `fs.ErrNotExist` is the datum. Do **not** add flock-on-Load or rename-save unless a test first shows a tear. Prefer rename-over (with a Windows answer) only after that measurement.

## Alternatives Considered

- **flock-on-Load tonight — rejected: no failing measurement.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-043-torn-load-is-measured-before-anyone-locks-it/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- flock-on-Load or rename-save this turn (permanent: boundary: measure first)
- Changing `internal/apply` (permanent: boundary: not this leftover)
- A syntax-aware engine (permanent: boundary: ADR-001 / ADR-048)
- Implementing an engine dream in the same turn as this record (permanent: boundary: M said record only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reads Accepted as "implement now" | Med | High | Decision says record only; T1 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Delete the record. Nothing in the engine depends on it.

## Measurement (2026-09-12)

Attempted on darwin: 8000-entry ledger (672013 bytes); one `Record` loop and one unlocked `Load` loop for 2s (same process — child `go test` binaries each mint their own `XDG_STATE_HOME` via `TestMain`, so a two-process recipe that does not pin that env cannot see the same file).
Result: **not observed** as a torn parse (`Load` error other than `fs.ErrNotExist`). 398 Records, 1189 Loads, 0 such errors, 6 empty Loads (`Load` treats a truncated header as stale). No flock-on-Load.

## Follow-ups

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said execute all till 050; measured; tear not observed; Load stays unlocked.
