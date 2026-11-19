# ADR-203: A line number alone is refused on a line not shown since mrw wrote the file

**Status:** Accepted
**Accepted by:** Zy, 2026-11-19. Asked "The stale-line-number refusal (ADR-200) … is read once on 2026-12-18 and not built before. Keep that date?", Zy chose *"Build it now"*. The refusal ADR-200 deferred is the one its count measures, and that is what this record builds, with no narrowing and no ledger change. A first draft narrowed it to lines an earlier write had moved, with new ledger state; two challenges before any code (Codex xhigh on gpt-6.2-sol: REJECT; adr-challenger: ACCEPT AFTER CHANGES) found that cutoff unsound, and the draft is withdrawn (History, below). The three options were put to Zy the same day; this is the first of them, the one the owner's words name. If Zy meant otherwise, this record is the thing to reverse.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-035, ADR-111, ADR-118, ADR-161, ADR-190, ADR-197, ADR-200
**Supersedes:** in part. ADR-200's Served-path sentence "No write is refused that was not refused before", its rejected alternative "Build the refusal now", its contract §343's "is not refused" rows and the gate of its Decision 5 (the reading of 2026-12-18, set aside by the owner). ADR-005 §4's "a chain of edits needs no re-read" now holds for an edit that says which line it means: by pattern, by `anchor=`, or on a line shown since the write.
**Governs:** `internal/apply/apply.go`, `internal/refusal/refusal.go`, `cmd/mrw/main.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/apply/stale203_test.go::TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite`
**Served-path change:** a plan hunk that names a line by number alone, in a file mrw wrote, on a line no read has shown since, is refused: exit 1, nothing written. It lands with `anchor=`, with a pattern address, after a read that shows the line, or under `--force`. The ledger and every receipt key are unchanged.

## Context

**The hazard.** mrw licenses every line of a file it wrote (ADR-005), so a second edit needs no read. A write that adds or removes lines moves the lines below it, and a number taken from an earlier read then names another line. The second plan passes every guard and edits the wrong line. Probed on v1.91.0, 2026-11-19: ten lines read whole, `3 insert-after`, then `8 replace` with no read between exits 0 and replaces what was line 7; `4-6 delete` removes the inserted line and two others. Claude Code's Edit tool has no such hazard: it addresses by text.

**What ADR-200 did.** It counted the shape in `mrw stats` and fixed the instructions, and deferred the refusal to a reading on 2026-12-18. The owner has set that date aside.

**What the ledger already knows.** A written file's record keeps `Shown`, the lines a read has served of this version (ADR-118). A write that changes the file replaces the record, so `Shown` starts empty after every such write (`merge` in `internal/seen/seen.go`; confirmed by probe: `mrw seen` says `no lines, and licensed the whole file` after each write). "Shown since the write" is therefore already exact, with no new state.

**What it costs, as far as it was measured.** E5's third run (2026-11-18, entries before 2026-11-18T10:00Z, without this repository) classed 339 of 6,700 landed write calls that typed a plan as refused at this width, 5.06%, with another 13.90% it could not class, and its verdict was STOP. 276 of the 620 refused hunks were on a file the session had created. That is the price the owner's decision accepts unmeasured: an edit by number alone right after a create or a same-size replace is refused too, though its number is not stale, because mrw cannot tell a number the caller computed from one it remembered.

**Audit of the class**: *the ways a plan hunk says which line it means.* `staleMarks` (`internal/apply/apply.go`) is the one place that sorts them, and its switch is the list: a number (`N`, `N-M`, `N,+L`, `N-$`) on `replace`, `delete`, `insert-after`, `insert-before` is a candidate; `$` as the start, `insert-after 0`, `insert-before 1`, `insert-before` one past the last line name a position; a pattern, `anchor=` and a verified quote say what the line holds. `create`, `unlink` and `rename` name no line.

## Existing Primitives Audit

- `staleMarks` returns, for a validated hunk, whether it is ADR-200's candidate and whether an addressed line is unshown. It already excludes a forced write, a plan mrw computed in the call, a record that is not whole and written, a quoted hunk, `anchor=`, a pattern start and the position-only inserts. The refusal is its second result.
- `unshown` judges every line of a range, so a range is refused when any line of it is unshown.
- A refusal carries a `refusal.Kind`. `NotRead` already drives two consumers (the newer-ledger wording, and the MCP surface's acknowledgement advice), neither of which fits here.
- ADR-190's quoted-only write records read-shaped spans, not a written record: a later edit there is judged by the ordinary per-line licence, as today.

## Decision

1. **The refusal.** A hunk for which `staleMarks` reports a candidate with an unshown line fails validation: exit 1, nothing written, siblings `skip`. Its kind is new, `refusal.StaleNumber`. The reason names the first addressed line that is unshown and the ways out: `line N of PATH has not been shown since mrw wrote the file, and a line number alone cannot say which line you mean: an earlier write may have moved it. Read it (mrw read PATH:N, a read that shows it), address it by pattern, or give anchor=`.
2. **What still lands**, exactly as `staleMarks` already sorts it: a hunk with `anchor=`; a pattern address; a quoted hunk; a line shown since the write (every line of a range); `$` as the start and the position-only inserts; a plan mrw computed in the call (`--sub`, `--rewrite`, a foreign format, the MCP forms); a forced write; a file whose record is not a whole written one (a file read whole and not written since, a partial licence).
3. **A create is not an exception.** A numbered edit right after `--create` is refused like any other: the ledger holds no difference between a number the caller computed and one it remembered. The instructions say to name the line by pattern or `anchor=` in a second edit, which costs one token and is what recipe 21 already teaches.
4. **No ledger change, no receipt change.** The format stays v6. `mrw_write` refuses by the same rule, with `isError` and the reason in the hunk as for any failed hunk; it has no `--force`.
5. **ADR-200's counters stay and say what they are.** `landed_writes` and `numbered_candidates` keep counting. `numbered_would_refuse` counted landings this refusal now stops, so from this release it grows only by writes made with an older mrw on the same root; `mrw stats` says that in place of the sentence about 2026-12-18. The keys keep their names, types and meaning (ADR-111).
6. **What would make this decision wrong.** The refusal costing more than it saves: BACKLOG.md ("Pre-registered for ADR-203") names a friction reading thirty days after the release. It measures how often the refusal fires, not how often a refused number was in fact stale, which no transcript can show.

## Alternatives Considered

- **A selective refusal, only below a line an earlier write moved.** The first draft. Withdrawn: it needs new ledger state (a format bump that locks an older mrw out of the checkout), a cutoff that two challenges showed wrong for inserts and growing replaces, history carried through renames and aliases, and it still over-refuses after cancelling hunks.
- **Exempt a file mrw created.** Rejected: the ledger does not record that, a later write to the file would have to clear it, and it is the same new state by another name.
- **Renumber for the caller.** Rejected: mrw would guess which version the number came from, and a wrong guess edits silently.
- **Keep the date.** What ADR-200's pre-registration says. The owner chose not to.

## Component / Boundary Impact

One engine package, owned here: `internal/apply` (the refusal at `staleMarks`' three call sites). `internal/refusal` gains a kind. `cmd/mrw` changes the `mrw stats` sentence. `internal/seen`, `internal/plan`, `internal/read`, `internal/check`, `internal/state` and `internal/writer` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a refused hunk | new reason and kind, exit 1 | T1 | every caller of `mrw write` and `mrw_write` |
| `mrw stats` | the stale-number block's last sentence | T1 | its readers |
| `mrw instructions` (core and full), `AGENTS.md`, `README.md`, `BESTPRACTICES.md` | rule 2 and recipes 7 and 21 say a second edit names its line | T1 | every caller |
| `scripts/contract.sh` §343 and §349 | §343's "is not refused" rows become a seed that lands by `anchor=`; §349 is the refusal and its pairs | T1 | the contract gate |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the one way a licensed write could edit the wrong line with no guard firing is closed, with no new state.
- **Negative:** a new refusal on plans that landed before, including ones whose number was right; at the one measured width, about one typed-plan landing in twenty, with a large uncertain share.
- **Neutral:** exit codes, receipt keys and the ledger are unchanged.

## Out of Scope

- A selective refusal with ledger state (deferred: docs/adr/BACKLOG.md)
- A receipt row naming the shift (deferred: docs/adr/BACKLOG.md)
- A numbered edit on ADR-190's partial record (permanent: boundary: it is judged by the per-line licence, which already refuses a line the caller was not served)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The refusal rate is higher than callers will bear | Med | Med | the reason names three one-step ways out; the instructions change with it; the BACKLOG reading; `--force` on the CLI |
| Tests and contract rows that chain numbered writes go red | High | Low | each is fixed by an anchor, a pattern or a read; every record whose locked test moves is relocked after review |
| The instructions' core outgrows its 300 words | Med | Low | rule 2 is reworded, not lengthened; the guide's own test holds the limit |

## Rollback

Revert T1: no state was changed, so nothing is left behind.

## History

The first draft of this record (commit `c7186cc8` on the branch) proposed `Observation.Moved`, ledger v7 and a refusal only below the first line a write moved. Challenged before any code. Established against the code and by probe: after `3 insert-after`, a stale `4 replace` overwrites the inserted line and a stale `4-6 delete` lands, both passing that draft's rule; its partial-record row could never be reached; a rename or a write through another spelling lost the move; `merge` could not accumulate it; an mrw from v1.66.0 meeting a v7 ledger could only create or `--force` in that checkout. Its "Accepted" line overstated the owner's decision. None of that design is built.

## Follow-ups

- [ ] The reading of BACKLOG.md, "Pre-registered for ADR-203".
