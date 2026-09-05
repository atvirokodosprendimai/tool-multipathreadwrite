# Reading 11, plan: the same ranges, with reading 4's outer number put back exactly

**Written and committed before any trial ran.** Reading 10's result stands as collected;
nothing here changes it.

## Why an eleventh reading

Reading 10 put a second number beside mrw's inside the tool result and the client ignored it:
fourteen of fourteen compliant trials at offset 0. But that number restarted at 1 in every range,
so on the target's row it read 125 to 259 — not a number a client would mistake for a line
address. Reading 4's reader numbered the served text continuously from its first row, so its
number on the target's row was `T + 2`, everywhere. This reading reproduces that number exactly,
inside the tool result, so that the value of the outer number is no longer a difference between
the arms.

## Cells

Reading 4's fifteen 200,000-byte cells, byte-identical (verified before the first trial), window
from line 1: the cells of readings 4, 8 and 10.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: the scripted tool-result arm of reading 10, numbered continuously.** Bash and Write only;
thirteen listed commands — `cat task.json`, then twelve `sed -n 'A,Bp' served.txt | nl -ba -v A`
ranges, eleven of 300 rows and a tail `3301,N`. `nl -v A` starts each range's numbering at its
first row's index, so the outer number of every row is its row index in `served.txt`: line `T`
is row `T + 2`, and the outer number on the target's row is `T + 2` — reading 4's number.
Compliance as reading 8 pre-registered it: the listed set run, merges of consecutive listed
ranges tolerated, nothing else run, no Read/Grep/Glob before the Write, memory-tool calls after
the Write not the trial's, no spill.

## The two accounts, and what each predicts

- **H-reader-number:** the weaker client takes the outer number when it is a plausible line
  number. Misses reappear, each at exactly `T + 2`; at least three of fifteen (reading 4's rate
  on these cells was 8 of 15).
- **H-reader-delivery:** the miss belongs to the file-reader delivery — its number is the
  harness's own, presented as the file's, in a viewer the client trusts for line addresses — or
  to its windowing, and a number of the same value inside a tool result is not taken. Fifteen of
  fifteen again.

## Predictions, recorded before collection

1. After reading 10 the author expects **H-reader-delivery: no miss** — stated so it can be
   wrong; reading 10's author expected the opposite and was.
2. **Any miss sits at exactly `T + 2`**; a miss anywhere else is a third account, reported as
   such.
3. **Compliance at least 12 of 15.**
4. **Cost within 20% of reading 10's median spend of 96,184** — the same bytes, with numbers one
   digit wider.

## What this decides

Misses at `T + 2` here would mean the value of the second number is what the weaker client takes,
wherever it arrives, and the note's account of reading 4 generalises beyond the file reader. No
misses would mean the miss is a property of the file-reader delivery specifically, and the
note's scope sentence — "a client that saves a tool's numbered output to a file and reads it back
through a numbering viewer will meet two gutters, and a weaker client takes the first" — is the
whole of the claim.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
