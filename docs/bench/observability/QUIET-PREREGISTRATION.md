# Quiet receipts: MRW with `--quiet` writes versus native — registration, 2026-11-01

[NATIVE-RESULTS.md](NATIVE-RESULTS.md) found native cheaper than MRW in all four rounds, by a mean
$0.031 per five jobs, and MRW's write receipts at 23,396 characters across its 20 cells, where
native's `sed` edits print nothing. The owner asked, 2026-11-01, whether silencing the receipt would
close the gap. `mrw write --quiet` already prints only failed hunks and the summary line. This run
measures what that saves, without changing mrw. The owner's follow-up the same day: keep the receipts by default and add a switch that turns them off. The shim does what such a switch would do, so this run prices it; the switch itself is a later record.

## Arms and workload

- **A (native):** as in runs 6 and 7.
- **C (MRW, quiet):** as in runs 6 and 7 (MRW v1.81.0, core guide appended, Bash and Glob, the guard),
  except that the `mrw` on the agent's PATH is a shim, `bench/quiet-shim/mrw`, which adds `--quiet` to
  every `mrw write` that has neither `--quiet`, `--terse` nor `--json`, and passes every other call to
  the real binary unchanged. The appended guide is the real `mrw instructions --core`, so the agent
  is not told about the shim.
- The five grouped jobs, the same seed, order and arm alternation as run 6.

## Design and budget

One run of two rounds, round 2 reversing each pair, a canary pair per round, the round-2 gate, at most
12 launches a round, `--max-turns 10`, $0.50 a call, $3 for the run. Load and peer processes are
recorded at each round's start.

**The two-round rule**, as CORE-PREREGISTRATION.md's: eligibility needs all 20 cells solved; for a
measure, MRW is lower when A minus C is positive in both rounds and the mean is larger than both arms'
spreads (each arm's two-round difference); native is lower when both are negative and the mean's
magnitude is larger than both spreads; otherwise no result. Measures: cost, cache creation, turns,
output tokens; reported beside them: C's write-receipt characters and refused writes.

## Predictions, written before the first call

1. **Quality:** all 20 cells solved.
2. **Receipts:** C's `mrw write` results total under 4,000 characters over its 10 cells, against
   about 11,700 per 10 cells in runs 6 and 7.
3. **Cost: native lower** still. The quiet receipts are estimated to save about $0.005 per round of a
   $0.031 gap, so the gap stays: native's lead per round is $0.015–$0.035. A gap under $0.015 is a
   miss, and says the receipts mattered more than estimated.
4. **Turns: no result**, as in runs 6 and 7.

Runs 6 and 7 are not pooled with this one; a comparison with them is descriptive.

## Out of scope

- Making `--quiet` mrw's default (permanent: boundary: a later record decides, on this result)
- Read size and refused writes, the other measured costs (permanent: boundary: one change at a time)
