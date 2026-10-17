# ADR-162: Stage 5 runs the multi-hunk SWE-bench Verified pool until each verdict is conclusive, with the quality-harness plugin as a third configuration

**Status:** Accepted
**Accepted:** 2026-10-17 by Zy — picked "Accept (Recommended)" after two challenger reviews (adr-challenger, Codex xhigh) and a Codex xhigh re-check, every finding that held taken into the draft.
**Date:** 2026-10-17
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-158, ADR-159, ADR-160, ADR-161
**Amends:** ADR-156 Decision 1 (two configurations) and Decision 2 (the invocation, `--disable-slash-commands` included), and the registered bar's 95% interval level and its percentile bootstrap for the solved difference, for Stage 5 only
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestConfigurationPIsAPlusThePluginAndSkills`
**Invalidates:** none — checked: Stages 1 to 4b keep their verdicts, since `bench analyze` keys a verdict by stage, model and source, and their level stays 95%; `SelectSWEBench`'s default salt stays `adr-156:` and ADR-160's `adr-160:`, so neither earlier selection moves; configurations A and B are unchanged in every stage that does not set the Stage 5 profile
**Served-path change:** None — this ADR changes only measurement: `bench`, its registration and its stage.

## Context

Every verdict ADR-156's bar has produced is INCONCLUSIVE except Stage 4 on mrw v1.66.0, which LOSES (`docs/bench/RESULTS.md`, #453). Zy, 2026-10-17: "draft the plugin arm ADR, we need conclusive results, go for them".

**What the receipts say about reaching a verdict.** Measured 2026-10-17 from the published receipts. For each task: B's mean cost relative to A's, and B's solved rate minus A's. SD is their spread across tasks.

| Stage, population, model | tasks | cost (B−A)/A, SD | solved B−A, SD | tasks with A ≠ B solved |
|---|---|---|---|---|
| Stage 3, Verified with no hunk floor, Sonnet | 20 | −0.061, 0.256 | +0.000, 0.324 | 10% |
| Stage 3, Verified with no hunk floor, Opus | 20 | −0.053, 0.445 | +0.000, 0.000 | 0% |
| stage-4b-swe, Verified with 5+ hunks, Sonnet | 20 | −0.170, 0.222 | −0.150, 0.366 | 15% |
| stage-4b-swe, Verified with 5+ hunks, Opus | 20 | −0.209, 0.278 | −0.050, 0.394 | 15% |
| stage4b own tasks, Sonnet | 24 | −0.021, 0.104 | −0.083, 0.282 | 21% |
| stage4b own tasks, Opus | 24 | −0.016, 0.077 | −0.021, 0.179 | 12% |

- **PAYS asks three things.** The mean cost must be −10% or lower, the cost interval wholly below 0, and the solved interval's lower bound at −5 points or above. The first is a point condition: on Stage 3's population (about −6%) no sample size reaches it. On the 5+ hunk population (−17% to −21%) it holds. The 2+ hunk population this stage draws from is unmeasured and lies between.
- **The solved interval is what needs the sample.** For one endpoint, a normal approximation at the 99% this stage's looks read, with 80% power, needs about (3.42 × SD / |d|)² paired instances to separate a true difference d from 0:
  - about 70 at Sonnet's 5+ hunk d = −0.15 and SD 0.366;
  - at d = −0.05, about 626 at SD 0.366 and 725 at SD 0.394.

  These are single-endpoint planning figures from other populations. They are not the power of the whole stopped rule.
- **The pool.** SWE-bench Verified at the revision ADR-156 T5 recorded has 500 instances. Of these, 220 have 2 or more `^@@ -` hunk headers. Leaving out the 40 that Stages 3 and 4b selected leaves 191, of which 31 have 5 or more. Three repositories in Verified have no clone in `bench/repos.json`: psf/requests, mwaskom/seaborn and pallets/flask.

**Quality-harness ADR-106 asks this runner for a plugin arm.** It measures what the quality-harness plugin buys with a paired benchmark that hidden tests grade (`/Users/zy/CursorProjects/quality-harness/docs/adr/ADR-106-a-paired-benchmark-says-what-the-plugin-buys.md`, Accepted). It names this project's SWE-bench runner and says a plugin arm here "would be a change in that project, for its owner to decide". ADR-106 requires, of each arm:
- a fresh session with no other plugins, identical except for the plugin;
- the plugin shown engaged;
- a final `STATUS:` line;
- resolved, broke-tests, limit and unrun per run;
- a configuration line whose fields its Decision 2 lists.

Its Decision 4 grades a run that hit a limit by its diff, unlike ADR-156, which counts it unsolved.

**Zy's decisions, 2026-10-17:**
- "One run, A/B/P": A is the control for mrw (B) and for the plugin (P).
- "Sonnet and Opus".
- "Looks every 100": a sequential stage with pre-registered looks.
- "Verified 2+ hunks (191)": chosen after the review showed that mrw's cost saving on the no-floor population cannot reach PAYS. With 191 instances, the looks come every 40.
- "Prune other projects' unused images".
- "Amend it before Stage 5": ADR-106 is amended to take this stage's rows, and its pre-registration committed, before Stage 5's first counted attempt.

**Measured 2026-10-17 on this Mac, for the price and the plumbing:**
- **Time.** Stage 4b's slice ran 80 attempts in 157 minutes, image pulls and grading included: about 2 minutes per attempt, on 5+ hunk instances with 4 attempts per image. Stage 3's agent time averaged 35 s against 4b's 76 s. T4 measures it again on this stage's configurations.
- **Disk.** 13 GiB is free on the volume that holds Docker's disk image, and one SWE image is about 10.6 GB. `docker system df` reports images 31.1 GB (18.3 GB reclaimable), volumes 45.7 GB and build cache 9.0 GB. `Docker.raw` holds 83 GiB on disk (`du`), so trimming it would recover little.
- **The CLI.** Claude Code 2.1.302 loads a plugin for one session with `--plugin-dir PATH`. `--disable-slash-commands` "Disable[s] all skills", and bench passes it to every attempt today (`internal/bench/runner.go:665`). `--include-hook-events` puts hook events in the stream-json output.
- **What the bench's config directory holds.** `~/.local/share/mrw-bench/claude-config` has account-synced skills under `skills/synced/` (pdf, docx, xlsx, pptx, skill-creator and others) and account-synced plugins under `plugins/synced/` (figma, cowork-plugin-management). They are hidden today only because skills are disabled.
- **The plugin.** Quality-harness's latest tag is v3.8.39; its HEAD 7f4d60a6 says 3.8.40 and is untagged. Its hooks run `node`, which on this Mac is an nvm shell function and so on no PATH bench builds. Its hooks also call `git`, and `gh` with the user's HOME. Its review skills launch Codex through `plugin/scripts/host-review.mjs`, which reads `CODEX_CLI_PATH` first and then the ChatGPT-bundled binary by absolute path, so PATH isolation alone does not keep that external model out.
- **The canary.** bench's canary refuses `quality-harness:`, "SessionStart hook" and "UserPromptSubmit hook" in any transcript (`internal/bench/canary.go:18`), and the name mrw in any configuration without mrw (`canary.go:49`). It does not read the init line's skills or plugins.
- **The runner.** `bench run` refuses a second plan in a directory whose `run.json` differs (`runner.go:408-433`). It runs one list of configurations for every model. It sets `solved` false on an attempt over the budget before the receipt is written (`runner.go:580`). `bench swebench-select` appends the colon to `--salt` itself (`cmd/bench/main.go:309`).

## Existing Primitives Audit

- **Reused unchanged:**
  - the runner's attempt loop, retries and resume (ADR-156 Decision 6), receipts, exclusions and the budget (20 minutes or 60 turns);
  - the SWE-bench path: one image per instance, grading with swebench 5.0.2, then removal (ADR-156 T5);
  - the seeded attempt order, and configuration B's mrw, instructions and hook.
- **Reshaped:**
  - `Config` gains P.
  - `invocation` reads a stage profile (Decision 1) that sets skills, `Skill`, `node` and the Codex stub for every configuration of every block, whatever configurations a block runs.
  - `CheckCanary` judges the init line's skills and plugins, and judges each marker per configuration.
  - The receipt keeps the harness's own `resolved` beside ADR-156's budget-adjusted `solved`.
  - `grade` reads PASS_TO_PASS failures from the report it already opens.
  - The plan takes configurations per model, and one block is one directory.
  - `bench verify` checks a block's per-model configurations.
  - `bench analyze`:
    - reads the interval level and looks from the stage's `run.json`;
    - computes P against A by the same rule as B against A;
    - counts P's silent mis-edits as P's, where `analyze.go:241-244` would count them as B's.
  - `bench analyze --selection` repeats, so earlier stages keep their descriptive rows.
  - `bench swebench-select --exclude` repeats.
- **New:**
  - `bench export-paired`, which writes ADR-106's rows;
  - `bench look`, the stop rule's answer;
  - `bench run --block K`;
  - the Codex stub;
  - the clones of the repositories the pool needs.

## Decision

1. **Configuration P is A plus the quality-harness plugin, inside a Stage 5 profile shared by all three.**
   - **What P is.** P has A's tools, PATH and prompt, and loads the plugin with `--plugin-dir` from a detached worktree of quality-harness at its latest tag when T4 runs. The receipt and `run.json` carry `plugin: {tag, commit}`, and a dirty or untagged worktree is refused.
   - **The profile.** Stage 5's profile is written in `run.json` and set for every block, whichever configurations a block still runs. Under it every configuration:
     - runs with skills on (no `--disable-slash-commands`);
     - is offered `Skill` beside its own tools;
     - finds the same real `node` (`--node PATH`) on PATH;
     - has `CODEX_CLI_PATH` pointing at a stub that exits 2 and says external models are not part of this stage, with no `codex` on PATH.
   - **No external models, no subagents.** External model calls are prohibited, and the stub is how. `Agent`, `Task`, `WebFetch` and `WebSearch` stay offered to none (ADR-156). A plugin skill that spawns a subagent or a Codex review therefore runs without one, and that limitation is printed beside P's rows.
   - **No synced skills or plugins.** The account-synced skills and plugins in the config directory must load in no configuration. T1 finds the setting or layout that keeps them out, and stops back to Zy if there is none.
   - **The plugin's own effect.** Its hooks running `git` and `gh`, and their time counted against the budget, are part of what the plugin does, so they are measured, not removed.
2. **The canary proves the profile and the plugin's engagement per configuration.**
   - **The skills each init line lists.** A's and B's must equal a named set of CLI built-ins, recorded from T1's uncounted probe. P's must be that set plus skills under `quality-harness:` and nothing else. The init line must list no plugin in A and B, and only quality-harness in P.
   - **Hooks.** P must show at least one hook event whose source is the plugin; A and B none.
   - **The markers.** `quality-harness:`, "SessionStart hook" and "UserPromptSubmit hook" are refused in A and B as today. In P, `quality-harness:` is allowed in three places: the init line's validated plugin skills, the agent's `Skill` calls to them, and the plugin's own hook events. The two hook markers are allowed only inside the plugin's hook events. Any other plugin, or any hook source but the plugin, is refused in P as in A.
   - **mrw.** The mrw-name check applies to P as to A, over the text the agent wrote and the tools' results. Text the plugin injects is excluded only if T1 shows that the plugin itself names mrw.
   - A failed canary starts nothing.
3. **Every attempt is asked for a structured claim, and the receipt keeps what the harness said.**
   - **The claim.** In this stage the instruction ends with one fixed sentence asking for a final line `STATUS: done` or `STATUS: not done: <reason>`. It is the same in A, B and P, and its SHA-256 is in `run.json`. The receipt's `status` is `done`, `not done` or `unreadable` (neither line, both, or another value).
   - **What the harness said.** `oracle.harness_resolved` is the harness's own verdict, kept on an attempt over the budget. `oracle.solved` stays ADR-156's budget-adjusted outcome. `oracle.broke_tests` is true when any PASS_TO_PASS test fails.
   - **The rows for ADR-106.** `bench export-paired --pair A,P --model M DIR...` writes ADR-106's rows for one model:
     - a `{"config": …}` line with every field ADR-106 Decision 2 names. That includes the git blob sha of ADR-106's scorer, `scripts/eval-paired.mjs`, taken from its pre-registration (`--scorer-sha`); the exporter's own sha goes beside it, not in its place;
     - then one row per instance, arm and repeat, `{instance, arm, repeat, status, resolved, brokeTests, limit, unrun}`;
     - `resolved` from `harness_resolved`, read from the last try. An attempt is `unrun` only if every try was excluded.
4. **The pool and its order.**
   - **The pool.** SWE-bench Verified at revision `78f471bf655a3137b2e8a75af1501690ec009ec3`: every instance whose gold patch has 2 or more hunk headers, except the 40 that `bench/swe-bench/selection.json` and `selection-adr-160.json` selected. That is 191, ordered by `sha256("adr-162:" + id)`. The CLI's `--salt adr-162` appends the colon, and T3's fence regenerates the file and diffs it.
   - **Repositories.** T3 clones every repository the 191 need and adds it to `bench/repos.json`.
   - **Blocks.** Blocks 1 to 4 are instances 40(K−1)+1 to 40K, and block 5 is instances 161 to 191.
   - **Descriptive row,** beside the verdict and never in its place: the instances with 5 or more hunks.
5. **Five looks, registered now, read per comparison at a nominal 99%.**
   - **The interval.** A look follows each block. At each look, each model and comparison (B against A, P against A) gets ADR-156's verdict at 99% in place of 95%; 99% is 1 − 0.05/5, a Bonferroni split over the five looks.
     - **Cost** keeps ADR-156's percentile bootstrap.
     - **Solved** is read with Tango's score interval for a paired difference of proportions. It does not collapse when no instance is discordant: at 40 paired instances with none discordant it is far wider than 5 points, so PAYS cannot close on an empty difference.
   - **Not exact.** The allocation is nominal: each comparison's error stays near 5% only as far as both intervals cover at 99%. A review showed what the bootstrap alone allows at these look sizes. A true 5-point solved loss that happens to show no discordant instance in the first 40 reads [0, 0] and closes PAYS, with probability 12.9%. The score interval is what removes that case. No other case has been measured.
   - **Not a family.** The four verdicts (two models × two comparisons) are four claims, not one family held at 5%.
   - **Early stops overstate.** A verdict that closes early overstates its effect, and is published with the look it closed at.
   - **A look must be eligible.** For a model, a look is eligible only when:
     - every attempt scheduled in the blocks run so far has a receipt;
     - exclusions are at most 5% per configuration;
     - at least 95% of the instances run are paired for the comparison.

     An ineligible look closes nothing, and the next block runs.
   - **Closing and stopping.** A comparison closes at the first eligible look where it is PAYS or LOSES, and later attempts of its second configuration are not run. A model whose two comparisons are closed runs no more blocks. The stage ends after the fifth look or when no model runs on. An open comparison then is INCONCLUSIVE.
   - **Infrastructure.** Three consecutive start-error exclusions stop the run, exit 2 and resumable, rather than consuming retries.
   - **Fixed now.** The looks, their level, the block edges and the eligibility rule. Nothing else is chosen after a look.
6. **The bar is otherwise ADR-156's,** one verdict per model, comparison and stage: pairing per instance, outcomes, the PAYS / LOSES / INCONCLUSIVE rule with LOSES winning an overlap, admission and exclusions, the budget, and B's counted shell reads and slipped writes. P against A uses the same rule with P in B's place. Silent mis-edits are not measured on SWE-bench.
   - **The rows.** Stage 5's rows print in their own table, whose header names 99% and the look each verdict closed at; the source column says the comparison.
   - **Beside each verdict, never in its place:** each configuration's false completions (`STATUS: done` with `harness_resolved` false), its under-claims, its broke-tests count and its unreadable claims.
7. **One repeat per configuration, both models, and the run's shape.**
   - **One stage.** Sonnet 5.5 and Opus 5.5 run in one stage, `stage5`, on one mrw release, one plugin tag and one pinned CLI executable. A block whose CLI version differs from block 1's is refused, and an upgrade starts a new stage.
   - **A block.** Each block is one directory, `docs/bench/stage-5/block-K/`, with its own `run.json` and per-model configurations as `bench look` printed. Each instance's attempts run together with its image pulled once, in the seeded order bench already uses. `bench look` and `bench analyze` read across the blocks.
8. **Contamination has a rule before any counted attempt.**
   - **What is scanned.** The dataset file, the probe venv and any Hugging Face cache lie outside every attempt's tree, but the user's account can read them. After each block, every transcript is scanned for:
     - the dataset file's name and directory;
     - `swe-probe`, `huggingface` and the selection files' names;
     - the instance's FAIL_TO_PASS test names, except those its own problem statement already gives the agent.
   - **A hit.** It stops the stage before the next look and goes to Zy. No attempt is replaced, dropped or kept at the executor's discretion.
   - **What the scan misses.** It cannot see a read whose path never appears in the transcript.
9. **The machine.**
   - **Room on the disk.** Before block 1, `docker image prune -a` removes the images no container uses (Zy: "Prune other projects' unused images"; 18.3 GB reclaimable when measured); volumes and containers stay.
   - **A floor.** Before each image pull, bench checks free space on the volume that holds Docker's disk image. Below 25 GiB it stops at the instance boundary, exit 2, resumable, and names the shortfall. If the prune leaves less than 25 GiB, the stage waits for Zy.
   - **Load.** Each block starts only when the load average is below the core count (costly-runs).
10. **Quality-harness ADR-106 is amended first.** Before Stage 5's first counted attempt, ADR-106 is amended in the quality-harness repository, through its own PR and review, to take this stage's rows: k = 1, this pool, this stop rule. Its pre-registration is committed there. Its scorer's verdict on Stage 5 is then confirmatory; bench's stop rule still reads only Decision 5's verdicts.
11. **Price, said first.**
    - **Size.** Each instance is 6 attempts (2 models × 3 configurations), about 12 minutes at the measured 2 minutes per attempt. A block of 40 is about 8 hours, and all 191 about 38 hours, 1,146 attempts on the subscription. A model that leaves early halves the rest.
    - **B against A.** It can close LOSES if B's solved rate really drops, as Sonnet's did by 15 points on 5+ hunks. At 191, the 99% solved half-width is about 7 points, so PAYS needs B's solved rate at or above A's and a mean cost saving of 10% or more, which this population has not shown.
    - **P against A.** Unmeasured.
    - **INCONCLUSIVE stays likely for some comparisons,** and is published as plainly as PAYS or LOSES.

What would make it FAIL:
- a P whose plugin is not engaged;
- an A or B that carries it, or any configuration carrying a synced skill or plugin;
- a configuration that reaches an external model;
- a verdict read at a look other than the five, at another interval, or at an ineligible look;
- an attempt run past a closed comparison and folded into its verdict;
- an instance chosen, skipped or reordered for an outcome;
- a profile that changes between blocks;
- a README sentence stating more than its verdict.

## Alternatives Considered

- **A fixed n with one analysis at 95%.** Rejected by Zy ("Looks every 100"). At the fifth look, 99% intervals need about 1.49 times the n a fixed-n 95% analysis needs; that is the price of stopping early when a comparison closes.
- **An O'Brien–Fleming boundary instead of Bonferroni.** Rejected: it spends less early, but its boundaries depend on each look's share of the final information, which exclusions and closed comparisons move. Bonferroni needs no such input.
- **All 460 Verified instances.** Offered, and rejected by Zy for the 2+ hunk pool: on the no-floor population mrw's mean cost saving (about 6%) cannot reach PAYS's −10% at any n.
- **A harder dataset (SWE-bench Pro, SWE-rebench).** Offered and not taken now: new images and harness work first. It is deferred.
- **The plugin arm as ADR-106 first planned** (its own pilot of 30 × 3 × 2, then a main run). Rejected by Zy ("One run, A/B/P"). Sharing A saves a quarter of the attempts: 3 per instance and model instead of 4.
- **Sonnet only.** Rejected by Zy ("Sonnet and Opus").
- **Keep `--disable-slash-commands` in A and B, and drop it only for P.** Rejected: P would then differ from A by the built-in skills as well as by the plugin.
- **Offer `Agent` and allow Codex in all three configurations.** Rejected: ADR-156 keeps subagents out because their cost and work are invisible to the receipt, and a Codex call's cost is outside `total_cost_usd` altogether.
- **Prune only bench's own Docker data.** Zy's first choice, given up when the measurement showed it frees too little; replaced by Zy's "Prune other projects' unused images".

## Component / Boundary Impact

None: the changes are internal to `bench` (`internal/bench`, `cmd/bench`). They add:
- one configuration and the Stage 5 profile;
- the subcommands `export-paired` and `look`, and `run --block`;
- receipt fields: `status`, `oracle.broke_tests`, `oracle.harness_resolved` and `plugin`;
- per-model configurations in a plan;
- the sequential reading in `bench analyze`.

The plugin is read from another repository's worktree and never changed. ADR-106's amendment and its scorer stay in that repository.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run --profile stage5 --configs …,P --plugin-dir DIR --node PATH --codex-stub` | configuration P; the Stage 5 profile in `run.json`, applied to every configuration and block | T1 | T4, T5 |
| `bench canary` | judges skills, plugins, hook sources and markers per configuration | T1 | T4, T5 |
| receipt and `run.json` `plugin: {tag, commit}` | new | T1 | T2, T6 |
| `bench run --status-line` | the registered `STATUS:` sentence; its sha in `run.json` | T2 | T4, T5 |
| receipt `status`, `oracle.broke_tests`, `oracle.harness_resolved` | new fields | T2 | T3, T6 |
| `bench export-paired --pair A,P --model M DIR...` | new: ADR-106's rows for one model | T2 | quality-harness ADR-106's scorer |
| `bench swebench-select --exclude FILE` | repeatable | T3 | T3's selection |
| `bench run --block K --configs-for MODEL=LIST` | new: one block, configurations per model | T3 | T5 |
| `bench look DIR` | new: per model and comparison, the verdict at the stage's level, the look it closed at, eligibility, next block's configurations; exit 0 when the stage is complete | T3 | T5, T6 |
| `bench analyze --selection FILE` | repeatable; level and looks read from each stage's `run.json`; Stage 5's own table | T3 | T6 |
| `bench/swe-bench/selection-adr-162.json`, `bench/repos.json` | new selection; the repositories it needs | T3 | T5 |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| configuration P, the Stage 5 profile, the canary, `plugin` | T1 | T2, T4, T5 | No — additive; A and B unchanged without the profile |
| `--status-line`, `status`, `broke_tests`, `harness_resolved`, `export-paired` | T2 | T3, T4, T5, T6 | No — additive |
| the selection, `--block`, `--configs-for`, `bench look`, `analyze`'s levels and table | T3 | T4, T5, T6 | No — additive; earlier stages read at 95% |
| the pilot's sign-off: P engaged, the built-in skill set, time per attempt, disk | T4 | T5 | No |
| `docs/bench/stage-5/block-K/` receipts and `bench look` per look | T5 | T6 | No |

## Implementation

See `tasks/README.md`:
- T1: configuration P and the profile;
- T2: the claim line, the harness's own result and the paired rows;
- T3: the pool, the blocks and the looks;
- T4: the harness pilot and the disk;
- T5: ADR-106's amendment, then the stage, block by block;
- T6: the verdicts in RESULTS.md and the README, and the rows handed to quality-harness.

## Consequences

- **Positive:**
  - a stage sized and stopped so that each comparison can close, on the population where mrw's mechanism should matter;
  - the first measurement of the quality-harness plugin against a control on hidden tests, confirmatory on both records;
  - A shared as the control.
- **Negative:**
  - up to about 38 machine hours and 1,146 attempts on the subscription;
  - up to 191 image pulls of about 10 GB, one at a time;
  - other projects' unused Docker images removed, rebuilt by them when next used;
  - Stage 5's A and B run with skills on, so its rows are not comparable row for row with Stages 3 to 4b, and each stage keeps its own verdict.
- **Neutral:** some comparisons may end INCONCLUSIVE at 191, and are published as computed.

## Out of Scope

- A plugin arm with `Agent` or external models allowed (deferred: docs/adr/BACKLOG.md — "From ADR-162", after this stage's P against A is read)
- More own tasks (permanent: boundary: Decision 4 draws from SWE-bench because its pool is the larger; the own corpus stays ADR-160's)
- Other datasets: SWE-rebench, SWE-bench Pro (deferred: docs/adr/BACKLOG.md — "From ADR-162", if a comparison stays INCONCLUSIVE at 191)
- Other hosts and mrw over MCP (deferred: docs/adr/BACKLOG.md — "Pre-registered for ADR-156", not measured now and recorded so it is not forgotten)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| P's hooks or skills do not engage headless | Med | High | Decision 2's canary starts nothing; T1's probe and T4's pilot read P's transcripts whole |
| Account-synced skills or plugins load once skills are on | High | High | Decision 1: T1 keeps them out or stops; Decision 2's canary refuses any unnamed skill or plugin |
| The plugin reaches an external model | Med | High | the Codex stub; the transcript scan counts attempts that tried |
| The subscription rate-limits a multi-day run | Med | Med | three start-errors in a row stop the run, resumable; ineligible looks close nothing |
| Disk runs out mid-run | Med | High | the prune before block 1; the 25 GiB floor before every pull |
| A model's comparison never closes | High | Med | INCONCLUSIVE after the fifth look, said in Decision 11 before any run |
| Contention with peers distorts wall time | High if run beside builds | Low | cost and solved are the bar; wall time is descriptive; costly-runs before each block |

## Rollback

None: this is measurement only. The configuration, profile, flags and subcommands can be removed. The stage's receipts, selection and clones are data. Nothing on mrw's served path changes. The images that are pruned are rebuilt by their projects.

## Follow-ups

None yet.
