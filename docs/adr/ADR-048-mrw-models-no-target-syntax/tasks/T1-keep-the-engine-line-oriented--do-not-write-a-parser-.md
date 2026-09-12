# Task ADR-048-T1: Keep the engine line-oriented. Do not write a parser.

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

Keep the engine line-oriented. Do not write a parser.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | none — record only | apply stays line-oriented |

## Ordered Steps

1. [S1] Confirm the Acceptance fence is RED if `**Status:** Accepted` or the record-only / measure-first phrase is deleted from this ADR. [proof: test]
2. [S2] Do not implement a parser, streamer, extra MCP tool, flock-on-Load, or LOCALAPPDATA move. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -F -q '**Status:** Accepted' docs/adr/ADR-048-mrw-models-no-target-syntax.md \
  && grep -E -q 'record only|measurement only|measure first' docs/adr/ADR-048-mrw-models-no-target-syntax.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `Acceptance fence` | `docs/adr/ADR-048-mrw-models-no-target-syntax.md` | Status is Accepted and the Decision names record-only or measure-first | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | this record |
| 2 — something selects it | a later execute names this number |
| 3 — the caller can discover it | BACKLOG points here |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log

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
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:c24acce6e9ea379381ff6467de2b7c3dfc46b4f6b43f53b44e91f7564d374693 · ms:75
