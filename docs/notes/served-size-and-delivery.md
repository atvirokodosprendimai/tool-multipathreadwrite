# Served size did not bend the curve; the delivery did

*A measurement note on how much text an AI coding agent can be handed before its edits go wrong,
and what actually made them go wrong. mrw v1.0.0, 2026-09-05.*

## Abstract

mrw serves a client many ranges of many files in one call and applies many edits in one plan,
refusing any plan whose hunks cannot all land. Its MCP read tool advertises a 200,000-character
result cap, enforced as 200,000 bytes (the CLI is uncapped). Nothing in the tool's design said
whether that number was right, and the common instinct in the field —
serve about ten thousand tokens and stop — had never been measured against the alternative. We
built an instrument (`curve`) that generates a fixture, records exactly what mrw would serve, has
a fresh client author one plan against it, and scores the plan by applying it: the line that
changed is the measurement. Under a criterion pre-registered before any cell existed, eleven
readings were taken, each plan committed before its trials, and every score file of the ten
scored readings committed (reading 1 is a void notice, not scores).

Two results. First, for a strong client, serving a hundred times more bytes cost 2.4–2.5× the
tokens with no measurable reduction in correct addressing at these sizes, on two fixtures. Second,
and the one we did not expect: for a weaker client the curve bent — 15, 12, 8 of 15 across 2 KB,
20 KB, 200 KB — and its ten misses all addressed the same wrong line, exactly two below the target;
across readings 2–4, all thirteen misses by either client shared that offset. Two further readings
identified the offset as the row index of the served text as the client's own file reader numbered
it, laid beside mrw's numbers. Delivering the identical text as a tool result — with mrw's numbers
the only numbers, and in `sed` ranges rather than a reader's windows — took the same client on the
same cells to 15, 15, 15, at a cost within 3% of the read arm's at 2 KB and 20 KB and 7% lower at
200 KB. Two more readings separated the two things that delivery changed: the same ranges with a
second number that restarts per range left the client at 14 of 14; the same ranges with a second
number equal to the reader's — the row index from the top — took it back to 10 of 15, every miss
at exactly that number. Served size did not bend the curve. A second number that reads as a line
number did.

## 1. The question

`mrw read` renders each served range as numbered rows under a header:

```
==> services.conf  3621L  200039B  sha 5a9440d4
@@ 1-3621
    1| # service registry — every service has the same shape
    2| ...
```

A plan then addresses a line by the number in that gutter: `@@ services.conf 2898 replace`. The
cap on an MCP `mrw_read` result is 200,000 characters as advertised, enforced as 200,000 bytes
(ADR-011); the CLI is uncapped. The question was the throughput one: as
the served window grows toward that cap, does a client's ability to address the right line hold,
degrade, or collapse — and at what token cost?

## 2. The instrument

`curve generate` writes a fixture (`services.conf`: many `[service …]` blocks, each with a
`retries` and a `timeout` line, padded with inert comment text to a byte target), records what mrw
serves for it (`served.txt`, measured), and plants one target: the `timeout` line of the one
service whose retry budget differs from every other's. Two selectors exist: the named one, where
the instruction names the service (readings 1 and 2), and the relational one, where the client is
told only the relation — never the service's name or its line (readings 3 onward). Either way it
must author one plan with one `replace` hunk.
`curve score` applies the plan to the fixture and reports which line changed. `curve tally`
groups by served bytes and target position and reports each cell as a rate with a Wilson interval.

The criterion was written into the backlog before the generator existed: correct-address rate
against served bytes, stratified by target position (early, middle, late), refusals reported
beside the cell and never in it, five repeats per cell, and **a flat curve accepted as an answer**.

Each reading's plan — cells, client, arm, predictions — is committed before its first trial, and
the plan in the tree is byte-identical to that commit; corrections live in the result document. Compliance is verified from each
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
| 10 | Haiku | tool result, `nl` per range | — | — | 14/14 | Second number restarts per range: no miss; one trial void (spill). |
| 11 | Haiku | tool result, `nl -v` from the top | — | — | 10/15 | Second number equal to the reader's: five misses, all at target+2, all late. |

Every score file is in the repository under `docs/curve/reading-0N-scores/`; the rates, intervals,
offsets and pairings in this note recompute from them. Compliance, coverage, cost and the quoted
transcript row come from transcripts and request records that are not committed, and are reported.

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

Two things changed between readings 4 and 8, not one: the second gutter went, and the text arrived
in `sed` ranges rather than a reader's windows. Readings 10 and 11 separated them on the same
cells. Reading 10 kept the ranges and piped each through `nl -ba`, so a second number stood beside
mrw's again but restarted at 1 in every range — on the target's row it read 125 to 259 — and the
client ignored it: 14 of 14 compliant trials at offset 0 (one trial merged two ranges and spilled;
void under the pre-registered rule). Reading 11 piped each range through `nl -ba -v A`, so the
second number was the row index from the top of the served text — `T + 2` on the target's row,
reading 4's number exactly — and the miss came back: 10 of 15, five misses, every one at `T + 2`,
and all five in the late position, where reading 4's misses had been spread across positions.
Against reading 10 on the fourteen cells compliant in both, five discordant pairs, all one way,
p = 0.0625. The chunking is not what removed the miss; a second number brings it back under
chunked delivery too, provided its value reads as a line number. What chunking contributes on its
own, these readings do not measure: reading 4 against reading 11 is 8 and 10 of 15, discordant
both ways.

What this is and is not. It is a measurement that, for this client on this fixture family, the
only recurring miss is induced by a second number laid beside mrw's whose value is plausible as a
line address, wherever it arrives — a file reader's gutter or a `nl -v` column — and that a
delivery without one showed no miss at any size. It is not a claim about the MCP transport, which
was not run, nor about clients or fixtures not measured; and the late-only pattern of reading 11
is observed on five cells and not explained. Reading 4 stands as a fact about a real path: a client
that saves a tool's numbered output to a file and reads it back through a numbering viewer will
meet two gutters, and a weaker client takes the first.

## 5. Result B: what a hundred times more bytes costs

Spend is input plus cache-creation plus output tokens summed over a trial's requests, medians per
tier; peak (the largest single request's context) is in the result documents.

| Client, delivery | 2 KB | 20 KB | 200 KB | 200 KB ÷ 2 KB |
|---|---|---|---|---|
| Sonnet, file reader, named fixture (reading 2) | 48,680 | 54,412 | 118,879 | 2.44× |
| Sonnet, file reader, relational fixture (reading 3) | 49,050 | 55,426 | 123,307 | 2.51× |
| Haiku, file reader (reading 4) | 31,062 | 36,469 | 91,546 | 2.95× |
| Haiku, tool result (readings 8, 9) | 31,909 | 36,444 | 84,789 | 2.66× |

The fixed cost of a session dominates: a hundredfold increase in served bytes is a 2.4× to 3.0×
increase in tokens, and the 20 KB tier costs 12–17% more than the 2 KB tier. A ten-thousand-token
window was not a tier and is not measured here. The same bytes cost the same
through either delivery — within 3% at 2 KB and 20 KB, 7% less at 200 KB — which is what a
delivery-not-size account predicts.
(Cost figures come from request records that are not committed; the result documents say so.)

## 6. What the method cost, and what it bought

Three readings were void under their own rules. Reading 1 because clients given a search tool
searched for the target's name instead of reading, so served bytes were never manipulated.
Readings 6 and 7 because a weaker client given a shell and a rule — read in ranges under a cap, no
search — broke the rule four different ways, and given the exact commands merged two of them. The
rule that finally held was written from how the client behaves and pre-registered before the run
that counted. Each void is recorded with its observations; none contributes to a counted rate.

Two plan files were edited after collection during this series — a wrong count corrected, a dated
note added — and both edits were reverted at review, because a plan that can be edited whenever the
edit looks harmless is not a pre-registration. Every plan in the series is byte-identical to the
commit that added it. Because `main` takes squash merges, the plan-before-trials ordering is
checkable in the pull requests rather than on `main`: reading 2 in #90, 3 in #96, 4 in #98, 5 in
#103 (plan commit `e65a684`), 6 to 8 in #104 (`5ebb34d`, `700133e`, `bfc27fa`), 9 in #106
(`30d5637`). And every rate, interval, offset and pairing in every result document is
computed by the script that reads the scores; the two numbers in this series that were typed from
memory — a paired count and a range width — were both wrong, and were caught by review.

## 7. Limits

One fixture family, five repeats per cell, two clients from one vendor; the strong client sits at
the ceiling on both fixtures, so its curve cannot bend and says nothing about where it would. The
weaker client's ceiling through a tool result is fifteen of fifteen per tier, whose interval's
lower bound is 0.796, not 1. Reading 9's 200 KB tier is reading 8's data, pooled under a plan that
said so. Readings 10 and 11 separate the gutter from the chunking on fifteen cells each, and
reading 11's misses fall in one position only, which no plan predicted and this note does not
explain. The MCP tool-result path, which carries no outer numbering, was not measured.

## 8. Reproduction

```sh
go build -o bin/curve ./cmd/curve
bin/curve generate -out CELL -bytes 200000 -position late -distractors 3 -seed 1 -selector odd-retries
bin/curve tally docs/curve/reading-09-scores/*.score.json docs/curve/reading-08-scores/*.score.json
```

Cells regenerate deterministically from their parameters; a reading's plan names them. The
transcripts and request records that compliance and cost come from are not committed.
