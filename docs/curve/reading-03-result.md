# Reading 3, result: 45 of 45 on the fixture built to be failed, and the prediction was wrong

**Collected 2026-09-05 under `reading-03-plan.md`, committed before any cell of this reading was
generated.** Forty-five trials, one client population (fresh Sonnet subagents), read arm only.

## The headline is a wrong prediction, and that is the point of having written one

The plan predicted, before collection:

> **The overall rate will be lower than reading 2's 42/45.** If it is not, the relational fixture is
> not meaningfully harder and T2/T3 bought nothing measurable — a real and publishable outcome.

It was not lower. It was **higher: 45 of 45, against reading 2's 42 of 45.** So the second half of
that sentence is what this reading reports. The relational fixture did not make the task measurably
harder for this client, and the honest conclusion is the one the prediction committed to in advance
rather than the one that would be more flattering to two tasks of work.

## The curve

| Served bytes | early | middle | late |
|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 |
| 20,000 | 5/5 | 5/5 | 5/5 |
| 200,000 | 5/5 | 5/5 | 5/5 |

Every cell 1.00, 95% Wilson interval [0.566, 1.000]. Zero misses, zero refusals. Flat, and flat at
the top.

## Compliance, checked per trial and both halves

All 45 trials used **zero** search-tool calls and read `served.txt` **whole** — coverage computed by
unioning each read call's offset and limit against the served row count, and equal to it in every
trial (47/47 at 2 KB up to 3,643/3,643 at 200 KB). No trial was void.

## The comparison is paired, which makes the null sharper

Readings 2 and 3 share sizes, positions, seeds and distractor count; only the SELECTOR differs. The
fixtures are the same layout, so the planted line is the same line — 725, 1450, 2898 and so on appear
in both. The two readings can therefore be compared trial by trial rather than cell by cell.

**The three trials that missed in reading 2 all hit in reading 3:**

| Trial | Reading 2 | Reading 3 |
|---|---|---|
| 2000-late-3 | miss, addressed 41, target 39 | hit, 39 |
| 20000-early-1 | miss, addressed 78, target 76 | hit, 76 |
| 200000-late-5 | miss, addressed 2919, target 2917 | hit, 2917 |

All three of reading 2's misses were `target+2`, and reading 3 has none. Two readings of five repeats
cannot separate "the relational task removed an addressing error" from "three misses in 90 trials is
noise that fell in reading 2". **This reading does not claim the first**, and the second is at least
as consistent with the data.

## Cost

Same accounting as reading 2: duplicates collapsed by message id, **spend** is
`input + cache_creation + output` summed and excludes cache reads, **peak** is the largest single
request's `input + cache_creation + cache_read`.

| Served bytes | median spend | vs 2 KB | median peak | vs 2 KB |
|---|---|---|---|---|
| 2,000 | 49,050 | 1.00× | 72,420 | 1.00× |
| 20,000 | 55,426 | 1.13× | 78,852 | 1.09× |
| 200,000 | 123,307 | 2.51× | 146,303 | 2.02× |

Reading 2 gave 1.00× / 1.12× / 2.44× on spend. So the harder instruction costs essentially the same:
**a hundredfold increase in served bytes is about 2.5× the spend and 2.0× the peak context, on both
fixtures.** That replicates the throughput answer on a second task rather than repeating it on the
same one.

## What this reading establishes, and what it does not

**Establishes.** For this client, at these sizes, serving a hundred times more bytes does not
measurably reduce accuracy, on a task whose target is identified by a relation between blocks and not
by a name — and it costs about 2.5×. That is now measured twice, on two different tasks.

**Does not establish.** That the relational fixture is harder: it is not, measurably, and the
pre-registered prediction that it would be is refuted. Nor does it discharge the pre-registration's
across-models requirement; this is one client population, like reading 2.

## Where the ceiling actually is

Two fixtures now sit at or near 100%, so the difficulty this harness can generate is not what limits
the client. Candidates the corpus has not tried, in the order they seem worth trying:

- **More candidates.** Distractors are held at 3 in every reading so far, so the target is one of
  four. The generator takes the count and nothing has varied it; "more context" and "more candidates"
  were deliberately separated, and the second has never been manipulated.
- **A target requiring more than one comparison** — a property over pairs rather than over one field.
- **A weaker client.** Every reading here uses the same model; a smaller one may sit below the
  ceiling and show the curve these three could not.

The instrument's own output is committed: `docs/curve/reading-03-scores/` holds the 45 `curve score`
results and `docs/curve/reading-03-tally.json` is `curve tally` over them.
