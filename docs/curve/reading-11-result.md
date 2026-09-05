# Reading 11, result: with reading 4's number put back, the miss came back — at `T + 2`, and only late

**Collected 2026-09-05 under `reading-11-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the scripted tool-result arm with every `sed` range piped through
`nl -ba -v A`, on reading 4's fifteen 200,000-byte cells, verified byte-identical against
reading 10's copies.

## Compliance: 15 of 15

Every trial ran its thirteen listed commands, in order, and nothing else: no other command, no
other tool, no memory call after the Write, no spilled result, no merge. Prediction 3 holds.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 4 | reading 8 | reading 10 |
|---|---|---|---|---|---|---|---|---|
| 200,000 | 5/5 | 5/5 | **0/5** | **10/15** | [0.417, 0.848] | 8/15 | 15/15 | 14/14 |

## Prediction 1 fails, prediction 2 holds: five misses, every one at `T + 2`

The author expected no miss. There were five, and **each addressed exactly `T + 2`** — the outer
number on the target's row, which this arm made equal to reading 4's reader number. The row as
the client saw it in `200000-late-1`, and its one-line reply:

```
  2900	 2898| timeout = 30
```

> 2900

No miss anywhere else; no third account is needed. The outer number's *value* is what mattered
for this client: when it looked like a line number it was taken in five of fifteen trials, and reading 10's per-range numbers,
which did not, were not.

## A position effect this plan did not foresee

All five misses are the five late cells; the ten early and middle cells are ten hits. Reading 4's
seven misses on these same cells were spread — early 1, middle 3, late 3 — so this is not
reading 4's pattern reproduced; it is sharper. The plan predicted no position effect and offers
no account of one; it is reported as observed, with a Wilson interval on 0 of 5 whose upper bound
is 0.434. The read arm's misses in readings 2 to 4 were not late-only either (reading 2: one at
each size, two of them late). One possibility, not tested: the two numbers on a late row are four
digits each and two apart, and the same is true of the early and middle rows, so it is not the
digits; what differs late is how much numbered text precedes the target, about 2,900 rows. That
is a hypothesis for a plan, not a finding.

## The pairings

- Against reading 10 on the fourteen cells compliant in both: reading 10 hit and this reading
  missed on 5, the reverse on 0. Exact two-sided sign test p = 0.0625. The only difference between
  the two arms is the value the outer number carries.
- Against reading 4 on all fifteen: reading 4 missed and this reading hit on 4, the reverse on 2.
  The two arms differ in where the number comes from, not in whether one is there, and the rates
  — 8 and 10 of 15 — are not distinguishable at this size.

## Cost

| Arm | median spend | median peak |
|---|---|---|
| Reading 10, `nl` per range | 96,184 | 114,719 |
| Reading 11, `nl -v A` | 99,088 | 117,765 |

Prediction 4 holds: 3.0% above reading 10's median, against 20% allowed.

## What this decides

With reading 10 it separates the two things reading 8 changed at once, and separates them the
other way from what reading 10 alone suggested. Readings 10 and 11 chunk identically and differ
only in the value of the second number, and they differ on five of the fourteen comparable cells,
all one way. So the chunking is not what removed the miss: restoring a plausible outer line number
reproduced the `T + 2` miss in five of fifteen trials under chunked delivery. What the chunking may contribute on its own these
two readings cannot say — reading 4 against reading 11 is 8 and 10 of 15 with six discordant
pairs both ways, and a different position pattern. Restoring the plausible number reproduced five
`T + 2` misses in the chunked arm — a number whose value is the row index of the served text from
its first row, which is what a file reader shows and what `nl -v` reproduced. The note's account
of reading 4 therefore stands for both delivery forms measured, a file reader's gutter and a
`nl -v` column beside mrw's in a tool result: in each, a weaker client took the second number
some of the time, and when it did it was off the target by exactly the difference, here two, the
header rows.

What stays the same for mrw: served size did not bend the curve, at 200,000 bytes or below; the
cap stays; through a delivery where mrw's number is the only plausible line number, the weaker
client is at the ceiling (readings 8, 9, 10).

## What it does not decide

One client, one fixture family, five repeats per cell; the position effect is observed, not
explained, and rests on five cells; the MCP delivery was not run. Fifteen trials cannot tell
8 of 15 from 10 of 15.

## Provenance

Fifteen scores in `docs/curve/reading-11-scores/`; `reading-11-tally.json` is computed over
them:

```sh
bin/curve tally docs/curve/reading-11-scores/*.score.json
```

Compliance, the outer numbers and cost come from transcripts that are not committed.
