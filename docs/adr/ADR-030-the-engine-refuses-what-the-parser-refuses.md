# ADR-030: The engine refuses what the parser refuses

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, standing direction for this session: *"keep closing the open issues, highest rank. the product must not deteriorate and make the premise of his - false."* `docs/adr/BACKLOG.md` deferred this and asked for the enumeration first; the enumeration found seven, one of which deletes lines and reports ok.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md`, `docs/adr/ADR-006-the-root-confines-reads-too-and-a-replace-must-replace-something.md`, `docs/adr/ADR-026-an-address-may-say-how-many-lines-follow.md`, `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md`
**Governs:** `internal/apply/apply.go`
**Enforced-by:** `internal/apply/apply_test.go::TestTheEngineRefusesEveryShapeTheParserRefuses`
**Invalidates:** none — checked
**Served-path change:** none — every shipped caller reaches `Apply` through `plan.Parse`, which already refuses these shapes. What changes is that `Apply`'s own doc comment stops being false.

## Context

`Apply`'s doc comment says it "validates every hunk". Measured 2026-09-07 against `d26e39a` by
driving `apply.Apply` directly, one `Input` per rule `plan.validate` enforces:

| shape | verdict | result |
|-------|---------|--------|
| `replace` with an empty body | **accepted** | lines 1-2 DELETED, `Failed: 0` |
| `insert-after` over a range `1-3` | **accepted** | inserted after line 1 |
| `insert-after` with an empty body | **accepted** | nothing changed, `ok` |
| `insert-before` with an empty body | **accepted** | nothing changed, `ok` |
| `create` with a line address | **accepted** | — |
| `create` with `anchor=` | **accepted** | — |
| `create` with `lines=` | **accepted** | — |
| `replace` with `Start: 0` | refused | — |

Seven of eight. **The first is the failure this format exists to refuse, and `validate`'s own comment
says so**: *"A replace with no body DELETES the addressed lines while reporting `ok`. A plan whose
body was lost in transit … would remove code and hand back a receipt saying it succeeded."* The
insertion rows are the premise directly: a write that changes nothing, reported as `ok`.

**This is the third and fourth time this hole has been found one field at a time.** ADR-026 closed it
for a relative end on an op that cannot honour one; ADR-027 closed it for a `create` carrying no
body; each fixed the instance in front of it. `docs/adr/BACKLOG.md` deferred the general question and
said the honest first step was ENUMERATING what `validate` checks that `Apply` does not, "rather than
assuming the list is those two". The enumeration above is that step, and the list is seven.

**Reachability, stated plainly.** No shipped caller reaches these shapes: the CLI, the MCP server and
the curve scorer all build `Input`s from `plan.Parse`, which refuses them first. So this is not a
live defect for a user of `mrw` — it is a false claim in an API this repository's own code and tests
call directly, and the record says so rather than dressing it as an exploit.

## Existing Primitives Audit

- **The boundary block in `Apply` (`internal/apply/apply.go:594-610`)** — already refuses a relative
  end on an op that cannot honour one (ADR-026) and a body-less `create` (ADR-027). **Extended in
  place**: this record adds the remaining rules to the block that exists, rather than introducing a
  second validation site.
- **`plan.validate` (`internal/plan/plan.go:592`)** — stays the parser's gate and the message author.
  **Not reused, and not importable**: `internal/apply` cannot call `internal/plan` without inverting
  the dependency the two packages are split to keep, which is ADR-027-T3's recorded Stop Condition.
- **The refusal strings** — copied verbatim from `validate` so the two sites cannot say different
  things about the same mistake.

## Decision

**Every rule `plan.validate` enforces that does not need the parse tree is asserted again at
`Apply`'s boundary, and the list is derived by enumeration rather than by memory.** The duplication
is accepted and is the point: `Apply` is a public entry point whose doc comment makes the promise, so
the promise is kept where it is made.

The two sites are kept honest by the test, which drives `Apply` directly with one `Input` per rule
and is written as a TABLE — a new rule added to `validate` with no engine counterpart shows up as a
missing row rather than as silence.

**What would falsify this:** a rule that cannot be checked without the parse tree — `patterned`, for
instance, is a parser fact. Those stay in `validate` alone, and the test names them as deliberately
absent rather than omitting them.

## Alternatives Considered

- **Make `Apply` call `plan.validate`.** Rejected: it inverts the package dependency, which is
  ADR-027-T3's Stop Condition and the reason the split exists.
- **Move validation OUT of the parser into the engine, so there is one site.** Rejected: the parser
  must refuse a malformed hunk before anything is staged, and its messages name plan-line numbers the
  engine does not have. One site would have to lose one of those.
- **Leave it, since no shipped caller can reach it.** Rejected on the doc comment: an API that says
  it validates every hunk and does not is a trap for the next caller, and this repository's own tests
  are already such callers. Fixing it one field at a time is what produced ADR-026 and ADR-027.
- **Delete the promise from the doc comment instead of keeping it.** Rejected: the promise is the
  right one. `Apply` stages files; a caller that reaches it with a shape the format forbids should be
  refused, not trusted.

## Component / Boundary Impact

None — internal to `internal/apply`'s per-hunk validation. No signature, struct, package boundary or
persisted format changes.

## Wiring & Contract Changes

None — implementation-internal only. No shipped caller's behaviour changes, because none of them can
present these shapes.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the enumerated engine-boundary refusals | T1 | — | No — every shipped caller is parsed first |

## Implementation

See `docs/adr/ADR-030-the-engine-refuses-what-the-parser-refuses/tasks/README.md`.

## Consequences

- **Positive:** `Apply`'s doc comment becomes true, and the class is closed by enumeration rather
  than by the next record finding the eighth field.
- **Positive:** the table test makes a future divergence visible: a rule added to `validate` with no
  row here is a missing row, not silence.
- **Negative:** the same rule is now written twice, and the two could drift. Accepted deliberately —
  the alternative is inverting a package dependency — and mitigated by copying the message strings
  verbatim so a drift in wording shows up in the test.
- **Neutral:** no contract row. `scripts/contract.sh` drives the built binary, which cannot reach
  `Apply` without the parser, so a row would prove the PARSER's refusal and credit it here. Said
  plainly rather than adding a row that looks like coverage.
- **One existing engine message CHANGES, and it changes to the parser's.** ADR-026 gave `create` with
  a relative end the INSERTION's wording — "takes a single line, not the range" — while
  `plan.validate` says "create takes no address, so it takes no relative end either". That divergence
  is the thing this record removes, so the engine now uses validate's message and
  `TestTheEngineRefusesARelativeEndTheOpCannotHonour` asserts per op rather than one string for
  three. Not a served-path change: the parser refuses these before any shipped caller reaches the
  engine, so no caller has ever seen the old wording.

## Out of Scope

- The rules that need the parse tree, `patterned` above all (permanent: boundary: a pattern address has no line bounds until the file is read, so a check that reasons about Start and End cannot speak about it; the test names these as deliberately absent)
- Typed error kinds shared by the two sites, so a refusal could be compared without matching message text (deferred: `docs/adr/BACKLOG.md`)
- Any change to `internal/plan` (permanent: boundary: the parser is not what is wrong here)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The enumeration is incomplete, and the record repeats the mistake it names | **High** | High | The list came from driving `Apply` with one `Input` per `validate` rule and recording the verdict, not from reading. The test carries the same table, so an eighth rule is a missing row |
| A refused shape was actually reachable, making this a live defect the record calls harmless | Low | High | All three `plan.Hunk → apply.Input` conversions were checked: each is fed by `plan.Parse`. If a fourth appears that is not, this record's reachability paragraph is wrong and must be corrected, not quietly kept |
| The duplicated messages drift | Medium | Low | Copied verbatim and asserted by substring in the test, so a change to one and not the other goes red |

## Rollback

Revert the commit; the boundary block returns to the two rules ADR-026 and ADR-027 added. No
persistent state is involved.

## Follow-ups

- [ ] If a fourth `apply.Input` construction site appears that is not fed by `plan.Parse`, revisit this record's reachability paragraph — it is what makes this a doc-comment repair rather than a live defect.
