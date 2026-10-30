# SWE-bench Verified results and limits

[Back to the quick start](../../README.md#quick-start). These are recorded results, not a claim about the current release.
This overview covers MRW (B) against the baseline (A). [RESULTS.md](RESULTS.md) includes all registered configurations, including P.

191 bug-fixing tasks per model, each run once as A and once as B and paired by task (190 pairs counted on Opus, where one attempt was excluded by the registered rule), graded by the official SWE-bench harness, against a bar registered before the first run ([ADR-162](../adr/ADR-162-stage-5-runs-until-the-verdict-is-conclusive.md)). A is Claude Code alone; B is Claude Code with mrw.

| | Opus 5.5 | Sonnet 5.5 |
|---|---|---|
| **cost per task** | **−19.4%** [−25.5%, −12.7%] | **−7.0%** [−12.1%, −1.8%] |
| turns per task | 14.6 → 11.5 (−21%) | 7.8 → 6.8 (−13%) |
| wall time per task (descriptive) | 115 s → 68 s (−40%) | 36 s → 25 s (−30%) |
| output tokens per task | 5,489 → 3,888 (−29%) | 2,274 → 1,721 (−24%) |
| tasks solved | 167 → 166 of 191 | 134 → 129 of 191 |

The cost intervals are 99% and lie wholly below zero. Both overall verdicts are **INCONCLUSIVE**:
the registered bar asks for a solved-rate lower bound no worse than −5 points, and the intervals
reach −6.7 on Opus and −8.7 on Sonnet. Sonnet's −7.0% is also short of the 10% mean saving PAYS
asks for. These results do not establish unchanged solving. Wall time was measured on a shared
machine and is descriptive only.

## Every SWE-bench stage, per attempt

Means per attempt at its final try. Tokens count input, output and cache tokens together. Every figure comes from the receipts under [docs/bench/](./), and every registered verdict is in [docs/bench/RESULTS.md](RESULTS.md).

| stage | model | config | attempts | solved | turns | wall s | $ per attempt | $ total | tokens (k) | output tokens |
|---|---|---|---|---|---|---|---|---|---|---|
| 5 | Opus 5.5 | A | 191 | 167 | 14.6 | 115 | 0.260 | 49.73 | 223 | 5,489 |
| 5 | Opus 5.5 | B | 191 | 166 | 11.5 | 68 | 0.200 | 38.20 | 182 | 3,888 |
| 5 | Sonnet 5.5 | A | 191 | 134 | 7.8 | 36 | 0.059 | 11.23 | 82 | 2,274 |
| 5 | Sonnet 5.5 | B | 191 | 129 | 6.8 | 25 | 0.053 | 10.13 | 91 | 1,721 |
| 4b | Opus 5.5 | A | 20 | 17 | 17.1 | 170 | 0.362 | 7.23 | 306 | 7,875 |
| 4b | Opus 5.5 | B | 20 | 16 | 12.4 | 75 | 0.277 | 5.54 | 220 | 5,834 |
| 4b | Sonnet 5.5 | A | 20 | 17 | 11.8 | 36 | 0.092 | 1.84 | 127 | 3,788 |
| 4b | Sonnet 5.5 | B | 20 | 14 | 8.0 | 25 | 0.072 | 1.43 | 107 | 2,616 |
| 3 | Opus 5.5 | A | 20 | 19 | 12.0 | 60 | 0.197 | 3.94 | 155 | 4,210 |
| 3 | Opus 5.5 | B | 20 | 19 | 10.4 | 44 | 0.170 | 3.40 | 137 | 3,452 |
| 3 | Sonnet 5.5 | A | 20 | 16 | 6.6 | 19 | 0.045 | 0.91 | 54 | 1,640 |
| 3 | Sonnet 5.5 | B | 20 | 16 | 5.4 | 18 | 0.039 | 0.78 | 55 | 1,290 |

## What the numbers say

- **Fewer reported turns.** The table records fewer agent turns in every SWE-bench stage on both models. These are the CLI's reported turns, not tool-call or API-request counts; the receipts do not establish which calls were batched. The lower wall times are descriptive observations from a shared machine.
- **Lower measured cost.** The Stage 5 estimate is −19.4% on Opus and −7.0% on Sonnet. Stage 4b, 20 multi-hunk tasks, estimated −20.9% [−32.3%, −8.6%] on Opus and −17.0% [−26.6%, −7.6%] on Sonnet. Those cost estimates do not by themselves satisfy the overall quality-and-cost bar.
- **Fewer tokens written, not always fewer read.** Output tokens fell 29% on Opus and 24% on Sonnet. Total tokens fell 18% on Opus but rose 11% on Sonnet, and that rise is all cache reads (72k to 82k per attempt) while Sonnet's cache writes and output fell. Cost still fell on Sonnet, because cache reads are the cheapest tokens and the output that fell is priced far above them.
- **Solving remains uncertain.** The observed counts are 166 against 167 on Opus and 129 against 134 on Sonnet. Neither interval rules out a loss beyond the registered five-point margin.

These measurements concern the tested bug-fixing tasks and versions. They do not establish a
speed or cost benefit for other repositories, feature work, reviews or long sessions.

## How mrw is judged

Every stage is paired, pre-registered and graded against the bar in [ADR-156](../adr/ADR-156-whether-mrw-pays-for-itself-is-measured-on-agent-runs.md): PAYS needs the mean per-task cost down by 10% or more with its interval wholly below zero, **and** a solved-rate lower bound no worse than −5 points; LOSES is a cost interval wholly above zero or a solved-rate interval wholly below it; anything else is INCONCLUSIVE. Each verdict is recorded in [docs/bench/RESULTS.md](RESULTS.md) exactly as `bench analyze` printed it. Stage 5's two B-against-A verdicts, both open when the stage ended at look 5 of 5: Sonnet 5.5 INCONCLUSIVE, cost −7.0% [−12.1%, −1.8%]; Opus 5.5 INCONCLUSIVE, cost −19.4% [−25.5%, −12.7%].

Before SWE-bench, four stages ran mrw on this project's own tasks (Stages 1, 1b, 2 and 4b, 24 to 40 tasks each, Sonnet and Opus): all INCONCLUSIVE, cost within ±3%. One stage went against mrw: Stage 4, 24 tasks chosen where mrw's mechanism should matter, on mrw v1.66.0, LOSES at cost +17.1% [+9.6%, +24.7%]. [ADR-161](../adr/ADR-161-instructions-lead-with-recipes.md) changed what mrw teaches in reply, and the rerun on v1.68.0 as Stage 4b came out INCONCLUSIVE on both models.

## Evidence, for anyone who wants to check

- **Receipts.** One JSON per attempt under [docs/bench/stage-5/](stage-5/)`block-1` … `block-5`, named `<instance>.<config>.<model>.<repeat>[.rN].receipt.json`; a `.rN` file is a retry of a run that did not finish and supersedes its base. Each carries the model, configuration, `num_turns`, `wall_ms`, `total_cost_usd`, `usage` by token kind, the harness verdict (`oracle.harness_resolved`), the size of the diff the agent produced (files, insertions, deletions) and the number of tool calls. The patch text was not kept: the attempt's worktree was discarded after grading.
- **Grader.** The official SWE-bench harness (swebench 5.0.2) on `SWE-bench/SWE-bench_Verified`, split `test`, dataset revision `78f471bf`. The 191 instance ids and the rule that picked them are in [bench/swe-bench/selection-adr-162.json](../../bench/swe-bench/selection-adr-162.json).
- **Pre-registration.** The bar, the pool rule, the five looks and the stop rule were written in [docs/adr/BACKLOG.md](../adr/BACKLOG.md) ("Pre-registered for ADR-162") and in ADR-162, merged as [1fc00bad](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/commit/1fc00bad) on 2026-10-17 at 08:42 UTC; the instance list was committed as [8b435f8f](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/commit/8b435f8f) at 10:55 UTC; the first counted attempt started at 13:21 UTC the same day (`started` in its receipt).
- **Run setup.** `run.json` in each block: Claude Code 2.1.302, mrw v1.68.0 (975c3ba), a 1,200 s budget per attempt, one repeat, the same prompt and tool set for A and B. `canary.json` per block proves each configuration's setup before the block; `exclusions.json` names the one excluded attempt.
- **Verdicts.** [docs/bench/stage-5/looks/look-5.txt](stage-5/looks/look-5.txt), as `bench analyze` printed them, copied unchanged into [RESULTS.md](RESULTS.md); ADR-162's acceptance fence diffs the two.
- **Independent scoring.** The paired rows were handed to the quality-harness project's own scorer, which reads them against its own pre-registration (its ADR-106).

Reproduce the Stage 5 rows of the table with jq alone, from a checkout:

```sh
jq -s 'map(select(.stage=="stage5" and .config!="P"))
  | group_by(.attempt|sub("\\.r[0-9]+$";"")) | map(max_by(.attempt))
  | group_by([.model,.config]) | map({model:.[0].model, config:.[0].config, attempts:length,
      solved:(map(select(.oracle.solved))|length), turns:((map(.num_turns)|add)/length),
      wall_s:((map(.wall_ms)|add)/length/1000), cost_total:(map(.total_cost_usd)|add)})' \
  docs/bench/stage-5/block-*/*.receipt.json
```

`go run ./cmd/bench analyze --selection bench/swe-bench/selection-adr-160.json --selection bench/swe-bench/selection-adr-162.json docs/bench` recomputes every verdict and interval in RESULTS.md.

What this does not show:

- One agent, Claude Code 2.1.302, and two models. Nothing here says what mrw does for another agent or another model.
- One kind of work: a one-shot bug fix from an issue text, with no human in the loop and a 1,200 s budget. Long sessions, refactors and greenfield work were not measured on SWE-bench.
- The agent could not run the repository's tests in Stage 5: the checkouts carry the code but not the test dependencies, so every solve was graded afterwards by the harness, and neither A nor B could check its own work. [ADR-163](../adr/ADR-163-the-plugin-is-measured-in-checkouts-that-run-their-tests.md) is the next stage, in containers where the tests run.
- Solving is not proven equal. 191 tasks per model put the solved-rate interval at about ±7 points; the registered bar wanted the lower bound at −5 or better, and it was not met.
- Wall time is descriptive. The attempts ran on one shared Apple Silicon machine, with the SWE-bench images under amd64 emulation, beside other work.
- Cost is what Claude Code reported per attempt (`total_cost_usd`), at the prices in force in October 2026.

Scripted interface-call and byte measurements are in [docs/measure.md](../measure.md), the six-guarantee comparison in [docs/comparison.md](../comparison.md), and model × score readings in [docs/model-benches.md](../model-benches.md).
