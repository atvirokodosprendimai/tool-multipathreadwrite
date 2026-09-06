# Reading 20, result: 30 of 30 at 2 KB and 20 KB through the delivery mrw ships

**Collected 2026-09-06 under `reading-20-plan.md`, committed before any trial ran and amended twice
before its first trial, both amendments recorded in the plan. Thirty trials. One of the plan's three
rules was not met, which costs this reading two of the four counts it set out to publish — and two
of the plan's own void conditions rest on the same evidence the tree does not carry, so this
document asserts neither that they occurred nor that they did not.**

**Published: 30 of 30 correct addresses, and 0 paged reads. Both computed from committed data.**

**Withdrawn: the compliance count and the `no_answer` count.** Rule 3 of the plan promised both from
committed data and named cost as its one exception; neither turned out to be derivable from the
tree. They are not published as numbers here, in either direction. "Rule 3 was not met" below says
exactly what was and was not preserved, and a reviewer who held that this should void the reading
whole is quoted at the end with the argument on both sides.

This is the first number this corpus has for the MCP delivery arm. Readings 12, 18 and 19 all tried
and all voided; `reading-12-void.md`, `reading-18-result.md` and `reading-19-void.md` say why.

## The curve

| Served bytes | early | middle | late | pooled | 95% Wilson | reading 9 (bare tool result) | reading 13 (numbered) | reading 4 (file reader) |
|---|---|---|---|---|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 | 14/15 | 15/15 |
| 20,000 | 5/5 | 5/5 | 5/5 | **15/15** | [0.796, 1.000] | 15/15 | 13/15 | 12/15 |

Pooled across both sizes: **30 of 30**, 95% Wilson [0.886, 1.000]. Every outcome is `hit`; there
were no misses, no `refused_parse` and no `refused_apply`.

## Predictions

1. **Holds.** At least 27 of 30 was required; the author expected 30 of 30 and got it. That is the
   first time in this run of readings the author's expectation was right — it was wrong in 10, 11
   and 13, and readings 12, 18 and 19 all ended somewhere their plans did not foresee.
2. **Vacuous.** No miss occurred, so there is no third account to report. The `T + 2` offset that
   explains every miss in readings 4, 11 and 13 has no analogue in this arm, which lays no second
   number beside mrw's gutter.
3. **Not met, and the count is withdrawn.** The prediction was compliance 30 of 30. Three of its
   four clauses do hold on committed data — every trial reported every service block and the true
   last line, both recomputable because `reading-20-compliance.json` carries the claimed AND true
   values, and none used a forbidden tool. The fourth, `coverage.json` written strictly after
   `result.json`, was checked on mtime at collection time and **cannot be re-derived from the
   tree**: git preserves no mtimes and the `result.json` files were not committed. Rule 3 promised
   this count from committed data, so the count is withdrawn rather than reported at three clauses
   out of four. See "Rule 3 was not met" below.
4. **Unevaluated, as the plan requires.** Cost comes from request records this corpus has never
   committed. Under rule 3 it is an observation and never a published number, and it is reported as
   unevaluated rather than estimated. No claim above rests on it.
5. **Holds exactly.** `next_read_sends` is 0 in all thirty. Both sizes sit far below
   `mcp.MaxResultChars`, so nothing paged and nothing was truncated.
6. **Withdrawn.** The prediction was at most one `no_answer` trial. Thirty committed score/coverage
   pairs cannot distinguish thirty first attempts from a discarded no-answer followed by a retry —
   which is exactly reading 19's failure — so the tree does not carry this count, and rule 3
   promised that it would. Reading 19's void run saw one no-answer at 20 KB. What happened in this
   run is not established by anything committed, in either direction.

## What it decides

**mrw's gutter, delivered by mrw's own tool, is the address the answer needs, and this client took it
every time.** On these exact cells the shipped delivery equals the best previously measured — reading
9's bare tool result, 15/15 at both sizes — and beats both arms that lay a second plausible line
number beside mrw's: reading 13's numbered tool result (14 and 13 of 15) and reading 4's file reader
(15 and 12 of 15). At these two sizes the flatness readings 8 and 9 found is reproduced through the
delivery mrw ships, and not only through a harness built for the measurement. **That is a claim
about 2 KB and 20 KB and about nothing else**: readings 8 and 9 also cover 200 KB, and this reading
does not, so the shipped path's accuracy above 20 KB stays unmeasured.

ADR-020's open item is answered for the sizes it can be answered for:

- **At 2 KB and 20 KB the arm is at the ceiling** — 30 of 30 here.
- **At 200 KB the arm cannot be measured on this host at all**, because the host truncates mrw's page
  before the model sees it while the ledger records the page whole, so a write to the unseen middle
  is licensed and applies. That is reading 18's finding: a defect in `docs/adr/BACKLOG.md` under
  ADR-023, not a point on a curve and not an accuracy result. **No correct-address rate exists for
  this arm at 200 KB, and none is claimed.**

Nothing here moves mrw's cap or its served format.

## Agreement with reading 19's void run

Reading 19 collected these same thirty cells and voided whole. Its trials are **not** pooled with
these and no score of its carries over. Reported only as an observation: every reading-19 trial that
produced an answer also addressed the planted line, and its `next_read_sends` were 0 throughout. The
one trial that produced nothing (`20000-middle-5`) produced an answer here.

## What it does not decide

Two sizes, one client (`claude-haiku-4-5`), one fixture family, five repeats per cell. **Thirty
trials at the ceiling cannot distinguish 30 of 30 from 28 of 30**, and the pooled interval reaches
down to 0.886. It says nothing about 200 KB — reading 18 says what there is to say there, and it is a
defect rather than a rate, so these thirty trials must not be read as a curve extending to the cap.
All thirty cells had been seen once by reading 19's void run with a fresh client per trial.

The in-band coverage report is each client's own account of what it was served. Committing it would
have made a compliance count reproducible, not truthful — and here it did not even manage
reproducible, because one of the four clauses rested on mtimes the tree does not carry. The mtime
rule constrains the order of the two writes; it never constrained their honesty.

## Provenance

Thirty scores in `docs/curve/reading-20-scores/`, thirty raw client coverage reports in
`docs/curve/reading-20-coverage/`, and the derived per-trial compliance table with claimed and true
values in `docs/curve/reading-20-compliance.json`. `reading-20-tally.json` is computed over the
scores:

```sh
bin/curve tally docs/curve/reading-20-scores/*.score.json                    # 30 of 30
jq -s 'map(.next_read_sends) | add' docs/curve/reading-20-coverage/*.json    # 0 paged reads
```

Every number this reading PUBLISHES — the correct-address rate and the paged-read count — is
computed from that committed data, none typed. The two counts that could not be are withdrawn, and
the next section says why.

## Rule 3 was not met, and two counts are withdrawn

Rule 3 of the plan is unambiguous, and it is worth quoting rather than paraphrasing: "**Every RATE
and COUNT this reading publishes — the correct-address rate, the compliance count, the paged-read
count and the `no_answer` count — is computed from committed data**", with "**the one exception**"
being cost, "named here rather than left to be noticed."

Two of those four are not. Cost's exception does not stretch to cover them: the plan named one
exception, and extending it after the results are in is the same move reading 19 was voided for,
run in the author's own favour. So the two counts are **withdrawn** — not restated as observations,
not reported at a lower figure, not published at all. What remains published is what rule 3 was
actually met for.
- **The compliance table's `order` clause.** It was computed from file mtimes while the cells
  existed; git preserves no mtimes and the `result.json` files were not committed, so a reader
  cannot re-derive it. `reading-20-compliance.json` therefore carries `"order": null` and
  `"compliant": null` rather than a conclusion nothing in the tree supports — the count cannot be
  recomputed from it in either direction. The other three clauses — `last`, `blocks`, `tools` — are
  fully recomputable from the committed claimed-and-true values beside them, and are kept.
  blocks, last line, tools — are fully recomputable from the committed claimed-and-true values.
- **Whether any trial was retried, and whether any produced no answer.** Thirty complete
  score/coverage pairs cannot distinguish thirty first attempts from a discarded no-answer followed
  by a retry, which is exactly reading 19's failure. This one matters twice over: **any retry is on
  the plan's list of void conditions**, and so is a `coverage.json` not strictly newer than its
  `result.json`. The tree settles neither, so this document asserts neither that they happened nor
  that they did not — which is also why the reviewer's argument at the end is not disposed of by
  anything here.

**A future reading of this arm should commit, per attempt, the `result.json`, the `coverage.json` and
a dispatch receipt recording the order of the two writes and the outcome of every attempt including
the ones that produced nothing.** That is the smallest change that would let rule 3 be met, and it is
cheap; it was simply not foreseen when this plan was frozen.

Cost is the third thing the tree does not carry, and it is the only one of the three the plan
provided for: it comes from uncommitted request records and is reported unevaluated, as rule 3 says.

## A reviewer held that this reading should void

Recorded because the argument is worth more in the open than overridden in silence. Reviewing
PR #116, Codex held that compliance evidence which cannot be re-derived from the tree should void
this reading whole: the missing evidence is exactly what would rule out two of the plan's own
enumerated void conditions, and accepting the author's word for it after a favourable result is the
asymmetry reading 19 was voided for.

The reading was kept, on the plan's own text rather than on the author's confidence. **What voids
this reading is enumerated**, in a section headed "What would void this reading": a change to the
cells, the plan or the harness after the first trial; a client reaching `answer.json`; a
`coverage.json` not strictly newer than its `result.json`; a cell failing the three-field identity
check; any retry of any trial. A shortfall against rule 3 is not on that list. Voiding for a reason
the plan does not list is itself a rule invented after the results were seen — the same error as
keeping a number the plan does not license, run in the other direction.

**Where the reviewer was right, and it cost this reading two counts.** The first version of this
document defended those counts by citing `reading-04-plan.md`, which pre-registered that compliance
"will be reported rather than reproducible, as in readings 2 and 3". The citation is accurate and
the defence is not: reading 4 pre-registered non-reproducible evidence, and reading 20 pre-registered
the opposite and then did not deliver it. A stricter promise, broken, cannot be excused by an
earlier reading's looser one. That is why the two counts are now withdrawn rather than qualified.

**What a sceptical reader is left with**, and it is the whole of what this reading asserts: 30 of 30
correct addresses and 0 paged reads, both recomputable from the committed scores and coverage files
by the two commands above. That is the first number this corpus has for the arm mrw actually ships,
it is a claim about 2 KB and 20 KB and nothing else.

## Provenance, continued

The thirty cells were verified before the first trial against `docs/curve/reading-13-scores/` on
`trial_id`, `served_bytes` and the planted line, and against the plan's own committed sha256 table;
all thirty matched on all four fields. The thirty dispatched prompts were verified byte-identical
apart from the cell name.
