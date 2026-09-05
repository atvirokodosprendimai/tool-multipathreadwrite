# Reading 10, result: an outer number in the tool result, and the client still took mrw's

**Collected 2026-09-05 under `reading-10-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the scripted tool-result arm of reading 8 with every `sed` range piped
through `nl -ba`, on reading 4's fifteen 200,000-byte cells, verified byte-identical.

## Compliance: 14 of 15

Fourteen trials ran the thirteen listed commands, in order, and nothing else. One trial
(`200000-early-4`) merged the last two listed ranges into one `sed -n '3001,3627p'` — a merge
the rule tolerates — and the merged output was 38 KB, which the harness spilled to a file and
replaced with a 2 KB preview. The plan pre-registered "no spill" as a compliance condition, so
that trial is **void and reported, not counted**, though its plan addressed the target (an early
one, at line 726, in a range the client had seen whole). Prediction 3 (at least 12 of 15) holds.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 4, same cells | reading 8, same cells |
|---|---|---|---|---|---|---|---|
| 200,000 | 4/4 | 5/5 | 5/5 | **14/14** | [0.785, 1.000] | 8/15 | 15/15 |

Fourteen of fourteen compliant trials at offset 0, zero refusals. The void trial was also at
offset 0.

## Prediction 1 fails: no miss at the outer number, or anywhere

The plan's expectation — at least three misses, each at the outer number of the target's row —
did not happen. **Zero misses.** Every plan addressed the target by mrw's number.

A correction to the plan's own arithmetic, recorded here and not there: the plan wrote the outer
number as `T − A + 1`. `served.txt` opens with two header rows (`==>` and `@@`), so line `T` is
row `T + 2`, and the outer number `nl` gives it in the range starting at row `A` is `T − A + 3`.
The transcripts show it: for the target on line 725 in `200000-early-1`, the row arrived as

```
   127	  725| timeout = 30
```

and the client's plan said 725. No plan in the reading addressed either number's neighbourhood
but the target's own; the correction moves the predicted miss by two rows and changes nothing
about a reading with no misses.

Prediction 2 holds: no miss at `target+2`.

## The pairing with reading 4

On the fourteen compliant cells, reading 4 hit 7 and this reading hit 14: 7 discordant pairs,
all one way, exact two-sided sign test p = 0.0156 — the same seven cells and the same p as
reading 8 against reading 4.

## Cost

| Arm | median spend | median peak |
|---|---|---|
| Reading 8, tool result, mrw's number only | 84,789 | 103,428 |
| Reading 10, tool result, `nl` number beside it | 96,184 | 114,719 |

Prediction 4 holds: 13.4% above reading 8's median, inside the 20% the plan allowed — the
extra bytes are the outer numbers, about eight per row over 3,600 rows.

## What this decides, and what it does not

The plan said fifteen of fifteen here "would mean the ranges did the work, and the note's account
would have to change." Taken as pre-registered: **the two accounts reading 8 could not separate are
now separated one way** — a second number beside mrw's, delivered inside a tool result, did not
reproduce the miss. What took the weaker client from 8 to 15 on these cells was not the absence of
a second number as such.

But this reading's outer gutter is not reading 4's, in one respect the plan did not weigh.
Reading 4's reader numbered the whole file from 1, so its number on the target's row was `T + 2`
at every size and every position — a number that looks like a line number. Reading 10's `nl`
restarts at 1 in every range, so the outer number on the target's row was between 127 and 261
for targets at lines 725 to 2917: a number that does not read as a line address in a 3,600-line
file, and one this client did not take in these trials. The parsimonious account of reading 5 —
the client took the reader's number — is therefore neither confirmed nor refuted here. It has been
narrowed: in these trials this client did not take *any* second number; the one it took, in
readings 4 and 5, was continuous from the top of the file and presented by its own file reader.

The two remaining accounts, and the reading that separates them, are in `reading-11-plan.md`:
the same ranges with `nl -ba -v A`, so the outer number is the row index of the whole served
text — `T + 2` on the target's row, indistinguishable in value from reading 4's.

## Provenance

Fifteen scores in `docs/curve/reading-10-scores/`, the void one included and named above;
`reading-10-tally.json` is computed over the fourteen compliant scores:

```sh
ls docs/curve/reading-10-scores/*.score.json | grep -v early-4 | xargs bin/curve tally
```

Compliance, the outer numbers and cost come from transcripts that are not committed.
