# ADR-193 T2: Stage 5's B under the frozen counting rule

Counted 2026-11-03 over Stage 5's stored B transcripts, the rule of ADR-193 Decision 4 and the resampling of Decision 5,
both in `scripts/bench_edits.py`. Stage 5 ran mrw v1.68.0 (before ADR-171 made `body=` optional) on Claude Code 2.1.302;
no B attempt used `--format=search_replace` or `--format=apply_patch`, so the attributed columns are zero.

script sha256: 16e62d90339b7159e1cf74fa7cbc3197b6afabbd1075edd264b8c45754f31487
helper sha256: 49001558d004aec635965ec4a53c7628ff5aaffcd80a91aec1f44255f1fcc1ac

transcripts: ~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5 (382 B transcripts, 191 per model, in `block-1` to `block-5`; no B receipt there is excluded or replaced)

```
S5=~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5
python3 -B scripts/bench_edits.py count --receipts $S5 --transcripts $S5 --configs B
python3 -B scripts/bench_edits.py null --transcripts $S5
```

## Stage 5's B per model, against the Context

| | Opus 5.5 | Context | Sonnet 5.5 | Context |
|---|---|---|---|---|
| B attempts counted | 191 | 191 | 191 | 191 |
| decided write calls | 555 | 555 | 359 | 359 |
| landed | 461 | | 253 | |
| refused | 94 | 94 | 106 | 106 |
| refused rate, of decided | 16.9% | 16.9% | 29.5% | 29.5% |
| undecided | 3 | 3 | 9 | 9 |
| blocked by bench's own Bash hook | 17 | 17 | 11 | 11 |
| attempts with a refusal | 73 | 73 | 74 | 74 |
| untokenized | 0 | | 0 | |
| output tokens per landed write call | 1,611 | 1,611 | 1,299 | 1,299 |
| attributed SEARCH/REPLACE calls | 0 | 0 | 0 | 0 |

Every count the Context quotes reproduces exactly under the frozen rule. Output tokens per landed write call is the sum of
`usage.output_tokens` over the 191 receipts divided by landed calls (742,652 / 461 and 328,753 / 253); `count` does not print it.

## The null rate

`null` draws, per iteration and per model, 36 of that model's 191 B attempts without replacement with Python's
`random.Random(193).sample` (the models in the order of their names, one generator for the run), takes the first 18 as B and the
last 18 as S, and judges Decision 5's rule with uptake and coverage taken to hold, 20,000 times. The record fixes the count and
the seed but not the sampler, so the figures below carry Monte Carlo error of about 0.3 points (0.15 on the joint rate).

| | Opus PASS | Sonnet PASS | both PASS | both, (i) unavailable |
|---|---|---|---|---|
| The draft's (Decision 5's first figures) | 24.8% | 24.5% | 6.1% | 4.1% |
| `null`, missing output and cost left as the receipts hold them | 25.1% | 24.9% | 6.2% | 4.5% |
| `null`, missing output and cost imputed against S (the frozen rule) | 22.4% | 24.9% | 5.6% | 4.0% |

**The difference, written down and not smoothed.** Two Opus B attempts (`sympy__sympy-16597`, `django__django-16032`) have no `result`
line in the transcript; their receipts hold zero output tokens and no cost. Decision 5's imputation treats those two as missing
and gives a B attempt the smallest value the model recorded; counted as the receipts hold them (zero), the Opus rate moves from
22.4% to 25.1%. Other seeds (`null --seed 1` to `--seed 6`, same sampler) give, for the frozen rule, Opus 22.2-22.7%, both
5.4-6.0% and both with (i) unavailable 3.8-4.2%; and for the receipts-as-held reading Opus 24.6-25.2%, both 6.1-6.7% and
(i) unavailable 4.5-4.9%. The draft's Opus 24.8% matches the as-held reading and its (i)-unavailable 4.1% matches the imputed
one: no single reading reproduces all four of its figures, and the sampler it used is not recorded. The frozen rule's false-SHIP
rate is **5.6%** at seed 193 (4.0% with (i) unavailable). The record now quotes the frozen figures: the session lead decided
that on 2026-11-03 and reported it to Zy; it changes no rule. Sonnet has no attempt without a result, and its two readings are
identical. The counts above reproduce exactly.
