# Reading 5, result: the miss is a row index, and the reading that decides whose is named

**Collected 2026-09-05 under `reading-05-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the read arm, the 200,000-byte tier, every window served from line 120.

## The tally

| Served bytes | early | middle | late | pooled (n=15) | 95% Wilson |
|---|---|---|---|---|---|
| 200,000, window from line 120 | 3/5 | 4/5 | 5/5 | **12/15** | [0.548, 0.930] |

Twelve hits in fifteen, zero refusals. Reading 4's 8/15 at this tier has interval [0.301, 0.752];
the two overlap, so the offset window did not measurably change the rate either way — prediction 3
holds. The tally is not what this reading is for.

## Every miss is `target − 117`, and none is `target + 2`

| Trial | Target | Addressed | Offset | H1 predicted (target − 117) | H2 predicted (target + 2) |
|---|---|---|---|---|---|
| 200000-early-4 | 751 | 634 | -117 | 634 | 753 |
| 200000-early-5 | 753 | 636 | -117 | 636 | 755 |
| 200000-middle-4 | 1502 | 1385 | -117 | 1385 | 1504 |

Three of three at the row-numbering prediction, zero at the intrinsic one, 119 lines apart.
**H1 is confirmed and H2 is refuted.** Predictions 1 and 2 hold. Across readings 2–5 that is 16
misses in 150 read-arm trials, and every one of them is the row index of the served text — the row
the target sits on, counted from the `==>` header — not the line number mrw printed beside it.

## What the client actually saw, from the transcript

The read arm delivers `served.txt` through the client's own file reader, which numbers what it
shows. At the target row of `200000-early-4` the client saw:

```
634	  751| timeout = 30
```

Two numbers on one row: the harness reader's row number first, mrw's line number second. The
client's own summary before it wrote the plan named the right service with the wrong number —
*"svc-nofdh: retries = 6 (line 634: timeout = 30)"* — and addressed 634. It found the right block
and wrote the right text. That it READ the first number rather than counted rows is what this
excerpt suggests, and no more: the scores cannot tell reading a gutter from counting rows, and the
transcript is not committed. With a window served from line 1 the row index is always mrw's number
plus two, because two rows above the first numbered line carry no number; that is the whole `+2`
of readings 2, 3 and 4.

## What this decides, and what it only suggests

- **The miss is the row number of the served text.** That is what the scores establish: with the
  window from line 1 every miss was the target's row counted from the `==>` header (+2), and with
  the window from line 120 every miss is that same row index (−117). mrw's two unnumbered rows add
  2 to the row index in both cases; they set the offset's magnitude, and the window start sets the
  rest.
- **Which gutter the client read is suggested, not established.** Two accounts give the row index:
  a client that reads the number its own file reader lays beside each row, and a client that counts
  mrw's rows itself. The transcript excerpt above — the reader's `634` beside mrw's `751|`, and the
  client quoting 634 as a "line" — points at the first, and the author finds it persuasive; but a
  transcript is not committed, and a reported excerpt is not a score. What separates the two is a
  delivery with no outer gutter, and that is the next reading, not this one.
- **The current evidence does not justify changing the cap or the served format.** Every miss in
  four readings matches one row-index account; none is a failure to read 200 KB (coverage is whole
  in every trial) or to find the block (every miss wrote the right text for the right service). So
  ADR-011-T3's revisit of 200,000 is deferred rather than ruled out — whether the observed bend
  transfers to the real delivery path is what the gutter-free reading decides — and no served-path
  record is opened until that reading says whether mrw's own rendering induces the row count when
  nothing else numbers the rows.
- **Readings 2–4 were collected through a delivery that adds a gutter.** The read arm was chosen in
  reading 2 to stop clients searching instead of reading, and it did that; it also put the reader's
  numbering in front of every client. The curve those readings report is real for that delivery
  path. What it says about the real one — a Bash result, an MCP tool result, where mrw's gutter is
  the only one — is not measured, and is not claimed here.

## What it does not decide

It does not show that Haiku is at the ceiling over the real delivery path; that is a reading, not an
inference. The reading that settles it delivers the served text to the client WITHOUT an outer
gutter — a Bash arm restricted by transcript to `cat served.txt`, or `mrw_read` over MCP, which is
the product's own path — at 200 KB with the same client. If that comes back at or near 15/15, the
bend was the harness's; if it bends again, the miss has a second cause this reading did not see.
That is the next reading, and it is the one that licenses a stability claim.

## Compliance: 15 of 15

Every trial made zero search-tool calls and read `served.txt` whole — coverage from each read call's
offset and limit equalled the served row count in all 15. The plan quoted one row count, 3,615, and
that is seed 1's; the five seeds pad differently and serve 3,615, 3,625, 3,630, 3,632 and 3,645 rows
(two unnumbered each), which the plan did not say and this does. Prediction 4 holds. No trial was
void.

## The guard, again

All three misses apply silently through a green receipt without a guard (exit 0) and all three are
refused with `anchor="timeout = 30"` (exit 1), run against each cell's own fixture with the built
binary after a read of the served window, and the fixtures restored. Sixteen of sixteen across
readings 2–5. The case for the guard does not depend on whose gutter the client took.

## Cost

Same accounting as readings 2–4: spend is input + cache-creation + output summed over the trial's
requests, peak is the largest single request's context.

| Served bytes | median spend | median peak | reading 4 at 200 KB |
|---|---|---|---|
| 200,000, from line 120 | 90,187 | 108,882 | 91,546 / 109,025 |

The same tier costs the same whether the window starts at 1 or at 120, as it should.

## Provenance

Scores in `docs/curve/reading-05-scores/`, tally in `docs/curve/reading-05-tally.json`; both
recompute the tables above. The transcript excerpt, compliance, coverage and cost come from
transcripts and request records that are not committed, and are reported rather than recomputable.
Cells regenerate from the plan's parameters; trial ids carry `from=120`.
