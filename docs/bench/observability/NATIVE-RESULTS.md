# Native versus MRW on v1.81.0 — results, 2026-11-01

Registered in [NATIVE-PREREGISTRATION.md](NATIVE-PREREGISTRATION.md), pushed in `51d1a6c4` before the
first call. Runs 6 and 7, two rounds each, 48 launches. Every canary passed and all 40 counted cells
were solved, with exact trees and verified final checks. All-in usage: $0.289870, $0.258189,
$0.241419 and $0.277449 by round, $1.066927 together, against one $3 cap. One-minute load at the
round starts was 4.04, 2.11, 2.31 and 2.61; no peer `go test`, `cargo` or `xcodebuild` was running.
Every cost reproduces from its tokens within 7e-18. Exported records: [native-pilot.json](native-pilot.json).

| Five-job total | Run 6 r1 | Run 6 r2 | Run 7 r1 | Run 7 r2 |
|---|---:|---:|---:|---:|
| First pair's arm order | AC | CA | CA | AC |
| Native (A) cost | $0.109610 | $0.104658 | $0.100065 | $0.109263 |
| MRW (C) cost | $0.143487 | $0.133961 | $0.121770 | $0.148786 |
| A minus C | $-0.033877 | $-0.029304 | $-0.021706 | $-0.039523 |
| A / C first-request context | 35,069 / 25,178 | 35,069 / 25,185 | 35,069 / 25,183 | 35,074 / 25,182 |
| A / C CLI turns | 17 / 18 | 17 / 19 | 16 / 19 | 17 / 21 |
| A / C output tokens | 3,073 / 3,578 | 3,188 / 3,780 | 2,780 / 3,205 | 3,100 / 4,065 |
| C guard refusals | 0 | 1 | 0 | 0 |
| C refused MRW writes | 0 | 0 | 0 | 3 |

## The registered verdicts

- **Cost: native lower.** Native was cheaper in all four rounds, by a mean $0.031103 per five jobs,
  larger than both ranges (A $0.009546, C $0.027016). Over the four rounds MRW cost $0.548006 against
  $0.423596, 29% more.
- **First-request context: MRW lower**, by 9,884–9,892 tokens over five jobs in every round
  (ranges 5 and 7): about 1,980 tokens a request, since native describes six tools.
- **Turns: no result.** MRW took more in every round (mean 2.5), but C's range was 3.
- **Output tokens: no result.**
- **Cache creation: native lower** (reported, not registered as a verdict): MRW created 6,422 more
  tokens per round on average, about $0.026 of the $0.031 gap.
- **Predictions:** 1, every cell solved: held. 2, MRW's starting context lower: held. 3, native
  cheaper: held. 4, turns no result: held. 5, at most 2 guard refusals in C's 20 cells: held (1).

## Where MRW's extra cost comes from

Its requests start smaller, and it read less from cache, but what it put into the conversation was
larger. Over the 40 cells, the tool results MRW's agent received came to 95,373 characters against
native's 59,334:

- **write receipts, about 23,400 characters.** Native's `Edit` and `sed` edits print next to nothing;
  each `mrw write` prints a row per hunk, a summary, and the file's new sha.
- **reads, about 17,200 characters more**, from line numbers and file headers on what native reads with
  `cat`.
- **refused writes:** four of C's tool results were refusals, one by the guard and three by mrw, each a round paid. The mrw ones were of two kinds. One is a plan with
  the wrong relative paths. The other is a JSON line's anchor written `anchor="    "timeout": 30"`: the
  core shows only `anchor="TEXT"`, the parser reads that value as `    timeout: 30`, and the refusal
  ("anchor … not in line 5") does not say that single quotes, `anchor='    "timeout": 30'`, were needed.

These are accounting facts over 40 synthetic cells, not a causal proof. They point at what a
following change would have to shrink — receipt and read size, and the refusals — for MRW to cost what
native does on edits this simple. No guidance changes from this run.
