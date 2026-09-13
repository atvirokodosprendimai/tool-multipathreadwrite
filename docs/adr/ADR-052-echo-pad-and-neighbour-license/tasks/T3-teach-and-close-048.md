# Task ADR-052-T3: Teach the two arms; close 048 / BACKLOG

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** neighbour license in `Apply` (T1), `HunkResult.Echo` / `--echo-pad` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `048 names 052`, `BACKLOG receipts the leftover`

## Goal

ADR-048's "Wrap-tail stays teaching" is narrowed for the license half only. BACKLOG's padded-echo leftover is receipted. README / AGENTS.md name the refuse and the opt-in flag. Shared() stays five sentences.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/adr/ADR-048-mrw-models-no-target-syntax.md` | edit | Decision names ADR-052's licence; keep `record only` so T1's fence still matches; padded-echo Out of Scope is no longer deferred. |
| `docs/adr/BACKLOG.md` | edit | Inventory + 2026-09-06 leftover point at ADR-052. |
| `README.md` | edit | Neighbour license + `--echo-pad` next to the wrap-tail habit. |
| `AGENTS.md` | edit | Same two facts on the write rules. |

## Ordered Steps

1. [S1] Amend ADR-048 and BACKLOG so a grep for `ADR-052` on those files is true, and confirm the greps are RED before the edit. [proof: mutation]
2. [S2] Teach README and AGENTS.md the refuse and `--echo-pad`. Shared() unchanged. [proof: acceptance]
3. [S3] Confirm ADR-048 still contains `record only` (its T1 fence). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q 'ADR-052' docs/adr/ADR-048-mrw-models-no-target-syntax.md \
  && grep -q 'ADR-052' docs/adr/BACKLOG.md \
  && grep -q -- '--echo-pad' README.md \
  && grep -q -- '--echo-pad' AGENTS.md \
  && grep -E -q 'record only|measurement only|measure first' docs/adr/ADR-048-mrw-models-no-target-syntax.md \
  && ! grep -q 'Wrap-tail stays teaching\.$' docs/adr/ADR-048-mrw-models-no-target-syntax.md \
  && grep -q 'mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.' internal/guide/guide.go \
  && [ -z "$(gofmt -l .)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `Acceptance fence` | `docs/adr/ADR-048-mrw-models-no-target-syntax.md` / `docs/adr/BACKLOG.md` / README / AGENTS | 048 names 052; leftover receipted; Shared() intact | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the greps |
| 2 — something selects it | a reader of 048 / BACKLOG / README |
| 3 — the caller can discover it | README / AGENTS / `write --help` (T2) |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-13 · 2f49847* · mutant survived · exit 0 · `docs/adr/ADR-048-mrw-models-no-target-syntax.md` · 048 must name ADR-052; dropping it fails the T3 grep · acceptance-sha256:53bd1cbc35cdf3742f054394b2e92b0e9c9bee70bd9ed7933ede1e6fe7f4ef90
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-13 · 2f49847* · mutant killed · exit 1 · `docs/adr/ADR-048-mrw-models-no-target-syntax.md` · restoring the teaching-only sentence must fail T3 · acceptance-sha256:53bd1cbc35cdf3742f054394b2e92b0e9c9bee70bd9ed7933ede1e6fe7f4ef90

## Invariants

- Shared() five sentences unchanged.
- 048 T1 fence still sees `record only`.
- Echo is not taught as a checker.

## Risks

- Dropping `record only` from 048 makes its own T1 fence red.

## Stop Condition

A rewrite of Shared() to mention the license.

## Out of Scope

- Engine changes (T1, T2)

## Verification Log
- 2026-09-13 · 2f49847* · exit 0 · `set -o pipefail …` · acceptance-sha256:53bd1cbc35cdf3742f054394b2e92b0e9c9bee70bd9ed7933ede1e6fe7f4ef90 · ms:45
- 2026-09-13 · 2f49847* · exit 0 · `set -o pipefail …` · acceptance-sha256:53bd1cbc35cdf3742f054394b2e92b0e9c9bee70bd9ed7933ede1e6fe7f4ef90 · ms:47
