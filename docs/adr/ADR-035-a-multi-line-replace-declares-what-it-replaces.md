# ADR-035: A multi-line replace declares what it replaces

**Status:** Accepted
**Date:** 2026-09-08
**Owner:** M
**Accepted:** M, 2026-09-08, *"Okay that is serious. Do it"*, then *"Proceed with the suggested"* — the second confirming the mechanism after the first proposal (mandatory `lines=`) was withdrawn as unsupported by its own evidence.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-008-a-delete-says-what-it-removed.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md`, `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md`, `docs/adr/ADR-026-an-address-may-say-how-many-lines-follow.md`, `docs/adr/ADR-028-a-guard-does-not-read-back-what-was-not-served.md`
**Governs:** `internal/apply/apply.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/apply_test.go::TestAMultiLineReplaceWithoutAnAnchorIsRefused`
**Invalidates:** none — checked
**Served-path change:** A plan whose `replace` addresses more than one line is refused unless it carries `anchor=`. Callers writing plans by hand pay one clause per multi-line replace; callers generating them from `mrw read` output can extract the anchor from the `NNN| content` form mechanically.

## Context

mrw is line-oriented and models no target syntax. A `replace` puts the lines it
was given where it was told, exactly. Every guard it has is about the *address*
being reachable and the file being the one the caller read; none is about the
address being the one the caller MEANT.

Two failures of that kind are measured, both `replace`, both multi-line:

1. **A short address.** `3104-3108` written where `3088-3108` was meant, leaving
   sixteen dangling lines above the new body. Recorded in `AGENTS.md` among the
   2026-09-06 measurements across four sessions and five stacks.
2. **A stale address.** 2026-09-08, in this repository, editing this very
   documentation: `internal/mcp/instructions.go` was addressed at `96-97` and
   `103` from a read whose output had been piped to `/dev/null`. Line 95 was
   duplicated and line 102 destroyed. The file was unmodified between the read
   and the write; only the caller's belief about which line held what was wrong.

`AGENTS.md` already carries the mitigation for a third shape — read on past the
replaced range until the enclosing structure closes, which is how a surviving
`@endif` or `</div>` below the body is caught. That is guidance. It is followed
by a caller who remembers to follow it, and the sessions that met the failure in
the field reported the condition was invisible to them at the time.

ADR-008 deferred the mandatory-guard question and pre-registered the condition
for reopening it, in `docs/adr/BACKLOG.md`:

> Requiring `lines=` or `anchor=` was the runner-up and is genuinely close: it
> taxes every correct two-line delete to catch the rare wrong one. Revisit once
> the receipt bounds have been in use — if a wrong range still reaches a build
> after ADR-008, the tax is worth paying.

Two wrong ranges have now reached a build. The condition is met. ADR-027's
deferral adds the bar for taking it up: *"a record and a measured reason, not a
symmetry argument."* The two incidents above are the measured reason, and they
are what bounds this record's scope.

## Existing Primitives Audit

Three guards already exist on a hunk, all opt-in. Each was checked against the
two measured incidents before a new one was considered:

| Guard | What it asserts | Incident 1 (short address) | Incident 2 (stale address) |
|---|---|---|---|
| `lines=N` (`apply.go:885`) | `end-start+1 == N` — the address against its own arithmetic | catches | **misses** — `96-97` does span 2 lines and `103` does span 1 |
| `sha=` (`apply.go:589`) | the whole file is the bytes the caller expects | catches | **misses** — the file was unchanged; only the belief was wrong |
| `anchor=` (`apply.go:899`) | `orig[start-1]` contains the declared text | catches | catches |

`anchor=` is the only one that speaks about the CONTENT AT THE ADDRESS, which is
the thing both incidents got wrong. `lines=` and `sha=` each cover one incident
and are dominated by it for this failure class; both remain available and remain
orthogonal — `sha=` in particular still answers a question `anchor=` cannot,
namely whether the file moved under the caller.

The first proposal for this record was mandatory `lines=`. It was withdrawn on
the finding in the table above: it would not have caught the incident cited as
its own evidence.

## Decision

**A `replace` whose address resolves to more than one line must carry
`anchor=`.** Without one the hunk fails, and by ADR-001 the whole plan writes
nothing.

A single-line `replace` is unchanged, and so is every other op.

Three scoping choices, each made against a specific alternative:

**Enforced in `internal/apply`, not in `plan.validate`.** `validate` runs at
parse time, where a pattern address (`/from/,/to/`) and `$` have no resolved
span — the existing `patterned` escape at `plan.go:599` says exactly this. A
guard there would therefore fire on `3-6` and not on `/a/,/b/`: a requirement
conditional on address form, which is the hole ADR-026 and ADR-027 each found
one field at a time. `apply` sees every address form after resolution, and it is
the single writer, so a direct `apply.Apply` caller is covered too.

**`replace` only; `delete` is left as ADR-008 has it.** Every measured incident
is a replace, and the difference is not cosmetic: a wrong `replace` writes new
content over lines nobody inspected, while a wrong `delete` removes lines and
leaves an absence. `delete` also already has the stronger opt-in guard — an
expected body, which declares every line rather than the first. Extending this
requirement to `delete` on the grounds that it is the same shape would be the
symmetry argument ADR-027's deferral names. ADR-008's BACKLOG entry stays
open, with its criterion recorded as still unmet for `delete`.

**`anchor=` only; `lines=` and `sha=` do not satisfy it.** Accepting a guard
that misses one of the two incidents would make the promise weaker than its
message claims.

## Alternatives Considered

- **Mandatory `lines=`.** The original proposal. Withdrawn: see the audit table.
- **Mandatory `anchor=` OR `sha=`.** Rejected. `sha=` misses incident 2, so the
  disjunction is only as strong as its weakest arm and a caller satisfying it
  with `sha=` gets a refusal message promising a guard that did not run.
- **Requiring the replaced body to be declared, as `delete` allows.** Strictly
  stronger, and rejected on cost: a replace's body is the NEW content, so this
  would mean carrying both old and new text in every multi-line hunk, roughly
  doubling a plan's size for a guard whose first line already catches both
  measured shapes.
- **Making it a warning rather than a refusal.** Rejected under ADR-003's
  reasoning: a verdict that does not change the exit code is not a verdict, and
  a caller generating plans mechanically never reads it.
- **Leaving it as the `AGENTS.md` guidance.** This is the status quo, and it is
  what both incidents happened under. Incident 2 was committed by a session
  editing the documentation of that very rule.

## Wiring & Contract Changes

- `internal/apply` gains one refusal on the resolved-range path. No exported identifier moves, no
  on-disk format changes, and no exit code changes — the hunk fails, which ADR-001 already maps to
  exit 1 with nothing written.
- `README.md`, `AGENTS.md` and the MCP `plan` tool description state the requirement where the plan
  grammar introduces `sha=`, `lines=` and `anchor=`.
- Contract §73 drives both spellings — refused without, applied with — through the built binary.

## Consequences

Every multi-line `replace` now costs one `anchor=` clause. That is the tax
ADR-008 named, now paid on the op where the evidence is.

The guard is honest about only one thing, and the record has to say it: **a
guard that restates the same wrong belief catches nothing.** An `anchor=` typed
from memory can be wrong in the same way the address is wrong. It is strong when
the text is extracted from `mrw read` output — the `NNN| content` form makes that
mechanical for a generated plan — and weak when hand-typed. mrw cannot tell which
it received, and this record does not claim otherwise.

It also does not cover the duplicated-closer shape at all: a body that re-emits
the structure's closing token spans the range it says it does and begins on the
line it says it does, so the anchor matches and the guard passes. That shape
remains covered only by the read-past-the-closer rule in `AGENTS.md`.

## Out of Scope

- Requiring a guard on a multi-line `delete` — ADR-008's criterion is met for `replace` and not for
  `delete`, and this record does not extend it on symmetry (deferred: `docs/adr/BACKLOG.md`)
- Any syntax or structure awareness in mrw, which is what would cover the duplicated-closer shape
  (permanent: boundary: mrw is line-oriented and models no target syntax by design — ADR-001)
- Deriving an anchor automatically from the ledger's served lines, so a generated plan need not carry
  one (deferred: `docs/adr/BACKLOG.md`)

## Risks

- **The tax falls on correct plans.** Every multi-line replace pays. Bounded by
  scoping to `replace`: the survey at the time of writing found 19 in-tree
  multi-line hunks without an anchor, of which 5 are replaces and 3 of those are
  fixtures already refused at parse time for another reason.
- **A caller routes around it with single-line hunks.** Replacing 20 lines as 20
  single-line hunks is legal and unguarded. Not mitigated: it is more work than
  writing the anchor, and a caller doing it has read the lines.

## Rollback

Delete the guard in `internal/apply/apply.go` and its contract row §73. Nothing
persists it; no on-disk format changes.

## Follow-ups

- If a wrong multi-line `delete` range reaches a build, ADR-008's criterion is
  met for `delete` too and this requirement extends to it.
