# Reading 19, result: at 2 KB the delivery mrw ships is at the ceiling; the 20 KB stratum is void

**Collected 2026-09-05 under `reading-19-plan.md`, committed before any trial ran.**

**2,000 bytes: 15 of 15, compliant 15 of 15, zero paged reads.** That stratum ran entirely under the
committed plan and is the reading's result.

**20,000 bytes: VOID**, on two deviations of the author's own making, described below. No rate is
published for it and its scores are not committed.

## The curve at 2 KB

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 9 (bare tool result) | reading 13 (numbered) | reading 4 (file reader) |
|---|---|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 | 14/15 | 15/15 |

Every trial addressed exactly the planted line. Every trial reported every service block, the true
last line, no forbidden tool, and the two writes in the required order. **`next_read_sends` was 0 in
all fifteen** — prediction 5 holds exactly: at this size mrw's render is far below its cap and
nothing pages.

Predictions 3 (compliance ≥ 27 of 30) and 4 (cost) were written across both sizes and cannot be
evaluated on one stratum; prediction 1 was likewise pooled across 30 trials and is **unevaluable as
written**. What can be said is narrower and is what this reading claims: at 2 KB, on these cells,
this client, through `mrw_read`, 15 of 15.

Prediction 2 is vacuously held: there were no misses, so there is no third account to report.

## What the 2 KB stratum decides

The delivery mrw actually ships matches the best delivery previously measured on these exact cells.
Reading 9's bare tool result was 15/15 and reading 13's arm — the same bytes with a second, plausible
line number laid beside mrw's — was 14/15. **mrw's gutter, delivered by mrw's own tool, is the
address the answer needs, and this client took it every time.**

This is the first number this corpus has for the MCP delivery arm, and it discharges ADR-020's open
item at the size it was run.

## Why the 20 KB stratum is void

Both causes are the author's, not the client's, and both are procedural rather than interpretive.

1. **The harness changed after the first trial.** After `20000-middle-3` reported the file's last
   line correctly by number and words but dropped its leading `# `, the coverage instruction in the
   final six dispatches gained the phrase "copied exactly, including any leading `#`". The plan
   names a harness change after the first trial as a void condition, and it fires. The materiality
   argument — that this touches a self-report and not the measured address — does not rescue it: a
   client reads its whole prompt before authoring, so it had been told something about file
   structure first, and the author would be adjudicating his own deviation by a standard invented
   after seeing results.

2. **A failed trial was retried, and the plan does not authorize it.** `20000-middle-5`'s first
   client ended without writing either file. A second was dispatched. **Selective retry of failures
   is a bias mechanism** — non-answers are retried and answers never are, so the surviving set is
   filtered. Disclosing it is not sufficient; the plan should have said in advance what a client
   that writes nothing means, and it did not.

The 2 KB stratum is untouched by either: all fifteen of its prompts were verified byte-identical
apart from the cell name, and none carries the added phrase.

**What the void trials showed, recorded so reading 20 knows what to expect and not as a published
rate:** all fifteen 20 KB trials that produced an answer addressed the planted line, and
`next_read_sends` was 0 throughout. One trial produced nothing at all, and one reported the last
line without its comment marker. None of that is a measurement here.

## Together with reading 18

The two readings answer ADR-020's item in two parts, and the pair is worth more than either number:

- **At 2 KB the shipped path is at the ceiling** — 15 of 15, at the best rate any delivery has
  reached on these cells.
- **At 200 KB it is not measurable on this host**, because mrw's page is truncated before the model
  sees it and the ledger records the page whole. That is the defect in `docs/adr/BACKLOG.md` under
  ADR-023, not a point on a curve.

Nothing here moves mrw's cap or its served format. What it does is locate where the shipped path
stops being trustworthy, and say that the boundary is the host's, not the tool's.

## What it does not decide

One client, one fixture family, one size actually reported, five repeats per cell. Fifteen trials at
the ceiling cannot distinguish 15 of 15 from 14 of 15. The 20 KB stratum is unmeasured until
reading 20 runs it. Compliance and cost come from transcripts and request records that are not
committed; the paged-read count comes from the clients' own in-band reports, which are weaker than a
transcript.

## Provenance

Fifteen scores in `docs/curve/reading-19-scores/`; `reading-19-tally.json` is computed over them:

```sh
bin/curve tally docs/curve/reading-19-scores/*.score.json
```

Every number above is computed from those scores, none typed. The 20 KB scores and all compliance
reports are not committed.
