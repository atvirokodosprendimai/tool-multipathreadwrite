# Reading 20, result: 30 of 30 at 2 KB and 20 KB through the delivery mrw ships

**Collected 2026-09-06 under `reading-20-plan.md`, committed before any trial ran and amended twice
before its first trial, both amendments recorded in the plan. Thirty trials, no void, no deviation.**

**30 of 30 correct addresses. 30 of 30 compliant. 0 paged reads. 0 no-answers.**

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
3. **Holds, with one clause not reproducible.** Compliance 30 of 30, from
   `reading-20-compliance.json`: every trial reported every service block and the true last line —
   both recomputable, because that table carries the claimed AND true values — and used no forbidden
   tool. The fourth clause, `coverage.json` written strictly after `result.json`, was checked on
   mtime at collection time and **cannot be re-derived from the tree**: git does not preserve mtimes
   and the `result.json` files were not committed. See "What is not reproducible" below.
4. **Unevaluated, as the plan requires.** Cost comes from request records this corpus has never
   committed. Under rule 3 it is an observation and never a published number, and it is reported as
   unevaluated rather than estimated. No claim above rests on it.
5. **Holds exactly.** `next_read_sends` is 0 in all thirty. Both sizes sit far below
   `mcp.MaxResultChars`, so nothing paged and nothing was truncated.
6. **Holds.** Zero `no_answer` trials, against at most one allowed. Reading 19's void run saw one at
   20 KB; this run saw none, and under rule 2 none was retried because none needed to be.

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

The in-band coverage report is each client's own account of what it was served. Committing it makes
the compliance number reproducible, not truthful; the mtime rule constrains the order of the two
writes but not their honesty.

## Provenance

Thirty scores in `docs/curve/reading-20-scores/`, thirty raw client coverage reports in
`docs/curve/reading-20-coverage/`, and the derived per-trial compliance table with claimed and true
values in `docs/curve/reading-20-compliance.json`. `reading-20-tally.json` is computed over the
scores:

```sh
bin/curve tally docs/curve/reading-20-scores/*.score.json
```

Every rate and count above is computed from that committed data, none typed.

## What is not reproducible from the committed data

Stated plainly because the plan's rule 3 promised that every rate and count would be recomputable,
and two things fall short of it. Neither is on the plan's enumerated list of void conditions, so
neither voids the reading; both narrow what it may be read to assert.

- **The compliance table's `order` clause.** It records the conclusion `"order": true`, not evidence.
  It was computed from file mtimes while the cells existed; git preserves no mtimes and the
  `result.json` files were not committed, so a reader cannot re-derive it. The other three clauses —
  blocks, last line, tools — are fully recomputable from the committed claimed-and-true values.
- **That no trial was retried and none produced a no-answer.** Thirty complete score/coverage pairs
  cannot distinguish thirty first attempts from a discarded no-answer followed by a retry, which is
  exactly reading 19's failure. The run is reported as conducted — one dispatch per cell, no retry,
  no trial without an answer — and that report is the author's, not the tree's.

**A future reading of this arm should commit, per attempt, the `result.json`, the `coverage.json` and
a dispatch receipt recording the order of the two writes and the outcome of every attempt including
the ones that produced nothing.** That is the smallest change that would make these two claims
reproducible, and it is cheap; it was simply not foreseen when this plan was frozen.

Cost is the third, and is the plan's own named exception: it comes from uncommitted request records
and is reported unevaluated.

## Provenance, continued

The thirty cells were verified before the first trial against `docs/curve/reading-13-scores/` on
`trial_id`, `served_bytes` and the planted line, and against the plan's own committed sha256 table;
all thirty matched on all four fields. The thirty dispatched prompts were verified byte-identical
apart from the cell name.
