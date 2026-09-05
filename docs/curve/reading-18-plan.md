# Reading 18, plan: the MCP delivery arm again, with compliance carried in band

**Written and committed before any trial ran.** Reading 12 is void and stays void;
`reading-12-void.md` records what it produced and nothing here revives its five scores.

## Why an eighteenth reading

Reading 12 failed on evidence, not on delivery. Its arm — mrw's served text arriving through
`mcp__mrw__mrw_read`, the delivery mrw ships — is still the one this corpus has never measured, and
ADR-020 has carried it open since reading 8. What stopped it is that every reading since reading 2
verified compliance from each trial's transcript, and a subagent's transcript is not readable by the
session that spawned it. This reading changes that one thing: **what a transcript would have proved,
the client writes down.**

## The build under test

Unchanged from reading 12: the registered `mrw mcp` server running the binary built at `d6c62e7`,
not rebuilt, whose engine is byte-identical to `v1.1.0` (`git diff v1.1.0 d6c62e7 -- internal cmd`
is empty). Confirmed live before reading 12's plan was written and not re-confirmed here: a
`mrw_read` returns the served lines as content block one and the receipt as block two, so ADR-023's
fix is in the path under test.

## Cells

Reading 4's fifteen 200,000-byte cells — 3 positions × 5 seeds, distractors 3, selector
`odd-retries`, window from line 1 — the same directories reading 12 staged, kept rather than
regenerated. Each was verified against `docs/curve/reading-11-scores/*.score.json` on three fields:
`trial_id`, `served_bytes` and the planted line. The sha256 of each `served.txt`, recorded here
because the cells are deleted once the scores are committed:

| cell | trial_id | served | target | sha256 of served.txt |
|---|---|---|---|---|
| `200000-early-1` | 3779f82d9879 | 200039 | 725 | `75a814982ad5dd9a53d05da7339730b4647550eebae67b44fd68c3f31aa9d3b9` |
| `200000-early-2` | b452f1a3d0bd | 200017 | 726 | `92ec04cb712c6904de23ee7e788243f30b836a6a622ff2b02607656ddc18d6ba` |
| `200000-early-3` | f273e867c370 | 200022 | 726 | `501bd5bff252cc1a588c8f917c7a3ee58082522fef00144686024ca569586721` |
| `200000-early-4` | 138d31a3f9e7 | 200013 | 726 | `f44d3e7ba8ff03443c2485d9af92ccaa1af53c2f77041a31f262910d3b144ad9` |
| `200000-early-5` | 19ea45d95094 | 200028 | 730 | `37c2f17cb7f2c5f8d47907866bc27f323b75a41184b1906605494a371e7fb135` |
| `200000-middle-1` | 11c1a43024e6 | 200039 | 1450 | `4a1716524e403bbc5879ca53c17e7332f5a8adda64e71a8bb038382fe0509ac8` |
| `200000-middle-2` | 7b73e158040b | 200017 | 1452 | `d5e6921aeffd766559b5ef6cba7d0509e8645b72d44df953933b7319415e4c1d` |
| `200000-middle-3` | 60744bda4a72 | 200022 | 1451 | `dad7b7689ed37d0731a7ffd3e199573f297400e2471ffc64f273db95cb72ae9c` |
| `200000-middle-4` | c8e854ec6b89 | 200013 | 1452 | `d68f49442701ce8556a305bd986b6222de44c9d841ec49f4451f49de82e37407` |
| `200000-middle-5` | 49e49889e911 | 200028 | 1459 | `3dd09e41e437358e019156724fb508667724d073d9294b6c34c4938dab5a20e2` |
| `200000-late-1` | 8d02e5e1d1f1 | 200039 | 2898 | `993ab2e9bd75856fa1e6aa34b18eb0f98a792d13f54d30205dff73d3537e2c3c` |
| `200000-late-2` | ade409eb03f8 | 200017 | 2904 | `e5ccb5825c57f999a816e7a13f50c70b9c0a62cbbee5c1c8ce5c476e76fbd0df` |
| `200000-late-3` | 0d4565fd70ab | 200022 | 2901 | `7679e42654d6fdc8de640b36480897843457529b3a118afbad191955d3838198` |
| `200000-late-4` | 938edee09d13 | 200013 | 2903 | `f42b07d835ca9564cc459bf732abbc38c55d278d6495c8beccba22e296aaeef9` |
| `200000-late-5` | 109fca722055 | 200028 | 2917 | `bb9f8dc512ede15799e456bfe7c829ee5e1bc845606da63908cb29b252f3f323` |

**The five `early` cells were answered once already, under reading 12.** They are re-run here with
fresh clients; no reading-12 score carries over, and the result reports the two runs' agreement as
an observation, never pooling them.

## The leak, closed structurally

As in reading 12: each cell's `answer.json` is moved out of the repository root after generation and
moved back only for `curve score`, so while any client runs the ground truth is not addressable by
`mrw_read`. "A client reaching `answer.json`" stays a void condition.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`), as in readings 4, 8 to 11 and 13.

**Arm:** `mcp__mrw__mrw_read` and `Write`, and nothing else. The listed calls:

1. `mrw_read` with specs `["tmp/curve/<cell>/task.json"]` — the instruction, `trial_id` and
   `served_bytes`.
2. `mrw_read` with specs `["tmp/curve/<cell>/tree/services.conf"]`, a bare path. Every cell's render
   exceeds `mcp.MaxResultChars`, so this returns ADR-014's first page: the lines that fit,
   `isError: true`, and a `next_read`. The prompt says a page is not a failure and that `next_read`
   is sent back until absent.
3. `Write` of `result.json` — `{"trial_id","served_bytes","plan"}`, the plan addressing
   **`services.conf`**, the name `task.json` carries in its `file` field and not the path passed to
   `mrw_read`. Reading 12's pilot established that a client given no other reader copies the
   header's root-relative path into its hunk and scores `refused_apply`.
4. `Write` of `coverage.json` — below.

**The plan address is mrw's gutter, not the row.** In `200000-early-1` the target is line 725 and the
served text's row 727 reads `  725| timeout = 30`; the two differ by the header rows. That the gutter
and the plan's address are the same number is a property of this delivery, checked on one cell before
this plan was written, and it is what makes the arm comparable to readings 8 and 9 at all.

## Compliance, in band — and why it is the SECOND write

`coverage.json` carries what a transcript would have shown:

```json
{"trial_id": "...",
 "last_line_number": N, "last_line_text": "...",
 "service_block_lines": [N, N, N, N],
 "next_read_sends": K,
 "other_tools_used": []}
```

`curve.Result` is unmarshalled without `DisallowUnknownFields`, so extra keys cost no engine change;
`coverage.json` is a separate file regardless, and `curve score` never reads it.

**The order is load-bearing and is checked.** Enumerating the service blocks is close enough to the
task that a client asked for it first would be nudged toward the comparison that answers it — the
instrument would be teaching the client. So the prompt requires `result.json` to be written first and
`coverage.json` second, and **the harness voids any trial whose `coverage.json` mtime is not strictly
later than its `result.json` mtime.** That restores the ordering guarantee a transcript used to give,
without one.

`service_block_lines` proves coverage without handing over the discriminating attribute: the blocks
sit roughly 700 lines apart across the whole window, so listing them requires having been served all
of it, while the retries values — the thing the task turns on — are not asked for.

**Named as harness protocol, not trial behaviour:** `ToolSearch` with `select:mcp__mrw__mrw_read`,
because the MCP tools are deferred; the reply channel; any `am_*` call after the writes; and **the
writes are the answer** — a subagent's final report is not read.

## Predictions, recorded before collection

1. **At least 12 of 15**, and the author expects 15 of 15. Readings 8 and 9 put this client at the
   ceiling on exactly these cells through a delivery whose only number was the served text's own, and
   this delivery's only number is mrw's gutter, which is the address the answer needs. Stated so it
   can be wrong; the author's expectation was wrong in readings 10, 11 and 13, and reading 12's five
   uninterpretable trials are a reason to hold it loosely.
2. **There is no `T + 2` to predict.** No row index is laid beside mrw's gutter in this arm, so the
   offset that explains every miss in readings 4, 11 and 13 does not exist here. **Any miss is a
   third account and is reported as one**, with its offset from the target and what sits at the
   addressed line.
3. **Compliance at least 12 of 15**, by the in-band report: full block coverage, the true last line,
   and no forbidden tool.
4. **Cost within 20% of reading 8's median spend of 84,789** at this size on these cells, plus the
   coverage write.
5. **The paged-read count** is reported as its own number: how many trials sent `next_read` back at
   all. The cut falls inside the file's last line, past every target, so this cannot move the
   primary; it is a fact about the shipped path.
6. **Reading 12's five `early` trials all addressed the file's last block.** If that recurs here in
   compliant trials, it is a finding about the delivery; if it does not recur, reading 12's pattern
   was the harness and `reading-12-void.md` is right to have refused to publish it.

## What this decides

If the ceiling holds, the delivery mrw actually ships is the delivery the flat curve of readings 8
and 9 was measured on, and every reading that bent did so through a harness mrw does not control. If
it does not hold — and reading 12 saw something that might be that — the miss belongs to mrw's own
delivery, the first such finding in this corpus, and the cap, the page boundary and the served format
are all in scope for a record.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching `answer.json`;
a `coverage.json` not strictly newer than its `result.json`; a cell failing the three-field identity
check. A void is a re-run under a new plan file, as reading 12 was.

## Known limitations, stated now

One size, one client, one fixture family, five repeats per cell: a point, not a within-arm curve.
The in-band coverage report is a client's own account of what it was served, which is weaker than a
transcript — it can be filled in wrongly or dishonestly, and the mtime rule constrains order but not
truthfulness. The five `early` cells have been seen by one earlier client population under a void
plan. Cost comes from request records that are not committed.
