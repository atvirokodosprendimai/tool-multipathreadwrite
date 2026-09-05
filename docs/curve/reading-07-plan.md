# Reading 7, plan: the same cells, the same arm, the commands given

**Written and committed before any trial ran.** Reading 6's result stands as collected; nothing
here changes it.

## Why a seventh reading

Reading 6 asked the question — does mrw's gutter alone, with no outer numbering, still produce
`target+2` misses — and could not answer it because its arm left the client three judgements to
make (where to cut ranges, how far to read, whether to search) and a Haiku client made each of them
wrong in some trial: 0 of 15 compliant. This reading removes the judgements. The question, the
client and the cells are unchanged.

## Cells

Reading 4's fifteen 200,000-byte cells, byte-identical (verified again before the first trial),
window from line 1. Reading 4: 8 of 15 hits, seven misses, all at `target+2`.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: the tool-result arm, scripted.** The client may use only the Bash tool and the Write tool,
and in Bash it may run only the commands listed in its prompt, in order: `cat task.json` and then
`sed -n 'A,Bp' served.txt` ranges of 300 rows, precomputed per cell, consecutive, non-overlapping,
the last ending on the file's last row. Reading 6 measured 545-row ranges arriving inline, so 300 is
safe by a wide margin. The served text arrives as the Bash tool's result, which carries no line
numbers of its own; mrw's gutter is the only gutter. The client writes one plan of one `replace` hunk
to the results file.

Compliance, per trial, from its transcript, matched on the prompt marker:

- the set of Bash commands run is exactly the listed set — every listed command run, nothing else
  run; a listed command run twice is tolerated; any unlisted command — a `grep`, a pipe, a different
  range, a `sed` pattern — is a void;
- no Read, Grep or Glob call;
- no Bash result spilled to a file.

**Pre-registered tolerance, from reading 6:** none. The commands are given, so there is nothing to
tolerate; a client that cannot run the listed commands in order is a result about the client.
*(Correction, 2026-09-05, after collection: the listed set is fourteen commands — `cat` and thirteen
ranges — not eleven as this paragraph first said. The count in the prompt and the compliance rule
were the fourteen; only this sentence was wrong.)*

## Predictions, recorded before collection

1. **Compliance of at least 10 of 15.** Reading 6's violations were judgements; with none to make,
   the remaining failure is a client that searches anyway, and the author expects some to.
2. **No compliant miss at `target+2`.** The author's expectation: the +2 was the reader's gutter.
   One compliant +2 miss refutes it and is H-render — mrw's own rendering induces the row count —
   and opens a served-path record.
3. **The compliant hit rate exceeds reading 4's on the same cells**, with the paired comparison
   reported: on the cells that are compliant here, reading 4's outcomes are known, and the
   discordant pairs are counted in each direction.
4. **Cost within 20% of reading 6's median spend of 82,976.**

## What this decides

If prediction 2 holds with at least 10 compliant trials and the paired comparison favours this arm,
the bend of reading 4 was the harness's delivery: the served format is not changed, the cap is not
changed, the BACKLOG read-format entry closes with no engine change, and the stability claim rests
on readings 3, 5 and 7 together. If a compliant miss sits at +2, a served-path record opens. If
fewer than 10 trials comply, the reading reports the client as the limit and decides nothing.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
