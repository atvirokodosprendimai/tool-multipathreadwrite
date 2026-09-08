# ADR-036: A paired pattern means one thing on both paths

**Status:** Accepted
**Date:** 2026-09-08
**Owner:** M
**Accepted:** M, 2026-09-08, *"NO divergence"* — choosing to remove the split rather than record it, and choosing the direction: align `read` to `write`, keeping `read`'s every-start behaviour.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-035-a-multi-line-replace-declares-what-it-replaces.md`, `docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-026-an-address-may-say-how-many-lines-follow.md`
**Governs:** `internal/read/read.go`, `scripts/contract.sh`
**Enforced-by:** `internal/read/read_test.go::TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF`
**Served-path change:** `mrw read f.go:/a/,/b/` now ends at the first `/b/` **at or after** the start, so an end on the start line yields one line; and when no `/b/` follows the start it is REPORTED as unreadable rather than silently served to the end of the file.
**Invalidates:** none — checked

## Context

Found by the eighth Codex review round of ADR-035, while checking whether a
documentation correction about the end pattern held on the read path too. It did
not. `/from/,/to/` meant three different things depending on which path resolved
it — measured against the resolvers, not inferred:

| case | `mrw read f.go:/a/,/b/` | `@@ f.go /a/,/b/ replace` |
|---|---|---|
| the start matches twice | serves BOTH spans (`read.go:523`) | refuses as ambiguous (`apply.go:728`) |
| the end matches on the START line | looks strictly after it (`read.go:529`), so the span runs on | accepts; the span is one line |
| no end matches after the start | extends silently to EOF (`read.go:527`) | refuses |

The third is the one that matters, and it is this project's own headline failure
wearing a different hat: **a read that quietly served more than the address named
is exactly as invisible as a write that quietly changed less.** A caller who
takes line numbers from such a read is holding numbers for a span mrw never
agreed to, and ADR-035 exists because those numbers then reach a plan.

`README.md` also stated flatly that "the same address means the same thing to
`read` and to `write`". For a paired pattern that was false.

## Existing Primitives Audit

`internal/read` already has the mechanism this needs: `missed`, the list that
prints `!! no match for <spec>` and makes the run exit 1 (`read.go:449`). A range that
resolves to nothing is already reported by name rather than served as an empty
success. Nothing new is required to refuse; only the decision to use it here.

`internal/apply`'s `resolve` (`apply.go:726`) is the write side and is not
touched by this record. Its exactly-once rule on the START answers "which site
did you mean", which a plan must know and a read need not.

## Decision

**A paired pattern resolves the same way on both paths, except where the two
paths differ in purpose.** Concretely, in `internal/read`:

1. The end is the first match **at or after** the start — `j >= i`, not `j > i`.
   An end on the start line gives a one-line span, exactly as a write does.
2. When no end matches at or after the start, the range is **reported in
   `missed`** rather than extended to the end of the file, and the report says
   which half failed so the caller does not re-read the file to find out.

**The every-start behaviour stays.** `read` continues to serve a span for every
match of the start THAT IS NOT ALREADY INSIDE A SPAN IT SERVED — the resolver
advances past each span it emits, so a start nested in the previous one is
skipped — while `write` continues to refuse unless the start matches
exactly once. That is not a divergence to remove: the exactly-once rule exists to
answer WHICH SITE a caller means, and a plan must resolve to one site while a
read is exploratory by construction. Making `read` exactly-once would refuse
`mrw read f.go:/func /,/^}/` on any file with two functions — the reading it is
most useful for. The two paths agree on what a span IS; they differ on how many
spans an exploratory read may return, which is a difference in purpose rather
than in grammar.

## Alternatives Considered

- **Align `write` to `read`.** Rejected, and it is the direction that looks
  symmetrical: take the first start match and extend to EOF when no end follows.
  It hands a plan an ambiguous site and an unbounded span it never named, which
  ADR-001 and ADR-002 exist to refuse.
- **Align `read` fully, including the start.** Rejected on cost: it breaks every
  exploratory read whose start matches more than once, which is the ordinary
  case for `/func /,/^}/`, and buys nothing the caller cannot get by narrowing.
- **Document the divergence and leave it.** This is what the eighth review round
  received and what the owner rejected. A documented trap is still a trap, and
  the silent-EOF case cannot be made safe by describing it.

## Wiring & Contract Changes

- `internal/read`'s paired-pattern branch changes; no exported identifier moves
  and no on-disk format changes.
- `README.md` states that a paired pattern now means one thing on both paths, and
  the round-8 paragraph that documented the divergence is withdrawn.
- Contract §74 drives both new behaviours through the built binary.

## Consequences

A read that relied on the silent extension to EOF now exits 1 and names the
range. That is the intended break: it was serving a span the address did not
describe. `mrw read f.go:/a/,$` is the way to ask for "from here to the end", and
it says so.

An end pattern that matches the start line now closes the span there. A caller
who meant "the next one after this" writes a pattern that does not match the
start line — the same thing they already do on the write path.

## Out of Scope

- The every-start difference, which this record keeps deliberately (permanent: boundary: a plan must resolve to one site and an exploratory read must not be forced to — ADR-001)
- Any change to `internal/apply`'s resolver (permanent: boundary: the write side is correct and ADR-035's tasks pin it byte-identical)

## Risks

- **A caller depended on the EOF extension.** Bounded: it exits 1 and names the
  range, so the failure is loud and the fix is one character (`,$`).
- **The MCP read path shares this resolver**, and is exercised by the existing
  suite and by contract rows that predate this record. `--grep` does NOT: it
  builds single-pattern ranges only (`internal/read/walk.go:219` sets no
  `ReEnd`), so it takes the branch below this one and is untouched. Checked
  rather than assumed — the first cut of this record claimed `--grep` was
  affected.

## Rollback

Restore `j > i` and the `end := total` default in `internal/read`, and remove
§74. No state persists across the change.

## Follow-ups

- If a caller reports the exactly-once asymmetry as surprising, the answer is a
  clearer refusal message on the write side, not a change of behaviour here.
