# ADR-028: A guard does not read back what was not served

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, standing direction for this session: *"keep closing the open issues, highest rank. the product must not deteriorate and make the premise of his - false."* This one is the premise being false in the shipped binary, and it was found while researching which path to take next.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-005-a-write-stays-inside-what-the-caller-scoped-and-saw.md`, `docs/adr/ADR-008-a-delete-says-what-it-removed.md`
**Governs:** `internal/apply/apply.go`
**Enforced-by:** `internal/adversarial/ledger_test.go::TestAFailedAnchorDoesNotReadBackAnUnservedLine`
**Invalidates:** none — checked
**Served-path change:** a failed `anchor=` on a line the caller was never served now reports the ledger refusal instead of quoting the line's text.

## Context

Reproduced against the v1.4.0 binary (`bd73ee0`) on 2026-09-07:

    $ mrw --root . read 'f.txt:1'          # line 1 only
    $ printf '@@ f.txt 2 replace anchor="zzz"\nX\n' | mrw --root . write -
    FAIL f.txt 2 replace (plan line 1): anchor "zzz" not in line 2: SECRET-VALUE-42

Line 2 was never served. The refusal quotes it.

**ADR-002 and ADR-005 say mrw does not tell you what it has not shown you**, and `anchor=` is the
guard that does. It reaches the file through TWO call sites, which is why the first cut of this
record fixed one and shipped the other: `replace` and `delete` compare the anchor inline, above
`covered()`; the two INSERTION ops call a guard closure BESIDE `covered()`, so moving the first
did nothing for them. All four print the line on a failed anchor. It is repeatable per address, so
what it can leak is bounded by `clip`'s 60 characters per failed hunk rather than by one line.

**This is not a new discovery and the record should not pretend otherwise.** It is
`docs/adr/BACKLOG.md:226`, reproduced 2026-09-01 during the PR #11 re-review, deliberately left, and
that entry already names the fix and the trap. Two corrections it records are worth repeating
because they are what make this a defect rather than a judgement call:

- *"the anchor is the caller's own text"* does not hold for the half that matters — the anchor is
  theirs, the PRINTED LINE is the file's, and that is the part never served.
- The blast radius is smaller than it looks: `--root` still refuses first, so nothing outside the
  tree mrw was pointed at is reachable, and the caller could read the file directly anyway. **So this
  is a contract violation, not a privilege boundary.** It is worth fixing because the contract is the
  product, not because a secret is at risk.

**ADR-008 already moved its own guard for exactly this reason.** The expected-removal comparison on
`delete` sits BELOW `covered()` with a comment saying why — *"A guard must not become the one thing
that reads a file back to someone who never read it"* (`apply.go:811`). Its sibling one line up was
noticed at the same time and left. This record closes the asymmetry rather than letting it read as
an oversight.

## Existing Primitives Audit

- **`covered()` (`internal/apply/apply.go:801`)** — the per-line ledger check, already the gate every
  other content-revealing guard sits behind. **Reused as-is**: this record moves a check below it and
  adds nothing.
- **ADR-008's ordering (`apply.go:804-815`)** — the precedent, the comment that states the principle,
  and a test that pins it. **Copied in shape**: the same ordering, the same reason.
- **`clip` (60 characters)** — bounds what a failed hunk prints. Unchanged, and it is why the current
  defect is a leak of bounded size rather than a file dump.

## Decision

**The anchor comparison is evaluated only after `covered()`, at BOTH call sites.** For `replace`
and `delete` the inline check moves below the ledger. For `insert-after` and `insert-before` the
guard closure is called after `covered()` rather than beside it — `||` short-circuits, so an
unserved line is refused by the ledger and the anchor is never reached. A hunk whose lines were not
served is refused in the ledger's own words and nothing of the file is printed; a hunk whose lines
WERE served is anchor-checked exactly as before, with the same message.

`lines=` keeps its position for `replace` and `delete`, above the ledger, because it reports only
arithmetic the caller supplied and reveals nothing of the file. For the insertions it moves below,
because it lives inside the same closure as the anchor comparison and the closure moves as one.

**What would falsify this:** if a caller relied on the anchor failure to discover WHY an unserved
write was refused, the ledger's own message would have to be worse than the anchor's. It is not — it
names the file, the address, and the served spans, and tells the caller to read the lines. If a real
caller is found who is worse off, the fix is a better ledger message, never a guard that reads the
file back.

## Alternatives Considered

- **Redact the line in the anchor message** (`anchor "zzz" not in line 2`, no text). Rejected: it
  keeps a guard that must remember to redact, and the anchor's value IS the quoted line when the
  caller is entitled to it. Ordering gives both properties with no conditional.
- **Leave it, as PR #11 decided.** Rejected now on the two corrections the BACKLOG itself records:
  the printed line is the file's, not the caller's, and "the caller could read it anyway" is an
  argument about privilege, which was never the claim ADR-002 makes.
- **Make `anchor=` the licence and drop the ledger check for anchored hunks.** Rejected, and worth
  naming because it is the shape a future reader will propose: an anchor matches a substring of the
  FIRST line of a range, so it proves the caller saw one line of many; and moving it above the ledger
  is precisely what creates this defect. It is the opposite of the fix.

## Component / Boundary Impact

None — internal to `internal/apply`'s per-hunk validation. One check moves below another in the same
function; no signature, no struct, no package boundary changes.

## Wiring & Contract Changes

None — implementation-internal only. The refusal a caller sees for an unserved anchored hunk changes
from the anchor message to the ledger message; both already exist and neither is a declared surface.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the anchor check sitting below `covered()` | T1 | T2 | No — a hunk whose lines were served behaves identically |

## Implementation

See `docs/adr/ADR-028-a-guard-does-not-read-back-what-was-not-served/tasks/README.md`.

## Consequences

- **Positive:** for a hunk addressed by the spelling the ledger recorded, the paths where mrw printed
  a line it had not served are gone — all four anchored ops, which is `replace`, `delete` and both
  insertions. The first cut of this record fixed the first two and claimed "every guard" while the
  insertions still leaked; the Codex review of PR #128 caught it. They reach the anchor through a
  guard closure invoked BESIDE `covered()` rather than after it, which is why moving one check did
  not move theirs, and why the test and §66 now drive all four.
- ⚠ **That qualifier is load-bearing, and it was added after the SECOND review.** An ALIAS spelling —
  an in-root symlink, or a case-only variant on a case-insensitive filesystem — misses `covered()`'s
  exact-key ledger lookup entirely, is therefore treated as covered, and the anchor quotes the line
  as before. Measured 2026-09-07 against this branch at `1efd1a6`: with `s.txt:1` served, a hunk
  spelled `al.txt 2 replace anchor="zzz"` printed `SECRET-VALUE-42`, while the same hunk spelled
  `s.txt` got the ledger refusal. It is a ledger-IDENTITY defect rather than a guard-ordering one —
  `internal/apply/apply.go:553` looks the ledger up again by exact key, discarding the alias recovery
  `:525` had just done — and it is receipted in `docs/adr/BACKLOG.md` rather than folded in here,
  because the same gap lets an alias-spelled write to unread lines APPLY. That is a larger claim than
  this record makes, and burying it under an anchor-ordering heading is how it stays unfound.
- **Positive:** the asymmetry with ADR-008 is closed, so the two guards no longer teach opposite
  lessons three lines apart.
- **Negative:** a caller whose anchor AND ledger are both wrong now learns about the ledger first,
  and must read the lines before the anchor mismatch is reported. That is the correct order of
  problems, but it is one more round trip for that caller.
- **Neutral:** nothing changes for a hunk whose lines were served, which is every hunk in this
  repository's own tests and contract.

## Out of Scope

- Making `anchor=` mandatory, or promoting it into the licence (permanent: boundary: an anchor matches the first line of a range, so it cannot carry a per-line claim; and the ledger is what ADR-002 rests on)
- The SENT-vs-SEEN gap itself — that mrw records what it served rather than what the caller saw (deferred: `docs/adr/BACKLOG.md`)
- The alias spelling that misses the per-line ledger altogether — an in-root symlink, or a case-only variant on a case-insensitive filesystem (deferred: `docs/adr/BACKLOG.md`)
- Whether `lines=` should also move (permanent: fact: it reports only arithmetic over values the caller supplied and prints no file content; citation: file `internal/apply/apply.go:793`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The fixture passes with the ordering reversed | **High** | High | `BACKLOG.md:226` pre-registers this trap: the obvious fixture trips the whole-file gate instead. T1 mutates the ordering back and the fence must go red, and the fixture serves a NARROW range so the hunk is refused for the line rather than for the file |
| A caller depended on the anchor message to locate a drifted line | Low | Low | The ledger message names the file, the address and the served spans; the anchor message returns as soon as the lines are read |

## Rollback

Revert the commit; the two checks swap back. No persistent state is involved — the ledger is read,
not written, by either guard.

## Follow-ups

- [ ] If a caller reports the ledger refusal being less useful than the anchor one for a drifted address, improve the ledger message rather than reordering the guards again.
- [ ] When the alias gap closes, this record's first Consequence loses its qualifier and §66 gains the alias case. Until then the qualifier stays: a record claiming more than its tree does is the failure `.claude/rules/reviews.md` names first.
