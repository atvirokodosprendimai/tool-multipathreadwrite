# Core-first MRW guide ablation — registration, 2026-10-31

The [trace diagnosis](TRACE-DIAGNOSIS.md) found that the grouped MRW arm started every request with
about 5,600 more input tokens than native. That arm appended the full `mrw instructions` guide
(23,187 characters) to every fresh job. [Current guidance](../../agents.md#1-a-standing-instruction-in-the-file-the-agent-always-reads)
says to load the 1,598-character core first and the full guide only when needed (lines 51–53).
This pilot changes that one variable inside the MRW arm, runs it twice, and measures what changes.

## Question and pre-registered predictions

Does appending `mrw instructions --core` instead of `mrw instructions` lower task usage and cost
at equal verified quality on the five grouped jobs, by more than two runs of the same arm differ?

The predictions are written before any model call, from the diagnosis's saved numbers. The primary
readout is the F-minus-C first-request context. It is the one clean measurement of what the appended
guide costs per request, because both arms share tools, settings and hook; the diagnosis could not
separate the guide from tool exposure.

1. **First-request context.** The F arm repeats the earlier 12,591–12,727 input/cache tokens. The C
   arm lands at 7,000–7,900. This range assumes the whole earlier 5,604-token prefix difference was
   the guide: 23,187 characters over 5,604 tokens is about 4.14 characters per token, so the 21,589
   characters removed are about 5,215 tokens. Arm B also had fewer tools than native, so the guide's
   real share may be larger, and C may land below 7,000. That counts as a miss of this range, and the
   measured F-minus-C difference replaces the assumption.
2. **Cost.** In each round, C's five-job total costs $0.004–$0.010 less than F's. The upper end is
   about 5,215 tokens × 17 requests × $0.1 per million cache-read tokens. The lower end allows
   on-demand guide loads or extra rounds to give back up to about eight requests' worth of that
   saving. This assumes cache creation does not change between arms; the diagnosis found the extra
   creation accumulating after an already-warm prefix (TRACE-DIAGNOSIS.md, "What changed", item 1),
   so the assumption is reported as checked or broken. A net saving below $0.002 means the guide's
   size does not drive the arm's net cost on these jobs; it is not, on its own, a measure of the
   prefix's cost, which prediction 1 measures.
3. **On-demand guide.** The core text names `mrw write PLANFILE`, a multi-line replace's `anchor=`
   and `@@ path 0 create`, but not the hunk header's shape (`@@ PATH N-M replace anchor=…`). In the
   grouped pilot every one of the five MRW writes ran a plan its Bash loop built
   ([results](GROUPED-RESULTS.md), "The five MRW writes are confirmed"). So, in a round, at least
   one of C's five jobs either runs the full `mrw instructions` or has an MRW write refused. A round
   in which neither happens is a miss for that round; the prediction holds only when both rounds hit.
   Two misses mean the core is enough for these jobs.
4. **Quality.** Both arms solve all five jobs in both rounds.

A result outside a predicted range is reported as a miss. It is never re-explained into a hit, and a
range is never widened after the fact.

**Repeat spread decides what counts.** The comparison population is fixed now: the five jobs, both
arms, both rounds, 20 counted cells. Canaries are outside every task outcome and inside the budget. A
comparative conclusion is eligible only when all 20 cells are solved and every measure it uses is
known; otherwise the raw totals are reported and the result is "no eligible comparison", with no
switch to a subset after the fact. When eligible: for each arm, the spread is the absolute difference
between its two rounds' five-job totals. C is reported as cheaper only when its total is lower than
F's in both rounds and the mean saving across the two rounds is larger than both arms' spreads.
Anything less is reported as "within repeat spread: no cost result", even when prediction 2's range
is met. The same rule applies to first-request context, turns and output tokens. Two rounds give a
spread, not a variance estimate: no interval or significance is computed.

**What this does not settle.** The earlier grouped gap to native was $0.0200, and $0.0088 of it was
cache creation that built up after the prefix. The estimate above suggests a core-only change closes
at most about 45% of that gap. Native is not re-run here, so this pilot can neither confirm nor refute
that figure. It is a reason for the separate workflow experiment under Out of scope, not a prediction.

## Fixed workload and arms

These are the same five synthetic jobs, ten registered edit sites each, frozen in
[`GROUPED-PREREGISTRATION.md`](GROUPED-PREREGISTRATION.md): routing, JSON configuration, Markdown
documentation, coupled source/test updates, and multi-line structural changes. The prompts, initial
files, reference patches, decoys and checkers are reused byte for byte, and their recorded hashes are
re-verified before the first call. One attempt per job, arm and round: two rounds, 20 counted cells.

Both arms are the earlier arm B except for the appended file: the installed Claude Code, which reports
2.1.316 at registration where the earlier pilots ran 2.1.314; requested and observed
`claude-sonnet-5-5`; installed MRW v1.79.0 with its recorded binary hash; Bash and Glob only, pinned
MRW on PATH, and the existing `bench hook` PreToolUse write guard. Subscription only, the
benchmark-only login, the already tested per-attempt context isolation. The CLI change is one more
reason the F arm is re-run and not borrowed, and why each round's canary must pass before any cell.

- **F (full):** `--append-system-prompt-file` holds `mrw instructions`, SHA256
  `a76dbd46ae65184aae51853b372c174f845a99fac6e2ad45c3a632a9fb7830f7`, 23,187 characters.
- **C (core):** `--append-system-prompt-file` holds `mrw instructions --core`, SHA256
  `3b90f436e1519f5afc95fce16bdfd97d955611e3d361240fd4dc3b41ae2227f3`, 1,598 characters.

The F arm is re-run instead of reusing the 2026-10-30 traces, so both arms share a day and a CLI
build. They do not share a cache state by construction: each arm's prefix is warmed by its canary,
the two prefixes may share a cached beginning, and provider cache state and service load can drift
between cells. Order is counterbalanced (below); cache and time effects are not controlled and are
reported as residual confounds. Native is not re-run. A comparison with the earlier native arm is
descriptive only and is not pooled.

## Rounds and order

Each round is its own run directory and its own adapter output directory, so cell records, canaries
and launch ledgers never meet across rounds. Round 1 takes seed 5030's job order and its FC/CF order,
as the grouped pilot did. Round 2 keeps the same job order and reverses each pair's arm order, so every
job runs FC once and CF once, and each arm goes first in five of the ten pairs.

The driver opens each round with a canary pair, F then C, through the same adapter in a tree with no
`TASK.json` (identities `canary.F` and `canary.C`). Counted cells start only when both canaries
completed with passing isolation. Stop on failed isolation; no paid retries. Keep all failures,
timeouts, unknowns and unexecuted cells.

Before round 2's first launch, its canaries included, the driver checks round 1: a summary with stop
reason `completed`, both canaries and every cell with passing isolation, and reported usage,
canaries included, below $1.50. If any fails, round 2 is not run and the results say so.

## Driver change, frozen before any call

`scripts/grouped_pilot.py` ties every arm setting to the labels A and B: tools (line 22), the hook
(line 94) and the appended full guide (line 106). The adapter, `scripts/claude_evidence.py`, names an
arm B whenever an appended file is present (line 100) and accepts only A and B identities (line 103);
it already records the appended file's SHA256 (line 110), and its $3 stop is fixed (line 59). Before
any call they change in exactly these ways:

- Driver: two arm labels, F and C, both taking arm B's tools, settings, isolation check, write guard
  and pinned MRW on PATH; it appends `mrw instructions` for F and `mrw instructions --core` for C.
- Driver: it launches the canary pair itself, as described above, and runs the round-2 gate first.
- Driver: a round setting. Round 1 runs the order `paired_plan` gives now (line 27); round 2 reverses
  each pair's arm order.
- Adapter: the driver names the arm in the adapter's environment; the adapter takes the identity's
  arm from it, refuses a name other than F or C when one is given, refuses F or C without an
  appended file, and does not pass the name on to Claude. With no name given it behaves as now.
- Adapter: the $3 aggregate stop becomes a configured ceiling, $3 by default. Round 2 is given
  $3 minus round 1's reported usage.

Tests assert: each arm's appended hash is the one named above; F and C cells and canaries get
distinct identities and arm B's tools, hook and MRW; the adapter refuses an unknown arm name, and F or
C without an appended file, while an unnamed call keeps its A/B behavior; the driver never launches
without naming F or C; round 2 reverses every pair; round 2 refuses, before its first canary, on each
failing gate condition; the ceiling stops a launch at the configured amount. No observer or Go
change. The diff and the test results are published with the results.

## Budgets

Per round, as in the grouped registration: `--max-turns 10` per attempt and at most 12 launches, two
canaries and ten counted attempts, one at a time. That is within the adapter's 20-launch ceiling
(`scripts/claude_evidence.py`, `cost_guard`), which is left unchanged. Every launch passes Claude
`--max-budget-usd 0.50` (line 132); the adapter does not enforce it afterwards, and a call can
overshoot it. Round 1 stops at $3; round 2's ceiling is $3 minus round 1's usage, and the gate
requires round 1 below $1.50. The stops act before a launch, so both rounds together stay below $3
plus whatever the last launch costs; no upper bound on that call is claimed. Actual expenditure and
any overshoot are reported. The grouped pilot's twelve launches reported $0.2390969 in all, so about
$0.50 is expected for both rounds. Unknown cost or usage, or an unfinished launch, blocks later calls.

Inner CLI deadline 75 seconds, outer cell deadline 90 seconds with process-group cleanup, observer
checker deadline 5 seconds. In each round the driver starts no new pair when fewer than 180 seconds
remain in a nine-minute envelope; the grouped pilot's ten cells took 6.6–14.2 seconds each. Load below
core count, free memory above 20%, and no peer build. No containers, services, installs or downloads.
Costs are subscription token-value figures, not an invoice.

## Scoring and reporting

Each cell is graded as in the grouped registration: valid completion within budget, matching
model/tools/isolation metadata, an exact reference match, and a verified final full check, each
verdict published separately.

Report per cell, per arm and per round:

- first-request input/cache tokens
- input, output, cache-read and cache-creation tokens, reconciled to the CLI cost with the coefficients
  the diagnosis states, plus any residual
- CLI turns, unique emitted assistant messages, and tool calls by kind
- MRW write invocations and refused MRW writes
- model-issued `mrw instructions` calls (in C, each one is an on-demand full-guide load, and its size
  is counted), counted apart from model-issued `mrw instructions --core` calls, which the core text
  may prompt even where it is already appended
- CLI wall, observer wall and total cell wall
- pair order and cache state

Then the eligibility verdict and, when eligible, per arm, the two rounds' spread for each measure and
the repeat-spread verdict above. Keep failed attempts in totals and list missing cells. Unknown usage
is never turned into zero. Each prediction gets a hit or miss line per round, and prediction 3 its
two-round line.

## Out of scope

- **Native re-run:** not run. The question is within the MRW arm. A same-day three-arm run that
  settles the gap question needs a separate registration and the owner's go-ahead.
- **The workflow change** that combines the Markdown write and its check, and supplies the
  configuration's active/legacy structure up front: a separate experiment, as the diagnosis names it.
  Running it together with this one would make the two causes impossible to tell apart.
- **Held-out or real work:** these are the five synthetic jobs the earlier pilot used. A guide choice
  for real work needs frozen held-out tasks, as the earlier evidence-pilot notes recommend.
- **Safety benefit** (stale or multi-file mistakes prevented): not measured by these fixtures.
- **Guidance, defaults, engine:** no change follows from ten synthetic pairs. No significance, p95
  or general savings claim, and no commit, push or release as part of this run.

Fixtures and publishable records contain no private customer or project identifiers.

## Amendment 1, 2026-10-31, before run 2

Run 1 ended with no eligible comparison ([results](CORE-RESULTS.md)). Run 2 repeats this registration
unchanged in design, predictions, eligibility, repeat-spread rule and budgets, with these differences:

- **The write guard's refusal names what passes.** `bench hook` still refuses a redirection to a path
  it cannot resolve; its refusal now says to put a plan file at a literal absolute path outside the
  task tree, or to pipe the plan to `mrw write -`. Run 2 uses a `bench` binary built from that source,
  recorded by hash. Both arms carry the guard, so the change applies to F and C alike.
- **Fresh run directories.** Run 1's records stay where they are and are not pooled with run 2's.
- **The CLI is whatever is installed at launch,** recorded per cell; 2.1.316 when this was written.
- **A peer process is present.** Another session on this machine runs a repeated `go test` loop
  capped at two cores (`GOMAXPROCS=2 -p 1`). It did not stop within 400 seconds of waiting. Run 2
  proceeds with load below the core count and records the load at start; this departs from the "no
  peer build" condition, and wall-clock results are reported as contended.

Prediction 2's range is kept although run 1 showed uncached prefixes; a miss is reported as a miss. If
run 2 also ends without an eligible comparison, it is reported as such and no run 3 starts without
the owner's go-ahead.

## Amendment 2, 2026-10-31, before run 3

Run 2 was eligible: C cost more than F in both rounds, and C ran the full `mrw instructions` in 8 of
its 10 cells ([results](CORE-RESULTS.md)). ADR-184 (Proposed, on branch
`adr-184-the-core-names-the-hunk-header`, not accepted or merged) adds the hunk header to the core's
rule 4. Run 3 repeats run 2's design, eligibility, repeat-spread rule, budgets and guard text, with
one difference: the MRW binary is built from that branch.

- **Binary.** `mrw` built from the ADR-184 worktree, which reports `dev (2c5a2ab, dirty)`, recorded
  by hash. Its full guide is byte-identical to run 2's (SHA256 `a76dbd46…`, 23,187 characters). Its
  core is SHA256 `11a023b125e5db3314babf90059a42705a67ad4155f3c434013b0b0dab7c550b`, 1,610
  characters, 298 words. The driver takes C's registered hash from the run's configuration, since
  the constant names run 2's core.
- **Peer processes** are recorded at each round's start, as in run 2.

Predictions for run 3, written before its first call, from run 2's numbers:

1. **First-request context.** C's lands at 4,960–5,110 tokens per cell: run 2's 4,953–5,092 plus the
   twelve characters added. F's repeats run 2's 12,562–12,699.
2. **On-demand guide.** C runs the full `mrw instructions` in at most one job per round. Two or more
   in either round is a miss: the header alone did not stop the fetch.
3. **Cost.** By the registered repeat-spread rule, C is lower: cheaper than F in both rounds, by a
   mean larger than both arms' spreads, with a saving of $0.005–$0.030 per round. The range is about
   7,600 tokens × 20–24 requests × $0.1 per million at the low end and the cache-creation gap of
   run 2 removed at the high end.
4. **Quality.** Both arms solve all five jobs in both rounds.

Run 3 is not pooled with runs 1 or 2. Its result informs ADR-184's acceptance; it does not accept it.
