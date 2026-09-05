# Reading 14, result: the strong client stays at the ceiling on the thirteen-service fixture

**Collected 2026-09-05 under `reading-14-plan.md`, committed before any trial ran.** Forty-five
trials, a Sonnet client, the bare tool-result arm, on forty-five new cells with twelve distractors
(thirteen `[service …]` blocks each), window from line 1.

## Corrections to the plan's own numbers, recorded here and not there

The plan quoted row counts from probe cells rather than from the generated set: the cells as
generated serve 71–73 rows at 2 KB, 395–403 at 20 KB and 3,644–3,670 at 200 KB (the plan said 72,
385–395 and 3,662–3,678). Every 200 KB cell took thirteen ranges plus the tail — fourteen listed
commands with `cat task.json`, where the plan said "twelve or thirteen … plus a tail". The cells
themselves are what the plan's parameters generate; only the prose counts were typed.

## Compliance: 45 of 45 under the rule as read here; 15 of 45 under the rule as written

Every trial ran its listed commands, in order, and nothing else in Bash: no other command, no merge,
no spill, no Read/Grep/Glob before the Write. Thirty trials, after the Write, called `SendMessage`
to deliver their one-line reply — the harness's reply channel, which this client uses where the
weaker one printed the line as its final text. The pre-registered rule names memory-tool calls
after the Write as the harness's protocol and not the trial's, and does not name `SendMessage`,
because no earlier reading's client used it. Reading it as the same category — a call after the
Write that carries nothing into the plan — gives 45 of 45; reading the rule literally gives 15 of
45 and fails prediction 4. **The interpretation was made after collection and is reported as such.**
The fifteen trials that are compliant either way are 5, 6 and 4 per tier at 2 KB, 20 KB and 200 KB,
and all fifteen hit; the tables below are over all forty-five, with that caveat standing.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 3 (four services, reader) |
|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 4/5 | **14/15** | [0.702, 0.988] | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 |
| 200,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 |

Forty-four of forty-five; the pooled interval is [0.884, 0.996]. **Prediction 1 holds:** at least
14 of 15 in every tier. **Prediction 2** (a bend with size) was not activated.

## The one miss is on the right service, one line above

`2000-late-5`: the target is line 69, `timeout = 30` of the one service with `retries = 5`; the plan
addressed line 68, that service's `retries` line. Right service, wrong line, offset −1. Not at
`target+2` — this arm has no second number — so **prediction 3 holds** as stated: the miss is
classified as an off-by-N on the right service, not a wrong-service miss.

## Cost

| Served bytes | median spend | median peak |
|---|---|---|
| 2,000 | 50,889 | 75,425 |
| 20,000 | 54,212 | 78,771 |
| 200,000 | 117,879 | 142,428 |

From 2 KB to 200 KB: 2.32×. **Prediction 5 holds** (2.2×–3.0×). Beside reading 3's 2.51× through
the reader on the four-service fixture.

## What this decides

On a fixture with three times the candidates, held across a hundredfold in served bytes, the
strong client through the bare delivery is at the ceiling in every tier, with one off-by-one at the
smallest size. The strong client's flat curve is not an artefact of an easy fixture, on this
fixture family. The cap question does not reopen for this client through this delivery.

## What it does not decide

Forty-five trials at a ceiling say a rate is high, not that it is 1. One client, one fixture
family, one delivery. The compliance caveat above: thirty of the forty-five rest on reading a
post-Write `SendMessage` as the harness's reply and not the trial's; on the fifteen that need no
such reading, 15 of 15.

## Provenance

Forty-five scores in `docs/curve/reading-14-scores/`; `reading-14-tally.json` is computed over
them:

```sh
bin/curve tally docs/curve/reading-14-scores/*.score.json
```

Cells regenerate from the plan's parameters. Compliance and cost come from transcripts that are
not committed.
