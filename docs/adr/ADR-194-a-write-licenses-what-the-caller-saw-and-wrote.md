# ADR-194: a write licenses the lines the caller saw and the lines it wrote, not the whole file

**Status:** Accepted
**Accepted:** 2026-11-04 by Zy — answered *"acept if this is beneficial"* to the two drafts, then *"go"*. "Beneficial" is T1's pre-registered verdict: T2, the code, starts only on GO (or on an override Zy gives and `replay.json` records).
**Date:** 2026-11-04
**Amended:** 2026-11-04, before acceptance, from an adr-challenger pass and one Codex xhigh challenge (a first run over both records was killed at its 600 s budget and returned nothing) (each finding checked against the code): the carry is computed in apply from the observation the plan was judged against, so an alias is carried and a rename is not a create; shown lines are not carried, so `from=` and `occurrence=` keep today's rule; the path-op refusal a partial licence brings is named; the refusal names no source of the licence (ADR-182); the replay's licence sources, uncertainty and verdict are pre-registered in full; one finding did not hold and is answered in Decision 6.
**Amended again:** 2026-11-04, before acceptance, from a second Codex xhigh challenge of the amended record (5 findings, each checked against the code): ADR-052's line after still counts against a forced hunk in the replay; a piped or clipped read makes its file UNCERTAIN; a quoted-only write's body is no longer copyable by `from=`, named in Decision 6; T2's fence allows `seen.go`'s comment edits; mutant-killing cases for uncertainty, a shrinking replace and `insert-before`.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-052, ADR-071, ADR-102, ADR-118, ADR-141, ADR-161, ADR-167, ADR-175, ADR-182, ADR-190, ADR-191, docs/adr/BACKLOG.md
**Supersedes:** ADR-005, in one clause of Decision 4: "a write observes everything, because mrw produced every line", for a file that was not wholly licensed before the write
**Governs:** `internal/writer/writer.go`, `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`
**Enforced-by:** `internal/writer/licence194_test.go::TestANativeWriteLicensesOnlyTheLinesTheCallerSawAndWrote`
**Invalidates:** ADR-071 — T5's Out of Scope item `(permanent: boundary: M, 2026-09-25, kept ADR-002)` and its sentence "A file mrw just wrote is wholly known" (`internal/guide/guide_test.go::TestInstructionsSayAWrittenFileIsWhollyKnown`); ADR-141's core rule 2 clause "except in a file mrw just wrote" (`internal/guide/core141_test.go`); and ADR-002's row "edit again straight after | applies", for a file not read whole. The texts change in T2, and those records' acceptance digests move with them.
**Served-path change:** after a write to a file that was only partly licensed, a later write is licensed for the lines the caller was licensed before (carried into the new line numbers) and the lines mrw wrote, and refused elsewhere with a reason saying a write does not license the rest of the file; unlink and rename of that file then need it read whole, as before any write; a file the plan created, or one wholly licensed before the write, stays wholly licensed as now.

## Context

ADR-002 licenses a line once mrw has served it. ADR-005 Decision 4 narrowed the ledger to what was shown, and added that "a write observes everything, because mrw produced every line". mrw did produce every line of the version it wrote, but the caller knows only the lines it was licensed for plus the ones it wrote.

Issue #602 (found in the Codex re-review of #600, 2026-11-02), reproduced on v1.84.0 on 2026-11-04 with a five-line file:

```sh
mrw read f:3                                        # serves lines 3-4 (ADR-161's line after)
printf '@@ f 3 replace\nTHREE\n' | mrw write -     # exit 0; mrw seen: "licensed the whole file"
printf '@@ f 5 replace\nFIVE\n'  | mrw write -     # exit 0: line 5 was never served
printf '@@ f 1 replace\nONE\n'   | mrw write -     # exit 0: nor was line 1
```

Without the first write, the same `@@ f 5 replace` is refused: `5 of f has not been read: mrw served lines 3-4`. A write to one line widens the licence from two lines to the whole file.

**The rule was kept once.** On 2026-09-25 M kept this design when two sessions of the v1.25.1 round read "per line" as broken (`internal/guide/guide_test.go:160-168`; ADR-071 T5, whose Out of Scope files it as a permanent boundary). The served text was changed to say so. Since then three records reach a partly licensed file with no `mrw read` at all: ADR-190 (an exact SEARCH quote), ADR-191 (the host's own Read) and ADR-175 (the ledger forgotten after a compaction, after which a quote or a host Read is the only licence). ADR-190's Consequences named the inherited hole and pointed at #602. On 2026-11-04 Zy asked for this record ("Draft ADR-194", to the question that cited the 2026-09-25 decision); keeping or reversing that boundary is decided at acceptance.

**Audit of the class**: *every site that marks a file as written, or records a written file in the ledger*. Enumerated with `rg -n 'Written:\s*true|\.Written\b.*=' internal cmd --glob '!*_test.go'` on 2026-11-04: 20 hits.
- Seven are in `internal/seen` (`merge` and the ledger parser), which combine observations and record nothing of their own.
- Three record a partial licence for a plan maker or a `--quiet` read, before any write: `internal/read/read.go:990` (`--sub`, ADR-161), `internal/read/rewrite.go:208` (`--rewrite`), `internal/mcp/licensed167.go:13` (`LicensedNotShown`: what a `--quiet` read licenses without showing, ADR-167).
- Seven in `internal/apply` set or clear `FileResult.Written` as a commit lands or is undone (`apply.go:517, 1001, 1179`; `pathop.go:342, 389, 403, 411`). `pathop.go:386-392` builds a rename destination's record with `Created: true`, so "created" in this record means a `create` hunk, never that flag.
- Two read the flag for a receipt (`cmd/mrw/terse179.go:42`, `internal/curve/score.go:163`).
- One records the whole file: `internal/writer/writer.go:108`, every write that landed, CLI and MCP alike (ADR-113: one write path), each file of a partial commit included (ADR-102). That one is decided here. Its ADR-190 sibling (`writer.go:106`, a file only quotes licensed) already records only the written lines; it becomes a case of the same rule.

## Existing Primitives Audit

- **The ledger already stores a partial licence on a written file.** ADR-161's `w<shown>|<licensed>` entry (`internal/seen/seen.go:547-560`) reads back as `Observation{Written: true, Spans: licensed, Shown: shown}`. `merge` (`seen.go:841-853`) keeps such a licence partial and never widens it, and `Served()` (`seen.go:120-133`) already says "shown X, and licensed Y". So no format change (ADR-150), and `internal/seen`'s code stays as it is.
- **Apply already knows where each body landed and which observation it judged.** `HunkResult.newSpan` (`internal/apply/apply.go:65-69`, set at `apply.go:2010-2015`) is a replace's or insert's body span in the written file; ADR-190 records exactly those spans for a quoted file (`apply.go:724-731`). Apply also resolves a file's observation under another spelling when the plan's spelling has none (the alias lookup #597 relies on). What is missing is the carry of that observation's licence into the new numbering, which apply is the one place able to compute.

## Decision

1. **A write keeps the whole-file licence only where the caller already had it.** After a write lands, mrw records the file as wholly licensed (`Written: true`, no spans, as now) when a `create` hunk of this plan made it, or when the observation apply judged the plan against was whole for the version the write replaced: a whole read, or a written record with no spans. A rename is not a create: its destination takes its source's result by these rules. A rename already needs its source licensed whole (`pathop.go:153`), so only a forced rename differs.
2. **Otherwise it records what the caller was licensed for and what mrw wrote.** Apply computes, per written file, the licence of the observation it judged (an alias's included) carried into the new numbering, plus every body span the write put in the file. A line outside every replaced or deleted range moves by the lines inserted and removed above it, by every hunk of the plan; a line inside one is dropped, and only the written body stands for it. The writer stores it as ADR-161's partial written record, with no shown lines, as a write records today.
3. **A forced write, or one whose observation describes another version, carries nothing.** Its licence is the lines it wrote; Decision 3 takes precedence over Decision 1 except for a `create` hunk, whose every line the caller wrote. Today `--force` leaves the file wholly licensed (contract §292's comment says so); it no longer does.
4. **The same rule holds for every way a write lands**: a native plan, a `--format` document, a `--sub` or `--rewrite` plan, `mrw_write`, and each file of a partial commit (ADR-102). A file ADR-190 licensed by quotes alone is the case of Decision 2, carrying the ledger's licence when it holds one for that version.
5. **The line refusal says why, without naming a source.** A hunk refused under a partial written licence keeps today's words (`N of f has not been read: mrw served …, and licensed …`) and adds that only the lines listed are licensed and a write adds the lines it wrote, not the rest of the file. It names no source of the licence, because a `--quiet` read and a `--sub` plan record the same shape (ADR-182). The guide's sentence "A file mrw just wrote is wholly known" and core rule 2's "except in a file mrw just wrote" are reworded to match.
6. **What else this changes, and what it does not.**
   - Changes: unlink and rename of a file written under Decision 2 are refused until it is read whole, with the path-op refusal's existing words (`pathop.go:153`), as before any write. A multi-line replace there needs ADR-052's line after inside the licence, as on any partly read file.
   - Changes too: a file only quotes licensed (ADR-190) is recorded today as a read of its body lines (`writer.go:106`), so `from=` may copy them; under Decision 2 it is a written record with nothing shown, so `from=` of those lines is refused, as for every other write.
   - Unchanged: shown lines for every other write, so ADR-118's `occurrence=` and ADR-167's `from=` refuse after a write exactly as today; the ledger format; `internal/seen`'s code (its comments that call a written file wholly licensed, `seen.go:57-58, 62-64`, are reworded); the ledger is still recorded before `--check` and `--then` run, so a failing check leaves the narrowed record.
   - Not a widening (a Codex finding answered): when the written version's sha equals the version an observation describes (a forced write that restores bytes the caller read whole), `merge` keeps that observation. It describes exactly the bytes on disk, which is what the ledger records.

**What would make this decision wrong, and the data that exists today.** (a) The rule refuses many writes agents make in practice, so the fix costs more re-reads than the hole is worth. T1 measures that before any code, on the Stage 8 pilot transcripts (Claude Code 2.1.317, mrw v1.83.0, both models, cells B and S), against the criterion pre-registered in `docs/adr/BACKLOG.md` (From ADR-194): GO only when the refused and the unplaceable hunks together are at most one landed hunk in 20; STOP, and T2 waits for Zy, when the refused alone are more. (b) The carry is wrong by a line and licenses a line nobody saw, the failure ADR-005 exists to prevent. T2's mutants each move the carry once.

## Alternatives Considered

- **Record only the lines written** (ADR-190's shape for every write). Simpler: no carry. Rejected: a caller who read lines 10-60 and edits line 20 would then need to read again before editing line 50, though it was shown line 50. That breaks ADR-002's chain of edits for lines the caller did see. The carry costs one function; the re-read costs every multi-site edit a call.
- **Keep the rule** (M's choice of 2026-09-25, ADR-071 T5). Decided at acceptance; this record proposes reversing it because three ways into a partly licensed file now exist that did not then.
- **Carry the prior spans without renumbering.** Rejected: after an insert or a delete the old numbers name other lines, which is the misplacement the guard prevents.
- **Whole licence when every hunk was inside the licence.** Rejected: it says nothing about lines outside it, and those are what a later write reaches.
- **Carry the shown lines too.** Rejected: after a write a whole read shows nothing (`merge` takes the new record on a new sha), so carrying shown lines only for a partial read would let a partly read file copy more than a wholly read one.

## Component / Boundary Impact

`internal/apply` (the carry, beside `newSpan`, from the observation apply judged; `pathop.go` so a rename's destination takes its source's result; an engine package, owned here) and `internal/writer` (which observation to record). `internal/guide` for two sentences. `internal/seen` changes comments only. `internal/read`, `internal/plan`, `internal/check` and `internal/state` stay byte-identical; the ledger format is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| The ledger entry after a write to a partly licensed file | ADR-161's partial written entry instead of a whole-file `w` | T2 | every later write, `mrw seen` |
| `mrw write` / `mrw_write` line refusal under a partial written licence | an added clause: only the listed lines are licensed, a write adds the lines it wrote | T2 | agents |
| unlink / rename of a file written under Decision 2 | refused until read whole, the existing words | T2 | agents |
| `mrw instructions`, `--core`, `mrw skill` | the write exception reworded | T2 | agents |
| `scripts/licence194_replay.py`, `docs/bench/licence194/` | new: the replay and its verdict | T1 | T2, Zy |
| `scripts/contract.sh` | §335 | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `docs/bench/licence194/replay.json` verdict | T1 | T2 | No — T2 starts only on GO, or on an override Zy gives and the file records |

## Implementation

`tasks/README.md`: T1, then T2. Two pull requests.

## Consequences

- **Positive:** a write no longer licenses lines nobody saw. The per-line rule holds after writes as it does before them, for every licence source, ADR-190's and ADR-191's included.
- **Negative:** an agent that edited one part of a file and then another part it never read is refused and must read it; so is an unlink or rename of a partly read file it wrote. T1 prices this.
- **Neutral:** a file read whole, or created by the plan, behaves as now, so the common read-whole-then-edit chain does not change.

## Out of Scope

- A host's own Edit and Write tools, which never reach the ledger (permanent: boundary: mrw records only what passes through it, ADR-191)
- Measuring the rule on a paid run after it ships (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The carry is off by one after an insert or a delete and licenses an unseen line | Med | High | T2's mutants move the carry once each way and after each op |
| The carry reads the plan's spelling and misses an alias's observation | Med | Med | computed in apply from the observation it judged; T2's alias case |
| Refusals rise enough that agents reach for `--force` | Low | Med | T1's pre-registered STOP; the refusal names the read to make |
| A test or contract row pins the whole-file licence and is silently rewritten | Med | Med | T2 lists each one it moves in the PR, with its record |

## Rollback

Revert T2: a write licenses the whole file again. The ledger entries T2 wrote read back as partial written records, which an older mrw already understands (ADR-161).

## Follow-ups

- [ ] The next paid benchmark counts the new refusal's clause per cell (BACKLOG.md, From ADR-194)
