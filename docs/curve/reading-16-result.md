# Reading 16, result: the second family at the ceiling at every size, forty-five of forty-five

**Collected 2026-09-05 under `reading-16-plan.md`, committed before any trial ran.** Forty-five
trials, `gpt-5.6-sol` at high reasoning effort through `codex-cli 0.153.0`, the prompt delivery
with the plan's two-line shape shown, on reading 4's forty-five cells.

## Compliance: 45 of 45

Every final message is exactly one parseable plan of one hunk; every process exited 0; none ran a
command in its empty directory. **Prediction 1 holds** (at least 42 parsed): 45.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 15 (shape not shown) | reading 4 (Haiku, reader) | reading 9/8 (Haiku, bare) |
|---|---|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 8 parsed, 8 hit | 15/15 | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 2 parsed, 2 hit | 12/15 | 15/15 |
| 200,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 1 parsed, 1 hit | 8/15 | 15/15 |

Forty-five of forty-five, zero refusals. **Prediction 2 holds** (at least 14 of 15 per tier);
**prediction 3 holds** trivially (no miss). Against reading 4 on the same forty-five cells: 10
discordant pairs, reading 4 missed and this reading hit, the reverse never.

## Cost

Not measured, as the plan said.

## What this decides

A client from a second family, shown mrw's served text with `N|` the only number and the plan's
shape once, addresses the target at every size and every position: 2 KB to 200 KB is flat for it,
as the bare tool result was flat for the weaker Claude client (readings 8, 9) and as both
fixtures were flat for the stronger one (readings 2, 3, and 14 on its strict trials). The account
"the number, not the size" gains a family. With reading 15: what the shape not shown cost was
thirty-four plans in a grammar mrw does not read, and what it cost in addressing was nothing —
every trial in both readings named the target line.

## What it does not decide

One family's one model at one effort setting; a prompt delivery, which is neither a file reader
nor a tool result; cost unmeasured; forty-five trials at a ceiling say a rate is high, not that
it is 1.

## Provenance

Forty-five scores in `docs/curve/reading-16-scores/`; `reading-16-tally.json` is computed over
them. The final messages are not committed.
