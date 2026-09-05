# Reading 9, plan: the tool-result arm at every size

**Written and committed before any trial ran.** Reading 8's result stands as collected; nothing
here changes it.

## Why a ninth reading

Reading 8 put the weaker client at the ceiling at 200 KB through a delivery with no outer gutter,
and said what it could not: whether that holds across served sizes. This reading takes the curve
within that arm — the same three tiers reading 4 took through the read arm — so the two curves can
be laid side by side for one client on identical cells.

## Cells

Reading 4's forty-five relational cells — 2,000, 20,000 and 200,000 bytes, three positions, five
seeds, distractors 3, selector `odd-retries`, window from line 1 — verified byte-identical to
reading 4's before the first trial. **The 200,000-byte tier is reading 8's fifteen trials,
unchanged and not re-run**: same arm, same client, same cells, collected under a committed plan.
This reading adds the 2,000 and 20,000-byte tiers, thirty trials.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: the scripted tool-result arm of reading 8.** Bash and Write only; the prompt lists the
exact commands — `cat task.json`, then `sed -n 'A,Bp' served.txt` ranges ending on the file's last
row: one range per cell here, 46 to 49 rows at 2 KB and 373 to 378 rows at 20 KB, all well inside
what reading 6 measured arriving inline (545 rows). Compliance as reading 8 pre-registered it: every listed command run or covered by a merge of
consecutive listed ranges printing the same rows; nothing else run; no Read, Grep or Glob before
the Write; memory-tool calls after the Write are the session harness's and not the trial's; no
spilled result.

## Predictions, recorded before collection

1. **Flat at the ceiling.** 15 of 15 at 2 KB and at 20 KB, as at 200 KB; the pooled intervals of
   the three tiers overlap. A miss at any tier is reported with its offset; a miss at `target+2`
   would say the row-count account has a second cause at that size.
2. **The paired comparison with reading 4 favours this arm at 20 KB** (reading 4: 12 of 15, three
   misses at +2) **and is level at 2 KB** (reading 4: 15 of 15).
3. **Compliance at least 27 of 30.**
4. **Cost ratio across the tiers within 20% of reading 4's** (31,062 → 36,469 → 91,546 median
   spend): the same bytes cost the same through either delivery.

## What this decides

With prediction 1, the weaker client's curve through a gutter-free delivery is flat from 2 KB to
200 KB, and the note can say "served size did not bend the curve; the delivery did" for both
clients at every size measured. A bend here is a bend in the tool's own delivery and reopens the
cap question at the size it appears.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`; a cell failing the byte-identity check against reading 4.
