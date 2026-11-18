# ADR-200: A line number alone does not say which line a hunk means; mrw counts what a guard would refuse

**Status:** Proposed
**Drafted for:** Zy, 2026-11-18. Asked "Should I merge #661 and draft that record?" after the reading below, with the recommendation to count would-be refusals inside mrw and report them in `mrw stats`, Zy answered *"Ues do not stop, continue"*. That approves the draft and the work on its branch. The predicate and the bar did not exist when Zy answered, so they await Zy's reading, and nothing here merges before it.
**Date:** 2026-11-18
**Amended:** 2026-11-18, before acceptance. The first draft chose a line tag; the second measured two rates from transcripts (M1, M2), which two challenges showed could not be read; the third priced an anchor guard from transcripts (E5) and its pre-registered verdict was STOP on three runs, because 13% of landed write calls could not be classed. This is the fourth form: a count taken inside mrw, from the ledger a guard would be judged on. Amended again the same day, before any code, from an adr-challenger pass and a Codex xhigh challenge (gpt-6.2-sol), each finding checked against the code: the status went back to Proposed; the predicate is stated on the ledger shape and not on "mrw wrote it"; `apply_patch` joined the computed plans; the landing is counted in one place; the over- and undercounts are named; the bar names its unit, its date and what a miss cannot say.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-009, ADR-035, ADR-056, ADR-111, ADR-118, ADR-161, ADR-167, ADR-190, ADR-194, docs/adr/BACKLOG.md, docs/bench/effect-study/README.md
**Supersedes:** none
**Governs:** `internal/apply/apply.go`, `internal/authoring/authoring.go`, `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `cmd/opencode/mrw-plugin/src/index.ts`
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

**What already refuses it.** Under the ordinary ledger check, a file whose content is not what mrw recorded is refused (ADR-002); `--force` waives that. A multi-line `replace` must carry `anchor=`, a substring of its first line (ADR-035). A pattern address names its line by content. A `from=` source is refused unless its lines were shown in this version (ADR-167), and an `occurrence=` count unless the matches before it were, unless forced (ADR-118): the ledger already tells a line *shown* from a line only *licensed* (`Observation.ServedLine`, `internal/seen/seen.go`).

**The guard that suggests, and its price so far.** Refuse a numbered hunk with no `anchor=` when a line it addresses was not shown since mrw wrote the file. Its price was pre-registered and counted from 769 Claude Code transcripts (`docs/bench/effect-study/`, E5; BACKLOG.md, "Measured 2026-11-18 for ADR-200"): third run, without this repository, entries before 2026-11-18T10:00:00Z: 5.06% of 6,700 landed write calls with a typed plan refused, 13.90% uncertain, against bars of 5% and 10% together. Verdict STOP, three runs. The uncertain calls are mostly reads piped through another command: the ledger recorded what they served and the transcript did not. A guard would be judged on the ledger, so the ledger is where its price can be counted. That count is of a different population (every landed write, not only those with a typed plan) and is not comparable to the 5.06%.

**Audit of the class**: *a hunk whose address is a line number, sent to a file whose ledger record licenses every line and shows none or some.* Counted with `mrw read --grep 'Op[A-Za-z]* *(Op)? *= *"' internal/plan/plan.go` (7 ops), `rg -n 'plan\.Parse\(' --glob '!*_test.go' --glob '!docs/**' .` (6 plan producers) and `rg -n 'seen\.(Record|RecordWithin)\(' --glob '!*_test.go' --glob '!docs/**' .` (9 ledger writers), 2026-11-18.

By op and address:
- `replace`, `delete`, `insert-after`, `insert-before` with an address typed in numbers (`N`, `N-M`, `N-$`, `N,+K`) and no `anchor=`: counted. A `sha=` or `lines=` on it changes nothing: neither says which line (ADR-164), and a write's receipt prints the new sha.
- the same with `anchor=`: the anchor is checked against the line. Not counted.
- a pattern address, and a SEARCH/REPLACE hunk (ADR-190): placed by content. Not counted.
- `$` alone, an insert at the top (`insert-after 0`, `insert-before 1`) and an `insert-before` one past the last line: name a position, not a line that can move. Not counted.
- `create`, `unlink`, `rename`: address a path. Not in the class.
- a `from=` source: already refused unless shown. Its numbered target is judged as any hunk's.

By where the plan came from:
- a plan the caller typed or filed, on the CLI or as `mrw_write`'s `plan`: counted.
- a plan mrw computed in the same call from the file as it is: `--sub`, `--rewrite`, `--format=apply_patch`, `--format=search_replace`, and their MCP forms (`grep`+`sub`, `ast_grep`+`rewrite`, `format`). Its numbers were found by content in this call. Not counted. A one-line `apply_patch` change compiles to a numbered `replace` with no anchor and no quote (`internal/ingest/applypatch.go`), so it needs this exemption said, not assumed.
- `mrw skill --dir`: lands with `Force`. Not counted.

By what the ledger holds for the file (the observation apply selected, alias lookup included):
- whole-file written-shaped (`Written` with no spans): an mrw write, **or a whole-file `--quiet` read**; the ledger cannot tell them apart (ADR-182). Counted. After such a record, a partial `--quiet` read or a `--sub` read adds nothing to it (`merge`, `internal/seen/seen.go`), so a plan from `mrw read --sub` sent in a later call is counted although its numbers are current: an overcount.
- a CLI read that served and recorded the whole file since the write: the record is no longer written-shaped. Not a candidate. A whole-file `--quiet` read keeps it written-shaped with nothing shown. MCP reads acknowledged for every line of the version on disk leave it written-shaped with every line shown: a candidate, and never unshown; lines of a page never acknowledged stay unshown.
- lines licensed in part (`--quiet` or `--sub` on a file with no whole record, ADR-161): not a candidate.
- a file last written by quoted hunks only: recorded as read spans, not written (`internal/writer/writer.go`). A later numbered hunk on those lines is not counted though its numbers moved: an undercount.
- a plain `rename`: the destination is recorded written though no number moved. Counted: an overcount.
- lines the host's own Read showed: shown only where `mrw hook read` is installed (ADR-191). Elsewhere counted unshown: an overcount.
- an MCP read never acknowledged: not in the ledger. Counted unshown: an overcount of what reached the caller, and exactly what a guard would refuse.

## Existing Primitives Audit

- ADR-056 prices `--strict-balance` nearby: apply marks each ok hunk with what the flag would have done (`singleLineCode`, `wrapTail`), `internal/writer/flow.go` records candidates through `internal/authoring`, and `mrw stats` prints the counters beside a pre-registered bar. Its record call is in `Settle`, which a write whose check could not start never reaches, and it records nothing for a non-candidate. This record reuses the marks, the counters file shape and the stats block, and not that call site.
- `Observation.ServedLine` answers "does this ledger snapshot record the line as shown in this version". No ledger change is needed, and none is made: `internal/seen` is not touched.
- `anchor=` exists on every line op and is checked; the guard, if a later record builds it, needs no new syntax.

## Decision

1. **Apply marks, and refuses nothing.** For each hunk that passed validation, in a write judged against a ledger, not forced, whose plan mrw did not compute in this call, apply records whether it is a *numbered candidate*: its op is `replace`, `delete`, `insert-after` or `insert-before`; its address was typed in numbers and names at least one line (the audit's exclusions); it carries no `anchor=`; it is not a quoted hunk; and the observation apply selected for the file is whole-file written-shaped. A candidate is *unshown* when any line it addresses (every line of a range; for an insert, the line it names) is not `ServedLine` in that observation. "Shown" means what that ledger snapshot records, which is what a guard would be judged on; it is not a claim about what reached the caller. The result carries the two counts as fields that are not receipt keys.
2. **A landing is counted once, in one place.** `Land` in `internal/writer/flow.go` records every call that left anything on disk (`MutationOf(res) != None`: a clean commit, a commit whose ledger update failed, a partial commit), once, before any check runs, and `Settle` records nothing. `internal/authoring` keeps three counters in a file of their own beside the tally: landings, landings with at least one candidate, landings with at least one unshown candidate. The marks are those of the hunks that passed validation, so a partial commit whose only unshown candidate did not reach disk still counts as would-refuse: a guard at validation would have stopped the whole plan. A dry run and a refused write count nothing. The counters live where the tally lives, in the state of the root mrw was run with, and `mrw stats --reset` leaves them as it leaves ADR-056's. They cost one small state write under the tally's lock per landing; a failure to read or write them is ignored and never changes a write's verdict. They hold counts only: no path, no content (ADR-009).
3. **`mrw stats` prints them, with the bar.** A `stale-number pricing:` block after the strict-balance one, printed where that one is, and `numbering` in `--json`, always present, with `landed_writes`, `numbered_candidates`, `numbered_would_refuse`. The four keys are appended to `docs/receipts.txt` (ADR-111). The opencode plugin's description of `mrw stats` names the block.
4. **The instructions say what is true today.** `mrw instructions` and `AGENTS.md` say that after a write that adds or removes lines, a number from an earlier read names another line: read again, address by pattern, or give `anchor=`.
5. **What would make this decision wrong, and what the count decides.** The bar is in BACKLOG.md ("Pre-registered for ADR-200's count"), written before the code. The count says how many landings a guard on this predicate would refuse. It does not say how many of those named the wrong line, and its over- and undercounts (the audit) are not separable by three counters, so a miss cannot say whether the guard or its ledger proxy is too wide. Met: a new record may build the refusal on this predicate and cite the count. Missed: the guard as designed is withdrawn. A narrower rule needs state the ledger does not hold today (which lines a write left in place; whether the caller created the file) and its own record. The count is wrong as a price if it differs from what the guard would do; T1's tests pin each clause, and a guard built later reuses the predicate.

## Alternatives Considered

- **Build the refusal now.** Rejected: its transcript price missed the bar set for it, and the bar was written first.
- **Price it again from transcripts with a better scanner.** Rejected: four review passes each found more ways a shell command differs from its text, and the uncertain share is not a scanner defect. It is what a transcript does not hold.
- **A receipt row that names the shift** (`lines after 1 are now +1`), staying `ok`. Not chosen here: it is a new receipt row for every caller, on evidence of a reproduced failure and no count. It can follow the count.
- **Narrow ADR-005 §4 so a write licenses only what it wrote.** ADR-194 priced that and stopped at INCONCLUSIVE; the licence is right and the number is what moved.
- **Count the narrower rule too** (width B of E5: a line shown before a write stays shown where the write left it in place; 4.52% against width A's 5.06% on E5's third run). Not here: it needs the write to carry surviving shown lines into the new ledger record, which changes what ADR-118's `occurrence=` and ADR-167's `from=` accept. That is a decision about the ledger and gets its own record.
- **A tag beside each served line number.** The first draft. It needs a host to turn it on and a caller to cite it.

## Component / Boundary Impact

`internal/apply` (one engine package: the marks, in the per-op validation where it holds the original op, the address form, the anchor, the quote and the resolved bounds), `internal/authoring` (counters), `internal/writer` (the record call in `Land`), `cmd/mrw` and `internal/mcp` (the stats block; the flag set by each of the four branches that compute a plan), `internal/guide` (one sentence), the opencode plugin (one description string). `internal/seen`, `internal/plan`, `internal/read`, `internal/check` and `internal/state` are not changed.

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

- **Positive:** the guard's price becomes a number taken from the ledger it would be judged on, in every root that runs the release. A caller sees no difference in any write; a landing costs one more small state write.
- **Negative:** the reproduced failure still lands until a later record acts on the count. The instructions name it meanwhile.
- **Neutral:** one more small state file per checkout.

## Out of Scope

- The refusal itself (deferred: docs/adr/BACKLOG.md)
- A receipt row naming where the numbers moved (deferred: docs/adr/BACKLOG.md)
- Telling an mrw write from a whole-file `--quiet` read, or a `--sub` plan sent later from a typed one (permanent: boundary: ADR-182, the ledger holds no provenance; the audit names both as overcounts)
- The narrower rule, width B (deferred: docs/adr/BACKLOG.md)
- Whether a `replace` range ends on the right line (permanent: boundary: ADR-035 checks a replace's first line; a delete's whole range only with ADR-008's expected body)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The count's predicate drifts from the guard a later record builds | Med | Med | Decision 5: that record reuses the predicate; T1's test names every clause |
| The overcounts make a met bar impossible where the guard itself would be cheap | Med | Med | the audit names each; the reading records whether `mrw hook read` was installed |
| A landing path is not counted, or counted twice | Med | Med | one call site in `Land`; the test moves `landed_writes` with the tally's landed count, a check that cannot start included |
| A count is read across checkouts as one population | Med | Low | the block says it is this checkout's; the bar names its reading |
| The mark changes what a write does | Low | High | the fence's test asserts the same exit code, receipt and file bytes with and without a candidate; a mutant makes the mark refuse |

## Rollback

Before a release, revert T1. After one, stop marking and keep `numbering` in `mrw stats --json` at zero: ADR-111 forbids removing a shipped key. The counters file is ignored by an older mrw, whose landings are then missing from the count.

## Follow-ups

- [ ] The reading against the bar (BACKLOG.md, Pre-registered for ADR-200's count)
