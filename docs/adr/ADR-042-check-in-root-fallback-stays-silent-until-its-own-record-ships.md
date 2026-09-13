# ADR-042: Check in-root fallback stays silent until its own record ships

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"* (T1, record only); *"accepted, close"* (this execute)
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-040, docs/adr/BACKLOG.md
**Governs:** `internal/check/check.go`
**Enforced-by:** `internal/check/check_test.go::TestAnInRootMissIsRefusedNotASilentPass`
**Invalidates:** none — checked
**Served-path change:** `mrw check` on a missing in-root path exits 2 and names the path; it no longer PASSes the whole project
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes (T1 was record only). Same day: *"accepted, close"* after 042 was named as the only armed leftover.

## Context

**The class this record governs.** The path where `mrw check` cannot place an in-root file and falls back to the whole-project command. Enumerated 2026-09-12 with

```
git ls-files internal/check/check.go
```

Parked since PR #15. An in-root typo can PASS the root's check. Inbox: `mrw check` never says whether the scope you asked for was honoured.

## Existing Primitives Audit

Audited at T1 and **not taken** then. The primitive is `confine`: it already refuses what cannot be honoured. This execute extends it to a miss.

## Decision

An in-root miss is refused, not a silent whole-project PASS. `confine` rejects a path that is not there (`os.IsNotExist` after `rooted.Resolve`) the same way it already rejects an unreadable directory or a path outside the root. The process exits 2. Nothing ran. No result document. ADR-003 rule 2 still owns a verdict that *did* run.

T1 was record only. This execute closes the fallback for a miss. Honouring it (report `Scoped` and still run the full check) was the other allowed reading; both were allowed, so refuse. A directory of prose or `testdata` still falls back — those paths are there.

## Alternatives Considered

- **Refuse every unplaceable in-root path tonight — rejected at T1: needs its own execute, not a drive-by in 040.** Taken now, for a miss only.
- **Honour the miss: keep the fallback and report `Result.Scoped`.** Rejected: both were allowed; M's close quote plus the execute brief pick refuse.
- **Fold this into ADR-040.** Rejected: 040 owns help / parse / version. This leftover is a different question.

## Component / Boundary Impact

None — internal to the named `Governs` files; no ownership change.

## Wiring & Contract Changes

§86 — a miss exits 2 and emits no result; a real package still scopes. §15/§15c stop asserting that a typo falls back.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships/tasks/README.md`.

## Consequences

- **Positive:** a typo in a check scope is visible (exit 2), not a green on the whole tree.
- **Negative:** callers who relied on the silent fallback must name a path that exists, or run the whole-project check.
- **Neutral:** T1 remains the pin that the leftover was recorded before it was closed.

## Out of Scope

- Honouring a miss by reporting `Scoped` (permanent: boundary: refuse when both were allowed)
- Scope derivation for languages other than Go (deferred: docs/adr/BACKLOG.md)
- Playtrix or another wing (external: wing_playtrix: wing_playtrix/inbox)
- T1's record-only pin (permanent: boundary: T1 was the record-only pin)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later turn reopens honour-by-reporting-Scoped | Low | Med | Decision names refuse; T2 Stop Condition |
| Folding this back into 040 | Low | Med | Separate number |

## Rollback

Revert T2. T1's record-only receipts stay. The previous silent fallback returns.

## Follow-ups

- [x] Execute only on a later quote that names this number. — 2026-09-12: M said *"accepted, close"*; T2 refuses a miss.
