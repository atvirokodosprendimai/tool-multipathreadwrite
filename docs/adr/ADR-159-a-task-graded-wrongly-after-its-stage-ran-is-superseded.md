# ADR-159: a task graded wrongly after its stage ran is superseded, and a descriptive row shows each stage without it

**Status:** Accepted
**Accepted:** 2026-10-15 by Zy — picked "Supersede + descriptive row", after Stage 1b's unsolved attempts showed `cac-rename-post-send`'s reference fix renames a second, unrelated function.
**Date:** 2026-10-15
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-158
**Governs:** `internal/bench/analyze.go`, `cmd/bench/main.go`, `bench/tasks/cac-rename-post-send/`, `bench/tasks/cac-rename-post-send-2/`, `docs/bench/README.md`
**Amends:** ADR-158 Decision 3's rule that a task is superseded "after the pilot only for an oracle defect shown on its base or its reference fix": that rule governs a stage while it runs; this record supersedes a task after its stage ran, for a later stage, and leaves every published verdict as it was computed.
**Enforced-by:** None — until T2 lands, which adds `internal/bench/analyze_test.go::TestARowWithoutTheTasksSupersededSinceIsDescriptive`
**Served-path change:** None — this ADR changes only measurement: `bench`, its corpus and its analysis.

## Context

Stage 1b ran on 2026-10-15 (160 attempts, receipts in `docs/bench/stage-1b-sonnet/`, ADR-158 T3) and its registered verdict is **INCONCLUSIVE**: cost (B−A)/A −0.004 [−0.050, +0.042], solved B−A −0.013 [−0.037, +0.000].

Its only unsolved attempts are three of `cac-rename-post-send` (A 1 of 4, B 2 of 4), and they are the whole of the solved difference. At `crossagentschat@ae4f7a9c`, `internal/httpapi` holds two things named `postSend`:

- the handler method `(h *handler) postSend` in `send.go`, with its route in `httpapi.go`;
- an unrelated package-level test helper, `func postSend(t *testing.T, url, body string)`, in `send_test.go`.

The instruction says "rename the function postSend to sendMessage, and update every reference to it". The reference fix renames both. Each unsolved attempt renamed the method and its references, left the helper, and said why: it is a separate function, not a reference to the method. That reading is defensible, and the oracle refuses it. Stage 1 graded the same task A 2/2, B 0/2 (`docs/bench/stage-1-sonnet/`), and the defect went unnoticed there.

**The class, audited 2026-10-15.** The class is every rename task whose old name has more than one definition in the scope the instruction names. Enumerated by listing the corpus's rename tasks (`bench/tasks/*-rename-*`, eight), then, for each Go one, searching its package at the pinned commit for `func NAME(` or `func (…) NAME(`, read through `mrw read --grep` over a `git archive` of the package. `buildClosetLines`, `copyConfigTree`, `endWingRootIfEntryRoomIsEmpty`, `extractContentDate` and `writeEvent` each have one definition. `postSend` has two (`send.go:20`, `send_test.go:11`). The two JavaScript renames (`qh-rename-find-git-dir`, `qh-rename-git-lines`) say "everywhere it is defined", so a second definition is in their instruction. The defect is this one task's.

ADR-158's registration supersedes a task after the pilot only for an oracle defect shown on its base or its reference fix, never for what an attempt did. This defect is in the reference fix, but it was found through attempts, after both stages ran. So both published verdicts stand as computed.

## Existing Primitives Audit

- `superseded_by` in `task.json`, `SelectTasks` and the shape counts (ADR-158 T2) already retire a task without editing it: reused.
- `Analyze`'s descriptive rows (ADR-156 T6 S5, ADR-158 Decision 6), marked in parentheses and never the verdict: reused for one more row.
- `LoadTasks` (ADR-156 T1) reads the corpus: `bench analyze` uses it to learn which tasks are superseded now.

## Decision

1. **`cac-rename-post-send` is superseded by `cac-rename-post-send-2`.** The new task's instruction names the handler method and says to leave the test helper; its reference fix is the old one's `httpapi.go` and `send.go` hunks, without `send_test.go`. Same repository, commit and shape (`multi-site`), `exact` kept. The old task's instruction, oracle and fix are not edited; it gains `superseded_by` only (ADR-158 Decision 3).
2. **One more descriptive row: each stage without the tasks superseded since it ran.** `bench analyze` loads the corpus (`--tasks-dir`, default `bench/tasks`) and passes the ids of superseded tasks to `Analyze`. For every stage, model and source whose attempts include one of them, a row follows, computed without them and labelled "without the tasks superseded after the stage ran; registered after it ran, descriptive, not the verdict". The verdict row is unchanged and still counts every task. For Stage 1 that row leaves out `site-html-cost-faq`, `am-go-sha-message` and `cac-rename-post-send`. For Stage 1b it leaves out `cac-rename-post-send`.
3. **ADR-158 T3's fence compares `run.json` with the corpus at the commit the stage ran from** (`edb65629`), not with the corpus as it stands, since a task superseded later must not make a finished stage's receipts look wrong.

## Alternatives Considered

- **Leave the task as it is** (Zy's third option): a later stage would carry the same misgrade. Rejected by Zy.
- **Supersede only, with no row:** the reader of Stage 1 and 1b could not see how much of each result the defect carries. Rejected by Zy.
- **Recompute the verdict without the task:** that changes a registered verdict after its stage ran, which ADR-156 forbids. Not considered further.

## Component / Boundary Impact

`internal/bench` (`Analyze` gains a variadic argument; no existing caller changes) and `cmd/bench` (`analyze` gains `--tasks-dir`). The corpus gains one task, and one task gains a field.

## Wiring & Contract Changes

| Change | Where | Task | Selected by |
|---|---|---|---|
| `cac-rename-post-send` gains `superseded_by`; `cac-rename-post-send-2` added | `bench/tasks/` | T1 | `bench run`'s default plan (`SelectTasks`) |
| ADR-158 T3's fence pinned to `edb65629` | `docs/adr/ADR-158-…/tasks/T3-stage-1b-sonnet.md` | T1 | `adr-verify` |
| a descriptive row without tasks superseded since | `Analyze`, `cmd/bench` analyze `--tasks-dir` | T2 | `bench analyze` |

## Inter-task Contracts

T1 produces `superseded_by` on `cac-rename-post-send`; T2's row over the published stages consumes it.

## Implementation

T1, then T2. No stage is run.

## Consequences

- **Positive:** a later stage grades the rename by what its instruction asks; a reader sees each published stage with and without the misgraded tasks.
- **Negative:** the row is registered after the stages ran, so it is evidence of how much a known defect moved a result, never a test of the bar; its label says so.
- **Neutral:** the published verdicts, their receipts and `TestStageOnesPublishedVerdictIsUnchanged`'s numbers are untouched.

## Out of Scope

- Running a stage with `cac-rename-post-send-2` (deferred: docs/adr/BACKLOG.md "From ADR-159")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| the post-hoc row is read as a verdict | Medium | Medium | the label says "registered after it ran" and "not the verdict"; the row sits in parentheses like the other descriptive rows |

## Rollback

Revert the commit: the field, the task and the row go together; no receipt changes.

## Follow-ups

The BACKLOG entry above, under "From ADR-159".
