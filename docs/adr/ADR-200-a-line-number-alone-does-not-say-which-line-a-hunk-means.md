# ADR-200: A line number alone does not say which line a hunk means; mrw counts what a guard would refuse

**Status:** Accepted
**Accepted:** 2026-11-18 by Zy. Asked "Should I merge #661 and draft that record?" after the reading below, with the recommendation to count would-be refusals inside mrw and report them in `mrw stats`, Zy answered *"Ues do not stop, continue"*. This session reads that as acceptance of the count this record decides, and of nothing that refuses.
**Date:** 2026-11-18
**Amended:** 2026-11-18, before acceptance. The first draft chose a line tag; the second measured two rates from transcripts (M1, M2), which two challenges showed could not be read; the third priced an anchor guard from transcripts (E5) and its pre-registered verdict was STOP on three runs, because 13% of landed write calls could not be classed. This is the fourth form: the same price, counted where it can be counted.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-009, ADR-035, ADR-056, ADR-111, ADR-118, ADR-161, ADR-167, ADR-190, ADR-194, docs/adr/BACKLOG.md, docs/bench/effect-study/README.md
**Supersedes:** none
**Governs:** `internal/apply/apply.go`, `internal/authoring/authoring.go`, `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/guide/guide.go`, `docs/receipts.txt`
**Enforced-by:** `internal/apply/stale200_test.go::TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused`
**Served-path change:** `mrw stats` prints a `stale-number pricing:` block and `mrw stats --json` carries `numbering`; `mrw instructions` says that line numbers move after a write. No write is refused that was not refused before.

## Context

**The failure, reproduced** on mrw v1.88.0 (0aaadb45), 2026-11-18, in a scratch root: `f.txt` holds `a b c d e`, one a line, all five read.

```
@@ f.txt 1 insert-after      →  ok, the file is now a NEW b c d e
NEW
@@ f.txt 4 replace           →  ok, exit 0
D-CHANGED
```

The caller meant the line it was served as 4, `d`. Line 4 is now `c`, and `c` is replaced. Nothing is refused and the receipt does not say what was removed. A file mrw has just written is wholly licensed (ADR-005 §4), so a number from before the write is accepted on whatever line now has it. It lands the same way after a delete, after a replace of another length, and across two plans in one shell command. Inside one plan it cannot happen: every address resolves against the original.

**Why it matters more than it did.** A tool that addresses by string does not have this failure: an inserted line does not move its target. Claude Code's Edit is that tool, and on current models it no longer requires a read first (`docs/comparison.md`, re-checked 2026-11-18).

**What already refuses it.** A file changed by anyone else is refused by its sha (ADR-002). A multi-line `replace` must carry `anchor=` (ADR-035). A pattern address names its line by content. A `from=` source and an `occurrence=` count are refused unless the lines were shown in this version (ADR-167, ADR-118): the ledger already tells a line *shown* from a line only *licensed* by a write (`Observation.ServedLine`, `internal/seen/seen.go`).

**The guard that suggests, and its price so far.** Refuse a numbered hunk with no `anchor=` when a line it addresses was not shown since mrw wrote the file. Its price was pre-registered and counted from 769 Claude Code transcripts (`docs/bench/effect-study/`, E5; BACKLOG.md, "Measured 2026-11-18 for ADR-200"): 5.06% of 6,700 landed write calls refused, 13.90% uncertain, against bars of 5% and 10% together. Verdict STOP, three runs. The uncertain calls are reads piped through another command: the ledger recorded what they served and the transcript did not. A transcript cannot settle them. The ledger can.

**Audit of the class**: *a hunk whose address is a line number, sent to a file whose numbers mrw's own write may have moved.* From the seven ops in `internal/plan/plan.go`:
- `replace`, `delete`, `insert-after`, `insert-before` with a numbered address and no `anchor=`: counted here.
- the same with `anchor=`: the anchor is checked against the line. Not counted.
- a pattern address, and a SEARCH/REPLACE hunk (ADR-190): placed by content. Not counted.
- an address that is only `$`, and an insert at line 0: name a position that does not move. Not counted.
- a plan mrw computed in the same call from the file as it is (`--sub`, `--rewrite`, their MCP forms): its numbers cannot be stale. Not counted.
- `create`, `unlink`, `rename`: address a path. Not in the class.
- a `from=` source: already refused unless shown. Not in the class.
- a file licensed in part by `--quiet` or a `--sub` read (ADR-161): the caller declared those lines for this version. Not counted; Out of Scope says why.

## Existing Primitives Audit

- ADR-056 prices `--strict-balance` the same way: apply marks each ok hunk with what the flag would have done (`singleLineCode`, `wrapTail`), `internal/writer/flow.go` records one line per landed write through `internal/authoring`, and `mrw stats` prints the counters beside a pre-registered bar. This record reuses that path and adds no mechanism.
- `Observation.ServedLine` answers "was this line shown in this version". No ledger change is needed.
- `anchor=` exists on every line op and is checked; the guard, if a later record builds it, needs no new syntax.

## Decision

1. **Apply marks, and refuses nothing.** For each hunk that is `ok` in a write judged against a ledger and not forced, apply records whether it is a *numbered candidate*: its op is `replace`, `delete`, `insert-after` or `insert-before`; its address was typed in line numbers (not a pattern, not `$` alone, not line 0); it carries no `anchor=`; it is not a quoted hunk; the plan was not computed by mrw in this call; and the file's observation is a whole-file written record (`Written` with no spans). A candidate is *unshown* when any line it addresses (for an insert, the line it names) is not `ServedLine` in that observation. The result carries the two counts as fields that are not receipt keys.
2. **A landed write is counted once.** `internal/authoring` keeps three counters beside ADR-056's: landed writes, landed writes with at least one candidate, landed writes with at least one unshown candidate. Every landed write counts, a partial commit included; a dry run and a refused write do not. Like the other tallies it never fails a write, and it holds counts only: no path, no content (ADR-009).
3. **`mrw stats` prints them, with the bar.** A `stale-number pricing:` block after the strict-balance one, and `numbering` in `--json` with `landed_writes`, `numbered_candidates`, `numbered_would_refuse`. The keys are appended to `docs/receipts.txt` (ADR-111).
4. **The instructions say what is true today.** `mrw instructions` and `AGENTS.md` say that after a write that adds or removes lines, a number from an earlier read names another line: read again, address by pattern, or give `anchor=`.
5. **What would make this decision wrong, and what the count decides.** The bar is in BACKLOG.md ("Pre-registered for ADR-200's count"), written before the code. If the would-refuse share is at most 5% of landed writes on the reading it names, a new record may build the refusal and cite the count. If it is over, the guard as designed is withdrawn and a narrower rule needs its own entry. If no checkout reaches the floor by the date, the record says so and the count stays. The count is wrong as a price if it differs from what the guard would do; T1's tests pin the predicate, and a guard built later must reuse it.

## Alternatives Considered

- **Build the refusal now.** Rejected: its transcript price missed the bar set for it, and the bar was written first.
- **Price it again from transcripts with a better scanner.** Rejected: four review passes each found more ways a shell command differs from its text, and the uncertain share is not a scanner defect. It is what a transcript does not hold.
- **A receipt row that names the shift** (`lines after 1 are now +1`), staying `ok`. Not chosen here: it is a new receipt row for every caller, on evidence of a reproduced failure and no count. It can follow the count.
- **Narrow ADR-005 §4 so a write licenses only what it wrote.** ADR-194 priced that and stopped at INCONCLUSIVE; the licence is right and the number is what moved.
- **A tag beside each served line number.** The first draft. It needs a host to turn it on and a caller to cite it.

## Component / Boundary Impact

`internal/apply` (one engine package: the marks, in the ledger check it already makes), `internal/authoring` (counters), `internal/writer` (the record call), `cmd/mrw` and `internal/mcp` (the stats block; the flag that says a plan was computed), `internal/guide` (one sentence). `internal/seen`, `internal/plan`, `internal/read`, `internal/check` and `internal/state` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `apply.Result` | two counts, not receipt keys | apply | writer |
| `apply.Options` | says the plan was computed by mrw in this call | cmd/mrw, mcp | apply |
| `<state>/numbering` | three counters | writer through authoring | `mrw stats` |
| `mrw stats`, `mrw stats --json` | the block, `numbering` | cmd/mrw | a person, an adapter |
| `docs/receipts.txt` | four lines appended | T1 | ADR-111's test |
| `mrw instructions`, `AGENTS.md` | the sentence | T1 | every caller |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the guard's price becomes a number with no uncertain share, taken in every checkout that runs the release, at no cost to a caller.
- **Negative:** the reproduced failure still lands until a later record acts on the count. The instructions name it meanwhile.
- **Neutral:** one more small state file per checkout.

## Out of Scope

- The refusal itself (deferred: docs/adr/BACKLOG.md)
- A receipt row naming where the numbers moved (deferred: docs/adr/BACKLOG.md)
- Lines licensed by `--quiet` or by a `--sub` read and then addressed by number (permanent: boundary: the caller declared those lines for the version on disk, or mrw computed them against it; neither is a number carried across a write)
- The last line of a multi-line range (permanent: boundary: ADR-035 checks the first line and the caller's body states the rest)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The count's predicate drifts from the guard a later record builds | Med | Med | Decision 5: that record reuses the predicate; T1's test names every clause |
| A count is read across checkouts as one population | Med | Low | the block says it is this checkout's; the bar names its reading |
| The mark changes what a write does | Low | High | the fence's test asserts the same exit code, receipt and file bytes with and without a candidate; a mutant makes the mark refuse |

## Rollback

Revert T1: the counters file is ignored by an older mrw, and no receipt key is removed from a release that shipped it (ADR-111), so a rollback after a release keeps the keys at zero.

## Follow-ups

- [ ] The reading against the bar (BACKLOG.md, Pre-registered for ADR-200's count)
