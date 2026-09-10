# Task ADR-019-T1: Encode M's naming pick into the Decision

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** Naming pick encoded in this Decision as `Naming pick:`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the Decision names exactly one of A, B, or C`, `Status is Accepted`, `BACKLOG receipts the pick`

## Goal

Write M's Fork 2 pick into ADR-019 as a single `Naming pick:` line and quote what they said, so T2
implements a chosen Desktop rather than a guessed one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md` | edit | Status → Accepted with M's quote; insert `Naming pick: …` under Decision; if B or C, add ADR-016's reach clause to Invalidates |
| `docs/adr/BACKLOG.md` | edit | the Desktop-reach entry records the pick so adr-debt does not keep treating it as unwritten |

## Ordered Steps

1. [S1] Confirm the failing test is RED: this task's Acceptance fence exits non-zero because `Naming pick:` is absent and Status is Proposed. That is T1's TDD red — there is no Go test until T2. Then stop until M names A, B, or C in one sentence. Do not infer a pick from README's current `--root` recipe, from the coder-plan count, or from ADR-011's `roots/list` follow-up being due. [proof: human: M's sentence is quoted on the Accepted line]
2. [S2] Insert exactly one line into the Decision, matching T1's Acceptance grep: `Naming pick: A — launch --root only` or `Naming pick: B — launch allow-list` or `Naming pick: C — MCP roots/list`. Quote M beside Status: Accepted. [proof: acceptance]
3. [S3] If the pick is B or C, add ADR-016's "narrower reach is not a defect to widen" clause to Invalidates in the same edit. If A, leave Invalidates as `none — checked`. [proof: acceptance]
4. [S4] Receipt the pick on the BACKLOG Desktop-reach entry. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -qE '^Naming pick: (A — launch --root only|B — launch allow-list|C — MCP roots/list)$' \
  docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md \
  && grep -qE '^\*\*Status:\*\* Accepted' docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md \
  && grep -q 'Naming pick:' docs/adr/BACKLOG.md
```

Every clause was grepped for BEFORE this fence was written and returned **zero hits** for the
`Naming pick:` line. Status is Proposed, so the Accepted grep is red on purpose.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | The Decision names one pick; no binary behaviour until T2 | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the `Naming pick:` line the fence greps |
| 2 — something selects it | T2 refuses to start without that line (its Stop Condition) |
| 3 — the caller can discover it | the Accepted quote |
| 4 — it is used | nothing measures Desktop uptake; ADR-009 refuses transmission of caller outcomes |

## Mutation Log

- 2026-09-10 · c444fc5* · mutant killed · exit 1 · `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md` · the fence greps the exact Decision line; a guessed wording must fail · acceptance-sha256:0ec86a042eedef5450f9176c423b5bbe1ae01925692da74d80cf05a088ec612b

## Invariants

- Do not implement Serve / schema / `--root` repeatability in this task. That is T2.
- Do not raise `maxInstructionsChars`.
- Do not invent a fourth pick.

## Risks

- A session writes `Naming pick: A` because it is the current README. Mitigated: S1 is human proof;
  the fence cannot tell a guessed A from a chosen A, which is why the Accepted quote is required.

## Stop Condition

Stop and ask if M names something that is not A, B, or C — including an unconstrained per-call
`root`, a per-hunk ledger, or MCP cargo tools. Those are different records, or an override of
Decision 3 / 5 that this task must not encode quietly.

## Out of Scope

- The binary — that is T2's job
- Teaching — that is T3's job
- Measuring whether Desktop sends `roots/list` (deferred: docs/adr/BACKLOG.md — this record's Follow-ups; required before T2 if the pick is C)

## Verification Log
- 2026-09-10 · human-observed · M named A on 2026-09-10; Accepted quotes that word. Ranking quote is not the pick.
- 2026-09-10 · c444fc5* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ec86a042eedef5450f9176c423b5bbe1ae01925692da74d80cf05a088ec612b · ms:26
- 2026-09-10 · c444fc5* · exit 0 · `set -o pipefail …` · acceptance-sha256:0ec86a042eedef5450f9176c423b5bbe1ae01925692da74d80cf05a088ec612b · ms:11
