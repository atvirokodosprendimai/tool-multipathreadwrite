# Quiet receipts versus native — results, 2026-11-01

Registered in [QUIET-PREREGISTRATION.md](QUIET-PREREGISTRATION.md), pushed in `a18e4860` before the
first call. Run 8, two rounds, 24 launches. Both canary pairs passed and all 20 counted cells were
solved, with exact trees and verified final checks. All-in usage $0.234586 and $0.243554, $0.478140
in all. One-minute load at the round starts 6.50 and 2.96; no peer build. Every cost reproduces within 3e-18.
Exported records: [quiet-pilot.json](quiet-pilot.json); tool results by kind: [receipt-breakdown.json](receipt-breakdown.json).

| Five-job total | Round 1 | Round 2 |
|---|---:|---:|
| Native (A) cost | $0.100540 | $0.103352 |
| MRW, quiet receipts (C) cost | $0.114335 | $0.120958 |
| A minus C | $-0.013795 | $-0.017607 |
| A / C cache creation | 15,381 / 17,849 | 15,394 / 18,739 |
| A / C CLI turns | 15 / 19 | 17 / 18 |
| A / C output tokens | 2,845 / 3,261 | 2,962 / 3,619 |
| C `mrw write` result characters | 2,331 | 1,341 |
| C refused writes | 0 | 0 |

## The registered verdicts

- **Cost: native lower**, by $0.013795 and $0.017607, mean $0.015701, larger than both spreads
  (A $0.002811, C $0.006623).
- **Cache creation, turns and output tokens: native lower** by the same rule; first-request context
  MRW lower.
- **Prediction 1, every cell solved: held.**
- **Prediction 2, C's write results under 4,000 characters: held** (3,672 over 10 cells, against
  about 11,700 per 10 cells in runs 6 and 7).
- **Prediction 3, native's lead $0.015–$0.035 a round: missed.** Round 1's lead, $0.013795, is under the
  registered $0.015 floor; round 2's, $0.017607, is inside it. The cost verdict above is unaffected.
- **Prediction 4, turns no result: missed.** MRW took 2.5 more turns a round, more than either spread.

## Against runs 6 and 7, descriptively

Not pooled; same jobs, binaries and setup but for the shim. Native's five-job cost stayed in its
band ($0.100–$0.110 then, $0.101–$0.103 now). MRW's fell from $0.122–$0.149 to $0.114–$0.121, and
native's lead from a mean $0.031 to $0.016 a round. The write results here are about 8,000 characters
per 10 cells shorter than in runs 6 and 7 (by the breakdown's command heuristic, which counts whole tool
results). Assuming about 4 characters a token and every such token paid once at the cache-creation
rate of $4 a million, that is near $0.004 a round: an illustrative estimate across different runs,
not an isolated effect of the shim. This run also had no refused writes against three in run 7's
second round, and other model behaviour differs between runs, so the narrowing cannot be divided
between the shim and those differences.

What is left of the gap is MRW's extra turns and output, and its numbered reads. On this evidence a
switch that turns the receipts off would narrow the gap, by an amount this design cannot isolate; it
did not close it here.

The shim as run joined `-C DIR` or `--root DIR` into one word before passing it on, which splits a
directory holding a space (the Codex re-review of #586); no MRW cell in run 8 called mrw with `-C` or
`--root` at all, so no call of this run went through that path. The shim is fixed after the run.
