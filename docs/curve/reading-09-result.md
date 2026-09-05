# Reading 9, result: flat at the ceiling, at every size, through a delivery with no outer gutter

**Collected 2026-09-05 under `reading-09-plan.md`, committed before any trial ran.** Thirty
trials, a Haiku client, the scripted tool-result arm, on reading 4's 2,000 and 20,000-byte cells,
verified byte-identical; the 200,000-byte tier is reading 8's fifteen trials, pooled as the plan
said and not re-run.

## Compliance: 30 of 30

Every trial ran its two listed commands and nothing else — no other command, no other tool, no
memory call after the Write, no spilled result. Prediction 3 (at least 27 of 30) holds.

## The curve

| Served bytes | early | middle | late | pooled (n=15) | 95% Wilson | reading 4, same cells |
|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 12/15 |
| 200,000 (reading 8) | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 8/15 |

Forty-five of forty-five across the three tiers, zero refusals; the three pooled intervals are the
same interval. **Prediction 1 holds: flat at the ceiling.** Laid beside reading 4's curve on the
same forty-five cells — 15, 12, 8 — the only thing that differs between the two is how the served
text reached the client.

## The pairing with reading 4, per tier

Discordant pairs, read arm missed and this arm hit versus the reverse: 2,000 bytes 0–0;
20,000 bytes 3–0; 200,000 bytes 7–0 (reading 8). Prediction 2 holds: level at 2 KB,
this arm favoured at 20 KB, and at 200 KB. Across the forty-five pairs, 10 discordant, all one way.

## No miss at `target+2`, and no miss at all

Forty-five plans, forty-five at offset 0. The row-index miss that reading 5 identified does not
appear at any size once mrw's gutter is the only gutter.

## Cost

| Served bytes | median spend, this arm | reading 4 | ratio | median peak, this arm | reading 4 |
|---|---|---|---|---|---|
| 2,000 | 31,909 | 31,062 | 1.03× | 50,654 | 48,758 |
| 20,000 | 36,444 | 36,469 | 1.00× | 55,173 | 53,910 |
| 200,000 | 84,789 | 91,546 | 0.93× | 103,428 | 109,025 |

Prediction 4 holds: the ratio from 2 KB to 200 KB through this arm is 2.66×, against reading
4's 2.95×, and each tier's median is within 3% of the read arm's. The same bytes cost the same
through either delivery — which is what a delivery-not-size account predicts.

## What this decides

For the weaker client, served size did not bend the curve at any size measured; the delivery did.
With readings 5 and 8 this closes the question reading 4 opened: the bend was the harness read
arm's second gutter, and through a tool-result delivery the weaker client is at the ceiling from
2 KB to 200 KB at the same cost per byte the read arm paid. The cap stays; no served-path change.

## What it does not decide

One client (Haiku), one fixture family, five repeats per cell, a Bash-result delivery; the MCP
delivery was not run. Fifteen trials at a ceiling say a rate is high, not that it is 1: the pooled
interval's lower bound at each tier is 0.796. The 200 KB tier is reading 8's data, carried over
under a plan that said so, not an independent replication.

## Provenance

Thirty scores in `docs/curve/reading-09-scores/`; `reading-09-tally.json` is computed over those
and `reading-08-scores/` together, and the tables recompute from the three score directories
(4, 8, 9). Compliance and cost come from transcripts that are not committed.
