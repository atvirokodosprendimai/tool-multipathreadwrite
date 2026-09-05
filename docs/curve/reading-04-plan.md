# Reading 4, plan: the same cells, a second model

**Written and committed before any trial ran.** Reading 3's result stands as collected; nothing here
changes it.

## Why there is a fourth reading

The pre-registration in `docs/adr/BACKLOG.md` requires the measurement **across models**. Every
reading so far — 1, 2 and 3 — has used one client population, and ADR-020's Follow-up has said since
reading 2 that the criterion is discharged only once a second client is measured. This reading is
that second client, and nothing else: it changes the model and holds every other variable where
reading 3 left it.

It is also the cheapest place a curve might appear. Two fixtures now sit at 100% for the first client,
so difficulty is not what limits it; a weaker client may sit below that ceiling and show the shape two
fixtures could not.

## The criterion, unchanged

Correct-address rate against served bytes, stratified by target position, refusals reported
separately, **a flat curve accepted as the answer.** Not re-derived here.

## Cells: reading 3's, byte for byte

The 45 relational cells of reading 3 — 3 sizes × 3 positions × 5 seeds, distractors 3, selector
`odd-retries`. They are regenerated from the same parameters into a fresh directory and **verified
byte-identical to reading 3's before the first trial**: fixture, served rendering and answer. That
check is the pairing. If any cell differs the reading does not start.

Issue #97 — the retry pair is keyed on the seed alone, so all nine cells of a seed share one pair —
is noted and is not a threat here: every trial gets a fresh client that sees exactly one cell, as in
reading 3. It would matter for a design that reuses a client, which this is not.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`). Reading 3 used Sonnet.

**Arm:** the read arm, identical to readings 2 and 3 — a file reader and a file writer, no search tool
of any kind, the same prompt text with only the paths changed. Keeping the arm identical is what makes
this a one-variable change.

Compliance is verified per trial from its own transcript, matched on the arm's prompt marker and never
on the cell id:

- zero calls to any search tool, and
- full coverage of `served.txt`, from each read call's offset and limit against the served row count.

**A weaker client is more likely to fail compliance than a stronger one**, and the two failure modes
are reported apart from each other and apart from misses: a trial that searched, a trial that did not
read the whole window, and a trial that read it all and addressed the wrong line are three different
things. A void trial is never counted as a miss. If the void count is large the reading says so as a
result in its own right, because "this client cannot be made to read 200,000 bytes" is an answer to
the throughput question too.

## What is scored

Unchanged: `curve score` applies the plan; the changed line number is the measurement. The 45 scores
and the tally are committed with the result.

## Predictions, recorded before collection

1. **The overall rate will be below reading 3's 45/45.** If it is not, two models sit at the ceiling
   on this fixture and the fixture, not the client, is what the next reading must change.
2. **If a curve exists it bends downward with served bytes**, and the 200,000-byte cells are where it
   shows first.
3. **Compliance will be lower than reading 3's 45/45.** The specific expectation is coverage failures
   at 200,000 bytes — a client that stops reading before the last row — more than search-tool use.
4. **Cost per trial will be lower in absolute tokens and the same in ratio**: about 2.5× spend from
   2 KB to 200 KB, as in readings 2 and 3, because the ratio is a property of what is served and not
   of who reads it.
5. **A flat curve remains an acceptable answer**, and would mean the ceiling is not a property of one
   strong model.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching `answer.json`;
a cell failing the byte-identity check against reading 3. As before, that is a re-run under a new plan
file.

## Known limitations, stated now

Five repeats per cell resolve only large effects. Two models discharge the pre-registration's wording
and no more; they are not a survey of models. Compliance, cost and any mechanism narration come from
transcripts and request records that are not committed, and will be reported rather than reproducible,
as in readings 2 and 3.
