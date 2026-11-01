# Native versus MRW on v1.81.0 — registration, 2026-11-01

The 2026-10-30 grouped pilot ([results](GROUPED-RESULTS.md)) found MRW 20.3% dearer than native on five
synthetic jobs, with the full guide appended and the benchmark guard refusing plan files at variable
paths. Since then: the core guide teaches the hunk header (ADR-184, ADR-185), core-first costs the same
as the full guide ([CORE-RESULTS.md](CORE-RESULTS.md), runs 4 and 5), the guard resolves the plan paths
agents write (ADR-186), and a write no longer runs the project's check unless asked (ADR-187, v1.81.0).
This run asks whether MRW, as `docs/agents.md` now tells a host to set it up, still costs more than
native on the same jobs. The owner chose this measurement on 2026-11-01.

## Arms and workload

- **A (native):** Claude Code with Bash, Read, Edit, Write, Glob and Grep, no MRW on PATH, nothing
  appended: the grouped pilot's arm A.
- **C (MRW, core first):** Bash and Glob, MRW v1.81.0 on PATH, `mrw instructions --core` appended
  (SHA256 recorded per cell), the `bench hook` write guard built from `main` at the run (ADR-186).
- The same five grouped jobs (`bench/grouped-pilot/tasks`), byte for byte, ten edit sites each. Each
  prompt says to run the declared checker, so both arms check their own work; with ADR-187, MRW's
  writes run no check of their own.
- Model `claude-sonnet-5-5` requested and observed; the installed Claude Code, its version recorded.

## Design

Two runs of two rounds, as Amendment 3 of [CORE-PREREGISTRATION.md](CORE-PREREGISTRATION.md) ran F and
C: each round opens with an A and a C canary, then the five jobs in seed 5030's order with the arm
order alternating by pair; round 2 reverses each pair; run 2 starts from the reverse. Over the four
rounds every job runs each arm first twice. Each round has its own run and adapter directories.

**Budget:** at most 12 launches per round, `--max-turns 10`, $0.50 per call, and one $3 cap across both
runs: run 2's first round is given $3 minus run 1's usage, its second round $3 minus everything
before it, and run 2 starts only if run 1's two rounds completed, isolated, and used under $1.50.
Load below the core count; the load and the output of `pgrep -fl 'go test|cargo|xcodebuild'` are
recorded at each round's start. The driver change below is the only code change.

**Driver change:** the canary pair and the per-round gate, today wired only for arms `FC`, run for any
pair the run names (`AC` here); arm A launches unnamed, as the adapter has always taken it.

## The rule, fixed now

Eligibility: every counted cell of both runs solved, every measure known; otherwise "no eligible
comparison". For a measure, s1…s4 are A's five-job total minus C's in each round, and each arm's range
is its highest five-job total minus its lowest:

- **MRW lower:** s1…s4 all positive and their mean larger than both ranges.
- **Native lower:** all four negative and the mean's magnitude larger than both ranges.
- Otherwise "no result" for that measure.

Measures: cost, first-request context, CLI turns, output tokens. Also reported: guard refusals in C
cells, MRW write refusals, model-issued `mrw instructions` calls.

## Predictions, written before the first call

1. **Quality:** both arms solve all 20 of their cells.
2. **First-request context: MRW lower.** C's first request was 4,977–5,118 tokens in runs 4 and 5;
   native's was 6,985–7,124 on 2026-10-30, with six tools described.
3. **Cost: native lower.** On 2026-10-30 MRW cost 20.3% more; since then its starting context fell
   below native's, but its cache creation and receipts remain. A "no result" or "MRW lower" is a miss.
4. **Turns: no result.** 16 against 17 on 2026-10-30.
5. **Guard refusals:** at most 2 in C's 20 cells, against about one cell in three before ADR-186.

## Out of scope

- Real or held-out work, other models, and the full-guide setup (permanent: boundary: one question)
- Changing guidance from this run (permanent: boundary: a later record would decide that)
