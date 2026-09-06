# ADR-027: An empty file is created on purpose, or not at all

**Status:** Accepted
**Date:** 2026-09-06
**Owner:** M
**Accepted:** M, 2026-09-06, standing direction for this session: *"keep closing the open issues, highest rank. the product must not deteriorate and make the premise of his - false."* A `create` whose body went missing reports `ok` for a file with no content, which is the premise this tool is built on failing inside the tool.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-006-the-root-confines-reads-too-and-a-replace-must-replace-something.md`, `docs/adr/ADR-015-a-refusal-names-the-fix-for-the-two-mistakes-the-syntax-invites.md`, `docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md`
**Governs:** `internal/plan/plan.go`
**Enforced-by:** `internal/plan/plan_test.go::TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero`
**Invalidates:** none — checked
**Served-path change:** `@@ new.txt 0 create` with no body lines now fails that hunk and names the fix; `@@ new.txt 0 create body=0` creates the empty file as before.

## Context

Measured 2026-09-06 against the binary built from `bab2128`:

    $ printf '@@ new.txt 0 create\n' > a.plan && mrw --root . write a.plan
    ok   new.txt 0 create  -0 +0
    created new.txt  0L -> 0L  sha e3b0c442
    1 hunk(s), 1 file(s), 0 failed — applied
    EXIT=0

A hunk that carried no content created a file with no content, reported `ok`, and exited 0.

**ADR-006 already refuses the same shape one op over.** `replace` with an empty body is rejected —
*"replace with an empty body would delete N — say delete if that is what you mean, and check the
body did not go missing if it is not"* (`internal/plan/plan.go:631`). The reasoning there is not
about deletion being wrong; it is that **a body lost in transit is indistinguishable from a body
that was never written**, and the receipt cannot tell the caller which happened. A truncated
emission, an editor eating the last line, a pipe that closed early.

That reasoning applies unchanged to `create`, and the exposure is worse in one specific way: a
`create` is very often the LAST hunk of a plan, which is exactly where a truncated emission loses
its body. The result is a file that exists, is empty, and is reported as applied.

This is the failure the whole tool exists to prevent, in the tool itself: *a read that returns
nothing is visible; a write that changes nothing is not.*

**The counter-argument is real and is why this was deferred rather than fixed** (`docs/adr/BACKLOG.md:275`,
filed 2026-09-01): an empty file is a legitimate thing to create, and `touch` is not a mistake. The
entry says so, and says the format has no way to spell the deliberate case today.

It does. Measured 2026-09-06: `@@ n2.txt 0 create body=0` is already accepted and already creates
the empty file, and so is `body=0 raw=true`. `body=0` parses today and means exactly "this hunk's
body is zero lines" — it is simply indistinguishable from writing nothing, because nothing currently
asks.

**The blast radius is measured, not assumed.** Every `create` hunk in this repository carries a body:
18 of them across `internal/plan/plan_test.go`, `internal/apply/apply_test.go`, `scripts/contract.sh`
and `README.md`, checked 2026-09-06. Nothing in the tree relies on the behaviour this record removes,
so the cost falls entirely on external callers, and for them the fix is one token.

## Existing Primitives Audit

- **`body=N` (`internal/plan/plan.go:141-207`)** — the counted-body guard. It exists precisely
  because a body that goes missing is silent, and it already carries the count through parsing,
  overcount detection and the `raw=` interaction. `body=0` is a value it already accepts.
  **Reused as-is; no new grammar.**
- **The empty-`replace` refusal (`internal/plan/plan.go:631`)** — the wording and the reasoning this
  record extends to a second op. **Reshaped**: the message is parameterised rather than duplicated,
  so the two refusals cannot drift into saying different things about the same hazard.
- **ADR-015's refusal shape** — a refusal names the fix. Reused: the message names `body=0` for the
  caller who meant an empty file, and says to check for a lost body for the caller who did not.

## Decision

A `create` hunk carrying no body lines is **refused**, failing that hunk and therefore the whole
plan (ADR-001). The refusal names both readings:

    create with an empty body: say body=0 if you mean an empty file, and check the body did not go
    missing if you do not

A `create` written `body=0` applies and produces an empty file, exactly as today. The deliberate
case keeps working and gains a spelling that says it is deliberate; the accidental case stops
being reported as success.

**What would falsify this:** if callers create empty files often enough, without `body=0`, that the
refusal is mostly noise rather than mostly a caught truncation. Nothing measures that today —
ADR-009 refuses telemetry — so the honest position is that this trades a known silent failure for
an unknown volume of refusals, and the refusal is cheap to satisfy (`body=0`, one token). If the
noise shows up in the field, the fix is to revisit this record, not to relax the guard quietly.

## Alternatives Considered

- **Leave it.** The backlog's position since 2026-09-01. Rejected now because "a write that changed
  nothing reported success" is the one failure this project's own README puts at the centre of the
  design; carrying it in the tool while asserting it about other tools makes the premise false.
- **A new keyword — `create-empty`, or `empty` as a guard.** Rejected: `body=0` already exists,
  already parses, and already means it. A second spelling for a thing the format can say is how a
  grammar becomes a language.
- **Warn rather than refuse.** Rejected by ADR-003's rule applied to the plan surface: a verdict
  that prints a caveat and exits 0 is a pass. The whole point is that the receipt currently says
  `ok`.
- **Refuse only when the create is the last hunk in the plan.** The truncation case is
  disproportionately last. Rejected: it makes the rule depend on position, so the same hunk means
  two different things depending on what follows it, and a caller cannot reason about it.

## Component / Boundary Impact

None — internal to `internal/plan`'s validation. `validate` already owns "is this body meaningful
for this op"; this is one more case in the place that already answers that question.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `@@ <path> 0 create` plan grammar | a body-less `create` is refused; `body=0` is the deliberate empty file | `internal/plan.validate` | CLI callers, `mrw_write` MCP tool |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the body-less `create` refusal and its wording | T1 | T2 | Yes — a plan that creates an empty file without `body=0` stops applying. Named in Consequences and in the Rollback |


## Implementation

See `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all/tasks/README.md`.

## Consequences

- **Positive:** the one shape where mrw reported success for a hunk that carried nothing is gone.
- **Positive:** the deliberate empty file becomes legible in the plan text itself — `body=0` says
  what the author meant, where before the plan and a truncated plan were byte-identical.
- **Negative:** it is a breaking change for any caller creating empty files today. The refusal names
  the fix and the fix is one token, but a script that generates plans has to be edited.
- **Neutral:** `touch`-like behaviour is unchanged in capability, only in spelling.

## Out of Scope

- Applying the same rule to `insert-after` / `insert-before` (permanent: fact: both already refuse an empty body with "would change nothing", so `create` is the last op where a lost body is reported as success; citation: file `internal/plan/plan.go:617`)
- A general "this hunk intends nothing" marker across every op (permanent: boundary: `body=0` answers the one op where an empty body is legitimate; a cross-op marker would be a grammar for a case that does not exist)
- Requiring `body=N` on every op rather than as an opt-in guard (deferred: `docs/adr/BACKLOG.md`)
- Telemetry on how often the new refusal fires (permanent: fact: this corpus refuses telemetry; citation: file `docs/adr/ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given.md:1`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller's existing plans break | Med | Low | The refusal names `body=0`; the change is in the release notes and README; ADR-001 means the plan fails whole, so nothing is half-applied |
| The two empty-body refusals drift apart in wording | Low | Med | T1 parameterises one message rather than writing a second; §65 asserts both in one section |
| `body=0` interacts with `raw=` or the overcount check in a way not considered | Low | Med | T1's test covers `body=0` with and without `raw=`, and keeps the existing body= tests in the fence |

## Rollback

Revert the commit. No persistent state is involved: the plan format is read fresh each run, and the
read-before-modify ledger records resolved lines, not hunk bodies. A caller who adopted `body=0`
keeps working after a revert, because `body=0` was already legal before this record.

## Follow-ups

- [ ] If the refusal is reported as noise by a real caller creating empty files in bulk, revisit this record rather than relaxing the guard.
