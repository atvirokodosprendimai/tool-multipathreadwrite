# Reading 12, plan: the served text delivered by the tool that serves it

**Written and committed before any trial ran.** Readings 1 to 11 and 13 to 17 stand as collected;
nothing here changes them.

## Why a twelfth reading

Seventeen readings have measured a client reading mrw's served text through a file reader, through a
Bash tool result, or from a prompt. None measured it arriving through `mrw_read`, the MCP tool mrw
ships — the one delivery a caller gets without building a harness first. ADR-020's follow-up has
carried it as open since reading 8, and the number it produces is the one this corpus does not have.

The arm was staged once, on 2026-09-05, and did not run: over Claude Code 2.1.261 a successful
`mrw_read` reached the model as its `structuredContent` alone, so the client would have authored
against a receipt naming no lines (issue #109). ADR-023 removed the envelope. This reading is that
arm against the build that removed it.

## The build under test

The registered server is `mrw mcp` from the binary built at `d6c62e7` — the ADR-023 commit. It is
**not** rebuilt for this reading: the running server holds that image, and a receipt naming a build
that did not serve the trials would be false. `git diff --stat v1.1.0 d6c62e7 -- internal cmd` is
empty, so the engine and commands under test are the v1.1.0 tree; the difference between the two
commits is documentation.

Confirmed before this plan was written, on one pilot cell scored nowhere: a narrow `mrw_read`
returned the served lines as its first content block and the receipt as its second. The envelope no
longer stands in for the text on this host.

## Cells

Reading 4's fifteen 200,000-byte cells — 3 positions × 5 seeds, distractors 3, selector
`odd-retries`, window from line 1 — regenerated from those parameters into `tmp/curve/`, which
`.gitignore` excludes. They must be inside the repository root because the registered server serves
only its own checkout.

**Cell identity is verified against committed data before the first trial, on three fields per
cell**: `trial_id`, `served_bytes` and the planted line, checked against
`docs/curve/reading-11-scores/*.score.json`, which recorded all three for these same fifteen cells.
A `served_bytes` match alone would not establish identity; a `trial_id` match is the generator's own
digest of the parameters. If any cell differs on any field the reading does not start. The sha256 of
each regenerated `served.txt` is recorded in the result, because the cells are deleted once the
scores are committed.

## The leak, closed structurally

Every previous arm listed the client's exact shell commands, so the ground truth could not be
reached. This arm hands the client a general file reader pointed at the repository root, and
`answer.json` is a sibling of `tree/`. Policing that with an instruction would be the weakest
control available.

So each cell's `answer.json` is **moved out of the root** after generation and moved back only for
`curve score`. While any client is running, the answer is not in a place `mrw_read` can address.
"A client reaching `answer.json`" stays a void condition anyway.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`), as in readings 4, 8 to 11 and 13.

**Arm: `mcp__mrw__mrw_read`, and nothing else that reads.** The client is given exactly two tools,
`mcp__mrw__mrw_read` and `Write`, and the calls are listed in its prompt:

1. `mrw_read` with specs `["tmp/curve/<cell>/task.json"]` — the instruction, `trial_id` and
   `served_bytes`. This replaces the `cat task.json` of readings 6 to 14 and keeps the trial on one
   reading tool. It differs from those arms in one declared way: the JSON arrives with mrw's gutter
   on it.
2. `mrw_read` with specs `["tmp/curve/<cell>/tree/services.conf"]` — a bare path, no range. mrw
   renders the whole file, which is what `served.txt` holds.
3. `Write` of `tmp/curve/<cell>/result.json` as `{"trial_id","served_bytes","plan"}`, the plan
   addressing **`services.conf`** — the name `task.json` carries in its `file` field.

**The prompt names the file the plan addresses, and this arm is the reason.** A `mrw_read` spec is
relative to the repository root, so the served text arrives under a header reading
`==> tmp/curve/<cell>/tree/services.conf`; the plan is applied against a copy of the fixture tree,
where the file is `services.conf`. Nothing tells a client the two differ. The pilot cell was run
without the file named and the client did the natural thing — it copied the path out of the header
into its hunk, and the plan scored `refused_apply` with "does not exist" while having addressed a
line it chose freely. No earlier arm could meet this: readings 6 to 14 served the text with `sed`
from `served.txt` and never showed the client a path that looked authoritative. Naming the file
keeps the measured variable the address, as in every reading since reading 2, and readings 15 and 16
already established that showing a client what shape an answer takes is how a format failure is kept
from being counted as an addressing failure.

**This arm pages, and that is part of what it measures.** The served rendering of a 200,000-byte
cell is 200,039 characters against `mcp.MaxResultChars` of 200,000, so step 2 comes back as
ADR-014's FIRST PAGE: the lines that fit, `isError: true`, and a `next_read` spec for the rest. The
prompt says a page is not a failure and that the client must send `next_read` back until it is
absent. No other arm in this corpus asked a client to do anything after an `isError`.

**Named as harness protocol, not trial behaviour** — reading 14 was evidence-limited because its
reply channel went unnamed, and this is the same shape:

- `ToolSearch` with `select:mcp__mrw__mrw_read`, because the MCP tools are deferred and the tool
  cannot be called until its schema is loaded;
- the subagent reply channel, and any `am_*` memory call, after the Write;
- **the Write is the answer.** A subagent's final report is not read and is not the trial.

Compliance, as reading 8 pre-registered it: the listed calls made, nothing else read, no Read, Grep,
Glob or Bash before the Write, no spilled result.

## Predictions, recorded before collection

1. **15 of 15.** This arm has no second number — mrw's gutter is the only gutter, laid by mrw and
   delivered by mrw, with no reader and no `nl` beside it. Readings 8 and 9 put the weaker client at
   the ceiling on exactly these cells through a delivery with no outer gutter, and this delivery has
   none either. Stated so it can be wrong; the author's expectation was wrong in readings 10, 11 and
   13.
2. **There is no `T + 2` to predict.** Every miss in readings 4, 11 and 13 sat on the served text's
   row index, and no such number exists here. **Any miss is a third account and is reported as
   one**, with its offset from the target stated.
3. **Compliance at least 12 of 15**, reading 11's bar at this size.
4. **Cost within 20% of reading 8's median spend of 84,789** at this size on these cells — the same
   bytes through a different envelope.
5. **A secondary, pre-registered because the arm is the first to produce it: the paged-read
   count.** How many trials treat `isError: true` as a failure and stop with part of the file is
   reported as its own number, apart from misses and apart from compliance. A client that answers
   from page one has covered 200,000 of 200,039 characters, so this is not expected to move the
   primary; it is a fact about the shipped path worth having.

## What this decides

If the ceiling holds, the delivery mrw actually ships is the delivery the flat curve was measured
on, and every reading that bent did so through a harness the tool does not control. If it does not
hold, the miss belongs to mrw's own delivery — the first such finding in this corpus — and the cap,
the paging boundary and the served format are all in scope for a record.

Either way this is one client, one fixture family, one size, fifteen trials.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching `answer.json`;
a cell failing the three-field identity check against reading 11's scores. As before, that is a
re-run under a new plan file.

## Known limitations, stated now

One size only: this arm is measured at 200,000 bytes, where the cap and the page boundary are, and
not at 2 KB or 20 KB, so it yields a point and not a within-arm curve. One client, five repeats per
cell. Compliance, cost and the paged-read count come from transcripts that are not committed, and
are reported rather than reproducible, as in every reading since reading 2.
