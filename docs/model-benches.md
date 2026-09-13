# Model benches

These are the **model × score** readings, recovered from the pre-tidy README.
They are not the `measure.sh` byte table — that lives in
[measure.md](measure.md). Status of the tree they describe is still
**v1.15.0 / `31422d8`**. Re-run `mrw stats` in your own checkout rather than
quoting the 65/68 cell.

The measurement note that draws the served-size readings together is
[notes/served-size-and-delivery.md](notes/served-size-and-delivery.md). Score
files recompute from `docs/curve/reading-NN-scores/`.

## Stats — can a caller actually author a plan?

```sh
mrw stats            # what became of the plans this checkout has been given
mrw stats --json     # the same numbers, machine-readable
mrw stats --reset    # empty the tally, saying how many records it discarded
```

Every number this project publishes about mrw — the byte savings, the round
trips — assumes the plan was authored correctly. Nothing measured that. `mrw
stats` does:

```
  applied           1 of 3 plan(s) (33.3%)
  refused_apply     1 of 3 plan(s) (33.3%)
  refused_parse     1 of 3 plan(s) (33.3%)
```

**`refused_parse` is the one that matters.** It is the only outcome that says
the FORMAT was the problem rather than your picture of a file. ADR-009
pre-registers the reading: above **5% of plans**, the format is what needs
changing, not the caller.

### The first reading — above the floor, and still not a general number

Taken 2026-09-04 on this repository, on an Apple M5, from the tally this
repository's own development produced:

```
  applied          65 of 68 plan(s) (95.6%)
  refused_apply     2 of 68 plan(s) (2.9%)
  refused_parse     1 of 68 plan(s) (1.5%)
```

**`refused_parse` is 1.5% of RECORDED OUTCOMES, and ADR-009's pre-registered
criterion is 5%.** The criterion was written before the number, which is the
only order in which a threshold means anything: above 5% the FORMAT is what
needs changing rather than the caller. At 1.5% this reading does not ask for a
format change. One parse refusal in 68 is a single malformed plan, not a
pattern.

**"Recorded outcomes" is not "plans handed to mrw", and the difference is not
rounding.** A plan that fails to parse is counted immediately, but several other
terminal paths in `mrw write` return before any outcome is recorded — an
unreadable plan file, a working-set pointer that resolves to none or many, a
ledger or check-config that will not load. `authoring.Record` also fails open by
design: it must never fail a write, so a tally it cannot write is silently not
written. Every one of those is a plan mrw was handed and this denominator does
not contain. The direction of the bias is unknown, which is worse than a known
one, and it is the reason the comparison above is stated against recorded
outcomes rather than against plans.

**Sixty-eight is above the floor of thirty, and the floor was the smaller
problem.** An earlier reading here published nine plans and said so — nine is
below the floor, and a percentage on nine samples is noise wearing a decimal
point. That has been fixed by time rather than by design: the sample grew
because the authoring machine was finally running a binary new enough to record
one. Which is the caveat that outlived the floor.

**The population is still the narrowest one possible.** Sixty-eight plans, one
repository, one model, one family of sessions, authored by whoever had most
recently read the format's documentation. Crossing a sample-size floor does not
make a population representative — it only stops the arithmetic being silly.
Nothing here supports a claim about a different model, a different repository or
a caller meeting the format for the first time.

**The tally under-counts by construction, and this is the sharpest caveat.** It
records only what a binary carrying the recorder applied, so plans applied by an
older binary are invisible. That is not hypothetical: this reading sat at nine
for weeks while the authoring machine ran a v0.0.14 binary that predates the
recorder entirely, and it jumped to sixty-eight the day that machine was
rebuilt. The heaviest users of a tool are the likeliest to be running a stale
copy of it, so the denominator is biased toward the traffic that already
upgraded.

**What the tally cannot show, at any sample size.** It counts parse refusals,
and only those are about the format. It cannot distinguish a model that could
not author a plan from a model that authored a correct plan for a file that had
moved — that shows up as `refused_apply`, which is about the caller's picture of
the tree rather than about the document. And it says nothing about plans that
were never attempted because the author reached for a different tool.

A reading from a different model or a different repository is the one that
would test ADR-009's criterion; this one only fails to trip it.

**Counts only.** No plan text, no paths, no anchors, no SHAs, no command lines —
the tally is something you can read in full and find nothing of your work in,
and a test reads the written bytes to keep it that way. Nothing is ever
transmitted; it lives beside the ledger in the directory `mrw seen` names, and
this command is the only reader.

## Does serving more hurt? — the served-size curve

The DEFAULT `MaxResultChars` is 200,000, and `mrw mcp --max-result-chars N`
overrides it. Nothing in this repository knew whether that number was right, so
ADR-020 built an instrument to find out rather than argue about it: `curve`
generates a fixture, a client authors a plan against what mrw would serve, and
the scorer applies the plan and reports which line changed. The pre-registration
in `docs/adr/BACKLOG.md` fixed the criterion before a cell existed — correct-
address rate against served bytes, stratified by position, **a flat curve
accepted as an answer** — and twenty readings have been taken under it: seven
void under their own rules (1, 6, 7 and 15 on format; 12 on unverifiable
compliance; 18 on a host defect; 19 on the author's own deviations), one
evidence-limited under its own (14), and twelve with results — the last of them
reading 20, which measured the MCP delivery arm, the path mrw ships, at 30 of
30 at 2 KB and 20 KB. Every plan was committed before its trials ran, every
score file is committed, and every table below recomputes from them.

| Reading | Client | Fixture | 2 KB | 20 KB | 200 KB | What it settled |
|---|---|---|---|---|---|---|
| 1 | Sonnet | named | — | — | — | **Void.** Clients searched for the name instead of reading, so served bytes were never manipulated. `docs/curve/reading-01-void.md` |
| 2 | Sonnet | named, read arm | 14/15 | 14/15 | 14/15 | Flat, at a ceiling. Three misses, one at each size. |
| 3 | Sonnet | relational | 15/15 | 15/15 | 15/15 | Flat. **Refuted its own prediction** that the harder fixture would be harder. |
| 4 | **Haiku** | relational | 15/15 | 12/15 | **8/15** | **The curve bends.** Intervals at 2 KB and 200 KB do not overlap. |
| 5 | Haiku | relational, window from line 120 | — | — | 12/15 | **Every miss moved from +2 to −117.** The miss is the row index of the served text; whose number the client took, readings 10 and 11 separated. |
| 6 | Haiku | relational, tool-result arm | — | — | (15/15) | **Void under its own rule**: 0 of 15 compliant — ranges over the cap, searches, early stops. Reported, not counted. |
| 7 | Haiku | relational, scripted arm | — | — | (15/15) | **Void under its own rule**: 14 of 15 merged the two listed tail ranges; no tolerance granted. Reported, not counted. |
| 8 | Haiku | relational, scripted arm | — | — | **15/15** | **Compliant 15 of 15.** mrw's gutter the only gutter, no miss at all; 7 discordant pairs against reading 4, all one way. |
| 9 | Haiku | relational, scripted arm | 15/15 | 15/15 | (15/15, reading 8) | **Flat at the ceiling through the tool-result arm**, 45/45 pooled; the read arm's 15, 12, 8 on the same cells becomes 15, 15, 15. Cost within 3% of the read arm's at 2 KB and 20 KB, 7% lower at 200 KB. |
| 10 | Haiku | relational, scripted arm, `nl` per range | — | — | 14/14 | **A second number that restarts per range was not taken by this client in these trials.** Compliant 14 of 15 (one merge spilled, void); no miss; predicted misses did not appear. |
| 11 | Haiku | relational, scripted arm, `nl -v` from the top | — | — | **10/15** | **Reading 4's number put back, the miss comes back**: five misses, every one at +2, all five late. Against reading 10, 5 discordant pairs, all one way. |
| 13 | Haiku | relational, scripted arm, `nl -v` from the top | 14/15 | 13/15 | (reading 11) | **The same number at the smaller sizes**: three misses in thirty, every one at +2, two late and one middle. Observed points with the number 14, 13, 10; without it (readings 9, 8) 15, 15, 15; no size trend established. |
| 14 | Sonnet | relational, twelve distractors, scripted arm | (6/6) | (7/7) | (2/2) | **Evidence-limited.** Thirty of forty-five replied through a channel the rule did not name and are void; the fifteen strict trials all hit; all forty-five as sensitivity: 14, 15, 15, the one miss the right service's line above the target. Reading 17 re-runs it with the channel named. |
| 15 | **gpt-5.6-sol** (Codex) | relational, prompt delivery, shape not shown | 8 parsed | 2 parsed | 1 parsed | **Void on format, 34 of 45**: the client wrote its own grammar (apply_patch, JSON, prose headers); every parsed plan hit, and every void message named the target line. A finding for ADR-012, not about size. |
| 16 | **gpt-5.6-sol** (Codex) | relational, prompt delivery, shape shown | 15/15 | 15/15 | 15/15 | **A second family at the ceiling at every size**, 45/45; 10 discordant pairs against reading 4, all one way. |
| 17 | Sonnet | relational, twelve distractors, scripted arm | 15/15 | 15/15 | 15/15 | **The strong client at the ceiling on a thirteen-service fixture**, 45/45, compliant 45 of 45 under a rule that names the reply channel. Cost 2.41× from 2 KB to 200 KB. |
| 20 | Haiku | relational, MCP delivery (`mrw_read`) | 15/15 | 15/15 | — | **The delivery mrw ships, at the ceiling at both sizes.** 30/30 correct addresses, 0 paged reads, both computed from committed data. Two secondary counts its own plan promised from committed data are not derivable from the tree and are withdrawn; 200 KB cannot be measured on this host at all (reading 18). |

**For a strong client, serving a hundred times more bytes costs about 2.5× the
tokens and loses nothing.** Measured twice, on two different tasks, through the
reader; a thirteen-service fixture through the tool result says the same at the
pre-registered strength (reading 17: 45 of 45, cost 2.41×; reading 14 before it,
evidence-limited). The "serve 10k and call it a day" instinct is not supported:
the fixed cost of a session dominates until the window is very large, so a small
window buys almost nothing.

**For a weaker client through the harness's read arm it costs 2.95× and loses
47 points at 200 KB; through a tool-result path it costs 2.66× and loses
nothing at any size.** Reading 4's 45 trials read every byte at every size,
verified from the transcripts, and still missed 7 of 15 at 200 KB; readings 8
and 9 delivered the same client the same forty-five cells as a Bash tool result
and it scored 15, 15, 15 — the cost within 3% of the read arm's at 2 KB and
20 KB, and 7% lower at 200 KB.

**Every miss in readings 2–5 is the same miss, and reading 5 says what it is.**
Across the 150 read-arm trials of readings 2–5 there are 16 misses; the
committed scores show all 16 changed exactly one line, and the offset is the
**row index of the served text** — the target's row counted from the `==>`
header — two below the target when the window starts at line 1 (13 of 13 in
readings 2–4), and 117 above it when the window starts at line 120 (3 of 3 in
reading 5). The transcript shows the row as the client saw it,
`634	  751| timeout = 30`: the harness reader's number first, mrw's second, and
the client addressing 634. That excerpt suggests the client read the first
number; the scores cannot tell reading a gutter from counting rows, and the
transcript is not committed. Each miss found the right service and wrote the
right text; the plans are not committed, so that half is reported rather than
recomputable.

All 16 apply silently through a green receipt without a guard, and **all 16 are
refused with `anchor=`** — run against each cell's own fixture with the built
binary, and reported in each result document rather than reproducible from a
committed receipt. That is the case for the guard, measured.

What readings 5, 8, 9, 10 and 11 settle between them: the bend was the harness
read arm's delivery, not mrw's rendering. The miss is the row index of the
served text (reading 5: −117 with the window from line 120), and when mrw's
`N|` is the only number on any row — the served text arriving as a Bash tool
result, the delivery most readings ran, and through `mrw_read` itself at 2 KB
and 20 KB (reading 20, 30 of 30) — the client that missed seven of fifteen at
200 KB and three of fifteen at 20 KB through its file reader addressed all
forty-five exactly, at every size, at the read arm's cost or below it (readings
8 and 9, compliant 45 of 45 under pre-registered rules). Readings 10 and 11
then separated the second number from the chunking: with a second number that
restarts per range this client addressed 14 of 14, and a second number equal to
the reader's — the row index from the top — brought the miss back at exactly
that number under the same chunking (10 of 15, all five misses late), and
reading 13 took that number to 2 KB and 20 KB: 14 and 13 of 15, three misses at
+2 — so with a plausible second number this client missed at every size
measured, through either delivery, where the bare arm had no miss at any
(observed points 14, 13, 10 against 15, 15, 15; thirty trials do not establish
a size trend). So the default stays at 200,000 with evidence, the served format
is not changed, and the stability claim rests on readings 3, 5, 8, 9, 10 and 11
together. What stands from reading 4 is a fact about the two delivery forms
measured: lay a plausible line number beside mrw's and a weaker client takes it
some of the time. Two readings between 5 and 8 were void under their own
compliance rules and are recorded, not counted.

Compliance, coverage and cost come from transcripts and request records that
are not committed, and each result document says so. Reading 20 is the one
attempt to do better: it commits its coverage reports, and it records the two
counts it promised to derive from them and could not. The tables, the intervals
and the offsets recompute from `docs/curve/reading-NN-scores/`.
