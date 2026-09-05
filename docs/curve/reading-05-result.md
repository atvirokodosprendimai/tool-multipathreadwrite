# Reading 5, result: the miss is the outer gutter, and it is the harness's, not mrw's

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
misses in 150 read-arm trials, and every one of them is the row number of the served text where the
client's reader put the target, not the line number mrw printed beside it.

## What the client actually saw, from the transcript

The read arm delivers `served.txt` through the client's own file reader, which numbers what it
shows. At the target row of `200000-early-4` the client saw:

```
634	  751| timeout = 30
```

Two numbers on one row: the harness reader's row number first, mrw's line number second. The
client's own summary before it wrote the plan named the right service with the wrong number —
*"svc-nofdh: retries = 6 (line 634: timeout = 30)"* — and addressed 634. It found the right block,
wrote the right text, and took the first gutter. With a window served from line 1 the first gutter
is always mrw's number plus two, because two rows above the first numbered line carry no number;
that is the whole `+2` of readings 2, 3 and 4.

## What this decides

- **The cap is not the knob.** The bend in reading 4 is not a retrieval or a reading failure at
  200 KB; it is an addressing choice between two gutters, and its frequency with size is the
  frequency of that choice. ADR-011-T3's revisit of 200,000 is not warranted on this evidence.
- **mrw's served format is not the cause either.** The candidate named in reading 4 and in the
  README — that the tool's own two unnumbered rows induce the miss — is refuted as stated: those two
  rows fix the SIZE of the offset when the window starts at 1 (+2) and do nothing when it starts at
  120 (−117). What induces the miss is a second numbering laid over mrw's by the reader the harness
  hands the text through. That numbering does not exist when mrw's output reaches a client as tool
  output — a Bash result, or an MCP tool result — which is every real delivery path this tool has.
- **So readings 2–4 measured the harness's delivery as much as the client.** The read arm was chosen
  in reading 2 to stop clients searching instead of reading, and it did that; it also put a second
  gutter in front of every client, and the weaker client took it 10 times in 45. The curve those
  readings report is real for that delivery path and is an upper bound on the harm for the real one.

## What it does not decide

It does not show that Haiku is at the ceiling over the real delivery path; that is a reading, not an
inference. The reading that settles it delivers the served text to the client WITHOUT an outer
gutter — a Bash arm restricted by transcript to `cat served.txt`, or `mrw_read` over MCP, which is
the product's own path — at 200 KB with the same client. If that comes back at or near 15/15, the
bend was the harness's; if it bends again, the miss has a second cause this reading did not see.
That is the next reading, and it is the one that licenses a stability claim.

## Compliance: 15 of 15

Every trial made zero search-tool calls and read `served.txt` whole — coverage from each read call's
offset and limit equalled the served row count in all 15 (3615/3615, 3625/3625, 3630/3630, 3632/3632, 3645/3645 rows across the five
seeds). Prediction 4 holds. No trial was void.

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
