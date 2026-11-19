# ADR-203: A line number a write moved is refused until the line is shown

**Status:** Accepted
**Accepted by:** Zy, 2026-11-19, answering "ADR-200's pre-registration says it is read once on 2026-12-18 and not built before. Keep that date?" with the option *"Build it now"*. The owner set the dated reading aside; this record does not claim the count supports the refusal. The width built here is narrower than the one the count prices, and that choice is this record's, put to Zy in the pull request.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-035, ADR-111, ADR-118, ADR-150, ADR-161, ADR-190, ADR-197, ADR-200
**Supersedes:** none. It builds the refusal ADR-200 deferred, in a narrower width, and sets aside the reading ADR-200 Decision 5 dated 2026-12-18 on the owner's word.
**Governs:** `internal/seen/seen.go`, `internal/apply/apply.go`, `internal/writer/writer.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/apply/moved203_test.go::TestANumberAWriteMovedIsRefusedUntilTheLineIsShown`
**Served-path change:** a plan hunk that names a line by number alone, in a file whose lines an earlier mrw write moved, is refused (exit 1, nothing written) unless the line was shown since, the hunk carries `anchor=`, addresses by pattern, or the write is forced. The ledger's format becomes v7.

## Context

**The hazard.** mrw licenses every line of a file it wrote (ADR-005), so a second edit needs no read. But a write that adds or removes lines moves every line below it, and a number taken from an earlier read then names another line. The second plan passes every guard and edits the wrong line. Claude Code's Edit tool has no such hazard: it addresses by text. ADR-200 counted the shape and fixed the instructions; its refusal waited for a reading on 2026-12-18.

**What the count prices, and why this is not that.** ADR-200's candidate is any numbered hunk with no `anchor=` on a file mrw wrote, at a line not shown since. E5 priced that width from transcripts at 5.06% of landings outside this repository, and 276 of its refused hunks were on files the session had just created, where every number is the one the caller typed and nothing has moved. Refusing those teaches nothing. The hazard is only below a hunk that changed how many lines the file has.

**What mrw knows at the write.** For each file a plan changes, apply knows every hunk's place and whether its body has as many lines as it replaced. The first hunk that does not is where numbering stops being what the caller knew: the lines after its body. The ledger does not keep that today.

**Audit of the class**: *what the ledger says of a file mrw wrote.* `internal/seen/seen.go`, `Observation`: `Written` (ADR-118), `Shown` (the lines a read served of this version, which `occurrence=N` needs), `Spans` (a partial licence, ADR-161, ADR-190). None says whether a line's number survived the write. `Shown` cannot be widened to mean "known by number": ADR-118 reads it as lines a read served.

## Existing Primitives Audit

- `staleMarks` (`internal/apply/apply.go`, ADR-200) already decides the candidate: op, numbered start that names a line, no `anchor=`, not quoted, not computed in the call, not forced, record written and whole, and whether an addressed line is unshown. The refusal reuses it and adds one condition.
- The ledger line of a written file is `w<shown>` or `w<shown>|<licensed>`. An mrw that reads v6 takes any other shape for a path and keeps it, which is why ADR-161 bumped the format: ADR-150's rule is that an older mrw leaves a newer ledger alone.
- `merge` (`internal/seen/seen.go`) carries `Shown` across a write that produced the recorded version.
- `writer.go` builds the observation of each written file from apply's per-file result.

## Decision

1. **A written record says where numbering moved.** `Observation` gains `Moved int`: the smallest line number, in this version, from which a line's number is not the one it had before mrw's writes since the caller last read the file whole. `0` means no line moved. For one write of a file it is the line after the body of the first hunk, in file order, whose body has a different number of lines from what it replaced (a `delete`, an insertion, a `replace` of n lines by m). A `create` moves nothing. A file whose hunks all keep their line counts moves nothing.
2. **It accumulates and is cleared by a whole read.** A later write of the same file keeps the smaller of the two non-zero values, the older one being still true where it is above the new write's first moving hunk, and replaced by the new one where it is not. A read of the whole file makes the record a read one, with no `Moved`. A partial read does not change it: the lines it shows are shown, which is what Decision 3 asks.
3. **The refusal.** A hunk that is ADR-200's candidate and unshown (`staleMarks` true, true) is refused when the record's `Moved` is not `0` and the hunk's first addressed line is at or after it. The hunk fails validation: exit 1, nothing written, siblings `skip`. The reason names the line, where the move began, and the three ways out: `line N of PATH: an earlier write moved the lines from K on, and line N has not been shown since, so the number may name another line: read it (mrw read PATH:N), address it by pattern, or give anchor=`.
4. **What is not refused.** A hunk above `Moved`; a line shown since; `anchor=`; a pattern address; a quoted hunk; a plan mrw computed in the call (`--sub`, `--rewrite`, a foreign format); `--force`; a position-only insert (ADR-200 Decision 1). Every write records `Moved`, a computed one included: it moves lines as any other does.
5. **The ledger is v7.** The written line becomes `w<shown>~K` or `w<shown>~K|<licensed>` when `Moved` is `K`, and is unchanged when it is `0`. `formatVersion` is 7 and `oldestCurrent` stays 4: v4 to v6 records are read as `Moved` 0, so an upgrade refuses nothing for writes made before it. An mrw up to v1.90.3 leaves a v7 ledger alone and says a newer mrw wrote it (ADR-150).
6. **ADR-200's count stays, and its dated reading is set aside.** `mrw stats` keeps the three counters with their meaning. Landings that this refusal stops no longer reach them, so the reading of 2026-12-18 cannot price width A any more; BACKLOG.md says so, with the owner's word. The receipt keys are unchanged (ADR-111).
7. **What would make this decision wrong.** The refusal firing where no number is stale: a tally of BACKLOG.md ("Pre-registered for ADR-203"), thirty days after the release, counts refusals with this reason against landed writes; over 2% of landed writes in a root with at least 200, it goes back to Zy. And the refusal missing the hazard: a move mrw did not make (a formatter, a `git checkout`) changes the file's sha, which the ledger already refuses; nothing here covers a tool that rewrites a file to the same bytes.

## Alternatives Considered

- **Width A, every unshown numbered hunk on a written file.** What the count measures. Rejected: it refuses the edit after a create, where the number is the caller's own, and E5 put it at about one landing in twenty.
- **Wait for 2026-12-18.** What the pre-registration says. The owner chose not to.
- **Renumber for the caller.** mrw could map an old number through the move. Rejected: it guesses which version the caller's number came from, and a wrong guess edits silently.
- **Keep it in `Shown`.** Rejected: ADR-118 needs `Shown` to mean lines a read served.
- **No format bump.** Rejected: an older mrw would read `w~12` as a path and keep the line, or write the record back without it (ADR-150, ADR-161's precedent).

## Component / Boundary Impact

Three engine packages, owned here: `internal/seen` (the field, the line's shape, the format number, `merge`), `internal/apply` (the per-file move, the refusal), `internal/writer` (the observation carries the move). `internal/plan`, `internal/read`, `internal/check`, `internal/state` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `seen.Observation.Moved`, ledger v7 | new field and line shape | T1 | apply, `mrw seen` |
| apply's per-file result | where the write moved numbering | T1 | writer |
| a refused hunk | new reason, exit 1 | T1 | every caller of `mrw write` and `mrw_write` |
| `mrw instructions`, `AGENTS.md`, `README.md` | the refusal and its ways out | T1 | every caller |
| `scripts/contract.sh` §349 | the refusal and its pairs | T1 | the contract gate |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the one way a licensed write could edit the wrong line without any guard firing is refused, where mrw can know it.
- **Negative:** a new refusal on plans that landed before; a ledger an older mrw will not read; three engine packages change.
- **Neutral:** exit codes and receipt keys are unchanged.

## Out of Scope

- Width A and a receipt row naming the shift (deferred: docs/adr/BACKLOG.md)
- Mapping an old number through the move (permanent: boundary: mrw does not guess which version a number came from)
- A move made by another tool that leaves the same bytes (permanent: boundary: the ledger knows a file by its sha)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The refusal rate is unmeasured | High | Med | narrower than the 5.06% width by construction; Decision 7's tally; `--force`, `anchor=` and a read are each one step |
| An older mrw on the same checkout stops licensing from the ledger | Med | Med | ADR-150's notice names the newer writer; the release note says to upgrade every mrw that shares a checkout |
| `Moved` computed wrong for several hunks | Med | High | a table test over one to three hunks in each order, compared with the numbers of the written file; mutants on the choice of hunk and on the line after its body |
| A partial licence record (ADR-161, ADR-190) carries a stale move | Low | Med | the field is set on every written shape and tested on both |

## Rollback

Before a release, revert T1. After one, the format number cannot go back: a v7 ledger stays unread by older mrw; the refusal would be turned off and the field kept.

## Follow-ups

- [ ] The tally of BACKLOG.md, "Pre-registered for ADR-203".
