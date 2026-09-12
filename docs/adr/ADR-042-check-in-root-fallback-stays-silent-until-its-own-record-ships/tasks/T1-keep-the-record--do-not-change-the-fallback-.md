# Task ADR-042-T1: Keep the record. Do not change the fallback.

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the record existing`

## Goal

Keep the record. Do not change the fallback.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | none — record only | the fallback stays |

## Ordered Steps

1. [S1] Confirm the Acceptance fence is RED if `**Status:** Accepted` or the record-only / measure-first phrase is deleted from this ADR. [proof: test]
2. [S2] Do not implement a parser, streamer, extra MCP tool, flock-on-Load, or LOCALAPPDATA move. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -F -q '**Status:** Accepted' docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships.md \
  && grep -E -q 'record only|measurement only|measure first' docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `Acceptance fence` | `docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships.md` | Status is Accepted and the Decision names record-only or measure-first | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | this record |
| 2 — something selects it | a later execute names this number |
| 3 — the caller can discover it | BACKLOG points here |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · b29f7a0 · mutant killed · exit 1 · `docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships.md` · without Accepted the record-only fence must fail · acceptance-sha256:d7abf5959869e668087b932367a747bb737ba37db4e4f76dc49f6b2a8177d2aa

## Invariants

- No engine change this turn.
- Do not reopen ADR-019 B/C.
- Do not touch Playtrix.

## Risks

- Reading Accepted as implement. Mitigation: Stop Condition.

## Stop Condition

Stop if the proposed work is a parser, streamer, extra MCP tool, flock-on-Load, or LOCALAPPDATA move in this turn.

## Out of Scope

- Engine change (permanent: boundary: record only)

## Verification Log
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:d7abf5959869e668087b932367a747bb737ba37db4e4f76dc49f6b2a8177d2aa · ms:36
- 2026-09-12 · b29f7a0 · exit 0 · `set -o pipefail …` · acceptance-sha256:d7abf5959869e668087b932367a747bb737ba37db4e4f76dc49f6b2a8177d2aa · ms:31
