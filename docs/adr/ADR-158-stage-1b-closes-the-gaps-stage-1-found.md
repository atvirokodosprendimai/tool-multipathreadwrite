# ADR-158: Stage 1b closes the gaps Stage 1 found, and adds edits a global substitution gets wrong

**Status:** Accepted
**Accepted:** 2026-10-15 by Zy — picked "Accept, run on this Mac (Recommended)", after the adr-challenger's 13 findings were taken into the draft.
**Date:** 2026-10-15
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-054, ADR-133
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/**`, `docs/bench/**`
**Amends:** ADR-156 Decision 2's "only the word mrw is taken out" (a section a repository declares is taken out too, Decision 2 here) and ADR-156 Decision 5's "one verdict per model and per task source" (per stage as well, Decision 6 here)
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestAnAttemptReadsNoneOfTheHostsShellOrGitSetup`
**Served-path change:** None — this ADR changes only measurement: `bench`, its corpus and its registration.

## Context

ADR-156's Stage 1 ran on 2026-10-14 (120 attempts, Sonnet 5.5, receipts in `docs/bench/stage-1-sonnet/`) and its registered verdict is **INCONCLUSIVE**: cost (B−A)/A +0.026 [−0.039, +0.090], solved B−A +0.050 [−0.083, +0.167]. Reading the transcripts found five things that are the harness's or the corpus's, not mrw's (ADR-156 `tasks/README.md`); the transcripts stay on this Mac and are not published:

1. agentsmemory's `AGENTS.md` (commit d038def, the one every agentsmemory task pins) opens with "## Gate — verify the `am_*` tools before you do anything else" (lines 15–82), which sends a session without the tools to "## When the tools are absent" (lines 298–397: tell the user, ask six times, never work around the gate). The bench withholds every MCP server (ADR-156 Decision 2), and 11 attempts (A 8, B 3) obeyed and stopped after one turn with no tool call, graded unsolved at about $0.117 each.
2. `site-html-cost-faq`: the sentence the task changes is also in the page's JSON-LD (base line 1125); every attempt of both configurations changed both, and the oracle counted the second as a mis-edit.
3. `am-go-sha-message`: a test asserts the old message. B's automatic check (ADR-054) showed it red, and both B attempts fixed it and said so; the oracle counts that test file against B, while A's attempts left the suite red and were graded solved.
4. The attempt's Bash tool ran the user's own zsh: Claude Code snapshots `$SHELL`'s startup files, and `~/.zshrc` aliases `grep` to `grc` when `grc` is found. 19 attempts (A 10, B 9) met `command not found: grc`.
5. A backgrounded command's output went to the CLI's task directory, `/private/tmp/claude-501/<project>/<session>/tasks/`, shared with this machine's other Claude sessions; one attempt read its own output there through `/private/tmp/claude-501/*/*/tasks/<id>.output`, a path that reaches every session's.

By shape (Stage 1 receipts, measured 2026-10-15), B was cheaper only on repeated-anchor tasks (B/A cost 0.86, turns 3.3 against A's 5.9), even on single-line ones (1.00), and dearer on the rest (1.11–1.13). The eight multi-site tasks are all renames, which A finishes with one `sed` or `perl` over every file: mrw's batched plan has no turns to save there. Where the anchor repeats and only some occurrences change, a global substitution is wrong, and that is where B gained.

Exploratory runs on a second host (WSL Ubuntu 26.04, 2026-10-14, 74 attempts, receipts kept on that host under `~/bench-out/explore/` and not published, part of no verdict) varied only B's instructions: the full `mrw instructions` cost about what A did on small tasks; `--core` cost about 2× and none about 2.4×, through extra turns spent learning the format. So the instructions stay as registered.

Zy chose, on 2026-10-15, to fix these and run a Stage 1b ("(a) Fix + Stage 1b").

## Existing Primitives Audit

- `invocation` (`internal/bench/runner.go`) already builds each attempt's environment from nothing; the new variables join that list.
- `PrepareInstructions` (`internal/bench/task.go:303`) already rewrites instruction files in both configurations; a declared section drop runs after it on the same tree.
- `Run`'s pre-stage loop (`internal/bench/runner.go:169-177`) already refuses a stage whose tasks cannot be set up, "rather than recording every attempt as a setup exclusion"; the declared sections are checked there too.
- `bench/repos.json` already says where each repository lives; it gains the one per-repository fact this needs.
- `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` (`internal/bench/oracle_test.go:39`) already proves every task's oracle against its base, its reference fix and a stray edit; the new tasks get that proof by existing.
- `Analyze` (`internal/bench/analyze.go`) already prints one descriptive row beside a verdict; a second, for the new shape, follows the same rule.

## Decision

1. **The host's shell, git and Python setup do not reach an attempt.** The class: every file the CLI or a tool on the attempt's `PATH` reads at startup from the user's home. `HOME` itself stays — measured 2026-10-15, with a fresh `HOME` the CLI answers "Not logged in", because it finds the subscription's credentials through it. Enumerated with `ls -a ~` and by reading each file, 2026-10-15, against the tools on the attempt's `PATH` (bash, zsh, git, go, node, npm, npx, python3):
   - zsh: `.zshenv` (prepends `/opt/homebrew/bin`), `.zprofile`, `.zshrc` (aliases `grep` and others to `grc`) — none is read once the shell is bash;
   - bash: `.bash_profile` (appends `~/.docker/bin`; sources `.bashrc`, whose one block is inert unless Otty sets `OTTY_SHELL_INTEGRATION`); `.profile` is not read while `.bash_profile` exists. Measured 2026-10-15 through `bench run` on the pinned CLI (T1 S6): the attempt's `PATH` is bench's alone in both configurations, without `~/.docker/bin`, `grep` is `/usr/bin/grep`, git has no user, Python's user site is off, and the CLI task directory is under the attempt's scratch;
   - git: `.gitconfig` (identity and credential helpers) and `~/.config/git/ignore` (one pattern);
   - python3 (the host's 3.14.8): user site-packages enabled, with a `.pth` file there that runs at every interpreter start;
   - go: `~/Library/Application Support/go/env` is absent; npm: no `~/.npmrc`.
   Every attempt of both configurations runs with `CLAUDE_CODE_SHELL=/bin/bash` (measured: `grep` is then the CLI's own shell function, with no `grc`), `CLAUDE_CODE_TMPDIR=<attempt scratch>/cctmp` (measured: `claude-501/` appeared under it), `GIT_CONFIG_GLOBAL=/dev/null`, `XDG_CONFIG_HOME=<attempt scratch>/config` and `PYTHONNOUSERSITE=1`.
2. **A repository may declare sections of an instruction file that the bench drops, in both configurations.** `bench/repos.json` gains `drop_sections`, a list of `{file, heading}`; after `PrepareInstructions`, `Materialize` removes the heading line and every line up to the next heading of the same or a higher level, a `#` line inside a code fence not counting as a heading. A declared file that is not in the materialized set is skipped; one that is present with the heading found other than exactly once is an error naming the file and the heading. `Run` checks every declared section against each task's commit before the first attempt, so a missing heading stops the stage instead of becoming a setup exclusion per attempt. agentsmemory declares `AGENTS.md` "## Gate — verify the `am_*` tools before you do anything else" and "## When the tools are absent": together they ask for tools ADR-156 withholds on purpose and tell the session to stop without them, so obeying them measures the harness, not the tool. Nothing else in the file is dropped.
3. **A task that was graded wrongly is superseded: its instruction, oracle and reference fix are never edited** (ADR-156 T3 Out of Scope). `task.json` gains `superseded_by`, the only field an old task gains; a superseded task stays in `bench/tasks/` for Stage 1's receipts, and `bench run` without `--task` skips it.
   - `site-html-cost-faq` → `site-html-cost-faq-2`: the instruction names both places the sentence appears, the visible answer and the page's JSON-LD, and the oracle requires both.
   - `am-go-sha-message` → `am-go-sha-message-2`: the instruction is unchanged; the reference fix also updates the test that asserts the message, so that file is an allowed region, and the oracle requires the source change only. A configuration that does not run the tests is not penalised for leaving the test, and one that fixes it is not penalised for fixing it.
4. **Ten varied multi-site tasks join**, shape `varied-multi-site`, each from one of the four repositories at the commit its existing tasks pin: at least three changed sites in at least two files, and at least one changed site whose old text also occurs, unchanged by the reference fix, in the files it touches, so a global substitution of that text is a mis-edit. A test reads every such task's `reference.patch` and holds it to this.
5. **Stage 1b is registered before it runs**, in `docs/adr/BACKLOG.md` "Pre-registered for ADR-158", in this record's commit: ADR-156's bar unchanged; the corpus every measured task not superseded (40); Sonnet 5.5; 2 repeats; every attempt on this Mac, Stage 1's host; a pilot first, not counted, holding at least one agentsmemory task. Its verdict is its own and is never pooled with Stage 1's.
6. **A verdict is per stage as well as per model and task source.** `Result` gains its stage; `Analyze` groups and sorts by stage, model and source; `bench analyze` prints a stage column. ADR-156's T6 builds `docs/bench/RESULTS.md` from `bench analyze docs/bench`, which walks every stage's receipts, and without this Stage 1's and Stage 1b's would be pooled — found while drafting this record (2026-10-15). Publishing stays T6's. A stage whose receipts include `varied-multi-site` tasks gets one more descriptive row over those tasks alone, registered here before any of them runs.

## Alternatives Considered

- **A fresh `HOME` per attempt.** Rejected: measured 2026-10-15, the CLI then answers "Not logged in", and the subscription is the only auth Zy chose (ADR-156 Decision 2).
- **Strip agentsmemory's instruction files whole.** Rejected: Zy chose "Redact, don't strip" on 2026-10-14 because agentsmemory's tests read its `AGENTS.md`; the rest of the file is the repository's real working environment.
- **Tell the agent in every task instruction that memory tools are not needed.** Rejected: it changes 13 tasks' instructions and both configurations' prompts for a fact about one repository's file.
- **Reclassify Stage 1's refusals and mis-edits after the fact.** Rejected: the registered verdict stands as computed; this record fixes the next stage instead.
- **Shorter mrw instructions for B.** Rejected by measurement: the exploratory runs cost more with less text.
- **Add the isolation checks to the canary.** Not now: the canary checks what ADR-156 T2 registered; T1's S5 measures the new environment on the pinned CLI and is rerun for any other CLI version.

## Component / Boundary Impact

Within `internal/bench`, `cmd/bench` and `bench/`; no mrw package changes, and the engine go/no-go packages stay byte-identical.

## Wiring & Contract Changes

| Change | Where | Produced by | Consumed by |
|---|---|---|---|
| five attempt environment variables | `invocation` | T1 | every attempt |
| `drop_sections` in `bench/repos.json`; its pre-stage check | `Repo`, `Materialize`, `Run` | T1 | every attempt of a declaring repository |
| `superseded_by` in `task.json`; default selection skips it | `Task`, `cmd/bench` run | T2 | `bench run` |
| shape `varied-multi-site`; its descriptive row | `task.go`, `Analyze` | T2 | `bench analyze` |
| verdicts grouped and sorted by stage; a stage column | `Result`, `Analyze`, `cmd/bench` analyze | T2 | `bench analyze`; ADR-156 T6's `RESULTS.md` |

## Inter-task Contracts

T1 produces the isolated environment and the section drop; T2 produces the corpus, the selection and the per-stage verdicts; T3 consumes both to run Stage 1b.

## Implementation

Three tasks in `ADR-158-stage-1b-closes-the-gaps-stage-1-found/tasks/`, in order.

## Consequences

- **Positive:** a second, cleaner measurement, and the first one with tasks where a global substitution is wrong.
- **Negative:** 170 more subscription attempts (a 10-attempt pilot, then 40 × 2 × 2).
- **Neutral:** Stage 1's verdict and receipts are untouched; `bench analyze docs/bench/stage-1-sonnet` prints the same numbers with a stage column.

## Out of Scope

- Stage 2 on Opus and Stage 3 on SWE-bench (deferred: ADR-156 T4 and T5, each on Zy's word)
- The bash startup files themselves (permanent: boundary: measured, the attempt's `PATH` stays bench's and they define no alias or function; removing them needs a fresh `HOME`, which loses the login)
- The system git configuration and the shared Go module cache (permanent: boundary: they are the machine's, not the user's home setup, and the same for both configurations)
- The host's global git ignore in bench's own git runs, whose one pattern is `**/.claude/settings.local.json` (permanent: fact: preparation removes every `.claude/` directory before the base commit, so the pattern can hide only a file an attempt creates there; citation: file `internal/bench/task.go:303`)
- Running Stage 1b on a host other than this Mac (permanent: boundary: one host per stage, so a machine difference cannot enter a pairing)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `CLAUDE_CODE_SHELL` or `CLAUDE_CODE_TMPDIR` is renamed by a later CLI | Medium | Medium | the CLI is pinned per stage; T1's S5 probe is rerun for any other CLI version; a unit test pins the variables |
| `GIT_CONFIG_GLOBAL=/dev/null` makes a repository's declared check fail for want of a git identity | Low | High | T1's S6 re-measures the four declared checks in the prepared tree under the new environment |
| A new task's oracle is wrong in a way the base/fix/stray test cannot see | Medium | Medium | the pilot reads every transcript; a task is superseded after the pilot only for an oracle defect shown on its base or its reference fix, never for an attempt's result |
| Still INCONCLUSIVE | High | Low | a registered, publishable outcome |

## Rollback

Revert the commits; `bench/tasks/` keeps every task, so Stage 1's receipts stay gradeable.

## Follow-ups

- ADR-156 T6 builds `docs/bench/RESULTS.md` and the README sentences from both stages' receipts (deferred to it, on Zy's word after Stage 1b).
