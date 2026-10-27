# Proposal: batching guidance versus mrw with runnable checks

**Status:** Draft protocol, 2026-10-26; no attempts run and no budget approved.
This is not a new benchmark stage or an amendment to existing pre-registrations.
Agree the execution budget and freeze the run inputs before running. Runner changes
follow [CONTRIBUTING.md](../../CONTRIBUTING.md#what-a-change-needs): an ADR is needed
only for a durable contract, exit code, persistent state or trust-boundary change.
Existing benchmark results are unchanged.

## Question and arms

Can the harness compare batching guidance and offering mrw while every arm can
run project tests, and retain enough evidence to design a larger study?

The letters here name proposed arms, not current `bench run --configs` values.
In the existing runner B already means mrw and P means the plugin; do not pass
this table's labels to it.

| Arm | Tools | Additional instruction |
|---|---|---|
| A — baseline | The ordinary tool set used by the existing benchmark, with no mrw or quality-harness plugin. | None. Natural batching remains allowed. |
| B — batching cue | Exactly A's tools. | The cue below. |
| C — mrw plus cue | B's tools, plus one frozen mrw binary and its generated instructions. | The same cue; teach mrw using that binary's instructions. |

Cue, verbatim for B and C:

> Before editing, collect the sites you need. Batch independent reads and edits where the tools permit it. Run the declared project check before claiming completion.

B-A and C-B are observed contrasts in this setup pilot. Nine attempts in a fixed
order cannot establish causal effects: run stochasticity, position and service-time
drift remain possible explanations. C-B compares the whole mrw treatment, including
its instructions and automatic checks; it does not isolate individual guards.
No arm is forced to use one shell call per edit. Causal effect estimates require
the separately registered randomized, repeated study proposed below.

## Small pilot and fixed order

One model, three tasks, three arms, one attempt per cell: **nine attempts**.
These tasks already have pinned base commits and existing oracles:

| Order | Task | Shape | Base commit |
|---|---|---|---|
| 1 | [cac-go-heartbeat](../../bench/tasks/cac-go-heartbeat/task.json) | one-line change | crossagentschat `ae4f7a9ca6c009863466dd92818f78bc9bcc8225` |
| 2 | [am-coupled-doctor-stale](../../bench/tasks/am-coupled-doctor-stale/task.json) | implementation plus an asserting test | agentsmemory `d038def25a7fadd2586674b954ea07896a330b4a` |
| 3 | [am-wide-config-from-cmd](../../bench/tasks/am-wide-config-from-cmd/task.json) | rename across files, including tests | agentsmemory `d038def25a7fadd2586674b954ea07896a330b4a` |

Run the arms in A/B/C order on task 1, B/C/A on task 2, C/A/B on task 3.
Use a fresh temporary checkout, fresh agent context and fresh state for every
attempt. Never edit a source repository named in `bench/repos.json`. This tiny,
deliberately varied sample is a setup pilot, not a population estimate.

## Preconditions before the first attempt

- Use `claude-sonnet-5-5`, as in the existing benchmark. Freeze the resolved model
  ID, agent version, mrw version and digest, prompts, tool permissions, task
  manifests, reference patches and runner revision in the run manifest. If the
  model is unavailable, amend this proposal before running.
- Materialize the pinned tasks using the existing benchmark machinery. Keep the
  task oracle and reference patch outside each agent's accessible tree.
- Establish one declared `go test ./...` check per task and run it successfully
  on the clean base. Resolve dependencies before attempts; make the same check
  and dependency environment available to all arms. If a base check cannot pass,
  stop the pilot and amend the task list before collecting any agent result.
- Give all arms the same task instruction and neutral common instructions.
  Verify that inherited user skills, MCP registrations, repository instructions,
  hooks and environment do not teach A or B to use mrw or introduce the plugin.
  Preserve project build instructions equally. Record any sanitization.
- Add a canary proving each arm's tools, instructions and declared check actually
  work. A C canary must run the frozen binary. An A/B canary must prove mrw is
  inaccessible. Stop on a failed canary.
- The existing runner does not directly provide the cue-only arm or this complete
  protocol. Implement and test those setup differences before running; do not
  relabel old receipts as this experiment.

## Limits and artifacts

Run one attempt at a time. Cap each at **180 seconds and 30 model turns**, with a
per-attempt dollar cap of **$0.50** enforced by the agent or runner before launch.
The nine-attempt model budget is therefore at most **$4.50**, and model-attempt
time at most **27 minutes**. Baseline checks, setup, canaries and grading cost
additional time and must be priced before execution. No containers are planned.

If an enforceable cost cap or required timing/usage accounting is unavailable,
do not start. Do not retry or replace an attempt after seeing its result; an
unfinished attempt remains unfinished. A setup failure discovered after launch
stops the pilot; retain the affected artifacts and mark the run invalid for
comparison. Do not extend this pilot into a larger study automatically.

For each attempt retain the base identity, prompt, arm, complete tool transcript,
usage and cost receipt, actual model-turn count, tool-call count, final binary
diff and untracked files, final declared-check output/exit, and the existing task
oracle's verdict. Capture the final patch and logs before removing its checkout.
Record checks invoked by the agent separately from grading performed afterwards.
Do not infer batched calls or guarded refusals from a lower turn count: inspect
the transcript and record unclassifiable calls as unknown.

## Outcomes and decision rule

Report each cell and the paired B-A and C-B differences in cost, model turns and
tool calls, separately by task. Also report completion, task-oracle correctness,
final check status, edit retries, and whether the agent claimed completion.
Review patches without arm labels against the task manifests: record unexpected
changes outside the requested scope and any demonstrated wrong-target edit,
with its diff as evidence. A failed test alone is not a wrong-target label.

This pilot passes its setup gate only if all nine cells have attributable
artifacts and accounting, every arm could run its declared check, and no
instruction/tool leak or oracle exposure occurred. Task failures are retained
as outcomes and do not invalidate the setup. There is no PAYS/LOSES verdict,
confidence claim or non-inferiority claim from three tasks.

If setup fails, repair the harness and record an amended setup and budget before
collecting replacement data. If setup passes, use observed cost and variation to
propose a separately registered study and budget. A larger study needs more tasks,
multiple repeats and randomized arm order, correctness as well as cost, and its
analysis and stopping rule fixed before any counted run. Other agents, long sessions,
human intervention, concurrent writers, and the causal effect of individual
safeguards remain outside this pilot.
