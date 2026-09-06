# Reading 20, result: 30 of 30 at 2 KB and 20 KB through the delivery mrw ships

**Collected 2026-09-06 under `reading-20-plan.md`, committed before any trial ran and amended twice
before its first trial, both amendments recorded in the plan. Thirty trials, no void, no deviation.**

**30 of 30 correct addresses. 0 paged reads. Compliance 30 of 30 — three of its four clauses
computed from committed data, the fourth attested. No retry and no trial without an answer,
attested.** Under the plan's rule 3 an attested item is an observation reported beside the result,
never a published number; "What is not reproducible from the committed data" below says which is
which and why.

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
3. **Holds on three clauses; the fourth is attested, not reproducible.** Compliance 30 of 30, from
   `reading-20-compliance.json`: every trial reported every service block and the true last line —
   both recomputable, because that table carries the claimed AND true values — and used no forbidden
   tool. The fourth clause, `coverage.json` written strictly after `result.json`, was checked on
   mtime at collection time and **cannot be re-derived from the tree**: git does not preserve mtimes
   and the `result.json` files were not committed. Under rule 3 that clause is an observation and
   not a published number. See "What is not reproducible" below.
4. **Unevaluated, as the plan requires.** Cost comes from request records this corpus has never
   committed. Under rule 3 it is an observation and never a published number, and it is reported as
   unevaluated rather than estimated. No claim above rests on it.
5. **Holds exactly.** `next_read_sends` is 0 in all thirty. Both sizes sit far below
   `mcp.MaxResultChars`, so nothing paged and nothing was truncated.
6. **Holds on attestation, not on the tree.** Zero `no_answer` trials against at most one allowed,
   and under rule 2 none was retried because none needed to be. Thirty committed score/coverage
   pairs cannot distinguish that from a discarded no-answer followed by a retry, so under rule 3
   this is an observation and not a published number. Reading 19's void run saw one no-answer at
   20 KB; this run saw none.

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
and two things fall short of it. Rule 3 already names the category they belong to: cost is "an
observation reported beside the result, never a published number", and these two are the same kind
of thing. Neither is on the plan's enumerated list of void conditions, so neither voids the reading
— but both are demoted from published numbers to attested observations, and every claim above that
rests on one now says so where the claim is made.

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

## A reviewer held that this reading should void

Recorded because the argument is worth more in the open than overridden in silence. Reviewing
PR #116, Codex held that compliance evidence which cannot be re-derived from the tree should void
this reading whole: the missing evidence is exactly what would rule out two of the plan's own
enumerated void conditions, and accepting the author's word for it after a favourable result is the
asymmetry reading 19 was voided for.

The reading was kept, for a reason that is checkable rather than a matter of confidence.
**Author-attested compliance is what every published result in this corpus already stands on.**
`reading-04-plan.md` says so in as many words — compliance and cost "come from transcripts and
request records that are not committed, and will be reported rather than reproducible, as in
readings 2 and 3." A rule that not-re-derivable implies void would void readings 2 through 17
retroactively, and that has never been this corpus's rule. The void conditions this plan lists are
events; no retry and no no-answer occurred. Reading 19 voided partly on an attested event its author
disclosed against interest, which is the evidence that the standard runs in both directions.

What the finding did change is the shape of the claims. The headline, prediction 3 and prediction 6
now carry the attestation with them rather than leaving it ninety lines below, and the two items are
demoted to rule 3's category. **A reader who thinks the reviewer had the better of it can discount
every attested item and is left with 30 of 30 correct addresses, three of four compliance clauses,
and `next_read_sends` 0 — all computed from committed data.** That subset is what the reading is
worth at its most sceptical reading, and it is still the first number this corpus has for this arm.

## Provenance, continued

The thirty cells were verified before the first trial against `docs/curve/reading-13-scores/` on
`trial_id`, `served_bytes` and the planted line, and against the plan's own committed sha256 table;
all thirty matched on all four fields. The thirty dispatched prompts were verified byte-identical
apart from the cell name.
