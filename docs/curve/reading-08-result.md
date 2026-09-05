# Reading 8, result: with mrw's gutter the only gutter, the weaker client is at the ceiling

**Collected 2026-09-05 under `reading-08-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the scripted tool-result arm, on reading 4's fifteen 200,000-byte cells,
byte-identical.

## Compliance: 15 of 15

Every trial ran the thirteen listed commands and nothing else — no merge to tolerate, no search,
no pipe, no other range, no spilled result, no other tool, and no memory call after the Write.
Prediction 1 (at least 12 of 15) holds with margin. The two readings that voided on this arm are
recorded in `reading-06-result.md` and `reading-07-result.md`; this one needed neither tolerance
it pre-registered.

## The tally

| Served bytes | early | middle | late | pooled (n=15) | 95% Wilson |
|---|---|---|---|---|---|
| 200,000, tool-result arm | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] |

Fifteen of fifteen, zero refusals. The interval separates from reading 4's [0.301, 0.752] on the
same cells.

## The pairing with reading 4

| Trial | Reading 4 (read arm) | Reading 8 (tool-result arm) | Offset here |
|---|---|---|---|
| 200000-early-1 | hit | hit | +0 |
| 200000-early-2 | hit | hit | +0 |
| 200000-early-3 | miss | hit | +0 |
| 200000-early-4 | hit | hit | +0 |
| 200000-early-5 | hit | hit | +0 |
| 200000-late-1 | hit | hit | +0 |
| 200000-late-2 | miss | hit | +0 |
| 200000-late-3 | hit | hit | +0 |
| 200000-late-4 | miss | hit | +0 |
| 200000-late-5 | miss | hit | +0 |
| 200000-middle-1 | miss | hit | +0 |
| 200000-middle-2 | hit | hit | +0 |
| 200000-middle-3 | miss | hit | +0 |
| 200000-middle-4 | miss | hit | +0 |
| 200000-middle-5 | hit | hit | +0 |

Discordant pairs: 7 where the read arm missed and this arm hit, 0 the other way.
Exact two-sided sign test on the 7 discordant pairs: p = 0.0156. Prediction 3 holds.

## No miss at `target+2`, and no miss at all

Prediction 2 holds. With no outer gutter — the served text arriving as the Bash tool's result, mrw's
`N|` the only number on any row — the client that missed seven of these fifteen through its file
reader, every one at the row index, addressed every target exactly.

## What this decides

Taken with reading 5, the account is now measured rather than suggested — for this client, on these
fifteen 200 KB cells, through a Bash-result delivery; the MCP path was not run:

- **The `target+2` of readings 2–4 was the harness read arm's delivery.** Reading 5 showed the miss
  is the row index of the served text (−117 from line 120). Reading 8 shows that when mrw's rendering
  is the only thing numbering the rows — the served text as a Bash tool result — the same client on
  the same cells does not count rows: it reads the number. mrw's own two unnumbered rows do not
  induce the miss on this delivery. The MCP tool-result path was not run; it carries no outer
  numbering either, and that is a statement about the transport, not a measurement.
- **The served format is not changed.** No served-path record opens; the BACKLOG read-format entry
  closes with no engine change.
- **The cap is not changed on this evidence.** ADR-011-T3's 200,000 stays as it is. The bend in
  reading 4 was not a failure to read or to find at 200 KB; it was a second gutter, and the tested
  delivery has none.
- **The throughput answer, scoped.** For the strong client, 100× the bytes cost 2.5× the tokens and
  lost nothing (readings 2, 3, three sizes). For the weaker client this arm ran at 200 KB only: 15 of
  15 at the same cost as the read arm's 200 KB trials. A served-size curve within this arm was not
  taken; the harm measured in reading 4 was the read arm's, and it is reported as such.
- **The stability claim rests on readings 3, 5 and 8 together**, each scoped to what it ran: a
  strong client at the ceiling on the relational fixture at three sizes, the miss identified as a
  row index, and the weaker client at the ceiling at 200 KB through a Bash-result delivery with no
  outer gutter.

## What it does not decide

Reading 4's read-arm result stands as a fact about that delivery: a client that saves mrw's output
to a file and reads it back through a numbering viewer recreates the collision, and the weaker
client then takes the first gutter. That path is real but not the tool's; it is documented, not
fixed. And the ceiling here is one client on one fixture at one size, fifteen trials: a second
client or a harder fixture through this arm is a budget decision, not a gap in this claim.

## Cost

| Arm | median spend | median peak |
|---|---|---|
| Reading 4, read arm | 91,546 | 109,025 |
| Reading 8, tool-result arm | 84,789 | 103,428 |

Prediction 4 (within 20% of reading 7's 84,972) holds. The same bytes cost the same through either
delivery.

## Provenance

Scores in `docs/curve/reading-08-scores/`, tally in `docs/curve/reading-08-tally.json`; the tally,
the interval and the pairing recompute from those and `docs/curve/reading-04-scores/`. Compliance
and cost come from transcripts that are not committed. Cells regenerate from reading 4's parameters
and are byte-identical to it.
