# Reading 13, result: with the number present the miss appears at 2 KB and 20 KB too, three times in thirty

**Collected 2026-09-05 under `reading-13-plan.md`, committed before any trial ran.** Thirty
trials, a Haiku client, reading 11's arm — one `sed` range piped through `nl -ba -v 1`, so the outer
number on every row is the served text's row index, `T + 2` on the target's row — on reading 4's
fifteen 2,000-byte and fifteen 20,000-byte cells, verified byte-identical to reading 9's copies.

## Compliance: 30 of 30

Every trial ran its two listed commands, in order, and nothing else: no other command, no other
tool, no memory call after the Write, no spilled result. Prediction 3 (at least 27 of 30) holds.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 4 (reader) | reading 9 (bare tool result) |
|---|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 4/5 | **14/15** | [0.702, 0.988] | 15/15 | 15/15 |
| 20,000 | 5/5 | 4/5 | 4/5 | **13/15** | [0.621, 0.963] | 12/15 | 15/15 |

Three misses in thirty, **every one at exactly `T + 2`** (prediction 2 holds): `2000-late-4`,
`20000-late-1`, `20000-middle-2`.

## Prediction 1 fails, and both accounts miss

The author expected no miss at 2 KB and at most one at 20 KB. There was one at 2 KB — a target on
line 38 with 39 numbered rows before it — and two at 20 KB. That is the third reading in a row in
which the author's expectation was wrong.

**H-preceding-text failed as written:** it said no miss at 2 KB, and there was one, with 39 rows of
preceding text. **H-position is inconclusive:** it said misses concentrate in the late cells; two of
three are late and one is middle, and three misses cannot carry a claim about position either
way. The plan's decision rule — "late-only misses at 20 KB, or any at 2 KB, mean position is the
variable" — fired on one 2 KB late miss, and one miss is too coarse a trigger to decide anything;
the rule was written too strong, and this result does not apply it. Reading 11's late-only pattern
at 200 KB is neither reproduced nor refuted here, and stays what reading 11 called it: observed on
five cells, not explained.

## What the three sizes say together

With the number present, this client's observed points through a tool result are 14, 13, 10 of 15
across 2 KB, 20 KB, 200 KB (readings 13, 13, 11). Reading 4's through its file reader — the same
number, laid by the reader — were 15, 12, 8. Through the bare tool result, with no second number,
they were 15, 15, 15 (readings 9, 9, 8). These are point estimates, and the ones at 2 KB and 20 KB
have overlapping intervals: **these trials do not establish a served-size trend.** What they
establish is that with the number present the miss occurred at every size measured, always at
`T + 2`, and that without it no miss occurred at any size. The pairings by cell:

- Against reading 9 (bare): 2 KB 1–0, 20 KB 2–0, reading 9 hit and this reading missed; the
  reverse never.
- Against reading 4 (reader, same number): 2 KB 0–1; 20 KB 2–1, with one cell (`20000-late-1`)
  missed by both. The two deliveries of the same number are not distinguishable at these sizes.

Thirty trials at rates this close to the ceiling cannot separate 14 of 15 from 15 of 15; the
finding is the offset — three of three at `T + 2` — and the direction of the pairings, all of which
run one way against the bare arm.

## Cost

| Served bytes | median spend, this arm | reading 9 | ratio | median peak |
|---|---|---|---|---|
| 2,000 | 32,670 | 31,909 | 1.02× | 51,452 |
| 20,000 | 38,177 | 36,444 | 1.05× | 56,944 |

Prediction 4 holds.

## What this decides

The second number's effect is not confined to 200 KB or to the late position: it appeared at 2 KB
and at 20 KB, at `T + 2`. What the late-only pattern of reading 11 was, this reading does not say,
and whether the miss rate rises with size, thirty trials with three misses cannot say either. What
it does say is that the flat curve of readings 8 and 9 is a property of the bare delivery: with a
plausible line number beside mrw's, this client missed at every size measured, through a file
reader (reading 4) and through a tool result (readings 11 and 13) alike. For mrw nothing moves: the
cap stays, the served format stays, and the bare delivery is the one the tool's own output is.

## What it does not decide

One client, one fixture family, five repeats per cell; three misses in thirty; the late-only
pattern of reading 11 neither reproduced nor refuted by three misses.

## Provenance

Thirty scores in `docs/curve/reading-13-scores/`; `reading-13-tally.json` is computed over them:

```sh
bin/curve tally docs/curve/reading-13-scores/*.score.json
```

Compliance and cost come from transcripts that are not committed.
