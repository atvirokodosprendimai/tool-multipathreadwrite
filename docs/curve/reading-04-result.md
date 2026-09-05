# Reading 4, result: the curve bends, and every miss is the same miss

**Collected 2026-09-05 under `reading-04-plan.md`, committed before any trial ran.** Forty-five
trials, a Haiku client, the read arm, on cells verified byte-identical to reading 3's before the
first trial.

## The curve

| Served bytes | early | middle | late | pooled (n=15) | 95% Wilson |
|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] |
| 20,000 | 5/5 | 3/5 | 4/5 | **12/15** | [0.548, 0.930] |
| 200,000 | 4/5 | 2/5 | 2/5 | **8/15** | [0.301, 0.752] |

Thirty-five hits in forty-five, zero refusals, and the first bend in four readings. The pooled
intervals at 2,000 and 200,000 bytes do not overlap. Five repeats per cell still cannot separate
adjacent cells, but fifteen per size tier can separate the ends of the range, and they do.

**Prediction 1 (lower than 45/45): confirmed. Prediction 2 (bends downward, 200 KB first):
confirmed.** Reading 3's Sonnet client scored 45 of 45 on these exact cells; the ten pairs that
differ all favour Sonnet, which is p = 0.002 on an exact two-sided matched test. This is the reading
the pre-registration asked for — across models — and the second model is not at the ceiling.

## Compliance: 45 of 45, and that refutes a prediction

Every trial made zero search-tool calls and read `served.txt` whole — coverage from each read call's
offset and limit equalled the served row count in all 45, from 47/47 at 2 KB to 3,643/3,643 at
200 KB. No trial was void.

**Prediction 3 said compliance would be lower, with coverage failures at 200 KB. It was wrong.** The
weaker client read every byte it was given at every size. So whatever it got wrong, it did not get
wrong by stopping early, and the misses below are misses by a client that had the whole window in
front of it.

## Every miss is `target+2`

| Trial | Target | Addressed | Offset |
|---|---|---|---|
| 20000-middle-1 | 152 | 154 | +2 |
| 20000-middle-5 | 152 | 154 | +2 |
| 20000-late-1 | 302 | 304 | +2 |
| 200000-early-3 | 726 | 728 | +2 |
| 200000-middle-1 | 1450 | 1452 | +2 |
| 200000-middle-3 | 1451 | 1453 | +2 |
| 200000-middle-4 | 1452 | 1454 | +2 |
| 200000-late-2 | 2904 | 2906 | +2 |
| 200000-late-4 | 2903 | 2905 | +2 |
| 200000-late-5 | 2917 | 2919 | +2 |

Ten of ten. Reading 2's three misses were `target+2`. Reading 3 had none. **Across the 135 read-arm
trials in readings 2, 3 and 4, there are 13 misses and all 13 are at exactly +2.** No trial in any
reading has missed by any other amount, in either direction.

That is a very strong constraint on what the error is. It is not a retrieval failure — every one of
these clients found the right block and wrote the right replacement text. It is not random — thirteen
random misses would not share one offset. It is an ADDRESSING error with a fixed magnitude, whose
FREQUENCY rises with served bytes (0 of 15, 3 of 15, 7 of 15) and falls with model strength (Sonnet 0
of 45 on these cells, Haiku 10 of 45).

**What this reading still cannot say is why it is two.** Every cell serves `@@ 1-N`, so a row count in
the served rendering — whose first two rows carry no line number — and the line number plus two are
the same integer in every trial, exactly as reading 2 noted. Reading 2 named the discriminating
experiment, a cell whose served window does not begin at line 1, and deferred it because a client
missing 3 in 45 would need many trials to show anything. **That is no longer true.** A client missing
7 of 15 at 200 KB would settle the question in a handful of trials, and this reading is the evidence
that the offset-window entry in `docs/adr/BACKLOG.md` should be promoted next.

## What the ten misses do to a plan

Run against each cell's own fixture with the built binary:

| | exit | file |
|---|---|---|
| the client's plan, no guard | **0, all ten** | a comment line replaced with `timeout = 45`; the target untouched two lines above |
| the same address with `anchor="timeout = 30"` | **1, all ten** | nothing written |

Ten of ten apply silently through a green receipt without a guard, and `anchor=` refuses ten of ten.
Reading 2 showed this on three misses; this is the same result on ten, from a different model.

## Cost

Same accounting as readings 2 and 3.

| Served bytes | median spend | vs 2 KB | median peak | vs 2 KB |
|---|---|---|---|---|
| 2,000 | 31,062 | 1.00× | 48,758 | 1.00× |
| 20,000 | 36,469 | 1.17× | 53,910 | 1.11× |
| 200,000 | 91,546 | 2.95× | 109,025 | 2.24× |

**Prediction 4 was half right.** Absolute tokens are lower than Sonnet's at every size (31,062 against
49,050 at 2 KB), as predicted. The ratio is not 2.5× but 2.95× — steeper, not the same — so the claim
that the ratio is a property of what is served and not of who reads it is refuted as stated. The
shape is the same; the slope is not.

## Where this leaves the throughput question

For the stronger client, serving a hundred times more bytes cost 2.5× and lost nothing. For the weaker
client it costs 2.95× and **loses 47 percentage points of accuracy at 200 KB** — but not by failing to
read, and not by finding the wrong block. It loses them to one fixed addressing error that a content
guard catches every time. So the honest answer to "does max throughput hurt or behave" now has two
halves: it behaves for the stronger client, and for the weaker one it hurts in exactly one way, which
mrw's `anchor=` turns from a silent wrong write into a refusal.

## What can be recomputed from this repository, and what cannot

- **The table and the tally: yes.** `curve tally docs/curve/reading-04-scores/*.score.json`
  reproduces `reading-04-tally.json`, and every offset in the miss table is `changed - target` from
  the committed score files.
- **The paired comparison with reading 3: yes.** The cells were verified byte-identical before the
  first trial, and all 45 planted lines match between `reading-03-scores/` and `reading-04-scores/`.
- **Compliance, coverage and cost: no.** From transcripts and request records that are not
  committed. Reported here rather than reproducible, as in readings 2 and 3.
