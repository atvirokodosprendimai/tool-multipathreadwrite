# ADR-192: The read hook is measured on SWE-bench, as a configuration H read against B and A

**Status:** Proposed
**Date:** 2026-11-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-162, ADR-163, ADR-191, docs/bench/observability/HOOK-RESULTS.md
**Amends:** ADR-162 Decision 5's sentence "later attempts of its second configuration are not run", for stage7 only (Decision 3)
**Governs:** `internal/bench/**`, `cmd/bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestConfigurationHIsBPlusReadAndTheReadHook`
**Invalidates:** none — checked: without `--profile stage7` every configuration's arguments, tools, prompt and refusal text stay as ADR-162 and ADR-163 left them, so Stages 1 to 6 stay reproducible; `bench look docs/bench/stage-5` stays byte-identical to `looks/look-5.txt` (T2's fence); Stage 5's receipts, looks and verdicts are untouched
**Served-path change:** None — this ADR changes only measurement: `bench`, its registration and its stage. Whether mrw's own guide stops telling agents to read with mrw is decided after the verdict (Out of Scope).

## Context

ADR-191 (`mrw hook read`, merged in #600 at `1f01dcdc`, in no tag as of 2026-11-03) licenses the lines a host's
own Read showed, so a plan to them needs no second read. Its motivation was one external recheck: 26 re-reads of
about 1,000 lines each in 17 runs (docs/agents.md §9).

Run 9 (docs/bench/observability/HOOK-RESULTS.md, 2026-11-02, five small grouped jobs) offered the host Read and
the hook and found **0 host Read calls in 10 cells**: the appended core guide's rule 1 says "Use mrw for every
file read, edit and create", and the agent obeyed it. The hook never ran. Native was also lower on cost by a
mean $0.035 per five jobs, at about $0.02 and 3 turns a job.

Stage 5 (ADR-162, SWE-bench Verified, 191 multi-hunk instances, closed at look 5, 2026-10-21) measured mrw where
an agent explores a repository it has never seen: B against A cost −19.4% [−25.5, −12.7] on Opus 5.5 at 14.6
native turns per task and −7.0% [−12.1, −1.8] on Sonnet 5.5 at 7.8 (`docs/bench/stage-5/looks/look-5.txt`).
Both are INCONCLUSIVE: solving's non-inferiority bound was not met on either, and Sonnet's −7% is also short of
the −10% saving PAYS needs (ADR-162 Decision 11). Zy, 2026-11-03: *"yes, draft, since this is where real work
happens. not a pilot bench"*.

Stage 5's B ran mrw v1.68.0 on Claude Code 2.1.302, tools `Bash, Glob`, the full `mrw instructions` appended
(`internal/bench/runner.go:42`, `:1071`). A stage keeps one mrw release (`runner.go:271`). The CLI on this machine
is 2.1.317 (2026-11-03), the hook's payload was captured on 2.1.316 (`cmd/mrw/hookread191.go:92`), and the
profile's built-in skill and plugin lists come from a 2.1.302 probe (`internal/bench/profile.go:28-35`). So B is
run again here on a newer mrw and a newer CLI, and Stage 5's numbers are context, never pooled.

`docs/adr/BACKLOG.md` "From ADR-190 and ADR-191" deferred "the measurement against native" to a grouped run; run 9
was that run and could not exercise the hook. This record takes up ADR-191's half on SWE-bench; ADR-190's
compacting half stays deferred there.

## Existing Primitives Audit

- **Reuse:** ADR-162's runner, pool (`bench/swe-bench/selection-adr-162.json`, 191 instances in its seeded order),
  blocks, Stage 5 profile, canary, five looks at a nominal 99%, eligibility, the contamination scan, the disk floor;
  ADR-156's bar (cost by percentile bootstrap, solved by Tango's score interval at 99%).
- **Reuse:** `scripts/bench_mechanism.py` at `3a970837` (#605) and HOOK-PREREGISTRATION.md's counting rules, over
  H's transcripts, as proxies (Decision 4).
- **Reuse:** `judgeAgainst` (`internal/bench/analyze.go:260`) already judges against a named baseline.
- **Reshape:** `bench look`, `bench analyze` and its table read every comparison against A (`internal/bench/look.go:134`,
  `:143`, `:163`, `:188`, `:229`, `:243`, `:277`, `:352`; `cmd/bench/main.go:447-457`). ADR-163's side arms
  (`internal/bench/sidearms.go`) read a configuration against P alone and are the precedent for a named baseline.
- **Not reused:** the `stage6` profile (ADR-163), whose surviving pool was never frozen (`bench/swe-bench/` holds no
  `selection-adr-163` file); ADR-162 Decision 10's quality-harness scorer and `export-paired`, which serve a plugin
  arm this stage does not run.

## Decision

1. **Configuration H is B plus the host Read, the read hook and one sentence.**
   - **Tools.** `Bash, Glob, Read`, and `Skill` as the profile offers every configuration. No `Grep`, `Edit` or
     `Write`.
   - **Hook.** A `PostToolUse` entry, matcher `Read`, whose command runs `<mrw> hook read`, appends its stderr to
     the attempt's `hook.stderr`, and appends one line to `hook.fired`. Both files are copied beside the receipt
     before the scratch is removed; the receipt counts `hook_runs` (lines of `hook.fired`) and `hook_stderr_bytes`.
   - **Prompt.** `mrw instructions` exactly as B gets it, then one fixed sentence:
     *"In this session you may also read files with the Read tool: the lines a Read shows you count as read for
     mrw, so a plan may edit them without reading them again with mrw. Edit and create files only with mrw
     write."* Its SHA-256 is in `run.json` and is refused if it changes between blocks. T3 may change the sentence
     once, before any counted attempt, and not after.
   - **Bench's own refusal text.** bench's Bash hook tells an agent "read them with mrw read"
     (`internal/bench/hook.go:52`); under stage7 H's says "with mrw read or the Read tool", so the arm is not told
     two things.
   - **Bundle, said plainly.** H against B measures *host Read permitted, plus the hook, plus the sentence*. It does
     not separate them; the counts in Decision 4 make the hook's part plausible, not proven.
2. **Stage 7: three configurations, both models, ADR-162's pool and blocks, one pinned CLI.** Profile `stage7` is
   `stage5` plus configuration H. Configurations A, B and H; Sonnet 5.5 and Opus 5.5; one repeat; the 191 instances
   in ADR-162's order and blocks; one mrw release and one Claude Code executable of 2.1.316 or later for the whole
   stage, as ADR-162 Decision 7. T3 re-probes the built-in skills and plugins on that CLI before any canary.
3. **Three comparisons per model, each with ADR-156's verdict at 99% over five looks.**
   - **H against B** (the question); **B against A** (Stage 5's again, on the new mrw and CLI); **H against A**
     (the claim a README could make). Eligibility is read per comparison against its own baseline.
   - Six verdicts in all, not a family held at 5%. A verdict that closes early overstates its effect and is
     published with its look (ADR-162 Decision 5).
   - **The stop rule, amended for stage7.** A configuration is scheduled while any open comparison includes it, so
     B runs on for H against B after B against A closes, and A stops once B against A and H against A are both
     closed. Closed rows are frozen (`look.go:157`): later attempts never move a closed verdict.
4. **An uptake gate, and mechanism counts beside every verdict.**
   - **What it counts.** Per model, over the block-1 instances, the final receipt of H that eligibility's walk
     reaches (`look.go:213-220`): excluded attempts are left out and replaced as ADR-156 replaces them; timed-out
     and over-budget attempts count. An attempt passes when its `host_reads` is at least 1.
   - **When it is read.** At the first look where H against B is eligible. If fewer than half of the counted
     attempts pass, H's two comparisons close **UNTESTED**, overriding a PAYS or LOSES reached at the same look,
     and H is not scheduled again for that model. B against A goes on.
   - **The counts,** per H attempt, beside the verdicts and never in their place: from the receipt, `host_reads`,
     `hook_runs` and `hook_stderr_bytes`; from `scripts/bench_mechanism.py` at `3a970837`, hook-licensed writes,
     overlapping mrw reads, refused unread writes and unknown calls, which are proxies (a failed Read followed by a
     legal `--create` counts as hook-licensed). `hook_runs` below `host_reads`, or hook-licensed writes at zero
     with Reads above it, says the hook did not work.
5. **What each outcome means, written now.**
   - **Expected:** H against B INCONCLUSIVE. No offline estimate of the re-reads H removes exists, since B never
     used the Read; PAYS needs about a 10% saving with solving at or above B's (ADR-162 Decision 11).
   - H against B **PAYS**: evidence for Zy to change rule 1 of mrw's guide; that change is its own record.
   - **LOSES**, or **INCONCLUSIVE** (including cheaper with solving's bound unmet): rule 1 stays.
   - **UNTESTED**: the sentence did not move the agent; no claim about the hook follows.
6. **Prerequisite: a release.** No counted attempt runs before a tag `v1.83.0` or later contains `1f01dcdc`
   (ADR-191), and the stage's `mrw_version` names that tag. Cutting it is Zy's call (the release skill).
7. **Price, said first.**
   - **Ceiling:** 1,146 attempts (191 × 3 × 2), as ADR-162 planned Stage 5. Stage 5 ran 850 receipts, since P
     closed LOSES at look 1 on both models: $131.12 in API prices and 15.7 hours of attempt wall time, in blocks
     started between 2026-10-17 and 2026-10-21 (measured 2026-11-03 over `docs/bench/stage-5/block-*/*.receipt.json`).
     At that pace the ceiling is about 21 hours of attempt time, spread over days.
   - **Projection:** A and B as Stage 5 spent ($49.73 and $38.20 Opus, $11.34 and $10.13 Sonnet) plus H priced as
     B: about $158, on the subscription.
   - **Savings:** an UNTESTED gate at look 1 saves 151 H attempts per model; each comparison closing early saves
     more. T3's uncounted pilot is 2 instances × 2 models × H plus the canaries.

What would make it FAIL:
- an H whose init tools are not `Bash, Glob, Read, Skill`, whose canary shows no `PostToolUse:Read` hook event,
  or whose sentence's sha differs from T3's registered one;
- an A or B that carries the Read hook, the sentence or H's refusal text;
- a verdict read against the wrong baseline, with eligibility read against another, at another level, or at an
  ineligible look;
- an H comparison reported as PAYS or LOSES after the uptake gate failed;
- an instance chosen, skipped or reordered for an outcome; a profile, sentence, release or CLI that changes
  between blocks;
- a README sentence stating more than its verdict.

## Alternatives Considered

- **A fourth configuration N, B plus the Read with no hook.** Separates the hook from the Read. Rejected for now:
  a third more attempts (about 382) for a question H against B only raises if it PAYS; it is the follow-up then.
- **Opus only.** Opus had the larger saving and the longer sessions. Rejected: Sonnet is about $31 of the $158
  projection, a fifth of the cost, and most adopters run it.
- **The `stage6` profile, checkouts that run their tests.** Closer to real work. Rejected: its surviving pool needs
  `bench baseline` over all 191 first (ADR-163 Decision 4) and was never frozen.
- **Change mrw's guide first, then measure.** Rejected: the guide is every adopter's served path; this stage decides
  whether to change it.
- **Run 10 on the grouped pilot.** Rejected (Zy, 2026-11-03): three-turn jobs leave no re-read to remove.

## Component / Boundary Impact

None — internal to `bench`: `internal/bench` (a configuration, a profile, a baseline per comparison, the gate) and
`cmd/bench` (the flags, the canary's configurations, the analysis table). No mrw engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run` / `bench canary` `--profile stage7 --configs A,B,H` | new profile and configuration | T1 | T3, T4 |
| `run.json` `h_sentence_sha256` | new field, refused if it changes between blocks | T1 | T4, T5 |
| receipt `host_reads`, `hook_runs`, `hook_stderr_bytes` (H) | new fields | T1 | T2, T5 |
| `<attempt>.hook.stderr`, `<attempt>.hook.fired` | kept beside the receipt | T1 | T5 |
| `bench look` / `bench analyze` comparison `H:B`, verdict UNTESTED | a comparison names its baseline | T2 | T4, T5 |
| `internal/bench/profile.go` built-in skills and plugins | re-probed on the stage's CLI | T3 | T4 |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| configuration H and profile `stage7` | T1 | T2, T3, T4 | No — new names |
| receipt `host_reads`, `hook_runs` | T1 | T2, T5 | No — added keys |
| comparison `H:B` and UNTESTED in `bench look` | T2 | T4, T5 | No — Stage 5's rows unchanged |
| the frozen sentence, its sha, the re-probed built-ins | T3 | T4 | No |

## Implementation

See `docs/adr/ADR-192/tasks/README.md`.

## Consequences

- **Positive:** the hook is measured where re-reads happen, on the pool and bar the published claim stands on.
- **Positive:** B against A is replicated on the release that carries ADR-191 and a current CLI, where Stage 5 ran
  v1.68.0 on 2.1.302.
- **Negative:** at most about 21 hours of attempt time on a shared machine, on the subscription, for an expected INCONCLUSIVE.
- **Negative:** H against B cannot say whether the Read, the sentence or the hook did the work.
- **Neutral:** bench's slipped-write count is biased low in H: a host Read records a file's sha, so a non-mrw change
  later Read is not counted (`internal/bench/hook.go:724-735`). Descriptive only.

## Out of Scope

- Changing rule 1 of mrw's guide (`internal/guide/guide.go`) (deferred: docs/adr/BACKLOG.md "From ADR-192")
- Configuration N, Read without the hook (deferred: docs/adr/BACKLOG.md "From ADR-192")
- ADR-190's compacting measurement (deferred: docs/adr/BACKLOG.md "From ADR-190 and ADR-191")
- Checkouts that run their tests (permanent: boundary: ADR-163's pool is unfrozen; this stage keeps Stage 5's profile so B against A stays comparable with Stage 5)
- ADR-162 Decision 10's scorer and `export-paired` rows (permanent: boundary: they serve quality-harness's plugin arm, which this stage does not run)
- Hosts other than Claude Code (permanent: boundary: `mrw hook read` reads Claude Code's PostToolUse payload, docs/agents.md §9)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The CLI differs from Stage 5's (2.1.302 to 2.1.316 or later) | Certain | Med | named in Context; T3 re-probes the built-ins; B against A is read as a replication on a new CLI |
| The sentence does not move the agent to Read | Med | High | T3's pilot measures uptake first; the gate closes UNTESTED |
| The hook runs but licenses nothing (a payload it does not know) | Med | High | T3's preflight: a Read then a plan edit that applies with the hook and is refused without it; `hook_runs` and the proxies per attempt |
| Shared-machine load skews wall time | High | Low | wall time descriptive only; each block starts below the core count |
| Disk for 191 images | Med | Med | ADR-162 Decision 9's prune and 25 GiB floor |

## Rollback

None — measurement only: `stage7` and H are opt-in names; removing them leaves Stages 1 to 6 as they were.

## Follow-ups

- [ ] If H against B PAYS on either model: a record for rule 1 of the guide, and configuration N.
