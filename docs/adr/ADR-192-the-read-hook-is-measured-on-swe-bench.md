# ADR-192: The read hook is measured on SWE-bench, as a configuration H read against B and A

**Status:** Proposed
**Date:** 2026-11-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-162, ADR-163, ADR-191, docs/bench/observability/HOOK-RESULTS.md
**Governs:** `internal/bench/**`, `cmd/bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestConfigurationHIsBPlusReadAndTheReadHook`
**Invalidates:** none — checked: without `--profile stage7` every configuration's arguments, tools and prompt stay as ADR-162 and ADR-163 left them, so Stages 1 to 6 stay reproducible; `bench look` keeps reading every Stage 5 comparison against A; Stage 5's receipts, looks and verdicts are untouched
**Served-path change:** None — this ADR changes only measurement: `bench`, its registration and its stage. Whether mrw's own guide stops telling agents to read with mrw is decided after the verdict (Out of Scope).

## Context

ADR-191 (`mrw hook read`, merged in #600 at `1f01dcdc`, in no tag yet) licenses the lines a host's own Read
showed, so a plan to them needs no second read. Its motivation was one external recheck: 26 re-reads of about
1,000 lines each in 17 runs (docs/agents.md §9).

Run 9 (docs/bench/observability/HOOK-RESULTS.md, 2026-11-02, Sonnet 5.5, five small grouped jobs) offered the
host Read and the hook and found **0 host Read calls in 10 cells**: the appended core guide's rule 1 says "Use
mrw for every file read, edit and create", and the agent obeyed it. The hook never ran. Native was also lower
on cost by a mean $0.035 per five jobs. Those jobs cost about $0.02 and 3 turns each.

Stage 5 (ADR-162, SWE-bench Verified, 191 multi-hunk instances, look 5 of 5, 2026-10-21) measured mrw where an
agent explores a repository it has never seen: B against A cost −19.4% [−25.5, −12.7] on Opus 5.5 at 14.6
native turns per task and −7.0% [−12.1, −1.8] on Sonnet 5.5 at 7.8, both INCONCLUSIVE because solving's
non-inferiority bound was not met (memoryd `405f5321`; `docs/bench/stage-5/looks/look-5.txt`). That is where a
re-read costs something. Zy, 2026-11-03: *"yes, draft, since this is where real work happens. not a pilot bench"*.

Stage 5's B ran mrw v1.68.0 on Claude Code 2.1.302 with tools `Bash, Glob` and the full `mrw instructions`
appended (`internal/bench/runner.go:42`, `:1071`). A stage keeps one mrw release (`runner.go:271`), so B is run
again here on the release that carries ADR-191, and Stage 5's numbers are context, never pooled.

`docs/adr/BACKLOG.md` "From ADR-190 and ADR-191" deferred "the measurement against native" to a grouped run;
run 9 was that run, and it could not exercise the hook. This record takes up ADR-191's half on SWE-bench.
ADR-190's half (its compacting arm) stays deferred there.

## Existing Primitives Audit

- **Reuse:** ADR-162's runner, pool (`bench/swe-bench/selection-adr-162.json`, 191 instances in its seeded
  order), blocks, Stage 5 profile, canary, five looks at a nominal 99%, eligibility and stop rule, the
  contamination scan, the disk floor; ADR-156's bar (cost by percentile bootstrap, solved by Tango's score
  interval at 99%).
- **Reuse:** `scripts/bench_mechanism.py` at `3a970837` (#605), HOOK-PREREGISTRATION.md's frozen counting
  rules, run over H's transcripts.
- **Reshape:** `bench look` and `bench analyze` read every comparison against A (`internal/bench/look.go:41`).
  H against B needs a second baseline; ADR-163's side arms (`internal/bench/sidearms.go`) read a configuration
  against P alone and are the precedent for a named baseline.
- **Not reused:** the `stage6` profile (ADR-163), whose surviving pool was never frozen
  (`bench/swe-bench/` holds no `selection-adr-163` file), so checkouts that run their tests are not available
  without first running its baseline (Alternatives).

## Decision

1. **Configuration H is B plus the host Read, the read hook and one sentence.**
   - **Tools.** `Bash, Glob, Read` (and `Skill`, as the profile offers every configuration). No `Grep`, `Edit`
     or `Write`: the difference from B is the Read and nothing else a tool list can carry.
   - **Hook.** A `PostToolUse` entry, matcher `Read`, command `<mrw> hook read 2>> <scratch>/hook.stderr`,
     beside B's `PreToolUse` Bash hook. The stderr file is kept with the attempt.
   - **Prompt.** `mrw instructions` exactly as B gets it, then one fixed sentence:
     *"In this session you may also read files with the Read tool: the lines a Read shows you count as read
     for mrw, so a plan may edit them without reading them again with mrw. Edit and create files only with
     mrw write."* Its SHA-256 is in `run.json`, as ADR-162 Decision 3 holds the STATUS sentence. T3 may change
     the sentence once, before registration, and not after.
   - **Bundle, said plainly.** H against B measures *host Read permitted, plus the hook*. It does not separate
     the two; the mechanism counts (Decision 4) say whether the hook did the work. A Read-without-hook
     configuration is the rejected alternative that would separate them.
2. **Stage 7: three configurations, both models, ADR-162's pool and looks.** Profile `stage7` is `stage5` plus
   configuration H. Configurations A, B and H; Sonnet 5.5 and Opus 5.5; one repeat; the 191 instances in
   ADR-162's order and blocks; one mrw release, one Claude Code executable, as ADR-162 Decision 7.
3. **Three comparisons per model, each with ADR-156's verdict at 99% over five looks.**
   - **H against B** (the question): what permitting the host Read with the hook buys over mrw as Stage 5 ran it.
   - **B against A**: Stage 5's comparison again, on this release.
   - **H against A**: the claim a README could make.
   - Six verdicts in all, not a family held at 5%; a verdict that closes early overstates its effect and is
     published with its look (ADR-162 Decision 5). A configuration runs on while any comparison it is in is open.
4. **An uptake gate, read at look 1, and mechanism counts beside every verdict.**
   - **The gate.** Per model, H must call the host Read in at least half of its block-1 attempts. Below that,
     H's two comparisons close **UNTESTED**: the hook was not exercised, so they are neither PAYS nor LOSES,
     and H runs no more blocks for that model. B against A goes on.
   - **The counts,** per H attempt, by `scripts/bench_mechanism.py` at `3a970837`: host Reads, hook-licensed
     writes, overlapping mrw reads, refused unread writes, unknown calls; and attempts whose `hook.stderr` is
     not empty. Descriptive, beside the verdicts, never in their place.
5. **What each outcome means, written now.**
   - H against B **PAYS**: evidence for Zy to change rule 1 of mrw's guide; the change is its own record.
   - **LOSES**: rule 1 stays.
   - **INCONCLUSIVE**, including cheaper with solving's bound unmet as in Stage 5: published as that, with
     the cost interval, and rule 1 stays.
   - **UNTESTED**: the sentence did not move the agent; no claim about the hook follows.
6. **Prerequisite: a release.** No counted attempt runs before a tag `v1.83.0` or later contains `1f01dcdc`
   (ADR-191). Cutting it is Zy's call (the release skill).
7. **Price, said first.** Stage 5 was 1,146 attempts, about 38 hours, at a descriptive total of $158 in API
   prices ($49.73 + $38.20 Opus, $11.23 + $10.13 Sonnet for A and B over 191; H priced as B), on the
   subscription. An UNTESTED gate at look 1 saves H's remaining 302 attempts; any comparison closing early saves
   more. T3's uncounted pilot is 2 instances × 2 models × H, about 10 minutes.

What would make it FAIL:
- an H whose init tools are not `Bash, Glob, Read, Skill`, or whose hook is absent, or whose sentence's sha
  differs from the registered one;
- an A or B that carries the Read hook or the sentence;
- a verdict read against the wrong baseline, at another level, or at an ineligible look;
- an H comparison reported as PAYS or LOSES after the uptake gate failed;
- an instance chosen, skipped or reordered for an outcome; a profile, release or CLI that changes between blocks;
- a README sentence stating more than its verdict.

## Alternatives Considered

- **A fourth configuration N, B plus the Read with no hook.** Separates the hook from the Read. Rejected for
  now because it adds a third to the attempts (about 13 hours) to answer a question H against B only needs if
  it PAYS; it becomes the follow-up then.
- **Opus only.** Opus had the larger saving and the longer sessions, so the most re-reads to remove. Rejected
  because Stage 5 ran both and a Sonnet answer is what most adopters run; dropping it halves the price.
- **The `stage6` profile, checkouts that run their tests.** Closer to real work. Rejected because its surviving
  pool needs `bench baseline` over all 191 first (ADR-163 Decision 4) and has never been frozen.
- **Change mrw's guide first, then measure.** Rejected: the guide is the served path for every adopter; this
  stage decides whether to change it.
- **Run 10 on the grouped pilot.** Rejected (Zy, 2026-11-03): three-turn jobs leave no re-read to remove.

## Component / Boundary Impact

None — internal to `bench`: `internal/bench` (a configuration, a profile, a baseline per comparison) and
`cmd/bench` (the flags and usage). No mrw engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run --profile stage7 --configs A,B,H` | new profile and configuration | T1 | T3, T4 |
| `run.json` `h_sentence_sha256` | new field | T1 | T4, T5 |
| receipt `host_reads`, `hook_stderr_bytes` (H only) | new fields | T1 | T4, T5 |
| `bench look` / `bench analyze` comparison `H:B` | a comparison names its baseline | T2 | T4, T5 |
| `bench look` UNTESTED verdict for H | new verdict state | T2 | T4, T5 |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| configuration H and profile `stage7` | T1 | T2, T3, T4 | No — new names |
| receipt `host_reads` | T1 | T2 (the gate), T5 | No — added keys |
| comparison `H:B` and UNTESTED in `bench look` | T2 | T4, T5 | No — Stage 5 comparisons unchanged |
| the frozen sentence and its sha | T3 | T4 | No |

## Implementation

See `docs/adr/ADR-192/tasks/README.md`.

## Consequences

- **Positive:** the hook is measured where re-reads happen, on the same pool and bar as the published claim.
- **Positive:** B against A is replicated on the release that carries ADR-191, where Stage 5 ran v1.68.0.
- **Negative:** about 38 hours of a shared machine at most, on the subscription.
- **Negative:** H against B cannot say whether the Read alone or the hook did the work; the counts can only
  make it plausible.
- **Neutral:** solving stays likely INCONCLUSIVE at 191 (Stage 5's half-width was about 7 points); a cheaper
  H with solving unmet is reported as exactly that.

## Out of Scope

- Changing rule 1 of mrw's guide (`internal/guide/guide.go`) (deferred: docs/adr/BACKLOG.md "From ADR-192")
- Configuration N, Read without the hook (deferred: docs/adr/BACKLOG.md "From ADR-192")
- ADR-190's compacting measurement (deferred: docs/adr/BACKLOG.md "From ADR-190 and ADR-191")
- Checkouts that run their tests (permanent: boundary: ADR-163's pool is unfrozen; this stage keeps Stage 5's profile so B against A is comparable with Stage 5)
- Hosts other than Claude Code (permanent: boundary: `mrw hook read` reads Claude Code's PostToolUse payload, docs/agents.md §9)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The sentence still does not move the agent to Read | Med | High | T3's pilot measures uptake first; the gate closes UNTESTED at look 1 |
| A newer Claude Code changes the Read payload | Low | High | one pinned CLI for the stage; `hook.stderr` and the counts show a silent hook |
| Shared-machine load skews wall time | High | Low | wall time descriptive only, as Stage 5; each block starts below the core count |
| Disk for 191 images | Med | Med | ADR-162 Decision 9's prune and 25 GiB floor |

## Rollback

None — measurement only: `stage7` and H are opt-in names; removing them leaves Stages 1 to 6 as they were.

## Follow-ups

- [ ] If H against B PAYS on either model: a record for rule 1 of the guide, and configuration N.
