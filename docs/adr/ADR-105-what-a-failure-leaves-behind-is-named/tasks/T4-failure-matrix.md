# Task ADR-105-T4: the failure matrix; the BACKLOG entries closed

**Depends-on:** T1, T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** README "What a failure leaves on disk"; AGENTS `left_behind`; three BACKLOG entries closed
**Consumes:** `Result.LeftBehind` (T1), the fsync outcome (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every stage has a row`, `the BACKLOG entries are closed`

## Goal

The README says, for each stage a write can fail at, what is on disk, what the receipt says, the exit code and
how to recover; AGENTS names `left_behind` beside `dirs_created`; the three BACKLOG entries this record answers
are closed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the matrix; `left_behind` beside `dirs_created` (`:199`) |
| `AGENTS.md` | edit | `left_behind` beside `dirs_created` (`:438`) |
| `docs/adr/BACKLOG.md` | edit | "A `.mrw-aside-*` left behind", "A probe left behind", "The cleanup errors mrw ignores" closed |

## Ordered Steps

1. [S1] Confirm the fence RED: none of the clauses holds before the edit. [proof: mutation]
2. [S2] The matrix, with a row each for validation, staging, commit, undo, ledger, check, process death and power loss; `left_behind` in README and AGENTS; the three entries closed with the commit. [proof: mutation] Mutant: the power-loss row removed.

## Acceptance

```bash
set -o pipefail
m=$(sed -n '/^## What a failure leaves on disk/,/^## /p' README.md) \
  && [ -n "$m" ] \
  && for s in validation staging commit undo ledger check 'process death' 'power loss'; do grep -qi "^| $s " <<<"$m" || exit 1; done \
  && grep -q 'left_behind' <<<"$m" \
  && grep -q 'left_behind' AGENTS.md \
  && [ "$(grep -c 'Closed\*\* by ADR-105' docs/adr/BACKLOG.md)" -ge 3 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| none | — | the fence reads the documents themselves | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the README section |
| 2 — something selects it | the README's table of contents links it |
| 3 — the caller can discover it | README and AGENTS |
| 4 — it is used | telemetry is refused (ADR-009) |

## Mutation Log

## Invariants

- No code changes.

## Risks

- A row drifts from the code: each names the receipt field or message it describes, so a reader can check it.

## Stop Condition

Stop and ask if a stage's behaviour is not what the code does.

## Out of Scope

- The other ADR-105 tasks, each in its own file.

## Verification Log
