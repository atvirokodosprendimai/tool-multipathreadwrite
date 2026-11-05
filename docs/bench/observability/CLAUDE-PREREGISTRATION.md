# Claude evidence pilot — registration, 2026-10-30

Owner requested creating and executing benchmarks with Claude. This is a new, synthetic smoke pilot,
separate from every published stage and the pending external assessment. Registration precedes model runs.

## Frozen comparison

Reuse `cmd/bench` (ADR-156) unchanged: native A versus forced-MRW B, one model, one repeat, seed 31030.
Use explicit `claude-sonnet-5-5`, installed Claude Code 2.1.314 and installed MRW v1.79.0; record executable
SHA256 and actual init model. Both arms must resolve the same model. Subscription only, the existing
benchmark-only login, no API key, no plugins/MCP, no user instruction inheritance. Reuse the runner's
B write guard and slipped-write check. Two passing isolation canaries for this CLI precede task runs.

Two tasks are frozen under `bench/evidence-pilot/tasks/`: a six-file status migration with unchanged
legacy decoys, and a one-line timeout change. Both use cheap Python acceptance scripts, held fixed with
`.quality-harness.json` and TASK.json. They are synthetic fixtures, never added to the measured historical
corpus. Reference patches define allowed regions and exact expected edits; the runner's oracle grades them.
Before model execution, every task must fail its checker at baseline and pass after its reference patch.
Record those controls. Reverting or weakening the acceptance script is not a solution.

The wrapper records a final `mrw check --full` observation in both arms, before the runner removes the tree,
using `scripts/mrw_observe.py`. It also hashes the exact task prompt and any appended instruction file.
The final verifier uses its own environment/state; A's Claude session never gains MRW access from it.
Its time is separately reported; the runner's total wall time includes this observation, so it is not
presented as pure model time. The underlying runner retains its independent oracle verdict.

## Budgets and stopping

Two canary invocations plus four task attempts, sequential. Outer budget 90 seconds per invocation,
inner Claude budget 75 seconds, verifier timeout 5 seconds. The CLI receives `--max-budget-usd 0.50`;
six invocations target at most $3 of CLI-reported token value, not a subscription invoice. A request can
overshoot a guard; record actual totals. Unknown reported usage/cost or an exhausted aggregate budget
prevents further model launches. No paid retries of started sessions; stop on failed canary. The existing
runner's start-error handling can record/retry setup failures that never launched a model; those remain visible.
Task turn cap is 20 (runner adjudication). No services, containers, corpus downloads or installs.

Load must be below core count, free memory above 20%, no peer build. One cached benchmark-binary build,
then the serial model calls. Preserve CLI transcripts/stderr and evidence outside the public checkout.
No private project/customer names or source enter fixtures or published artifacts.

## Scoring and claims

Report every planned cell: oracle solved, observer verified/rejected/missing, CLI exit, timeout/error,
resolved model, input/output/cache tokens, turns, tool calls, CLI reported cost, total wall time and
observer wall time. Missing values stay unknown. Include startup/isolation failures, never count them as
zero-cost solved tasks. Preserve binary/collector/wrapper/task/instruction hashes and final-tree digests.

Pilot acceptance: both task oracles reject baseline and accept reference; isolation canary passes; each
completed cell yields an evidence record; independent oracle and observed check agree on final correctness.
Any disagreement, gate change or missing record is reported as a failed pilot control, not discarded.
No statistical promotion, significance, p95 or general PAYS/LOSES claim from two pairs. Show descriptive
per-task differences only. A later representative, held-out and repeated campaign needs its own numerical
quality/cost bar written before collection. This pilot must not change existing registrations or defaults.

## Setup amendment — before any task cell

The first two canaries ran and are retained as setup-1. A's model tried MRW despite its absence, and
both init records listed synced custom plugins. Scratch paths also named the tool. No task cell ran.
Before unblinding task results, exclude all user/project/local instruction files with the documented
claudeMdExcludes glob, disable auto memory/synced skills and the observed synced plugins, and use a
neutral scratch path. Preserve the B PreToolUse hook by merging these settings into the explicit
per-attempt settings; do not use safe mode, which disables hooks too. Settings semantics:
[Claude settings](https://code.claude.com/docs/en/settings-reference#claudemdexcludes).

Allow exactly one replacement isolation pair, still no task retry: eight model launches maximum
including the retained failed pair, at the same $3 aggregate reported-cost stop and $0.50 per-call
guard. The first pair consumed 27.017 seconds and $0.1014632 reported value, leaving six calls with
75-second inner deadlines within the original nine-minute execution envelope. Record the adapter's
amended hash separately; task data, model, oracle, ordering seed and claim restrictions are unchanged.
If replacement isolation fails, no task run starts. Re-freeze instrument source before that pair.

## Fixture runner amendment — before any task cell

The normal `bench run` entry point refuses synthetic `shape=fixture` tasks by design. Its startup
refusal launched no model. Use the existing `bench canary` fixture entry point once per task pair:
it runs the same A/B invocations, hook/audit and independent oracle, retaining both receipts, while
keeping synthetic tasks out of measured stages. Adapt only the runner's task ID to `canary` in private
copies; TASK.json retains the public pilot ID. Instruction, base files, reference patch and gates stay
unchanged. Score correctness from each receipt's oracle, never from the canary's isolation pass alone.
Task order uses seed 31030; within each pair the fixture runner fixes A then B. This order/cache bias
is another reason all cost/time differences are descriptive only. No additional model calls or budget.
