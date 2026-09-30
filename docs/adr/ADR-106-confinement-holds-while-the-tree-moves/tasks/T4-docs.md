# Task ADR-106-T4: what confinement holds, written down

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** README confinement sentence; ADR-075 qualified; `sha=` pinning taught; BACKLOG entry closed
**Consumes:** the identity recheck (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the docs say what confinement holds`

## Goal

The README says a write is confined while the tree moves and a target changed after validation is refused;
ADR-075's sha-guard sentence says which window each mechanism covers; a caller sharing a checkout is taught to
pin with `sha=`; the BACKLOG entry is closed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the confinement sentence (`:213-216`); `sha=` pinning |
| `docs/adr/ADR-075-one-writer-per-checkout.md` | edit | a dated qualification of `:74` and `:126` |
| `docs/adr/BACKLOG.md` | edit | "Anchoring file operations to a handle" closed |

## Ordered Steps

1. [S1] Confirm the fence RED. [proof: mutation]
2. [S2] The edits. [proof: mutation] Mutant: the BACKLOG entry left open.

## Acceptance

```bash
set -o pipefail
grep -q 'ADR-106' README.md \
  && grep -qE 'sha=.*read header|read header.*sha=' README.md \
  && grep -q 'Qualified by ADR-106' docs/adr/ADR-075-one-writer-per-checkout.md \
  && grep -q 'Anchoring file operations to a handle.*Closed\*\* by ADR-106' docs/adr/BACKLOG.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | no unit test: the deliverable is documentation, proved by the fence's greps and its killed mutant | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the README and ADR-075 text |
| 2 — something selects it | the README's Safety section |
| 3 — the caller can discover it | README |
| 4 — it is used | telemetry is refused (ADR-009) |

## Mutation Log
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `docs/adr/BACKLOG.md` · the BACKLOG entry left open · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · covers:the docs say what confinement holds

## Invariants

- No code changes.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the README already makes a claim the code does not keep.

## Out of Scope

- The other ADR-106 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · ms:33
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · ms:15
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · ms:30 · test-lock-sha256:d0a88974807379b1a850b3ee812a98f67f902bffe32b9e36b9a17cac84f8be11 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMw
  ```
  ```
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · ms:39
- 2026-09-30 · 9582cb3* · exit 0 · `set -o pipefail …` · acceptance-sha256:71535c2ca8163dc867f22f382a04b44ecfd39c8025a776f3d6cce5a8b6fe53f5 · ms:40
