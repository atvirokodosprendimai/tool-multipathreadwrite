# Reading 5, plan: a served window that does not begin at line one

**Written and committed before any trial ran.** Reading 4's result stands as collected; nothing
here changes it.

## The question

Thirteen of thirteen misses across the 135 read-arm trials of readings 2–4 addressed exactly
`target+2`. Every served window in those readings began at line 1, and mrw's served rendering
opens with two unnumbered rows — the `==>` header and the `@@ 1-N` range — so in every one of
those windows "the row a client counts" and "the line number mrw prints" differ by exactly 2 for
every line. That is one account of the miss. It is not the only one: +2 could be intrinsic to the
fixture or to the client, and nothing served so far can tell the two apart. ADR-020 T4 built the
cell that can: a window served from a line other than 1.

## The two accounts, and what each predicts

A client that reads `served.txt` through its Read tool sees that file numbered from 1 — row 1 is
the `==>` header, row 2 the `@@` range, row 3 the first numbered line. If a client takes the Read
tool's row number for the address, then with a window from line 1 it addresses `target+2`, and
with a window from line F it addresses `target − F + 3`.

- **H1, row numbering:** with the window served from line 120, every miss is at
  **`target − 117`**, and none is at `target+2`.
- **H2, intrinsic +2:** every miss is at **`target+2`**, exactly as in readings 2–4.
- **Neither:** any miss at another offset, or a split, is reported as such and decides nothing.

The two predictions differ by 119 lines, so a single miss discriminates, and a reading with several
does so beyond argument. **A reading with zero misses is inconclusive**, not a null: it would mean
this client did not miss on this window, and the question stays open.

## Cells

Fifteen: the 200,000-byte tier only — reading 4's highest miss rate, 7 of 15 — at three positions
and five seeds, distractors 3, selector `odd-retries`, **`-from 120`**. The generator re-pads to the
byte target, so served bytes stay at 200 KB (200,014–200,056 measured at generation), the window is
`@@ 120-3732`, and `served.txt` holds 3,615 rows: two unnumbered, 3,613 numbered. The planted lines
move against reading 4's because the padding is refitted; they are recorded in each cell's
`answer.json` and never shown to the client. Cells regenerate deterministically from these
parameters; the trial ids carry `from=120` and differ from reading 4's by construction.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`), as in reading 4 — the client that
bends. **Arm:** the read arm, prompt text identical to reading 4's with only the paths changed:
Read and Write tools only, no search tool, `served.txt` read whole, one plan of one `replace` hunk
written to the results file. Compliance is verified per trial from its own transcript — zero
search-tool calls and full coverage of the served row count from each read call's offset and
limit — matched on the prompt marker, never on the cell id. A void trial is never a miss.

## What is scored

`curve score` applies the plan; the changed line is the measurement. The tally is reported as
before, but the result this reading exists for is the **offset table**: for every miss, target,
addressed, and the offset, against the two predictions above.

## Predictions, recorded before collection

1. **There will be misses.** Reading 4 missed 7 of 15 at this tier; the window start is not
   expected to change the rate much either way.
2. **H1: every miss will sit at `target − 117`.** This is the author's expectation, stated so it can
   be wrong: the +2 has been a constant across three readings and two models, which is the
   signature of a rendering, not of a judgement.
3. **The hit rate will not be lower than reading 4's 8/15 by more than noise.** If the offset window
   itself hurts — a client confused by a range that does not start at 1 — that is a finding about
   the rendering too, and is reported.
4. **Compliance will be 15 of 15**, as in reading 4.

## What this decides, and what follows

If H1: the miss is mrw's served format, and the fix belongs in the read path (a served-path ADR),
not in the byte cap; ADR-011-T3 is not revisited on this evidence. If H2: the cap is the knob and
reading 4 is its evidence, as ADR-020's Follow-up already says. Either way the offset-window entry
in BACKLOG.md is discharged by this reading.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`; a trial whose transcript shows a search call or incomplete coverage (void, reported,
not counted).
