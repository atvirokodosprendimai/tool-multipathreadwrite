# ADR-087: A refusal the parser and the engine share has a kind

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — chose "Do it now" for PR 7b of the approved backlog plan, having said "we can take our own decisison besides M, since we own this now"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-030, ADR-039, docs/adr/BACKLOG.md
**Governs:** `internal/refusal/refusal.go`, `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/mcp/tools.go`
**Enforced-by:** `internal/adversarial/kinds087_test.go::TestTheEngineAndTheParserRefuseWithOneKind`
**Served-path change:** none a caller sees — every refusal's text, exit code and receipt field is unchanged; `HunkResult.Kind` is `json:"-"`, and `plan.Parse`'s error prints the same bytes. The MCP acknowledgement remedy is chosen by kind instead of by matching "has not been read" in the message.

## Context

**What was deferred** (BACKLOG, "Typed error kinds shared by `internal/plan` and
`internal/apply`", from ADR-030): the engine re-checks every rule the parser refuses, and the two
are kept honest by copying `plan.validate`'s message strings verbatim and comparing them by string
equality (`TestTheEngineAndTheParserRefuseInTheSameWords`). And one production path classifies a
refusal by its text: `nameTheAck` (`internal/mcp/tools.go`) appends the acknowledgement remedy to
a hunk whose reason contains "has not been read". A reworded message would silently stop that.

**The class**, enumerated 2026-09-27 with
`mrw read --grep 'strings\.(Contains|HasPrefix)\([a-z.]*Reason' internal cmd --exclude '*_test.go'`:
that one text match. And the mirrored rules: the thirteen rows of the pairing test, seven kinds —
create with an empty body, with an address, with a relative end, with a guard; an insertion over a
range, with an empty body; a replace with an empty body.

## Existing Primitives Audit

- `authoring.Outcome` is a closed enum for the tally (ADR-009); it counts plans, not refusals, and
  its `RefusedApply` bucket deliberately does not split by reason. Kinds do not change that.
- The engine's `fail` helper records every refusal; a kinded variant sets one more field.

## Decision

1. `internal/refusal` names the seven mirrored kinds and `NotRead`. `refusal.Error` carries a
   kind and a message whose text is the message exactly.
2. `plan.validate` returns a kinded refusal for each mirrored rule, and `plan.Parse` returns a
   `*plan.ParseError` whose text is unchanged and whose `Kinds` has one entry per reported error,
   in the same order: the kind `validate` gave it, or `""` for an error nothing classifies.
3. The engine sets `HunkResult.Kind` (`json:"-"`) at each mirrored rule and at its three
   not-read refusals.
4. `nameTheAck` keys on `refusal.NotRead`, not on the message.

## Alternatives Considered

- **Type every one of the engine's refusals.** Rejected: fifty-odd `fail` sites with no second
  reader; YAGNI until a caller classifies one.
- **Put the kind in the JSON receipt.** Rejected for now: it is a public contract, and no caller has
  asked; the field stays `json:"-"` so adding it later is one tag.

## Component / Boundary Impact

`internal/refusal` (new), `internal/plan`, `internal/apply`, `internal/mcp`.

## Wiring & Contract Changes

None a caller sees. Contract §170 drives the acknowledgement remedy through `mrw mcp`.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- A refusal can be reworded at one site without breaking a classification, as long as its kind stays.

## Out of Scope

- Kinds for the unlink and rename copies in `internal/apply/pathop.go` and every other refusal (permanent: boundary: nothing classifies them; the pairing test does not cover path ops)
- A kind in the JSON receipt (deferred: docs/adr/BACKLOG.md, "A refusal kind in the JSON receipt")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller type-asserts `plan.Parse`'s error to `*fmt.wrapError` | Negligible | Low | the text is byte-identical; no in-repo caller does |

## Rollback

Revert the task.

## Follow-ups

None.
