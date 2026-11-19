# ADR-206: Stage 9 is run as three separate bench runs per model, with two builds of one tag that differ in one refusal

**Status:** Proposed
**Drafted for:** Zy, 2026-11-19. The pilot was pre-registered in `docs/adr/BACKLOG.md` the same day, from a design review of v1.92.0. Asked "shall I start on the Stage 9 stage record and runner changes?", Zy answered *"staert"*. That starts this draft; nothing is built, run or spent until Zy accepts the record, with the spend stop in Decision 8.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-156, ADR-160, ADR-192, ADR-193, ADR-200, ADR-203, ADR-205
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/stage-9/**`, `bench/tasks/**`, `scripts/bench_stage9.py`, `scripts/test_bench_stage9.py`, `docs/bench/**`
**Enforced-by:** None — T1 adds `internal/bench/stage9_test.go::TestARunPinsItsMrwItsInstructionsAndItsSpend`; this line names it in the pull request that merges T1
**Served-path change:** none to `mrw`. `bench run` and `bench canary` gain flags; a run without them behaves as before. The backlog's "no new flag while the pilot is open" is about mrw's surface; these flags are the bench's.

## Context

**Challenged before acceptance**, 2026-11-19, on `1e6d2b53`: Codex gpt-6.2-sol at xhigh (AMEND, twelve findings) and the in-repo challenger (AMEND, eighteen), each checked against the code. The first draft attributed a landed edit to "the wrong line" from one logged line of text, which duplicates, insertions and ranges defeat; gave three spend figures for six runs, so its $40 stop summed to $80; used the release asset for one arm and a patched local build for the other, whose `mrw version` strings cannot match; and had fences that passed without the runs they named. This text replaces it.

**What was registered.** The backlog entry "cost per correctly completed edit task (Stage 9, a descriptive pilot)" asks what each way of editing costs per task done right, and whether a number-only edit goes wrong when nothing refuses it. It decides nothing about ADR-203, the taught form or the standing instruction.

**What the bench does today**, read from the code at `755ab96f`:

- *The mrw of configurations B, H and S is one live path per run.* `RunOptions.Mrw` is linked onto PATH per attempt and `mrw version` is read once (`internal/bench/runner.go`). No hash of it is taken. The plugin's side arm G is different: `verifyAsset` and `stageQH` (`sidearms.go`) verify a binary against a registered SHA-256, hold its bytes and copy them into each attempt.
- *Instructions come from the arm's own binary*, by running `mrw instructions` per attempt; the text is neither stored nor hashed. `prompt_sha256` hashes the prompt template, and only when `--status-line` is given.
- *No stop on spend.* The stops at an attempt boundary are the disk floor and, in a block, three start errors. Stage 8's $15 stop was a person reading receipts.
- *The scratch directory is deleted after an attempt.* The bench keeps the transcript, stderr, hook counts and what `keepHookFiles` copies; nothing a mrw binary wrote in its state directory survives.
- *`--task` gives a run one task list for all its configurations*, and `analyze` and `look` do not know that two directories of configuration B are two arms.
- *Resume and `bench verify`*: `claimOut` requires `run.json` byte for byte; receipts read back on resume are not validated; `readsBack` checks identity and side-arm metadata.
- *The canary* (`canary.go`) runs A and B for one model on the supplied mrw and records neither a binary nor an instruction hash; a run's gate checks its pass and Claude Code version.
- *Grading.* The oracle's verdict is solved when `Missing`, `Outside` and `Structure` are empty; `settle` then clears it on a time or turn limit. `Outside` lists Git-visible changes outside the regions the reference allows: hunks, untouched files, created files, deletions, mode changes. Ignored untracked files are not seen. `exact` fails as a setup exclusion when the reference creates a file.
- *One prompt per attempt*; `num_turns` and `tool_calls` are recorded, model requests are not. A receipt's cost can be null without a timeout (a setup failure, a result with no cost) and present with one.

**Audit of the class**: *the ways two arms can differ without the receipts saying so.* Found by reading every field of the `plan` struct and every `env` the runner builds (`mrw read --grep 'env = append\(|env := \[\]string' --count internal/bench/runner.go internal/bench/sidearms.go`: 9 sites at `755ab96f`).

| Member | Disposition |
|--------|-------------|
| the mrw binary's bytes | pinned and recorded: Decision 2 |
| the appended instructions | one frozen file: Decision 3 |
| the bench binary (it is the hook and the grader) | its revision in `run.json`: Decision 2 |
| the Claude Code release, seed, budget, turn limit, prompt and status-line hashes | already in `run.json`; the script requires them equal across the six runs: Decision 7 |
| the model snapshot behind an alias | the init line's model is recorded in the receipt: Decision 2 |
| the task list | a committed class list: Decision 8 |
| run order and the time between runs | registered order, named as a limit: Decision 8 |
| host tools linked onto PATH (git, go, node, python3), the shared `CLAUDE_CONFIG_DIR` | not recorded: named as a limit in the backlog entry |

## Existing Primitives Audit

- `verifyAsset`, `stageQH` and the receipt's side-arm hash: the pattern Decision 2 follows for mrw.
- `keepHookFiles`: the pattern Decision 5 follows.
- `--status-line`, `--task`, `bench verify`, `export.go`'s reading of build info.
- `scripts/bench_edits.py` and `scripts/bench_mechanism.py` (ADR-193, ADR-192): the frozen-script model, and the transcript readers.
- ADR-203's refusal in `internal/apply` (`staleMarks`, `failStale`) and `writer.Land`'s call to `authoring.RecordNumbering`, which since ADR-203 passes 0 for what would be refused.

## Decision

1. **Each arm is its own `bench run`, in its own directory, under a configuration that exists.** Arm **A** is configuration A. Arms **M** and **N** are both configuration B with different `--mrw` binaries. No configuration letter and no profile is added (the S and stage8 pair touches 14 non-test lines in three files: `mrw read --grep '"S"|ProfileStage8' --count internal/bench`). The content-first arm **X** runs later and is not built by this record.
2. **A run pins what it ran.** `bench run --mrw-sha256 HEX` refuses to start unless the file at `--mrw` has that SHA-256; it holds the bytes, copies them to the attempt's `bin/mrw` (the path PATH and the H hook use), reads the version from those bytes, and records the hash in `run.json` and each receipt. Every run records the bench's own revision in `run.json` and the model the init line names in each receipt. A resume validates each receipt it reads back, and `bench verify` checks the hash in every receipt against `run.json`. `bench canary` takes the same flags and records the same hashes, and a run refuses a canary whose hashes are not its own.
3. **A run can be given its instructions.** `--mrw-instructions FILE` appends that file's bytes instead of running the binary's `instructions`, and records the file's SHA-256 in `run.json`. M and N are given one file, the output of M's binary, committed under `bench/stage-9/`. Equal hashes say the two arms were handed the same bytes; they say nothing of what a model did with them.
4. **M and N are two builds of one tag by one script; N has one patch.** `bench/stage-9/build.sh TAG OUTDIR` checks the tag out in a temporary directory and builds `mrw-m` from it, then applies `bench/stage-9/no-refusal.patch` and builds `mrw-n`, both with the release's flags, `-buildvcs=false` and `-X main.version=TAG`, on the toolchain `go.mod` names, and prints both SHA-256. Neither arm runs the release asset, whose version string carries a revision a patched tree cannot reproduce. The patch does two things and lives only under `bench/`:
   - in `internal/apply`, a hunk ADR-203 would refuse is applied, and reported in the result as an event: path, op, the resolved start and end, the first unshown line, the file's SHA-256 before the write;
   - in `internal/writer`, after a write that landed and was not a dry run, each event is appended as one JSON object per line to `mrw/stage9-exposed.jsonl` under the state home, created empty by the first landed write of any kind.
   What turning the refusal off also changes is part of the intervention and is said, not hidden: the plan lands with its siblings where M's is refused whole, so receipts, checks, the ledger's record of the written file and `mrw stats` differ from there on. A test holds what N is (T2): from identical states it lands the hunk M refuses and logs it; a dry run, a plan refused for another reason and an `anchor=` hunk log nothing; it still refuses a line no read has served; with the log's file unwritable the write lands and prints what it would have; and `version`, `instructions`, `instructions --core` and `skill` print byte for byte what M prints.
5. **A run can keep a file mrw wrote.** `--keep-state-file PATH` copies `PATH`, relative to the attempt's state home, beside the receipt before the scratch directory is deleted, and records its SHA-256 in the receipt; the key is absent when the file is. `run.json` records the setting, and `bench verify` checks each kept file against its receipt. Arm N keeps `mrw/stage9-exposed.jsonl`: absent means no write landed in the attempt, empty means writes landed and none was exposed.
6. **A run stops on known spend.** `--max-spend USD` is part of the run's plan and cannot be raised for that directory. Before each attempt the run sums `total_cost_usd` over every receipt in its directory, replacements and excluded attempts included, and ends with its own exit status when the sum has reached the figure, or when three receipts carry no cost. It is a stop on what was recorded: it can pass the figure by one attempt's cost, and an interrupted attempt that left only `.started` is not in the sum. A stopped run is not resumed; going on is Zy's, in a new directory, on an amendment to Decision 8.
7. **Counting is a frozen script, `scripts/bench_stage9.py`**, committed with its hash before the first counted attempt. It reads six run directories, their kept logs and the transcripts, keyed by model and arm; it refuses directories whose `run.json` differ in anything but model, configuration, task list, mrw hash, spend figure and kept-file setting, whose mrw hash is not the arm's in `pilot.md`, or whose M and N hashes are equal. Per attempt:
   - `correct`: the receipt is not an exclusion, not an error, within budget, and its oracle is solved, on a task that is `exact: true`, creates no file and whose `require` the reference satisfies (T3 holds all three).
   - `outside`: not `correct`, and `Outside` is not empty. `other`: not `correct`, and `Outside` is empty. An attempt with no receipt, or excluded, is `missing` and is in no rate's denominator; it is counted beside them.
   - `claimed`: `status` is `done`; every run passes `--status-line`.
   - `exposed`, per attempt, one unit in both arms: the attempt has at least one exposed hunk. In N that is a line in the kept log (a number-only hunk on a line not shown since mrw wrote the file, that landed). In M it is a tool result matching `line \d+ of \S+ has no recorded showing in this version` in the output of a command that ran `mrw write` (so the text of `mrw instructions` does not count); a result cut by the host or redirected away is not seen, and the report says how many write results it could read. The hunk counts are printed beside the attempt counts.
   - spend, `num_turns`, `tool_calls`, tokens, wall time. A null cost is missing, never zero, whether or not the attempt timed out.
   - comparisons as the backlog entry states them. Location is not attributed: the script does not say an exposed hunk hit a wrong line. It lists each N attempt that was `exposed` and ended `outside`, and T6 quotes those diffs for a person to read.
8. **The stage.** Models `claude-opus-5-5` and `claude-sonnet-5-5`, one repeat, seed 206, `--status-line`, the default budget and turn limit. Claude Code: the release T5's canary passes on, with the unprofiled invocation these configurations use, the same for all six runs. mrw: the newest tag at T2, built twice by Decision 4. Arms A and M run all 48 tasks and arm N the 24 follow-on and control tasks: 120 planned attempts per model, 240 in all, before replacements. Order: Opus A, M, N, then Sonnet N, M, A; the arms of one model are hours apart, which is a limit the entry names. **Cost.** Corpus tasks cost $0.16 to $0.25 an attempt on Opus and $0.07 to $0.11 on Sonnet in earlier stages (`docs/bench/stage-2-opus`, `stage-4b-opus`, `stage-1b-sonnet`, `stage-4b-sonnet`, October 2026), which puts the stage at $29 to $43. **The stop is $50**, as six figures that sum to it: Opus A $14, M $14, N $7; Sonnet A $6, M $6, N $3. The canaries and the smoke are outside it and are held to $5. A class is a committed list, `bench/stage-9/classes.json`.
9. **Before a counted attempt**, committed and pushed: the tasks and `classes.json`; the instructions file; `pilot.md` naming the tag, both binary hashes, the instruction, class-list, script and helper hashes, the Claude Code release, the bench revision, the seed and the six figures; a canary per model for each of the two binaries; and an uncounted smoke (T5) showing the hashes in `run.json` and receipts, a kept log from a scripted exposed write in N, and a run stopped by `--max-spend`. Nothing in Decisions 1 to 8 changes after the first counted attempt.
10. **What would make this record wrong.** N differing from M in something an agent sees before any refusal would have fired: T2's test stands against that. Agents solving every follow-on task in one plan, or avoiding bare numbers because both arms' instructions tell them to: then nothing is exposed, and the backlog entry says what that result means and does not mean.

## Alternatives Considered

- **A new configuration letter and profile, so the arms shuffle together in one run.** Rejected for a pilot: the registry, the gates, the canary, `look` and `analyze` all key on the letters. The cost is that arms are separated in time, which Decision 8 names.
- **A build tag or environment switch in shipped source that turns the refusal off.** Rejected: a released binary would carry a way to disable a guard.
- **The release asset as M.** Rejected: its `version` string cannot be matched by a patched build, and the agent can read it.
- **mrw v1.91.0 as the no-refusal arm.** Rejected: it differs from v1.92.0 in more than the refusal and cannot say which hunks it would have refused.
- **Attributing a landed edit to a wrong line** from the log. Rejected after the challenge: one line's text cannot tell a duplicate, an insertion or a range apart, and the first edit an agent makes need not be the reference's.

## Component / Boundary Impact

`internal/bench` and `cmd/bench` (flags, `run.json` and receipt keys, resume and verify checks, the canary). `bench/stage-9/` and `bench/tasks/`. `scripts/`. No package of `mrw` changes in this repository's builds; `internal/apply` and `internal/writer` are touched only by a patch file that `build.sh` applies in a temporary checkout.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run` and `bench canary`: `--mrw-sha256`, `--mrw-instructions`; `bench run`: `--keep-state-file`, `--max-spend` | new, each off by default | T1 | T5, T6 |
| `run.json`: `mrw_sha256`, `mrw_instructions_sha256`, `keep_state_file`, `max_spend_usd`, `bench_revision` | new keys, omitted when unset (`bench_revision` always) | T1 | the script, `bench verify` |
| receipt: `mrw_sha256`, `kept_state_file_sha256`, `init_model` | new keys | T1 | the script |
| a run's exit status for a spend stop | new, distinct from the disk floor's | T1 | T6 |
| `bench/stage-9/no-refusal.patch`, `build.sh`, `builds.md` | new | T2 | T5, T6 |
| `bench/stage-9/classes.json`, the tasks, the corpus test's registered shapes and counts | new or edited | T3 | T5, T6, the script |
| `scripts/bench_stage9.py`, `scripts/test_bench_stage9.py`, `.github/workflows/ci.yml` | new; the suite joins CI's | T4 | T6 |
| `docs/bench/README.md` | the flags and Stage 9 | T1, T6 | readers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the flags and their recorded keys | T1 | T4, T5, T6 | no |
| the kept log: `mrw/stage9-exposed.jsonl` under the state home, one JSON object per line with `path`, `op`, `start`, `end`, `first_unshown`, `sha_before` | T2 | T4, T5, T6 (`--keep-state-file` is given this path) | no |
| `classes.json`: class name to task ids | T3 | T4, T6 | no |

## Implementation

`tasks/README.md`: T1 to T6 in three waves. T6, the counted pilot, is a human-proof task behind T5's canaries and smoke.

## Consequences

- **Positive:** two arms can be shown to have run different, known binaries on the same words; a run cannot overspend unseen; the no-refusal behaviour never enters a shipped binary.
- **Negative:** a patch against a tag is brittle and is re-made when the pilot's tag changes. Six directories are more to keep straight than two; the script's refusals are what hold them. The arms of a model run hours apart.
- **Neutral:** stages that do not pass the new flags are unchanged, apart from two recorded keys.

## Out of Scope

- The content-first prototype and arm X (deferred: docs/adr/BACKLOG.md)
- A powered stage (deferred: docs/adr/BACKLOG.md)
- A second turn in the bench, so a follow-on task could be two prompts (deferred: docs/adr/BACKLOG.md)
- Any change to ADR-203, to what mrw teaches, or to the standing instruction (permanent: boundary: the owner's, on a record of its own)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Both mrw arms read instructions saying a bare number is refused, so agents in N avoid one | High | Med | named in the backlog entry; low exposure in N is not read as the hazard being absent |
| Agents make both follow-on changes in one plan | Med | Med | the instruction makes the second depend on the first's result; exposure is reported with its own floor |
| The stage costs more than estimated | Med | Low | the six figures; a stopped run goes to Zy |
| The patch no longer applies to the tag | Med | Low | `build.sh` fails loudly; T2's test builds both |
| A task's local repository is missing on the machine | Low | Med | `bench run` refuses; T3's operator fence requires every oracle check to run, not skip |

## Rollback

Revert the tasks' commits; the flags are off by default and no earlier stage names them.

## Follow-ups

- [ ] Arm X, when a record specifies the prototype: M is run again beside it.
