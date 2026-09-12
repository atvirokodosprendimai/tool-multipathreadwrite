# ADR-044: MCP cargo stays two tools

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-019, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/tools.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** CLI subcommands that are not `read` or `write` and are not on the MCP wire. Enumerated 2026-09-12 with

```
git ls-files internal/mcp/tools.go cmd/mrw/main.go
```

ADR-010 ships two tools because read and write are the product. `check` / `iter` / `seen` / `stats` are cargo for a Desktop folder of CSVs (ADR-019 Decision 5).

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record the parked cargo. Do **not** add extra MCP tools this turn. Two tools stand. Each of the four would need its own answer to "what did the caller see".

## Alternatives Considered

- **Add all four tonight — rejected: cargo, and 019 Decision 5 parked it.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-044-mcp-cargo-stays-two-tools/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- Extra MCP tools this turn (permanent: boundary: record only)
- Reopening ADR-019 B/C or `roots/list` (permanent: fact: Accepted Naming pick A is launch --root only; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
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

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said execute all till 050; T1 receipts; still two MCP tools.
