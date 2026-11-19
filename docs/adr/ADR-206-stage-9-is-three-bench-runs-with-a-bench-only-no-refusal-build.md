# ADR-206: Stage 9 is run as three separate bench runs, with a bench-only build that lacks one refusal

**Status:** Proposed
**Drafted for:** Zy, 2026-11-19. The pilot was pre-registered in `docs/adr/BACKLOG.md` the same day, from a design review of v1.92.0. Asked "shall I start on the Stage 9 stage record and runner changes?", Zy answered *"staert"*. That starts this draft; nothing is built, run or spent until Zy accepts the record, with the spend stop in Decision 8.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-156, ADR-160, ADR-192, ADR-193, ADR-200, ADR-203, ADR-205
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/stage-9/**`, `bench/tasks/**`, `scripts/bench_stage9.py`, `docs/bench/**`
**Enforced-by:** `internal/bench/stage9_test.go::TestARunPinsItsMrwItsInstructionsAndItsSpend`
**Served-path change:** none to `mrw`. `bench run` gains four flags; a run without them behaves as before.

## Context

**What was registered.** The backlog entry "cost per correctly completed edit task (Stage 9, a descriptive pilot)" asks what each way of editing costs per task done right, and whether a number-only edit to the wrong line happens when nothing refuses it. It names four arms (the host's tools; mrw; mrw with ADR-203's refusal off; a content-first prototype), four task classes of 12, and what must be committed before a counted attempt. It decides nothing about ADR-203, the taught form or the standing instruction.

**What the bench cannot do today**, read from the code at `755ab96f`:

- *One mrw binary per run, by live path.* `RunOptions.Mrw` is one string; each attempt symlinks it onto PATH (`internal/bench/runner.go`, `invocation`); only `mrw version` is read, once. No hash of the binary is taken anywhere.
- *Instructions come from the arm's own binary.* `exec.Command(o.Mrw, "instructions")` per attempt; the text is neither stored nor hashed. Two arms with different binaries cannot be shown to have read the same words.
- *No stop on spend.* The only stops at an attempt boundary are three start errors and the disk floor. Stage 8's $15 stop was a person reading receipts.
- *Nothing a binary writes survives the attempt* except what `keepHookFiles` copies for configuration H: the scratch directory, with mrw's state under it, is deleted.
- *A configuration cannot be limited to some tasks*, and `analyze` and `look` pair arms only inside one run directory and stop on a cost that is missing without a timeout.
- *The letter G is taken.* `Configs["G"]` is the plugin's Go-hooks side arm; its receipts go to `side/` and it demands the stage6 profile.
- *Grading.* `Verdict.Solved` holds when `Missing`, `Outside` and `Structure` are all empty. `exact` compares each file the reference touches with the reference result and fails as a setup exclusion when the reference creates a file. `Outside` lists, in the base tree's numbering, each changed hunk the reference does not allow, mixed with whole-file reasons. A task stores `reference.patch`, not a tree.
- *One prompt per attempt.* The bench cannot send a second turn, so a follow-on task is one instruction and an agent may make both changes in one plan.
- *Model requests are not recorded*; `num_turns` and `tool_calls` are.

**Audit of the class**: *the ways two arms of a bench stage can differ without the receipts saying so.* The binary's bytes (decided: Decision 2), the appended instructions (Decision 3), the Claude Code release (already pinned per run and compared across blocks), the task list (Decision 5), the prompt and status line (already hashed in `run.json`), the environment mrw reads (Decision 4 names the one variable), and what `mrw version` prints to an agent that asks (Decision 4).

## Existing Primitives Audit

- `verifyAsset` and `stageQH` (`internal/bench/sidearms.go`) already verify a binary against a registered SHA-256, hold its bytes, copy them into each attempt and record the hash in the receipt, for the qh side arm.
- `keepHookFiles` copies named files out of the scratch directory before it is deleted.
- `--status-line` records the agent's own STATUS line as `status`; `--task` selects tasks by id; `claimOut` refuses a second plan in one directory.
- `scripts/bench_edits.py` (ADR-193) is the model for a frozen counting script with its hash in the results.
- ADR-200's count: `writer.Land` passes `apply.Result.StaleNumbered` to `authoring.RecordNumbering`; since ADR-203 the second number is 0, because such a hunk is refused.

## Decision

1. **Each arm is its own `bench run`, in its own directory, under a configuration that exists.** Arm **A** is configuration A. Arms **M** and **N** are both configuration B, with different `--mrw` binaries. No configuration letter and no profile is added. The backlog's "G" is renamed **N** (no refusal); the content-first arm stays **X**, runs later, and is not built by this record.
2. **A run pins its mrw by hash.** `bench run --mrw-sha256 HEX` refuses to start unless the file at `--mrw` has that SHA-256, copies the bytes into each attempt instead of linking to the live path, and records the hash in `run.json` and in each receipt. Without the flag a run behaves as before.
3. **A run can be given its instructions.** `bench run --mrw-instructions FILE` appends that file's bytes instead of running the binary's `instructions`, and records the file's SHA-256 in `run.json`. M and N are given the same file, the output of M's binary, committed under `bench/stage-9/`. Equal hashes in the two `run.json` files are the proof that both arms read the same words.
4. **Arm N's binary is built from M's tag with one patch that lives under `bench/`, never in shipped source.** `bench/stage-9/no-refusal.patch` applies to the tag M is built from and changes two things: a hunk ADR-203 would refuse is applied, and its path, its line as addressed and that line's text before the write are appended to `mrw-stage9-exposed.log` in mrw's state directory. `bench/stage-9/build-n.sh` checks out the tag in a temporary directory, applies the patch, builds with M's release flags and `-buildvcs=false`, and prints the SHA-256. A test that drives both binaries holds what N is: it lands the hunk M refuses and logs it; it still refuses a line no read has served; it prints to the agent exactly what M prints for every other command in the test, `mrw version` and `mrw instructions` included; and with the log's directory missing it behaves the same and logs nothing. The patch is re-made if the tag moves; it is not carried in `internal/`.
5. **A run can keep a file mrw wrote.** `bench run --keep-state-file NAME` copies `NAME` from the attempt's mrw state directory beside the receipt before the scratch directory is deleted, and records its SHA-256 in the receipt (absent when the file was not written). Arm N keeps `mrw-stage9-exposed.log`.
6. **A run stops on spend.** `bench run --max-spend USD` sums `total_cost_usd` over the receipts in its directory before each attempt and stops, resumable like the disk floor, when the sum has reached the figure. A receipt whose cost is null is counted as missing, and three missing costs stop the run too: a stop that cannot see what was spent is not one. The stage's total is the sum over its run directories, which the script in Decision 7 prints and T6 reads before each run starts.
7. **Counting is a frozen script, `scripts/bench_stage9.py`**, committed with its hash before the first counted attempt, as ADR-193 did. It reads the run directories and the transcripts and never `bench analyze` or `bench look`. What it computes, each from fields that exist:
   - `exact`: the receipt's oracle is solved, on a task that is `exact: true`. Every Stage 9 task is, none creates a file, and each carries a `require` the reference result satisfies, so solved means the touched files equal the reference result and nothing else changed.
   - `wrong_location`: not `exact`, and `Outside` names a hunk or a file. `other_inexact`: not `exact`, and `Outside` is empty.
   - `claimed`: the receipt's `status` is `done`.
   - `exposed`, arm N: the lines of the kept log. `exposed`, arm M: the tool results in the transcript that carry ADR-203's refusal, counted by its fixed words (`has no recorded showing in this version`), with the line it names.
   - *an exposed call that landed on a wrong line*, arm N: a log entry whose recorded line text is not the text of any line the reference patch removes or replaces in that file. The text is used because the log's numbers are the file's at the time of the write and `Outside`'s are the base tree's.
   - *a refusal of a number that was still right*, arm M, control class: a refusal whose named line, in the base numbering, is a line the reference patch changes. On a control task the first change moves no line, so the two numberings agree.
   - spend, `num_turns`, `tool_calls`, tokens and wall time; a null cost is missing, never zero.
   - the comparisons, the INSUFFICIENT rules, the 95% percentile bootstrap (4,000 resamples, seed 206) and the dropped-resample rule, as the backlog entry states them.
8. **The stage.** Models: `claude-opus-5-5` and `claude-sonnet-5-5`, one repeat. Claude Code: the release current when T5's canary runs, the same for every run. M: the mrw release asset current at T2, verified against its `SHA256SUMS.txt`; N: built from that tag. Arm A and arm M run all 48 tasks; arm N runs the 24 follow-on and control tasks: 240 attempts. Stage 8's 72 attempts cost $8.61, on larger tasks; **the stop is $40 for the stage**, enforced per run by `--max-spend` figures that sum to it ($15 for A, $15 for M, $10 for N), and if it is reached the stage stops and goes to Zy. Seed 206. A class is a committed list, `bench/stage-9/classes.json`, not a task's `shape`.
9. **Before a counted attempt**, committed and pushed: the 48 tasks and `classes.json` with its hash; the instructions file and its hash; M's and N's hashes and the patch; the script and its hash; a canary per model and arm; and an uncounted smoke of two tasks per arm whose receipts show the hashes, the kept log in N and the stop working. Nothing in Decisions 1 to 8 changes after the first counted attempt.
10. **What would make this record wrong.** N differing from M in something an agent can see besides the refusal: the test of Decision 4 is what stands against that, and a difference found later voids the M-against-N comparison, not the rest. Or a follow-on class that agents solve in one plan every time, leaving nothing exposed: the backlog entry already says what that result means.

## Alternatives Considered

- **A new configuration letter and a stage9 profile.** Rejected: about twenty sites, the canary and the side-arm code, to say what two `--out` directories already say.
- **A build tag or an environment switch in `internal/apply` that turns the refusal off.** Rejected: a shipped binary would carry a way to disable a guard, for one experiment.
- **mrw v1.91.0 as the no-refusal arm.** Rejected: it differs from v1.92.0 in more than the refusal (`read --sub` anchors, the stats text), and it cannot say which hunks it would have refused.
- **Counting exposure in N from transcripts.** Rejected: it means modelling every way a line gets shown; the binary already knows.
- **Extending `bench analyze`.** Rejected for a pilot: it pairs within one run and stops on a missing cost; a frozen script is what ADR-193 used.

## Component / Boundary Impact

`internal/bench` and `cmd/bench` (four flags, their `run.json` and receipt fields). `bench/stage-9/` (the patch, the build script, the class list, the instructions file). `scripts/` (the counting script and its tests). No package of `mrw` changes; `internal/apply` is touched only by a patch file that no build of this repository applies.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run --mrw-sha256`, `--mrw-instructions`, `--keep-state-file`, `--max-spend` | new, each off by default | T1 | T5, T6 |
| `run.json`: `mrw_sha256`, `mrw_instructions_sha256`, `max_spend_usd` | new keys, omitted when unset | T1 | the script, `bench verify` |
| receipt: `mrw_sha256`, `kept_state_file_sha256` | new keys, omitted when unset | T1 | the script |
| `bench/stage-9/no-refusal.patch`, `build-n.sh` | new | T2 | T5, T6 |
| `bench/stage-9/classes.json`, 48 tasks | new or selected | T3 | T5, T6, the script |
| `scripts/bench_stage9.py` | new | T4 | T6 |
| `docs/bench/README.md` | the flags and Stage 9 | T1, T6 | readers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the four flags and their recorded keys | T1 | T2, T4, T5, T6 | no |
| the exposed log's line format: `PATH<TAB>LINE<TAB>TEXT` | T2 | T4 | no |
| `classes.json`: class name to task ids | T3 | T4, T6 | no |

## Implementation

`tasks/README.md`: T1 to T6. T6, the counted pilot, is a human-proof task behind T5's canary and smoke.

## Consequences

- **Positive:** two arms can be shown to differ in one thing; a stage can no longer overspend unseen; the no-refusal behaviour never enters a shipped binary.
- **Negative:** a patch against a tag is brittle, and is re-made when the pilot's mrw changes. Three run directories per model are more to keep straight than one; the script's own check that their hashes agree is what holds them.
- **Neutral:** stages that do not pass the new flags are unchanged.

## Out of Scope

- The content-first prototype and arm X (deferred: docs/adr/BACKLOG.md)
- A powered stage (deferred: docs/adr/BACKLOG.md)
- A second turn in the bench, so a follow-on task could be two prompts (deferred: docs/adr/BACKLOG.md)
- Any change to ADR-203, to what mrw teaches, or to the standing instruction (permanent: boundary: the owner's, on a record of its own)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Both mrw arms read instructions saying a bare number is refused, so agents in N avoid one | High | Med | stated in the backlog entry under what the pilot cannot show; low exposure in N is not read as the hazard being absent |
| Agents make both follow-on changes in one plan | Med | Med | the instruction makes the second depend on the first's result; exposure is reported, and under 3 of 12 is named as not provoked |
| The patch no longer applies to the tag | Med | Low | `build-n.sh` fails loudly; T2's test builds it |
| A task's local repository is missing on the machine | Low | Med | `bench run` already refuses; T3 uses repositories in `bench/repos.json` |
| Spend passes the stop between receipts | Low | Low | the stop is checked before each attempt; one attempt's cost is the most it can overrun |

## Rollback

Revert the task's commit; the flags are off by default and no stage before this one names them.

## Follow-ups

- [ ] Arm X, when a record specifies the prototype: M is run again beside it.
