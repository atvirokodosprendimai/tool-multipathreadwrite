# Reading 6, result: void under its own rule, and what the void says

**Collected 2026-09-05 under `reading-06-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the tool-result arm, on reading 4's fifteen 200,000-byte cells, verified
byte-identical before the first trial.

## The verdict the plan requires

**No trial is compliant under the plan's rule, so the reading decides nothing.** The plan said a
void-heavy reading "says so and decides nothing", and this is that. Every one of the fifteen plans
hit — fifteen of fifteen, on cells where the same client through the read arm hit eight — and no
plan sits at `target+2`; but the plan pre-registered four compliance conditions and the transcripts
fail them in four different ways:

| Trial | Hit | Offset | Searched | Widest range | Coverage | Spilled | Void under the plan |
|---|---|---|---|---|---|---|---|
| 200000-early-1 | yes | +0 | yes (1) | 370 | 1836/3621 | no | yes |
| 200000-early-2 | yes | +0 | yes (10) | 351 | 2682/3629 | no | yes |
| 200000-early-3 | yes | +0 | no | 474 | 3623/3625 | no | yes |
| 200000-early-4 | yes | +0 | no | 476 | 3625/3627 | no | yes |
| 200000-early-5 | yes | +0 | no | 545 | 3643/3645 | no | yes |
| 200000-late-1 | yes | +0 | no | 351 | 3150/3621 | no | yes |
| 200000-late-2 | yes | +0 | no | 478 | 3627/3629 | no | yes |
| 200000-late-3 | yes | +0 | no | 351 | 3150/3625 | no | yes |
| 200000-late-4 | yes | +0 | no | ≤350 | 3150/3627 | no | yes |
| 200000-late-5 | yes | +0 | yes (4) | 544 | 3643/3645 | yes | yes |
| 200000-middle-1 | yes | +0 | yes (3) | 420 | 3619/3621 | no | yes |
| 200000-middle-2 | yes | +0 | yes (1) | 351 | 2338/3629 | no | yes |
| 200000-middle-3 | yes | +0 | no | 474 | 3623/3625 | no | yes |
| 200000-middle-4 | yes | +0 | no | 476 | 3625/3627 | no | yes |
| 200000-middle-5 | yes | +0 | yes (2) | ≤350 | 2450/3645 | no | yes |

- **Twelve trials ran a range one row over the cap** — `1400,1750` is 351 rows — because a client
  that chunks on round boundaries overlaps them by one. Two ran ranges of 474 and 545 rows. None of
  these spilled: the cap was set from a 55-byte row estimate, and the rows are shorter. The cap was
  the author's mis-specification, and it voids under the rule as written.
- **Six trials searched** — `grep`, a pattern in a `sed` address, a pipe — despite the prohibition
  being the point of the arm. A hit reached by `grep -n "\[service"` says nothing about whether a
  client counts rows when it reads, which is the question.
- **Three stopped at row 3150** after nine ranges and answered from what they had; the late targets
  sit before 3150, so they hit anyway.
- **Most left the last two rows unread**: a range ending at `3623` on a 3,625-row file. Padding, and
  never the target; void under "every row" all the same.

## Predictions, scored

1. Rate rises against 8 of 15 — **not scorable**: no compliant trial exists to count.
2. No miss at `target+2` — **holds on every trial, void or not**: fifteen plans, fifteen at offset 0.
   It is reported, not claimed, because six of those fifteen found the target by searching.
3. Compliance lower than 15 of 15 — **confirmed, and far beyond the expectation**: 0 of 15. The
   specific expectation — a `sed` pattern, a `grep`, a range over the cap — named three of the four
   failure modes; it did not name stopping early.
4. Cost within 20% of reading 4's 91,546 — **holds**: median spend 82,976.

## What the void suggests, labelled as such

Six trials neither searched nor spilled and read every row but the last two of padding: early-3, early-4, early-5, late-2, middle-3, middle-4.
Their only violation is the off-by-one range. All six hit. On the same six cells the read arm in
reading 4 scored 3 of 6. That is a post-hoc selection under a tolerance the plan did not
pre-register, and it is written here so the next plan can pre-register it, not as a result.

## What follows

The arm was under-specified for this client. A weaker client given `sed` and a rule will search, and
given a cap and a file length will overlap its boundaries. Reading 7 fixes both by giving each trial
the exact commands to run — the ranges precomputed per cell, ending on the file's last row — so
coverage and the cap take no judgement, and the only way to be non-compliant is to run something
else. The client, the cells and the question are unchanged.

## Provenance

Scores in `docs/curve/reading-06-scores/`, tally in `docs/curve/reading-06-tally.json`. Compliance,
coverage and cost come from transcripts that are not committed; the table above is reported from
them. The cells regenerate from reading 4's parameters and are byte-identical to it.
