# Task ADR-200-T1: How often a numbered hunk names the wrong line, measured against its bars

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `scan.py` reports M1 (hunks sent after the session's own shifting write, classified) and M2 (single-line anchored replace attempts joined to their result rows); `test_a_replace_after_a_shift_is_classified`, `test_an_anchored_attempt_is_joined_to_its_row`; the reading in BACKLOG.md
**Consumes:** `docs/bench/effect-study/scan.py` (effect-study); the replay approach of `scripts/licence194_replay.py`
**Data dependency:** local transcripts — S4 reads `~/.claude/projects` on the machine it runs on and prints aggregates only; S1 to S3 are hermetic
**Proof map:** v1
**Rests-on:** `a hunk after a shift is found`, `a read between clears it`, `the body decides stale or renumbered`, `an attempt is joined to its own row`, `both tests run`

## Goal

ADR-200 picks its guard from two counts that do not exist yet.

**M1.** Replaying each transcript's mrw calls in order, the scanner keeps, per file, the text it was served by
line number and, after each `ok` write receipt, how the hunks of that write moved the numbers (an
`insert-after N` of k lines moves every line after N by +k; a `delete N-M` moves every line after M by
-(M-N+1); a `replace N-M` with a body of another length likewise). A later hunk with a numeric address A in
that file, sent before any read serves line A again, is an *after-shift hunk* when the earlier write moved the
numbers at or above A. Where the scanner holds both the text that had number A before the write (`was`) and
the text that has it now (`now`), and the hunk has a body, it compares the body's first line with each
(`difflib.SequenceMatcher` ratio on the stripped lines): **stale** when `was` scores at least 0.2 above `now`,
**renumbered** when `now` scores at least 0.2 above `was`, **unclassified** otherwise. Counts are by op.

**M2.** A single-line `replace` header with `anchor=` (address `N`, `$` or `N-N`) in a plan sent inline in a
shell call that runs `mrw write`, joined to the result of that same call by path, address and plan line: `ok`,
refused for the anchor (split by the refusal's ending into *in line N* or *lines N, …*, *no line holds it*,
and the quoting advice), refused for another reason, or `skip`. A header with no row, and a row with no
header, are counted as unjoined. A dry run is counted apart.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/effect-study/test_scan.py` | edit | the red tests |
| `docs/bench/effect-study/scan.py` | edit | M1 and M2 |
| `docs/bench/effect-study/README.md` | edit | the method, the reading, the limits |
| `docs/adr/BACKLOG.md` | edit | the reading against the bars |

## Ordered Steps

1. [S1] Write `test_a_replace_after_a_shift_is_classified` and `test_an_anchored_attempt_is_joined_to_its_row`, add both to the list `main()` runs, and see them RED: the keys `M1` and `M2` do not exist. M1's fixture is the record's own reproduction (five lines read, an insert after line 1, then a replace at 4 whose body resembles the old line 4) beside the same sequence with a body that resembles the new line 4, with a read between, and with an insert below line 4. M2's fixture is a plan of two anchored single-line replaces and one multi-line one, answered by one `ok` and one anchor refusal, then the same plan file run again with no header in the call. [proof: acceptance]
2. [S2] `scan.py` reports M1. Mutants, each killed: a write below the address counts as a shift; a read between does not clear the shift; the stale and renumbered labels are swapped. [proof: mutation]
3. [S3] `scan.py` reports M2. Mutants, each killed: a row is joined by path alone; a multi-line replace is counted as single-line. [proof: mutation]
4. [S4] Run the scanner twice with `--until 2026-11-18T10:00:00Z`, all transcripts and with `--skip` for this repository's, confirm identical output, and record the reading and each bar's verdict in BACKLOG.md and the README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
python3 -I docs/bench/effect-study/test_scan.py >/dev/null \
  && [ "$(grep -c 'test_a_replace_after_a_shift_is_classified' docs/bench/effect-study/test_scan.py)" -ge 2 ] \
  && [ "$(grep -c 'test_an_anchored_attempt_is_joined_to_its_row' docs/bench/effect-study/test_scan.py)" -ge 2 ] \
  && grep -q '"after_shift_hunks"' docs/bench/effect-study/scan.py \
  && grep -q '"anchored_single_line_attempts"' docs/bench/effect-study/scan.py \
  && grep -q '^\*\*Measured .*ADR-200' docs/adr/BACKLOG.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_a_replace_after_a_shift_is_classified` | `docs/bench/effect-study/test_scan.py` | a hunk after the session's own shifting write is found, cleared by a read between, not raised by a write below it, and labelled stale or renumbered by its body | none | S1, S2 |
| `test_an_anchored_attempt_is_joined_to_its_row` | `docs/bench/effect-study/test_scan.py` | a single-line anchored replace is joined to its own result row and its refusal split; an unjoined row and a multi-line replace are counted apart | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests, each named twice in the file: defined and listed |
| 2 — something selects it | `scan.py` prints M1 and M2 on every run |
| 3 — the caller can discover it | the README's method and the BACKLOG reading |
| 4 — it is used | the reading picks ADR-200's guard or withdraws the record (Decision 3) |

## Mutation Log

## Verification Log

## Invariants

- Every count effect-study already reports keeps its value on the same corpus and window.
- The scanner prints aggregates only: no path, content, project name or session id.

## Risks

- The replay knows a file's text only from what mrw served in the transcript. A file read by the host's own tool, or changed by a shell command, leaves M1 with nothing to compare: counted as unclassified.
- A body unlike both lines, or like both, is unclassified. The 0.2 margin is a choice; the counts at 0.1 and 0.3 are reported beside it so its effect is visible.
- M2 sees only plans typed inline in the call that runs them.

## Stop Condition

The fence exits 0 and the reading is in BACKLOG.md.

## Out of Scope

- Any change to `internal/read`, `internal/plan`, `internal/apply` or `internal/seen` (deferred: docs/adr/BACKLOG.md)
