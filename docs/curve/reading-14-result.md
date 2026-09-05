# Reading 14, result: evidence-limited — fifteen strictly compliant trials, all hits; forty-five as a sensitivity analysis

**Collected 2026-09-05 under `reading-14-plan.md`, committed before any trial ran.** Forty-five
trials, a Sonnet client, the bare tool-result arm, on forty-five new cells with twelve distractors
(thirteen `[service …]` blocks each), window from line 1.

## Corrections to the plan's own numbers, recorded here and not there

The plan quoted row counts and byte ranges from probe cells rather than from the generated set:
the cells as generated serve 71–73 rows at 2 KB, 395–403 at 20 KB and 3,644–3,670 at 200 KB (the
plan said 72, 385–395 and 3,662–3,678), and 2,009–2,063, 20,006–20,045 and 200,002–200,049 bytes
(the plan said about 2,009, 20,006–20,045 and 200,002–200,037). Every 200 KB cell took thirteen ranges plus the tail — fourteen listed
commands with `cat task.json`, where the plan said "twelve or thirteen … plus a tail". The cells
themselves are what the plan's parameters generate; only the prose counts were typed.

## Compliance: 15 of 45. The reading is evidence-limited

Every trial ran its listed commands, in order, and nothing else in Bash: no other command, no merge,
no spill, no Read/Grep/Glob before the Write. Thirty trials, after the Write, called `SendMessage`
to deliver their one-line reply — the harness's reply channel, which this client uses where the
weaker one printed the line as its final text. The pre-registered rule exempts memory-tool calls
after the Write and nothing else; it does not name `SendMessage`, because no earlier client used it.
Under the rule as written those thirty trials are void, **prediction 4 fails** (15 of 45 against
40 required), and the fifteen that remain — 6, 7 and 2 per tier at 2 KB, 20 KB and 200 KB — do not
fill the pre-registered five-per-cell design. A reading of the rule that admits them was considered
after collection and is not applied: a post-hoc exemption cannot carry a pre-registered result.
**Reading 17 re-runs this arm under a plan that names the reply channel before any trial.**

## The strict result: 15 of 15

| Served bytes | strict trials | hits |
|---|---|---|
| 2,000 | 6 | 6 |
| 20,000 | 7 | 7 |
| 200,000 | 2 | 2 |

Fifteen of fifteen. Prediction 1 (at least 14 of 15 per tier) is **not decidable** on these
counts; prediction 3 (no miss at `target+2`) holds trivially; prediction 5 (cost ratio) is computed
over all forty-five below and is therefore sensitivity, not result.

## Sensitivity: all forty-five, the thirty void trials included

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 3 (four services, reader) |
|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 4/5 | **14/15** | [0.702, 0.988] | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 |
| 200,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 |

Forty-four of forty-five; the pooled interval is [0.884, 0.996]. On this set prediction 1's
threshold is met in every tier and prediction 2 (a bend with size) is not activated — as a
description of what the forty-five did, not as the pre-registered result.

## The one miss is on the right service, one line above

`2000-late-5` (a void trial): the target is line 69, `timeout = 30` of the one service with
`retries = 5`; the plan addressed line 68, that service's `retries` line. Right service, wrong
line, offset −1. Not at `target+2` — this arm has no second number. Classified as the plan
required: an off-by-N on the right service, not a wrong-service miss.

## Cost

| Served bytes | median spend | median peak |
|---|---|---|
| 2,000 | 50,889 | 75,425 |
| 20,000 | 54,212 | 78,771 |
| 200,000 | 117,879 | 142,428 |

From 2 KB to 200 KB: 2.32×, inside prediction 5's band (2.2×–3.0×), over all forty-five — a
sensitivity figure. Beside reading 3's 2.51× through the reader on the four-service fixture.

## What this decides

Less than it measured. Fifteen strictly compliant trials, all hits, on a fixture with three times
the candidates, say nothing against the strong client's flat curve and cannot confirm it under the
pre-registered design. The forty-five say the same with the thirty void trials counted, which is
evidence of the ordinary kind and not of the pre-registered kind. Reading 17 is the reading this
one was meant to be.

## What it does not decide

Whether the flat curve survives a harder fixture, at the pre-registered strength. One client, one
fixture family, one delivery; and a rule that did not foresee a client's reply channel.

## Provenance

Forty-five scores in `docs/curve/reading-14-scores/`; `reading-14-tally.json` is computed over
them:

```sh
bin/curve tally docs/curve/reading-14-scores/*.score.json
```

Cells regenerate from the plan's parameters. Compliance and cost come from transcripts that are
not committed.
