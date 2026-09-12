# Task ADR-050-T1: Keep the XDG path. Do not move it.

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

Keep the XDG path. Do not move it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/state.go` | none — record only | XDG stays |

## Ordered Steps

1. [S1] Confirm the Acceptance fence is RED if `**Status:** Accepted` or the record-only / measure-first phrase is deleted from this ADR. [proof: test]
2. [S2] Do not implement a parser, streamer, extra MCP tool, flock-on-Load, or LOCALAPPDATA move. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -F -q '**Status:** Accepted' docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md \
  && grep -E -q 'record only|measurement only|measure first' docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `Acceptance fence` | `docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md` | Status is Accepted and the Decision names record-only or measure-first | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | this record |
| 2 — something selects it | a later execute names this number |
| 3 — the caller can discover it | BACKLOG points here |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 0c810a6 · mutant killed · exit 1 · `docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md` · without Accepted the record-only fence must fail · acceptance-sha256:6cd13861c384057ef9d5bd1da460b81679cddfa3a63beca0dc8b8bbd9db3a333

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
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6cd13861c384057ef9d5bd1da460b81679cddfa3a63beca0dc8b8bbd9db3a333 · ms:59
- 2026-09-12 · 0c810a6 · exit 0 · `set -o pipefail …` · acceptance-sha256:6cd13861c384057ef9d5bd1da460b81679cddfa3a63beca0dc8b8bbd9db3a333 · ms:18
