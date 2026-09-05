# Reading 13, result: with the number present the miss appears at every size, at a rate that rises with size

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

Neither pre-registered account survives as written. **H-preceding-text** said no miss at 2 KB; a
miss at 2 KB with 39 rows of preceding text refutes the amount of preceding text as what the miss
requires. **H-position** said the misses concentrate in the late cells; two of three are late and
one is middle, and three misses cannot carry a claim about position either way. Reading 11's
late-only pattern at 200 KB is therefore not reproduced as a rule at smaller sizes, and stays what
reading 11 called it: observed on five cells, not explained.

## What the three sizes say together

With the number present, this client's curve through a tool result is 14, 13, 10 of 15 across
2 KB, 20 KB, 200 KB (readings 13, 13, 11). Reading 4's curve through its file reader — the same
number, laid by the reader — was 15, 12, 8. Through the bare tool result, with no second number,
it was 15, 15, 15 (readings 9, 9, 8). So: in the presence of a plausible second number, served size
is associated with the miss rate through both deliveries measured; in its absence the curve was
flat at every size. The pairings by cell:

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

The second number's effect is not confined to 200 KB or to the late position: it appears at 2 KB
and at 20 KB, at `T + 2`, at a lower rate. What the late-only pattern of reading 11 was, this
reading does not say. What it does say is that the flat curve of readings 8 and 9 is a property of
the bare delivery, and that the delivery which lays a plausible line number beside mrw's has a curve
that bends with size — through a file reader (reading 4) and through a tool result (readings 11
and 13) alike. For mrw nothing moves: the cap stays, the served format stays, and the bare
delivery is the one the tool's own output is.

## What it does not decide

One client, one fixture family, five repeats per cell; three misses in thirty; the late-only
pattern of reading 11 neither reproduced nor refuted by three misses.

## Provenance

Thirty scores in `docs/curve/reading-13-scores/`; `reading-13-tally.json` is computed over them:

```sh
bin/curve tally docs/curve/reading-13-scores/*.score.json
```

Compliance and cost come from transcripts that are not committed.
