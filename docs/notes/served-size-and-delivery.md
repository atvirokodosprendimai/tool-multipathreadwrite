# Served size did not bend the curve; the delivery did

*A measurement note on how much text an AI coding agent can be handed before its edits go wrong,
and what actually made them go wrong. mrw v1.0.0, 2026-09-05.*

## Abstract

mrw serves a client many ranges of many files in one call and applies many edits in one plan,
refusing any plan whose hunks cannot all land. Its result cap is 200,000 characters. Nothing in
the tool's design said whether that number was right, and the common instinct in the field —
serve about ten thousand tokens and stop — had never been measured against the alternative. We
built an instrument (`curve`) that generates a fixture, records exactly what mrw would serve, has
a fresh client author one plan against it, and scores the plan by applying it: the line that
changed is the measurement. Under a criterion pre-registered before any cell existed, nine
readings were taken, each plan committed before its trials and every score file committed.

Two results. First, for a strong client, serving a hundred times more bytes cost about 2.5× the
tokens and lost nothing, on two fixtures. Second, and the one we did not expect: for a weaker
client the curve bent — 15, 12, 8 of 15 across 2 KB, 20 KB, 200 KB — and every one of its misses,
thirteen of thirteen across three readings, addressed the same wrong line, exactly two below the
target. Two further readings identified that offset as the row index of the served text as the
client's own file reader numbered it, laid beside mrw's numbers; delivering the identical text as
a tool result, with mrw's numbers the only numbers, took the same client on the same cells to
15, 15, 15 at the read arm's cost or below it. Served size did not bend the curve. A second gutter did.

## 1. The question

`mrw read` renders each served range as numbered rows under a header:

```
==> services.conf  3621L  200039B  sha 5a9440d4
@@ 1-3621
    1| # service registry — every service has the same shape
    2| ...
```

A plan then addresses a line by the number in that gutter: `@@ services.conf 2898 replace`. The
cap on a served result is 200,000 characters (ADR-011). The question was the throughput one: as
the served window grows toward that cap, does a client's ability to address the right line hold,
degrade, or collapse — and at what token cost?

## 2. The instrument

`curve generate` writes a fixture (`services.conf`: many `[service …]` blocks, each with a
`retries` and a `timeout` line, padded with inert comment text to a byte target), records what mrw
serves for it (`served.txt`, measured), and plants one target: the `timeout` line of the one
service whose retry budget differs from every other's. The client is told only that relation —
never the service's name or its line — and must author one plan with one `replace` hunk.
`curve score` applies the plan to the fixture and reports which line changed. `curve tally`
groups by served bytes and target position and reports each cell as a rate with a Wilson interval.

The criterion was written into the backlog before the generator existed: correct-address rate
against served bytes, stratified by target position (early, middle, late), refusals reported
beside the cell and never in it, five repeats per cell, and **a flat curve accepted as an answer**.

Each reading's plan — cells, client, arm, predictions — is committed before its first trial and
never edited after; corrections live in the result document. Compliance is verified from each
trial's transcript, matched on the prompt marker, never on the cell id: which tools were called,
whether the served text was read whole, whether anything searched. A non-compliant trial is void
and reported, never counted as a miss.

## 3. The readings

| Reading | Client | Delivery | 2 KB | 20 KB | 200 KB | What it settled |
|---|---|---|---|---|---|---|
| 1 | Sonnet | any tools | — | — | — | Void: clients searched for the target's name instead of reading. |
| 2 | Sonnet | file reader, no search | 14/15 | 14/15 | 14/15 | Flat at a ceiling. Three misses, each at target+2. |
| 3 | Sonnet | file reader | 15/15 | 15/15 | 15/15 | Relational fixture: predicted harder, was not. |
| 4 | Haiku | file reader | 15/15 | 12/15 | 8/15 | The curve bends. Ten misses, all at target+2. |
| 5 | Haiku | file reader, window from line 120 | — | — | 12/15 | Every miss at target−117: the row index of the served text. |
| 6, 7 | Haiku | tool result | — | — | (15/15) | Void under their own compliance rules; reported, not counted. |
| 8 | Haiku | tool result | — | — | 15/15 | mrw's gutter the only gutter: no miss. |
| 9 | Haiku | tool result | 15/15 | 15/15 | (reading 8) | Flat at the ceiling at every size. |

Every score file is in the repository under `docs/curve/reading-0N-scores/`; every table here
recomputes from them.

## 4. Result A: the miss was a row index, and the row index was the reader's

Across readings 2, 3 and 4 — 135 trials in which a client read the served text through its file
reader — there were 13 misses, and all 13 changed the line two below the target. Each had found
the right service and written the right replacement text. Thirteen misses do not share one offset
by chance, and the frequency rose with served bytes and fell with model strength, so it looked
like a property of the served text at scale.

Two accounts fit. mrw's served rendering opens with two rows that carry no line number; a client
that counted rows from the top would land at target+2. Or the client was reading a number from
somewhere else. Every window in those readings began at line 1, so the two accounts predicted the
same integer everywhere and could not be told apart.

Reading 5 served the window from line 120. If the miss was a row count, it would now sit at
target−117; if it was intrinsic, at target+2. Three misses in fifteen, all at exactly target−117.
The transcript showed the row as the client saw it:

```
634	  751| timeout = 30
```

Two numbers on one row. The first is the harness's: the read arm delivers `served.txt` through
the client's file reader, which numbers what it shows. The second is mrw's. The client took the
first and called it "line 634" in its own summary.

Reading 8 removed the first number: the same fifteen 200 KB cells, the same client, the served
text arriving as a Bash tool result in listed `sed -n` ranges, so that mrw's `N|` was the only
number on any row. Fifteen of fifteen, compliant under a rule pre-registered from two voided
attempts, no plan anywhere near +2. Against reading 4 on the same cells: seven discordant pairs,
all one way, exact two-sided sign test p = 0.0156. Reading 9 took the same arm through 2 KB and
20 KB: 15 and 15. The forty-five pairs against reading 4 are 0, 3 and 7 discordant, every one in
the same direction.

What this is and is not. It is a measurement that, for this client on this fixture family, the
only recurring miss was induced by a second numbering laid over mrw's by the delivery, and that
removing the second numbering removed the miss at every size. It is not a claim about the MCP
transport, which was not run, nor about clients or fixtures not measured. And reading 4 stands as
a fact about a real path: a client that saves a tool's numbered output to a file and reads it back
through a numbering viewer will meet two gutters, and a weaker client takes the first.

## 5. Result B: what a hundred times more bytes costs

Spend is input plus cache-creation plus output tokens summed over a trial's requests, medians per
tier; peak (the largest single request's context) is in the result documents.

| Client, delivery | 2 KB | 20 KB | 200 KB | 200 KB ÷ 2 KB |
|---|---|---|---|---|
| Sonnet, file reader, named fixture (reading 2) | 48,680 | 54,412 | 118,879 | 2.44× |
| Sonnet, file reader, relational fixture (reading 3) | 49,050 | 55,426 | 123,307 | 2.51× |
| Haiku, file reader (reading 4) | 31,062 | 36,469 | 91,546 | 2.95× |
| Haiku, tool result (readings 8, 9) | 31,909 | 36,444 | 84,789 | 2.66× |

The fixed cost of a session dominates: a hundredfold increase in served bytes is a two-and-a-half
to three-fold increase in tokens. The "serve ten thousand tokens and stop" instinct buys little,
because the window is not where the tokens go until it is very large. The same bytes cost the same
through either delivery — within 3% at 2 KB and 20 KB, 7% less at 200 KB — which is what a
delivery-not-size account predicts.
(Cost figures come from request records that are not committed; the result documents say so.)

## 6. What the method cost, and what it bought

Three readings were void under their own rules. Reading 1 because clients given a search tool
searched for the target's name instead of reading, so served bytes were never manipulated.
Readings 6 and 7 because a weaker client given a shell and a rule — read in ranges under a cap, no
search — broke the rule four different ways, and given the exact commands merged two of them. The
rule that finally held was written from how the client behaves and pre-registered before the run
that counted. Each void is recorded with its observations; none is in a table.

Two plan files were edited after collection during this series — a wrong count corrected, a dated
note added — and both edits were reverted at review, because a plan that can be edited whenever the
edit looks harmless is not a pre-registration. Every plan in the series is byte-identical to the
commit that added it. And every number in every result document is computed by the script that
reads the scores; the two numbers in this series that were typed from memory were both wrong.

## 7. Limits

One fixture family, five repeats per cell, two clients from one vendor; the strong client sits at
the ceiling on both fixtures, so its curve cannot bend and says nothing about where it would. The
weaker client's ceiling through a tool result is fifteen of fifteen per tier, whose interval's
lower bound is 0.796, not 1. Reading 9's 200 KB tier is reading 8's data, pooled under a plan that
said so. Reading 8's arm differs from reading 4's in two things — the gutter is gone, and the text
arrives in `sed` ranges rather than a reader's windows; the gutter is the parsimonious account,
because reading 5 put the misses exactly where it predicted, but the two were not separated. The
MCP tool-result path, which carries no outer numbering either, was not measured.

## 8. Reproduction

```sh
go build -o bin/curve ./cmd/curve
bin/curve generate -out CELL -bytes 200000 -position late -distractors 3 -seed 1 -selector odd-retries
bin/curve tally docs/curve/reading-09-scores/*.score.json docs/curve/reading-08-scores/*.score.json
```

Cells regenerate deterministically from their parameters; a reading's plan names them. The
transcripts and request records that compliance and cost come from are not committed.
