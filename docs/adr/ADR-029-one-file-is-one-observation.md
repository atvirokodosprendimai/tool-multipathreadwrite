# ADR-029: One file is one observation, whatever the plan calls it

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, standing direction for this session: *"keep closing the open issues, highest rank. the product must not deteriorate and make the premise of his - false."* This is the premise being false in the shipped binary: a write mrw refuses under one spelling applies under another.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-005-a-write-stays-inside-what-the-caller-scoped-and-saw.md`, `docs/adr/ADR-021-a-plan-names-a-file-once-however-it-is-spelled.md`, `docs/adr/ADR-028-a-guard-does-not-read-back-what-was-not-served.md`
**Governs:** `internal/apply/apply.go`
**Enforced-by:** `internal/adversarial/ledger_test.go::TestAnAliasSpellingIsTheSameFileToThePerLineLedger`
**Invalidates:** none — checked
**Served-path change:** a hunk addressing lines through an alias spelling — an in-root symlink, or a case-only variant on a case-insensitive filesystem — is now refused by the per-line ledger exactly as the recorded spelling is. It was applied.

## Context

Found by the second Codex review of PR #128 and reproduced against that branch at `1efd1a6` on
2026-09-07:

    $ mrw --root . read 'real.txt:1'                             # line 1 only
    $ printf '@@ link.txt 4 replace\nPWNED\n' | mrw --root . write -
    ok   link.txt 4 replace  -1 +1
    wrote link.txt  5L -> 5L

`link.txt -> real.txt`. The same hunk spelled `real.txt` is refused, correctly. A case-only
`REAL.txt` behaves like the symlink on APFS and on NTFS.

**This is not a weakened guard, it is an absent one.** `internal/apply/apply.go:510` looks the ledger
up by exact key and, on a miss, recovers an aliased observation with `sameFileEntry` — issue #47's
fix, which exists because a file that HAD been read was refused as unread on a case-insensitive
filesystem. That recovery stays local to the file-level check. `:553` then looks the ledger up
**again**, by exact key, and `covered()` treats the miss as covered (`:555`), because "no observation
for this path" is the shape of a caller who read nothing and is handled one level up. So for an alias
spelling every per-line check passes, and the caller's line numbers were counted in a file they were
never shown — which is the whole of what ADR-002 refuses.

ADR-028 is the visible symptom, one level up: with the per-line gate absent, a failed `anchor=`
prints the line as it always did. That record's fix is correct and is not disturbed here; its first
Consequence carries the qualifier this defect forced, and loses it when this lands.

**What it is NOT.** `--root` still refuses first, checked the same day: an in-root symlink pointing
outside the root is refused by name on read AND on write. So nothing outside the tree mrw was pointed
at is reachable, and this is a contract violation rather than a privilege boundary — the same reading
ADR-028 and `BACKLOG.md:226` already give.

**What ADR-021 does and does not cover.** ADR-021 refuses a plan that names one file under two
spellings, because both would stage a copy and the last rename would win. It is about two spellings
in ONE PLAN. This is one spelling in the plan and a different one in the ledger, which no record has
covered.

## Existing Primitives Audit

- **`sameFileEntry` (`internal/apply/apply.go:1214`)** — asks the filesystem, via `os.SameFile`,
  whether two ledger keys name one file. **Reused as-is.** It already answers exactly the question
  `covered()` needs; it is called once and the answer is thrown away. This record moves the call, and
  adds no new identity logic — folding case, or asking what kind of filesystem this is, is what #47
  rejected and it stays rejected.
- **`covered()` (`:554`)** — the per-line gate. **Reused**, with the observation it closes over
  supplied rather than re-derived.
- **`seen.Observation` and its `Whole` / `Covers` / `Served`** — unchanged; the defect is which
  observation is consulted, never what an observation means.

## Decision

**The ledger observation for a plan's file is resolved ONCE, alias recovery included, and both the
file-level check and the per-line `covered()` consume that one value.** The second exact-key lookup
is deleted rather than duplicated: two lookups of one fact is what let them disagree, and a fix that
copies the alias recovery into `covered()` leaves the same shape for the next reader to break.

A caller who genuinely read nothing is unchanged — no observation is found under any spelling, and
the file-level check refuses first. A caller who read the file WHOLE under one spelling may still
write it under an alias, which is #47's promise and is asserted in the same test.

**What would falsify this:** a filesystem where `os.SameFile` says two entries are one file when a
write through either does not reach the same bytes. Then the ledger would license an edit to a file
the caller had not read, and identity would have to come from resolving the path rather than from
statting it. Nothing in this repository's two CI operating systems behaves that way.

## Alternatives Considered

- **Copy the `sameFileEntry` recovery into `covered()`.** Rejected: it fixes the symptom and keeps
  two lookups of one fact, which is the defect's actual shape. The next guard added beside them
  inherits the same trap.
- **Key the ledger on the resolved path at write time, so aliases never appear.** Rejected as a
  larger and riskier change: the ledger is persisted state shared with `mrw seen`, whose output names
  the paths the caller typed, and re-keying it would invalidate every existing entry on upgrade for a
  defect that a read-side resolution closes.
- **Refuse any hunk whose spelling is not the one the ledger recorded.** Rejected: it undoes #47,
  whose whole point is that a file which HAS been read must not be refused as unread because the
  caller typed a different valid name for it. The T1 test asserts this half explicitly, because a
  one-sided fixture is green against exactly this wrong fix.
- **Leave it and document it.** Rejected: it makes mrw's central promise false in the shipped binary,
  and silently — a write that changes lines the caller never saw is the failure the tool exists to
  prevent.

## Component / Boundary Impact

None — internal to `internal/apply`'s per-hunk validation. One lookup is hoisted and one deleted; no
signature, struct, package boundary or persisted format changes.

## Wiring & Contract Changes

None — implementation-internal only. What a caller sees changes only for the alias case, where the
refusal is the ledger's existing message naming the file and the served spans.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| one resolved observation consumed by `covered()` | T1 | T2 | No — a hunk spelled as the ledger recorded behaves identically |

## Implementation

See `docs/adr/ADR-029-one-file-is-one-observation/tasks/README.md`.

## Consequences

- **Positive:** the per-line half of ADR-002 holds for every spelling of a file, not only the one the
  caller happened to type. ADR-028's anchor fix reaches the alias case at the same time, without a
  second change, and its first Consequence loses the qualifier this defect forced on it.
- **Positive:** one lookup, so the two cannot drift apart again.
- **Negative:** a caller who reads under one spelling and writes under another now meets a refusal
  where a write used to apply. That is the point, and it is the refusal the recorded spelling has
  always given.
- **Neutral:** `sameFileEntry` runs on the ordinary path now rather than only on the failure path, so
  a plan whose file IS in the ledger pays one `os.Stat` of a path it already stats. Measured in T1.

## Out of Scope

- Re-keying the persisted ledger on a resolved path (permanent: boundary: it is shared state whose keys `mrw seen` prints back to the caller, and a read-side resolution closes this defect without invalidating it)
- The second ledger ENTRY a successful alias-spelled write leaves behind, so one file is recorded under two keys with different SHAs (deferred: `docs/adr/BACKLOG.md`)
- The SENT-vs-SEEN gap — that mrw records what it served rather than what the caller saw (deferred: `docs/adr/BACKLOG.md`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The fixture passes against a fix that refuses EVERY alias, undoing #47 | **High** | High | Pre-registered in `docs/adr/BACKLOG.md` before the harness exists. T1 asserts both halves in one test: the partial read refuses the alias write, and a WHOLE read still licenses it. The S2 mutant is what proves the two are distinguished |
| The fixture is not portable and is silently skipped everywhere | **High** | High | Each half runs on exactly one CI operating system — symlink on Linux, case-only where the filesystem is case-INsensitive, which is Windows here. Probe at runtime and skip; never assert the platform. §67 carries the symlink half only, `scripts/contract.sh` being Linux-only, and says so |
| Hoisting changes behaviour for a file that does not exist | Low | Medium | The file-level block is gated on `existed`; the hoisted resolution must not be, and `sameFileEntry` already returns false when `os.Stat` of the target fails. Asserted by an existing test for a `create` into an unread path |

## Rollback

Revert the commit; the second lookup returns. No persistent state is involved — the ledger is read,
not written, by either check.

## Follow-ups

- [ ] When this lands, drop the qualifier from ADR-028's first Consequence and extend §66 with the alias case, so the two records stop disagreeing about what is closed.
