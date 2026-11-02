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

- The binary is built from `main` at a commit no older than `345157f2`, named in the results, not v1.81.0: ADR-188 (`MRW_RECEIPT`, unset here),
  ADR-189 (narrower gutter), ADR-190 (a quoted `search_replace` block licenses its lines) and
  ADR-191 are in it. The driver is `scripts/grouped_pilot.py` at `345157f2` (#603).
- Arm R is new. Arms A and C are unchanged in setup.

## Runs

Two runs, each two arms. Not three: the driver admits a pair while 180 s of the round's 540 s
remain, whatever the pair holds, so a three-cell group can outrun the round where a two-cell one
did not in run 8; and a three-arm order leaves the middle arm's position uncounterbalanced.

- **Run 9: A vs R.** The headline, comparable with runs 6–8.
- **Run 10: C vs R.** Prices the bundle R adds to C, which is the host `Read`, the host `Grep` and the hook
  together. It does not isolate the hook: no arm has `Read` without it. Run after run 9.

Each run: two rounds, round 2 reversing each pair, a canary per arm per round, the round-2 gate,
`--max-turns 10`, the same seed, jobs and order as run 6. That makes 12 launches a round, 24 a run and
48 in all, eight of them canaries. The budget is $0.50 a call and a nominal $3 stop per run, $6 in all.
As CORE-PREREGISTRATION.md says, the adapter checks spend before a launch without reserving that
launch's $0.50, and the CLI's own cap can overshoot, so the last launch of a run can carry it past $3.
Actual spend is reported. Neither run compacts (`compact_pct` unset). A compacting run is registered
by an amendment to this file before its first call, once one live cell has shown how many compactions
a given `compact_pct` causes.

**Load gate.** A round launches only when the one-minute load is at most 10 (the core count) and no
peer build runs, both recorded at the round's start; above it the run waits or is abandoned, never
run and discounted. The hook's one-second lock deadline makes load a confound specific to R: a hook
that misses it records nothing, and says so only on standard error, which the agent never sees; R
would then read as "no gain". Each R cell's hook stderr is kept and reported.

**The two-round rule**, as CORE-PREREGISTRATION.md's: eligibility needs all 20 cells of a run solved;
for a measure, an arm is lower when the difference is the same sign in both rounds and the mean is
larger than both arms' spreads; otherwise no result. Measures: cost, cache creation, turns, output
tokens. Reported beside them: refused writes, and the mechanism count below.

**Mechanism counts**, from each R cell's stream-json transcript by `scripts/bench_mechanism.py`, committed
with a test on synthetic transcripts before the first call. These rules are frozen with it:

- An *overlapping read* is a Bash call whose command runs `mrw read` naming a file F, made after a host
  `Read` of F and with no landed write to F in between. It is a proxy, not a measure of forced work:
  the core guide tells the agent to read with mrw, so an agent may repeat a read it did not need. A
  command that names F several times counts once for F. `--quiet` reads count. A host `Read` whose
  output was cut still counts as a `Read` of F.
- A *hook-licensed write* is an `mrw write` to F that applied, made with no `mrw read` of F before it in
  the cell and at least one host `Read` of F. Under the guard, only the hook can license such a write,
  so one such write shows the hook worked in that cell.
- A *refused unread write* is an `mrw write` to F refused for a line not read, made after a host `Read`
  of F. It is the hook failing.
- A command whose files the classifier cannot name (a variable, a subshell, a `--files-from` list, a
  `--grep` walk) is counted as *unknown* and reported by itself. It is in neither count.

The cell's `hook.stderr` is reported beside these counts.

## Predictions, written before the first call

1. **Quality:** all 20 cells of each run solved.
2. **The hook is exercised:** R calls the host `Read` in at least 5 of its 10 cells in each run, and
   makes at least 1 hook-licensed write in each run. Either one failing is a miss. It leaves
   predictions 3–5 untested, not refuted.
3. **Mechanism:** at most 1 overlapping read and 0 refused unread writes across R's 10 cells in each
   run.
4. **Run 9 cost: native lower** still, by less than run 8: native's lead per round $0.000–$0.015.
   R lower than A is a miss in R's favour. A lead over $0.015 is a miss: the bundle R adds did not pay
   for itself here. This design cannot tell the hook's share from `Read`'s or `Grep`'s.
5. **Run 10: R lower in turns.** Cost: no result. The extra tool definitions cost R on every
   request, and the turns it saves may not cover them.

Runs 6–8 are not pooled with these; a comparison with them is descriptive.

## Out of scope

- A compacting run (deferred: an amendment here, after the live compaction check)
- Making the read hook part of `docs/agents.md`'s default setup (permanent: boundary: a later record decides, on this result)
- A host `Edit` or `Write` in any MRW arm (permanent: boundary: edits go through mrw)
