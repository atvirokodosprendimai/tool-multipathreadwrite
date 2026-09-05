# Reading 7, result: void under its own rule, for one uniform reason

**Collected 2026-09-05 under `reading-07-plan.md`, committed before any trial ran.** Fifteen
trials, a Haiku client, the scripted tool-result arm, on reading 4's fifteen 200,000-byte cells.

## The verdict the plan requires

**One trial of fifteen is compliant, so the reading decides nothing** — the plan said fewer than
ten compliant trials "reports the client as the limit and decides nothing". Fifteen of fifteen hit,
every plan at offset 0, none at `target+2`; and fourteen transcripts fail the compliance rule in
the same way:

| Trial | Hit | Offset | Bash calls | Unlisted command | Listed commands not run | Other tools | Spilled |
|---|---|---|---|---|---|---|---|
| 200000-early-1 | yes | +0 | 13 | sed -n '3301,3621p'  | 2 | none | no |
| 200000-early-2 | yes | +0 | 13 | sed -n '3301,3629p'  | 2 | ToolSearch, mcp__agentsmemory__am_add_drawer | no |
| 200000-early-3 | yes | +0 | 13 | sed -n '3301,3625p'  | 2 | none | no |
| 200000-early-4 | yes | +0 | 13 | sed -n '3301,3627p'  | 2 | ToolSearch, mcp__agentsmemory__am_add_drawer | no |
| 200000-early-5 | yes | +0 | 13 | sed -n '3301,3645p'  | 2 | none | no |
| 200000-late-1 | yes | +0 | 13 | sed -n '3301,3621p'  | 2 | none | no |
| 200000-late-2 | yes | +0 | 13 | sed -n '3301,3629p'  | 2 | none | no |
| 200000-late-3 | yes | +0 | 13 | sed -n '3301,3625p'  | 2 | none | no |
| 200000-late-4 | yes | +0 | 13 | sed -n '3301,3627p'  | 2 | none | no |
| 200000-late-5 | yes | +0 | 13 | sed -n '3301,3645p'  | 2 | none | no |
| 200000-middle-1 | yes | +0 | 13 | sed -n '3301,3621p'  | 2 | none | no |
| 200000-middle-2 | yes | +0 | 13 | sed -n '3301,3629p'  | 2 | none | no |
| 200000-middle-3 | yes | +0 | 13 | sed -n '3301,3625p'  | 2 | none | no |
| 200000-middle-4 | yes | +0 | 14 | none | 0 | none | no |
| 200000-middle-5 | yes | +0 | 13 | sed -n '3301,3645p'  | 2 | none | no |

- **Fourteen trials merged the last two listed commands into one** — `sed -n '3301,3621p'` in
  place of `3301,3600p` and `3601,3621p` — a single range of 321 to 345 rows that prints exactly the
  rows the two listed commands print, in order, with nothing added and nothing searched. Every one
  of the fourteen made the same merge and no other change. The plan said "exactly the listed set,
  nothing else run" and pre-registered no tolerance; that rule voids them, and the author wrote it.
- **Two trials called `ToolSearch` and a memory tool after writing their plan.** That is the session
  harness's own persistence protocol, loaded into every subagent, firing at the end of the task; it
  reads nothing the trial is about and ran after the plan existed. The prompt said Bash and Write
  only, so under this plan those calls are a second violation in those two trials, recorded here;
  the void verdict on them stands either way, and reading 8's plan names such calls in advance.
- **No trial searched, spilled, skipped a row, or ran any other unlisted command.** The failure
  modes of reading 6 are gone; what remains is a client that does the listed work in one fewer call.

## Predictions, scored

1. Compliance of at least 10 of 15 — **refuted: 1 of 15**, by the merge above.
2. No compliant miss at `target+2` — **holds on the one compliant trial, and on all fifteen**.
3. Compliant hit rate above reading 4's, paired — **not scorable** on one trial.
4. Cost within 20% of reading 6's 82,976 — **holds**: median spend 84,972.

## What the void suggests, labelled as such

The fourteen merged trials read the same rows as the listed commands, through the same delivery,
with no outer gutter, and every one addressed the target exactly. On the same fifteen cells the
read arm scored 8 of 15 in reading 4. That is what the scripted arm was built to measure, and it
is written here as a post-hoc observation under a tolerance the plan did not grant, so that
reading 8 grants it in advance: a run of consecutive listed ranges merged into one that prints the
same rows is the listed work done in fewer calls, and the harness's persistence calls after the
Write are not the trial's.

## Provenance

Scores in `docs/curve/reading-07-scores/`, tally in `docs/curve/reading-07-tally.json`. Compliance
and cost come from transcripts that are not committed.
