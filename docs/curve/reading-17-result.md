# Reading 17, result: the strong client at the ceiling on the thirteen-service fixture, forty-five of forty-five

**Collected 2026-09-05 under `reading-17-plan.md`, committed before any trial ran.** Forty-five
trials, a Sonnet client, the bare tool-result arm, on reading 14's forty-five cells (twelve
distractors, thirteen services), byte-identical.

## Compliance: 45 of 45

Every trial ran its listed commands, in order, and nothing else in Bash: no other command, no
merge, no spill, no Read/Grep/Glob before the Write. Twenty-four delivered their one-line reply
through `SendMessage` after the Write, which this plan names as the harness's reply channel and
not the trial's. **Prediction 4 holds** (at least 40 of 45).

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 14 (strict) | reading 3 (four services, reader) |
|---|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 6/6 | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 7/7 | 15/15 |
| 200,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 2/2 | 15/15 |

Forty-five of forty-five, zero refusals; the pooled interval is [0.921, 1.000]. **Prediction 1
holds** (at least 14 of 15 in every tier); **prediction 2** (a bend with size) was not activated;
**prediction 3** holds trivially (no miss). Against reading 14's forty-five on the same cells,
counting its void trials: reading 14 missed and this reading hit on 1 (`2000-late-5`), the reverse
on 0.

## Cost

| Served bytes | median spend | median peak |
|---|---|---|
| 2,000 | 48,646 | 73,334 |
| 20,000 | 56,940 | 81,641 |
| 200,000 | 117,454 | 142,169 |

From 2 KB to 200 KB: 2.41×. **Prediction 5 holds** (2.2×–3.0×). Beside reading 3's 2.51× through
the reader on the four-service fixture and reading 14's 2.32× over its forty-five.

## What this decides

What reading 14 could not at the pre-registered strength: on a fixture with three times the
candidates, held across a hundredfold in served bytes, the strong client through the bare delivery
is at the ceiling in every tier. The strong client's flat curve is not an artefact of an easy
fixture, on this fixture family. The cap question does not reopen for this client through this
delivery.

## What it does not decide

Forty-five trials at a ceiling say a rate is high, not that it is 1. One client, one fixture
family, one delivery; a thirteen-way relation is harder than a four-way one and not the hardest
fixture there is.

## Provenance

Forty-five scores in `docs/curve/reading-17-scores/`; `reading-17-tally.json` is computed over
them:

```sh
bin/curve tally docs/curve/reading-17-scores/*.score.json
```

Cells regenerate from reading 14's parameters. Compliance and cost come from transcripts that are
not committed.
