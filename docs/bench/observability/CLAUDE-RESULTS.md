# Claude evidence pilot — results, 2026-10-30

**Completed:** two synthetic tasks × native/MRW arms, one attempt per cell, Claude Sonnet 5.5.
All four recorded file manifests match the registered references, and all four final checks verify.
The task calls used $0.0709984 of CLI-reported token value. Including every setup call: $0.2411861.
These are subscription usage estimates, not an invoice or a general savings result.

[Registration and setup amendments](CLAUDE-PREREGISTRATION.md) precede task calls.
[Retained metrics and fingerprint records](claude-pilot.json) include failures and every completed cell.
The [task corpus](../../../bench/evidence-pilot/tasks) is synthetic and separate from historical measured stages.

## What ran

| Task | Arm | Exact tree / final check | Reported cost | Claude CLI wall | Observer wall | Turns | Tool calls |
|---|---|---|---:|---:|---:|---:|---:|
| One-line timeout | Native A | Pass / verified | $0.020176 | 10.859 s | 130 ms | 6 | 5 |
| One-line timeout | MRW B | Pass / verified | $0.014129 | 7.131 s | 118 ms | 3 | 2 |
| Six-file status migration | Native A | Pass / verified | $0.019022 | 9.750 s | 140 ms | 5 | 4 |
| Six-file status migration | MRW B | Pass / verified | $0.017672 | 5.549 s | 202 ms | 3 | 2 |

Both arms preserved the legacy decoys and every unrelated fingerprinted file. Each B task used one MRW write,
with no `--no-check` command. Every cell additionally ended in the observer's full check.
Input/output/cache token counts, instruction hashes, actual model and final file hashes are in the JSON.
Claude CLI wall covers startup and the CLI process; observer wall is recorded separately. The original
runner's outer wall would include both, so it is not labeled model inference time.

The MRW arm had lower reported cost, fewer tool calls and shorter CLI wall in these four observations.
A always ran before B, instructions/cache state warmed during setup, and there is only one attempt
per task/arm. No confidence interval, p95, PAYS/LOSES verdict or general quality-equivalence claim follows.
No default instruction change or optimization was made from this result.

## Reproducibility and isolation

- Installed Claude Code 2.1.314; requested and observed model `claude-sonnet-5-5` in every call.
  No effort override; both arms used the same CLI default.
- Installed MRW v1.79.0, executable SHA256
  `5427ceaebc8d8c3514a62eb1bb102d304acacf203d3280914c9570b9eeee0abe`.
- Existing `cmd/bench` built without source changes; binary SHA256
  `df2ebb4f4cd19a8e76f72e4d7e06fa4fe96664c7b2cb54f0ed530c29b5d6e6d8`.
- Registered corpus SHA256
  `c278e40a0c1ab57159fb22c23458a05681892669a4bc9dd51b86d4b8467cce0a`.
  Corpus includes baseline files, task specifications and reference patches. The JSON lists per-file hashes.
- Both checkers failed on their initial trees and passed their reference fixes before model execution;
  forward and reverse patch controls passed.
- A had Bash/Read/Edit/Write/Glob/Grep and no MRW on PATH. B had Bash/Glob, the pinned binary and full
  MRW instructions, with the existing PreToolUse write guard configured. The explicit hook dictionary
  was preserved while suppressing inherited settings and context.
- Final isolation init records contain no plugins and no MCP servers. A's passing canary did not name
  MRW. No account/global instruction files were edited. The observer runs outside the agent's tool
  exposure and therefore does not give A access to MRW.
- All eight calls, including the failed setup pair, reported known cost. No model timeout, task retry,
  corpus download, container or service was used. Load before execution was 2.35 on 10 logical CPUs,
  with 35% free memory and no peer build observed.

The adapter is [scripts/claude_evidence.py](../../../scripts/claude_evidence.py); local controls are
[scripts/test_claude_evidence.py](../../../scripts/test_claude_evidence.py). Source snapshots for the
exact observed adapter/collector, original and amended registrations, commands, raw transcripts and
stderr remain outside the public checkout. The source used for these task calls predates the added
post-run grader and interrupted-launch marker hardening; its hash is retained with the local source snapshot.

## Setup failures and deviations retained

1. The first isolation pair failed: A tried MRW without having the binary, both arms loaded synced
   plugins, and the scratch path named the tool. That pair cost $0.1014632 and is retained in
   `setup_records`. No task call started from that setup.
2. Before any task calls, an amendment excluded inherited instruction files and synced context,
   preserved B's explicit hook, and moved scratch to a neutral path. This uses the documented
   [instruction-exclusion settings](https://code.claude.com/docs/en/settings-reference#claudemdexcludes).
   The replacement isolation pair passed and cost $0.0687245. Aggregate cap remained $3.
3. Normal `bench run` refuses fixtures, correctly. Its startup refusal launched no model. A second
   pre-task amendment used the existing fixture entry point, `bench canary`, once per task pair.
   Private task descriptors use runner ID `canary`; TASK.json and exported records keep the pilot ID.
   This entry point fixes A then B, replacing the originally intended randomized arm order.
4. The fixture entry point computes a Go oracle result but discards its per-attempt receipt. We do
   not claim to have retained that oracle verdict, hook counters, slipped-write audit or outer wall.
   After collection, the exact-tree grader applies the already registered reference patch to a
   disposable baseline, hashes every expected file and mode, and compares that map with the observed
   final map. The **criterion was frozen before task execution**; its report implementation was added
   afterward to recover the missing receipt. All four maps match, with no extra files in the recorded scope.
   A test accepts the reference and rejects wrong or extra files. This is separately labeled scoring,
   not the unretained Go receipt and not independent re-execution of the original task tree.
   Ignored untracked files, such as bytecode caches, are outside both the observer and this recovered grade.

## Evaluation

The pilot demonstrates an executable path from Claude task output to pinned final-check evidence and
reference-backed correctness scoring. It also caught a contaminated comparison before collecting task
results. The measured observer stage was 118–202 ms on these tiny fixtures, including a full cheap check.
That is a four-observation range, not a scaling estimate or a claim about real test suites.

To repeat, use fresh artifact and neutral scratch directories, the benchmark-only subscription profile,
and the adapter as `bench`'s Claude executable. Its config contains absolute `claude`, `mrw`, `out`
paths and `disabled_plugins` IDs. Keep native and MRW tool lists unchanged; create private `canary`
aliases of these fixture descriptors; run a fresh isolation pair before the two task pairs. Retain the
raw stream, all setup failures and both score types. Never reuse an output cell or count synthetic
fixtures as historical measured tasks.

A representative cost/quality campaign still needs held-out real tasks, repeated and counterbalanced
arms, a fresh corpus, frozen numerical promotion criteria and preserved per-attempt grading receipts.
The current observer's 120-second checker cap also needs a bounded extension for longer real suites.

## Local validation

Ten adapter controls and 21 observer controls pass locally. Ruff F/E9, workflow structure validation
(actionlint 1.7.12 with ShellCheck disabled), the 513-clause prose check and diff whitespace checks pass.
A fresh bounded review approved the pilot and a focused follow-up approved the unknown-usage stop guard.
The original broad review timed out before a verdict and is not counted as approval. No remote CI run,
full Go regression campaign, commit, push or release was performed for this pilot.
