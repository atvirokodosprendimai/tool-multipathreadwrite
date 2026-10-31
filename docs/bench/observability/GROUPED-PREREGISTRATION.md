# Grouped-edit Sonnet pilot — registration, 2026-10-30

Owner requested a bigger native Sonnet versus MRW pilot with grouped edits, clarifying approximately
50 turns per arm and the hypothesis that 50 native edits might fit in roughly five MRW writes.
Edit calls, Claude turns and edited sites are different units. Fix the work and measure the reduction;
never force native into one Edit call per site or pad the faster arm with extra work.

## Fixed workload and comparison

Five synthetic jobs, ten registered edit sites each: 50 sites per arm, one attempt per job/arm.
The families are routing, JSON configuration, Markdown documentation, coupled source/test updates,
and multi-line structural changes. Each has legacy decoys and a fixed acceptance script.
Reference patches, prompts, initial files and the corpus manifest are frozen before any model call.
Each baseline checker must fail, its reference must pass, and a wrong decoy must fail.

Use Claude Code 2.1.314, requested/observed `claude-sonnet-5-5`, installed MRW v1.79.0 with its
recorded binary hash. Subscription only, the benchmark-only login. A offers Bash/Read/Edit/Write/
Glob/Grep and no MRW on PATH. B offers Bash/Glob, pinned MRW/full instructions, and the existing
`bench hook` PreToolUse write guard. Both use the already tested per-attempt context isolation.
A fresh clean isolation pair precedes counted jobs. Stop on failed isolation; no paid retries.

Use a direct serial fixture driver because the Go fixture entry point fixes A-before-B and discards
its grading receipts. The new driver preserves exact-tree and check verdicts while the tree exists.
It reuses the existing adapter/observer and compiled write guard without modifying Go code.
Seed 5030 shuffles the five jobs; alternate AB/BA within pairs. Five pairs necessarily give a 3/2
order imbalance; publish it. Keep all failures, timeouts, unknowns and unexecuted cells.

## Budgets

A turn is the CLI result's `num_turns`, including its final response, not a tool call or edit site.
Pass `--max-turns 10` for each of five attempts per arm: a maximum 50-turn task budget per arm.
Actual turns may be lower and are an outcome. No extra jobs to equalize observed turn counts.
The adapter may make at most 12 launches: two canaries and ten counted attempts, one at a time.
Its existing $3 aggregate reported-usage stop and $0.50 per-call guard remain. Report any last-call
overshoot honestly; these are subscription token-value figures, not an invoice. Unknown cost/usage
or an unfinished launch blocks later calls. Inner CLI deadline 75 seconds, outer cell deadline
90 seconds with process-group cleanup, observer checker deadline 5 seconds. Driver stops new pairs
when fewer than 180 seconds remain in a nine-minute task-run envelope. Expected model time 2–5 minutes.
No containers, services, installs or corpus downloads. Load below core count, free memory above 20%,
no peer build. Preserve exact observed source snapshots, tool/binary versions and hashes.

## Scoring and reporting

Before teardown, grade each recorded final file manifest against a disposable tree with the frozen
reference patch applied. Scope: baseline/current tracked plus nonignored untracked paths. Ignored
files, external dependencies and transient restored changes are outside this unsigned observation.
A solved cell requires valid CLI completion within budget, matching model/tools/isolation metadata,
an exact reference match, and a verified final full check. Publish those component verdicts separately.
Persist baseline manifests, prompt/settings/instruction hashes and both grading receipts immediately.

Report actual edited sites/files, applied-result correctness, Claude turns, tool calls by kind,
Bash calls invoking MRW writes, input/output/cache tokens, CLI-reported cost, Claude CLI wall,
observer wall and total cell wall. Shell-command mentions are not a count of engine invocations.
Report pair order and cache state. Compare matched completed pairs, keep failed attempts in totals,
and list missing cells explicitly. Do not turn unknown usage into zero.

This is a larger mechanistic pilot, not held-out real work or a profile qualification. No significance,
p95, PAYS/LOSES, 10x-turn or general task-quality claim from five synthetic pairs. Do not pool it into
earlier registrations or change MRW guidance/defaults based on it. No commit, push or release is part
of this request. Fixtures and publishable records contain no private customer/project identifiers.
