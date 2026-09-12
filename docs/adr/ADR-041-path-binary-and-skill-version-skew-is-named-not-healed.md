# ADR-041: PATH binary and skill version skew is named, not healed

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-040, ADR-037, docs/adr/BACKLOG.md
**Governs:** `.claude/skills/mrw/SKILL.md`, `cmd/mrw/main.go`
**Enforced-by:** None — record only this turn; no engine change
**Invalidates:** none — checked
**Served-path change:** None — this ADR changes only the record
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes. This leftover is recorded, not executed as an engine dream.

## Context

**The class this record governs.** Every document a PATH caller uses to learn which binary they are talking to. Enumerated 2026-09-12 with

```
git ls-files .claude/skills/mrw/SKILL.md cmd/mrw/main.go AGENTS.md
```

Field PATH `mrw -v` → `dev (0b75313)` (mtime 8 Sep) while this tree and the skill were at v1.12.0. The caller trusted `--help` on the old binary.

## Existing Primitives Audit

Audited and **NOT taken** as an engine change this turn. The primitive that already exists is named in Decision.

## Decision

Record the skew. Do not auto-upgrade other machines, do not pretend an old binary grew `instructions` or `version`, and do not write a version-negotiation protocol this turn. A skill that teaches a subcommand an old PATH binary lacks is a documentation problem, not an engine one. `mrw version` (ADR-040 T3) is additive on new binaries; `-v` stays.

## Alternatives Considered

- **Install mrw as a side quest — rejected: consumer rule, not this contract.**
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

None — implementation-internal only. No contract row this turn.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-041-path-binary-and-skill-version-skew-is-named-not-healed/tasks/README.md`.

## Consequences

- **Positive:** the leftover is a numbered decision, not chat noise.
- **Negative:** the engine does not change.
- **Neutral:** a later execute starts from this record, not from BACKLOG prose.

## Out of Scope

- Installing mrw for the caller (permanent: boundary: consumer rule; if mrw is missing, say so)
- Pretending a pre-v1.10.0 binary grew `mrw instructions` (permanent: fact: `instructions` exists here since v1.10.0; citation: file `docs/adr/ADR-037-the-binary-teaches-the-format-it-demands.md:94`)
- Raising `maxInstructionsChars` so the handshake can carry a version warning (permanent: boundary: ADR-037's 4096 go/no-go)
- A parser, streamer, or extra MCP tool (permanent: boundary: not this leftover)
- Implementing an engine dream in the same turn as this record (permanent: boundary: M said record only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reads Accepted as "implement now" | Med | High | Decision says record only; T1 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Delete the record. Nothing in the engine depends on it.

## Follow-ups

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said execute 041; T1 receipts; no engine change.
