# Host Read with the read hook: MRW arm R versus native and versus arm C — registration, 2026-11-02

[QUIET-RESULTS.md](QUIET-RESULTS.md) (run 8) found native cheaper than MRW with quiet receipts by a
mean $0.015701 per five jobs, the rest of the gap being MRW's extra turns, output and numbered reads.
The v1.79.0 recheck traced 26 re-reads to a host `Read` followed by an `mrw read` of the same lines,
which the guard required. ADR-191 (`mrw hook read`, merged in #600) licenses the lines a host `Read`
showed, so an arm that keeps the host's `Read` no longer pays that second read. The owner asked,
2026-11-02, for a verification with numbers. This registration prices it.

## Arms

- **A (native):** as in runs 6–8.
- **C (MRW, quiet):** as in run 8: Bash and Glob, the core guide appended, the guard, and the agent's
  `mrw` is `bench/quiet-shim/mrw`.
- **R (MRW, host Read and the read hook):** C, plus the tools `Read` and `Grep` (still no `Edit` or
  `Write`; edits go through mrw), plus a PostToolUse hook on `Read` that runs `mrw hook read`
  (`cell_settings` in `scripts/grouped_pilot.py`). The guide is C's, byte for byte, so the agent is
  told nothing about the hook. The hook is run through the same shim, which `exec`s every call that is
  not a write, so the payload reaches the real binary on stdin unchanged.

## Differences from run 8, all of them

- The binary is `main` at the merge of #603, not v1.81.0: ADR-188 (`MRW_RECEIPT`, unset here),
  ADR-189 (narrower gutter), ADR-190 (a quoted `search_replace` block licenses its lines) and
  ADR-191 are in it. The driver is the one #603 merged.
- Arm R is new. Arms A and C are unchanged in setup.

## Runs

Two runs, each two arms, because three arms do not fit: a round's 540 s deadline and its 180 s
per-pair guard held 12 launches in run 8, and three arms make 18; a three-arm order also leaves the
middle arm's position uncounterbalanced.

- **Run 9: A vs R.** The headline, comparable with runs 6–8.
- **Run 10: C vs R.** Isolates the host Read with the hook. Run after run 9.

Each run: two rounds, round 2 reversing each pair, a canary per arm per round, the round-2 gate, at
most 12 launches a round, `--max-turns 10`, $0.50 a call, $3 a run, the same seed, jobs and order as
run 6. Neither run compacts (`compact_pct` unset); a compacting run is registered by an amendment to
this file, before its first call, once one live cell has shown how many compactions a given
`compact_pct` causes.

**Load gate.** A round launches only when the one-minute load is at most 10 (the core count) and no
peer build runs, both recorded at the round's start; above it the run waits or is abandoned, never
run and discounted. The hook's one-second lock deadline makes load a confound specific to R: a hook
that misses it records nothing, silently, and R would read as "no gain".

**The two-round rule**, as CORE-PREREGISTRATION.md's: eligibility needs all 20 cells of a run solved;
for a measure, an arm is lower when the difference is the same sign in both rounds and the mean is
larger than both arms' spreads; otherwise no result. Measures: cost, cache creation, turns, output
tokens. Reported beside them: refused writes, and the mechanism count below.

**Mechanism count.** In an R cell, a *forced re-read* is an `mrw read` whose served lines include a
line a host `Read` of the same file had returned earlier in the cell, with no write to that file in
between. Counted from the transcripts by a script committed with the results.

## Predictions, written before the first call

1. **Quality:** all 20 cells of each run solved.
2. **The hook is exercised:** R calls the host `Read` in at least 5 of its 10 cells in each run.
   Fewer is a miss and makes predictions 3–5 untested, not refuted.
3. **Mechanism:** at most 1 forced re-read across R's 10 cells in each run.
4. **Run 9 cost: native lower** still, by less than run 8: native's lead per round $0.000–$0.015.
   R lower than A is a miss in R's favour; a lead over $0.015 is a miss and says the hook bought
   nothing that the extra tool definitions did not spend.
5. **Run 10: R lower in turns.** Cost: no result (the extra tool definitions cost R on every
   request, and the turns it saves may not cover them).

Runs 6–8 are not pooled with these; a comparison with them is descriptive.

## Out of scope

- A compacting run (deferred: an amendment here, after the live compaction check)
- Making the read hook part of `docs/agents.md`'s default setup (permanent: boundary: a later record decides, on this result)
- A host `Edit` or `Write` in any MRW arm (permanent: boundary: edits go through mrw)
