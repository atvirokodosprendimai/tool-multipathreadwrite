# Reading 6, plan: the same cells, delivered with no outer gutter

**Written and committed before any trial ran.** Reading 5's result stands as collected; nothing
here changes it.

## The question

Reading 5 established that every miss in 150 read-arm trials is the row index of the served text,
and could not establish whose count it was: the read arm delivers `served.txt` through the client's
own file reader, which lays a second gutter beside mrw's, and the transcript suggested the client
read that one. Whether mrw's own rendering induces the row count when nothing else numbers the rows
is the question the served-format decision and the cap decision were deferred to. This reading
answers it by changing one thing: the delivery.

## Cells: reading 4's 200,000-byte cells, byte for byte

Fifteen: the 200 KB tier — reading 4's bend, 8 of 15 — at three positions and five seeds, distractors
3, selector `odd-retries`, window from line 1. Regenerated from the same parameters and **verified
byte-identical to reading 4's before the first trial**: fixture, served rendering and answer. That is
the pairing: the same Haiku client saw these exact cells through the read arm and missed seven of
them, every miss at `target+2`.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`), as in readings 4 and 5.

**Arm: the tool-result arm.** The client may use only the Bash tool and the Write tool. It reads the
instruction with `cat task.json` and the served text with `sed -n 'A,Bp' served.txt` in ranges of at
most 350 rows, until every row is covered; the served text arrives as the Bash tool's result, which
carries no line numbers of its own, so mrw's gutter is the only gutter. It writes one plan of one
`replace` hunk to the results file. The range cap exists because a Bash result larger than about
25 KB is spilled to a file with a preview (measured 2026-09-05: 25 KB inline, 29 KB spilled), and a
spilled result would put the client back in front of its file reader — which is the delivery under
test, so it is forbidden rather than tolerated.

Compliance is verified per trial from its own transcript, matched on the prompt marker and never on
the cell id:

- every Bash command is `cat task.json` or `sed -n 'A,Bp' served.txt` and nothing else — no `grep`,
  no `awk`, no pattern in the `sed` address, no pipe;
- the ranges cover every row of `served.txt`;
- no Read, Grep or Glob call;
- no Bash result was spilled to a file.

A trial that fails any of these is void, reported, and never counted as a miss.

## The two accounts, and what each predicts

- **H-render (mrw's rendering induces the count):** with mrw's gutter the only one, a client that
  counts rows still addresses `target+2`, the row index counted from the `==>` header. Misses at
  exactly +2 are this account, and they open a served-path record.
- **H-delivery (the harness's file reader induced it):** with no outer number to read, the client
  reads mrw's number. Misses vanish, or land at offsets other than +2.

## What is scored

`curve score` applies the plan; the changed line is the measurement. The tally is reported against
reading 4's on the same cells, and the offset table is the result this reading exists for.

## Predictions, recorded before collection

1. **The rate rises against reading 4's 8 of 15 on these cells.** The author expects 13 or more of
   15; the pooled Wilson interval at 15 of 15 is [0.796, 1.000] and separates from reading 4's
   [0.301, 0.752], while 13 of 15 does not. The rate is therefore secondary to the offsets.
2. **No miss sits at `target+2`.** This is the author's expectation, stated so it can be wrong: the
   +2 was the reader's gutter. A single +2 miss refutes it and is H-render.
3. **Compliance will be lower than 15 of 15.** A Bash arm with a range cap gives a weaker client more
   ways to fail compliance than a Read arm did: a pattern in a `sed` address, a `grep`, a range over
   the cap. Voids are expected and reported as a result in their own right.
4. **Cost per trial will be within 20% of reading 4's at this tier** (median spend 91,546): the same
   bytes reach the client either way.

## What this decides

If prediction 2 holds with at least three misses fewer than reading 4 on the same cells, the bend of
reading 4 was the harness's delivery, the served format is not changed, the cap is not changed, and
the BACKLOG read-format entry closes with no engine change. If any miss sits at +2, mrw's rendering
induces the row count and a served-path record opens with its own decision about the gutter. If the
reading is void-heavy or the rate does not move, it says so and decides nothing.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching `answer.json`;
a cell failing the byte-identity check against reading 4.
