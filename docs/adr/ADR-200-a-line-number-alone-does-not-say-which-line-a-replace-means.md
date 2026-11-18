# ADR-200: A line number alone does not say which line a replace means

**Status:** Proposed
**Date:** 2026-11-18
**Amended:** 2026-11-18, before acceptance and before any run, from an adr-challenger pass and a Codex xhigh challenge of the first draft (each finding checked against the code). The first draft chose a line tag and said the drift a 2026 paper measures "does not land here". The challenger showed that it does, after mrw's own write, and this session reproduced it. The record now states that failure, measures both ways a wrong number arrives, and picks the guard from the reading instead of ahead of it.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-005, ADR-008, ADR-035, ADR-052, ADR-169, ADR-179, ADR-194, ADR-197, ADR-009, docs/adr/BACKLOG.md, docs/bench/effect-study/README.md
**Supersedes:** none
**Governs:** `docs/bench/effect-study/scan.py`, `docs/bench/effect-study/test_scan.py`
**Enforced-by:** `docs/bench/effect-study/test_scan.py::test_a_replace_after_a_shift_is_classified`
**Served-path change:** none in this record's one task, which measures. Which guard is built, if any, is decided from its reading (Decision 3) and gets its tasks then.

## Context

**Where this came from.** A reading of 2026 work on edit formats (memoryd craft/34fd8f89): a paper that measured line-range edits misapplied silently in 99.1% of files after a one-line shift (arXiv 2609.11957), and an edit format whose edits cite a short content hash printed beside each served line (the "hashline" post of 2026-02-12, its author's own benchmark, no independent replication found). Zy, 2026-11-18: *"Ok go, build experiments"*, then *"Yes"* to drafting this record.

**The failure, reproduced** on mrw v1.88.0 (0aaadb45), 2026-11-18, in a scratch root: `f.txt` holds `a b c d e`, one a line, all five read.

```
@@ f.txt 1 insert-after      →  ok, the file is now a NEW b c d e
NEW
@@ f.txt 4 replace           →  ok, exit 0
D-CHANGED
```

The caller meant the line it was served as 4, `d`. Line 4 is now `c`, and `c` is replaced. Nothing is refused and the receipt does not say what was removed. A file mrw has just written is wholly licensed (ADR-005 §4, "a chain of edits needs no re-read"), so a number from before the write is accepted on whatever line now has it.

**The two ways a wrong number arrives**, in a file nobody else changed:
1. *A stale number after mrw's own write*, above. This is the paper's case, between two plans of one session. Inside one plan it cannot happen: every address resolves against the original.
2. *A wrong number in a range that was served*: line 41 typed where 42 was meant, both served.

**What already refuses or shows it.** A file changed by anyone else is refused whole by its sha (ADR-002). A line never served is refused (ADR-005), except in a file mrw just wrote. A `replace` of more than one line must carry `anchor=`, checked against its first line only (ADR-035; `internal/apply/apply.go`, the anchor comparison). An `anchor=` on a single-line `replace` is accepted and checked today, and nothing requires it. A `delete` says what it removed, after the fact (ADR-008). The balance and closer rows and `--echo-pad` look at shape, not at which line was named.

**What the transcripts say so far** (`docs/bench/effect-study/`, results.json at 4267acb9: 942 Claude Code transcripts on one Mac, entries before 2026-11-18T10:00:00Z):
- 22.3% of the hunks typed are a single-line `replace` with no anchor (6,356 of 28,563). Exposure, not a rate.
- Anchor-miss lines are 2.23% of the anchored hunks typed, 1.74% without this repository's sessions. The two counts are different populations (result lines against headers typed), over hunks of every length, and a miss may be a quoting mistake or an anchor that is nowhere.
- Anchor text is 0.16% of recorded output tokens.

**What is not known**, and what T1 measures: how often either wrong number is sent, and how often the stale one lands.

**Audit of the class**: *a hunk whose address is a line number and whose op changes or removes existing lines.* From the seven ops in `internal/plan/plan.go`:
- single-line `replace`: no guard. Decided here.
- multi-line `replace`: `anchor=` on its first line; its last line is not checked. Left as ADR-035 decided it (Out of Scope).
- `delete`, one line or many: no guard before it lands; the receipt says what went (ADR-008). T1 counts it beside `replace` so the reading says whether it belongs in the guard (Decision 2).
- `insert-after`, `insert-before`: change no existing line; a wrong number puts new lines in the wrong place. Counted by T1, not guarded here (Out of Scope).
- `create`, `unlink`, `rename`: address a path, not a line. Not in the class.
- a pattern address names its line by content. Not in the class.
- a `from=PATH:ADDR` source: a copy of lines the caller was served in the version on disk (ADR-167); a stale number there copies the wrong lines. Counted by T1, not guarded here (Out of Scope).

## Existing Primitives Audit

- `anchor=` on a single-line `replace`: the guard exists, optional.
- The read's ` NNN| ` gutter is parsed by `internal/mcp/ack.go` (checkpoints; a non-numeric gutter gets none), the awk recipe in `AGENTS.md` §3, `scripts/chaos.py`, `scripts/licence194_replay.py`, the copy-study, context-study and output-study scanners, rows of `scripts/contract.sh` and Go tests. `mrw hook read` and `.claude/hooks/rules-on-read.py` do not parse it: they read the host's own content and the `==>` headers. Found with `git grep -lE '\[0-9\]\+\\\||\\d\+\)?\\\||\\d\+\\\|' -- ':!docs/adr'` plus `ack.go`, which uses `Atoi`.
- `scripts/licence194_replay.py` (ADR-194) already replays a transcript's mrw calls in order and renumbers lines after a write. T1 reuses that approach.
- The write receipt prints `wrote f.txt 5L -> 6L`: the caller is told the length changed, not from which line the numbers moved.

## Decision

1. **Measure both wrong numbers first (T1).** No guard is chosen or built before the reading. The measures and the bars are in BACKLOG.md ("Pre-registered for ADR-200"), amended there before any run.
   - **M1, the stale number.** For every hunk with a numeric address sent to a file after the session's own mrw write changed that file's line count above the addressed line, with no read of the addressed line between: count them, by op. Where the session was served both the line that has the number now and the line that had it before the write, classify the hunk by which of the two its body resembles more. A body that resembles the line that *had* the number is a probable stale number that landed.
   - **M2, the wrong number in a served range.** Single-line `replace` attempts that carry `anchor=`, joined to their own result row by path, address and plan line: the share refused for the anchor, split into the anchor found on another line, found nowhere, and the quoting case. This is a rate among callers who chose to anchor, and is reported as that.
2. **The guard covers the ops the reading says it must.** If M1's probable stale hunks include `delete`, the guard covers `delete` with `replace`.
3. **The guard is picked from the reading**, by this rule, written before it:
   - M1 meets its bar: the failure is a number carried across the session's own write. The first guard is the cheapest one that reaches every caller: **the write receipt says where the numbers moved** (`f.txt: lines after 1 are now +1`), and a later hunk at a moved number in that file, sent with no read between, is **refused once** with that sentence, as a read-before-write refusal is. No gutter changes, nothing for a host to switch on.
   - M2 meets its bar and M1 does not: the failure is a mistyped number. The candidates are a required `anchor=` on a single-line `replace`, a tag beside the served line number that a plan cites, and a receipt row quoting the removed line. Each is priced in Alternatives; the choice is a second record's, with M2's split in hand.
   - Neither meets its bar: this record is withdrawn with the numbers.
4. **What would make this decision wrong.** If the guard of Decision 3's first arm ships: thirty days after its release, T1's M1 is run again over the transcripts since. If probable stale hunks that landed are not at most a quarter of the pre-release rate per thousand after-shift hunks, the guard is not reaching the failure and goes back to Zy.

## Alternatives Considered

- **A tag beside each served line number, cited in the plan** (the first draft's decision). Not chosen ahead of the reading. It needs a host to turn it on and a caller to cite it (ADR-169's optional guard: 27 uses in 19.9 million served characters, one 46-session Claude Code cohort); a second gutter form for every parser in the audit above; three characters a served line; a grammar for `N:hh` that does not collide with `from=PATH:ADDR`, whose parser splits at the last colon (`from=g.txt:1:12` is today the file `g.txt:1`, line 12); and a tag copied along with a stale number defeats it. Against the stale number it does work without a re-read, which is its case if M1 is large and the refusal of Decision 3 proves too blunt.
- **Require `anchor=` on a single-line `replace`.** The check exists. It refuses 22.3% of the hunks typed on this corpus until callers change, and ADR-035 scoped the requirement to more than one line on purpose. It catches both wrong numbers.
- **The receipt quotes the removed line**, as a `delete`'s does. Reaches every caller with no input change, and `--terse` keeps a delete's row today so it could keep this one; `MRW_RECEIPT=quiet` would not show it. It shows the mistake after it has landed.
- **Narrow ADR-005 §4** so that a write licenses only the lines it wrote. Rejected: the unchanged lines below an insert were served and are still what they were; the licence is right and the number is what moved.
- **Renumber the stale hunk for the caller.** Rejected: mrw applies what the plan names or refuses (ADR-001); a guess that lands is the failure this exists to prevent.

## Component / Boundary Impact

T1 touches `docs/bench/effect-study/` only. Decision 3's first arm, if built, changes `internal/apply` (the refusal) and `internal/seen` (what a write records about moved lines), and the receipt in `cmd/mrw`: two engine packages, each with a go-reviewer pass, in tasks written after the reading and Zy's acceptance.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `docs/bench/effect-study/scan.py` | M1 and M2 | T1 | this record's bars |
| `docs/adr/BACKLOG.md` | the bars, then the reading | T1 | Zy |
| the write receipt and a refusal after a shift | only if M1 meets its bar | tasks not yet written | every caller |

## Inter-task Contracts

None — one task until T1's reading.

## Implementation

`tasks/README.md`: T1, the measurement. Tasks for a guard are written after T1's reading and Zy's acceptance.

## Consequences

- **Positive:** the guard that is built answers a failure that was counted, not one that was read about.
- **Negative:** nothing changes for a caller until T1 is read; the reproduced failure stays possible meanwhile.
- **Neutral:** identical neighbouring lines stay indistinguishable by content to any guard.

## Out of Scope

- The last line of a multi-line `replace` (permanent: boundary: ADR-035 checks the first line and the caller's body states the rest; a range that ends one line off is a different mistake with its own evidence, recorded in memoryd craft/f6deec19 and 57c5384e)
- A guard on `insert-after`, `insert-before` and `from=` addresses (deferred: docs/adr/BACKLOG.md)
- The choice among anchor, tag and receipt quote when only M2 meets its bar (deferred: docs/adr/BACKLOG.md)
- Tags over `mrw_read` and its checkpoints (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| M1's resemblance test calls a correct renumbering stale, or the reverse | Med | Med | only hunks where both lines were served and the two resemblances differ by a stated margin are classified; the rest are counted as unclassified and shown |
| M2 is read as the rate among callers who do not anchor | Med | Med | Decision 1 says whose rate it is; it decides only between this record's arms |
| A plan built by a script, or a result that cannot be joined to its hunk, is missed | High | Low | joined and unjoined counts are both reported; fewer than the floor is "insufficient", not a miss |
| The refusal of Decision 3 fires on a caller that renumbered correctly | Med | Med | it fires once per moved file and the receipt has already said where the numbers moved; the tasks, when written, price it on M1's "renumbered correctly" count |

## Rollback

T1 is a scanner change and a BACKLOG entry: revert it.

## Follow-ups

- [ ] T1's reading against the bars (BACKLOG.md, Pre-registered for ADR-200)
- [ ] The guard's tasks, or the withdrawal, from that reading
