# Reading 10, plan: the same ranges, with an outer gutter put back

**Written and committed before any trial ran.** Reading 9's result stands as collected; nothing
here changes it.

## Why a tenth reading

Reading 8 changed two things against reading 4 at once: the outer gutter went, and the text
arrived in `sed` ranges rather than a file reader's windows. The gutter was the parsimonious
account — reading 5 put every miss exactly where a row-index account predicted — but the two were
not separated, and the note says so. This reading separates them: the ranges of reading 8, on the
cells of reading 8, to the client of reading 8, with one change — each range is piped through
`nl -ba`, so a second number stands beside mrw's on every row again, inside the tool result.

## Cells

Reading 4's fifteen 200,000-byte cells, byte-identical (verified before the first trial), window
from line 1: the cells of readings 4 and 8.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: the scripted tool-result arm of reading 8, numbered.** Bash and Write only; the prompt
lists thirteen commands — `cat task.json`, then twelve `sed -n 'A,Bp' served.txt | nl -ba` ranges,
eleven of 300 rows and a tail `3301,N`. `nl` numbers each range's output from 1, so the outer
number of a row is its index within its range: for a target on line T in the range that starts
at A, the outer number is T − A + 1. Compliance as reading 8 pre-registered it: the listed set run,
merges of consecutive listed ranges tolerated, nothing else run, no Read/Grep/Glob before the
Write, memory-tool calls after the Write not the trial's, no spill.

## The two accounts, and what each predicts

- **H-gutter:** the weaker client takes the outer number when there is one. Misses reappear, and
  each sits at `T − A + 1` for its target's range — a number between 1 and 300, hundreds of lines
  from the target and different for every cell. Reading 4's rate on these cells was 8 of 15.
- **H-chunking:** reading 8's ceiling came from the ranges, not from the missing gutter. The outer
  number changes nothing: 15 of 15 again, or misses at some other offset.

## Predictions, recorded before collection

1. **Misses reappear**, at least three of fifteen, and **every miss sits at exactly `T − A + 1`**.
   This is the author's expectation, stated so it can be wrong.
2. **No miss at `target+2`**, since no row's outer number is its line plus two here.
3. **Compliance at least 12 of 15.**
4. **Cost within 20% of reading 8's median spend of 84,789** — the same bytes, with a few more per
   row for the numbers.

## What this decides

Prediction 1 holding separates the two accounts: the gutter, not the chunking, is what took the
client from 8 to 15 of 15, and the note's parsimony hedge becomes a measurement. Fifteen of
fifteen here would mean the ranges did the work, and the note's account would have to change.
Misses at neither offset would be a third account this plan did not foresee, reported as such.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
