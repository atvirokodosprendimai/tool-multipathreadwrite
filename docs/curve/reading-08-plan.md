# Reading 8, plan: the same cells, the same scripted arm, the ranges as the client runs them

**Written and committed before any trial ran.** Reading 7's result stands as collected; nothing
here changes it.

## Why an eighth reading

Reading 7 removed the client's judgements and was still void: fourteen of fifteen clients merged
the two listed tail ranges into one that prints the same rows, and the plan had granted no
tolerance. The question, the client and the cells are unchanged for the third time. This plan
lists the ranges the way the client runs them, and says in advance what a merge means.

## Cells

Reading 4's fifteen 200,000-byte cells, byte-identical (verified before the first trial), window
from line 1. Reading 4: 8 of 15 hits, seven misses, all at `target+2`.

## Client and arm

*(Note, 2026-09-05, after collection: the commit that added this plan, `bfc27fa`, says "twelve
listed commands" in its message; the plan below says thirteen, and thirteen is what ran.)*

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: the scripted tool-result arm.** Bash and Write only. The prompt lists thirteen commands:
`cat task.json`, then twelve `sed -n 'A,Bp' served.txt` ranges — eleven of 300 rows and a final
`3301,N` ending on the file's last row (321 to 345 rows; reading 6 measured 545 rows arriving
inline). The served text arrives as the Bash tool's result, which carries no line numbers; mrw's
gutter is the only gutter.

Compliance, per trial, from its transcript, matched on the prompt marker:

- every listed command is run, or is covered by a merge: **a run of consecutive listed ranges
  merged into one command that prints exactly the same rows counts as those commands run** — this
  is the tolerance reading 7 lacked, granted here in advance;
- no other Bash command — a `grep`, a pipe, a pattern, a range that adds or drops a row — is run;
- no Read, Grep or Glob call before the plan is written;
- **calls to `ToolSearch` or an `mcp__agentsmemory__*` tool after the Write are the session
  harness's persistence protocol and are not part of the trial**; they are reported and do not void;
- no Bash result spilled to a file.

## Predictions, recorded before collection

1. **Compliance of at least 12 of 15.** With the tail range listed as one command and merges
   tolerated, the remaining way to fail is to search or to drop a row.
2. **No compliant miss at `target+2`.** One compliant +2 miss refutes it and is H-render: mrw's
   own rendering induces the row count, and a served-path record opens.
3. **The compliant hit rate exceeds reading 4's on the same cells**, with the paired comparison
   reported: discordant pairs counted in each direction, and an exact two-sided sign test.
4. **Cost within 20% of reading 7's median spend of 84,972.**

## What this decides

If predictions 1 and 2 hold and the paired comparison favours this arm, the bend of reading 4 was
the harness's delivery: the served format is not changed, the cap is not changed, the BACKLOG
read-format entry closes with no engine change, and the stability claim rests on readings 3, 5
and 8 together. A compliant +2 miss opens a served-path record. Fewer than 12 compliant trials
reports the client as the limit and decides nothing.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
