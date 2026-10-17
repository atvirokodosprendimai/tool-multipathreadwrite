# bench — does mrw pay for itself on agent runs (ADR-156)

`bench` runs an agent CLI over a fixed corpus of editing tasks, once in each of
two configurations, and grades what each run leaves behind. It calls no model
itself: it starts `claude -p` as a subprocess, reads the stream-json it prints,
and judges the tree with an oracle that knows the answer by construction.

**Zy chose, on 2026-10-14, to run attempts under their own account with
`bypassPermissions`** — over a dedicated macOS user and over Docker — and logged
in once in the dedicated config directory (ADR-156 Decision 2). `bench run` and
`bench canary` still refuse without `--containment bypass` and `--config-dir`,
and `bench run` refuses without a passing canary for the same CLI version.

## Configurations

| | Tools | mrw | Hook |
|---|---|---|---|
| A | Bash, Read, Edit, Write, Glob, Grep | not on PATH | none |
| B | Bash, Glob | a pinned build on PATH, `mrw instructions` appended | `bench hook --tree`: refuses writes by other means, command substitutions included, except a redirection or `tee` to an absolute path outside the attempt's tree (Zy, 2026-10-14, after the pilot); counts reads, not refuses them (Zy, 2026-10-14), and `bench analyze` reports the count beside each verdict |
| P | A's | not on PATH | the quality-harness plugin's own, loaded with `--plugin-dir` at a tagged checkout; runs only under the stage5 profile (ADR-162) |

**The stage5 profile (ADR-162 Decision 1)** belongs to the stage, not to a configuration: under `--profile stage5` every configuration of every block runs with skills on (no `--disable-slash-commands`; `--include-hook-events` in its place), is offered `Skill` beside its own tools, finds the same real `node` on PATH (`--node`, else the one PATH finds), and has `CODEX_CLI_PATH` at a stub that refuses, so no plugin skill reaches an external model. Its settings switch off the config directory's account-synced skills (`syncClaudeAiSkills`) and plugins (each by name in `enabledPlugins`, read from `plugins/synced/*/*/.claude-plugin/plugin.json`; `syncClaudeAiPlugins` is not used, since it moves them into the config directory's trash), the bundled skills (`disableBundledSkills`) and the three built-in plugins. `Agent`, `Task`, `WebFetch` and `WebSearch` stay offered to none, so a plugin skill that would spawn a subagent or a Codex review runs without one: that is P's limitation, and it is said beside P's rows. P's `run.json` and receipts carry `plugin: {tag, commit}`; a plugin checkout that is untagged or has uncommitted changes is refused.

Under the profile the canary (`CheckProfileCanary`) also reads the init line and the hook events, in the shapes Claude Code 2.1.302 printed on 2026-10-17: A's and B's skills are exactly the built-in `design` and `doctor`, with no plugin; P's are those and the plugin's own `quality-harness:*`, with that plugin alone. The hook events carry no source, so hooks are judged by configuration: A fires none; B only bench's own `PreToolUse:Bash`; P at least one of `SessionStart`, `UserPromptSubmit` or `Stop`, the plugin being the only hook source P has. In P the plugin's own text — hook output, its skills and agents in init, the `Skill` calls to its skills, their results and the skill text injected from the plugin's directory — is taken out before the mrw and marker checks, since the plugin itself names mrw and the markers; the agent's own text is read as A's.

Every configuration runs with `--strict-mcp-config` and an empty MCP config, a settings file
written per attempt, `--disable-slash-commands` (outside the stage5 profile), `--no-session-persistence`,
`--no-chrome` and `--permission-mode bypassPermissions`, in a fresh tree whose
`CLAUDE.md`, `CLAUDE.local.md`, `AGENTS.md` and `.cursorrules` keep every line
but those naming mrw (the letters in any case, with no letter or digit on either
side), the same in both configurations, and whose `.claude/`, `.cursor/` and
`.codex/`, and `.mcp.json` at the root, are removed, since a project's settings,
hooks and skills would load inside an attempt (a `.mcp.json` below the root is
kept: a plugin may ship one, and its tests read it). A file in the tree that a kept
instruction file imports with `@path` is redacted too, and an instruction file that
is a link into the tree stays a link with its target redacted. Until the pilot of 2026-10-14 the
instruction files were stripped whole; a repository whose tests read its
`AGENTS.md` then failed configuration B's automatic check for that reason alone.
Every attempt runs with `CLAUDE_CONFIG_DIR` set to the dedicated directory
(resolved, and refused if it
is `~/.claude` by any spelling or link). The CLI runs in a process group of its
own, stopped when it returns. Its `PATH` holds links to the host's `git`, `go`,
`gofmt`, `node`, `npm`, `npx` and `python3` (quality-harness's check runs node)
and the system directories, not the directories those tools live in, and
configuration A's is refused if `mrw` is anywhere on it. `USER` is passed
through, since the CLI finds the subscription's credentials in the macOS
keychain by it. The canary takes the config directory's own path out of a
transcript before it looks for `mrw`: the CLI prints that path in its init line,
and it is bench's name for a directory, not context the model was given.

Since ADR-158 an attempt reads none of the host's zsh, git or Python setup and
shares no directory with the host's own Claude sessions: `CLAUDE_CODE_SHELL` is
`/bin/bash` (the CLI otherwise takes the user's login shell, and Stage 1 met a
zsh alias of the host's in 19 attempts), `CLAUDE_CODE_TMPDIR` and
`XDG_CONFIG_HOME` are directories of the attempt's own scratch (`cctmp/`,
`config/`), `GIT_CONFIG_GLOBAL` is `/dev/null`, and `PYTHONNOUSERSITE` is `1`.
`HOME` stays, since the CLI finds the login through it, so bash still reads its
startup files: measured on this Mac (2026-10-15) they leave the attempt's
`PATH` bench's own and define no alias or function (ADR-158 Out of Scope).

## The corpus: `bench/tasks/<id>/`

- `task.json`:
  - `id` — the directory's name;
  - `shape` — `multi-site`, `structural`, `repeated-anchor`, `large-file`, `single-line`, `varied-multi-site`, or `fixture` (never measured). A `varied-multi-site` task changes three or more places in two or more files, and at least one text it removes also occurs, unchanged by the fix, in a file the fix touches, so one global substitution gets it wrong (ADR-158);
    ADR-160 adds `wide-rename` (a Go identifier declared once, renamed with its comments to a name of the same length across 15 or more places in 8 or more files) and `check-coupled` (one string literal's text changed in a source file and the test that asserts it), each drawn by `bench mine` in the order of `sha256("adr-160:" + key)`;
    Stage 4's own tasks — the ten `wide-rename`, the eight `check-coupled` and the six `single-line` controls — are listed in `bench/stage-4-tasks.txt`, which `bench run` takes as `--task` ids; how they were drawn, with each command and its output, is in `docs/bench/stage-4-draw/`, and its SWE-bench slice is `bench/swe-bench/selection-adr-160.json`;
  - `repo` — a name in `bench/repos.json`, or `fixture`;
  - `commit` — the full sha the base tree is taken from;
  - `instruction` — what the agent is told; it names no tool;
  - `slack` — how many lines either side of the fix's own insertion point new lines may also go (default 0); it never licenses changing a line the fix leaves alone;
  - `exact` — each touched file must equal the reference result, so a call site commented out or kept in a block comment fails though its tokens survive (the renames);
  - `test_package`, `asserting_tests` — on a `check-coupled` task only, together, never with `exact`: the directory whose `go test` grades the attempt, and the top-level tests in it that fail when only the source literal is changed. The attempt is solved when the `require` and `forbid` hold on the source file's code (its comments blanked, so a comment quoting the value is not the change), nothing outside that directory changed and no file in it was deleted, and `go test -json -count=1 ./<dir>` passes within 10 minutes with every asserting test run and passed, not skipped — with `GOWORK=off` and `GOFLAGS` cut to the `-mod` flags of its effective value (split by go's quoting, in order), so no host setting chooses which tests run while the go env file's proxy and private-module settings still apply; a `GOFLAGS` the oracle cannot read is a grading error (Zy, "Grade by the tests", after T3's pilot found answers `exact` refused that the instruction allows; ADR-160 T6 and the reviews of #434);
  - `check_red_at_base` — why the repository's own declared check fails at the pinned commit in the tree bench builds, so configuration B's automatic check is red whatever the agent does: a measured fact, carried by the quality-harness tasks (the seven of Stage 1, measured 2026-10-14, and the three ADR-158 added at the same commit) and absent elsewhere;
  - `superseded_by` — the task that replaced this one after a run found its grading wrong. A superseded task keeps its instruction, oracle and fix, is not run unless `--task` names it, and is not counted in the shape counts (`site-html-cost-faq` and `am-go-sha-message`, ADR-158; `cac-rename-post-send`, whose reference fix also renamed an unrelated test helper of the same name, ADR-159);
  - `require`, `forbid` — `{file, match}` regular expressions over the finished file.
- `reference.patch` — the fix, as `git diff` writes it. Its changed lines, not its context lines, are the allowed regions.
- `base/` — a fixture's whole base tree.

`bench/repos.json` maps each repository to its public URL and the local path
its clone is expected at (`~/` is the home directory). `drop_sections` lists
markdown sections, by file and exact heading line, that bench removes from the
tree in both configurations; a stage whose pinned commit lacks one, or has it
other than exactly once, does not start. agentsmemory declares its `AGENTS.md`
gate on the `am_*` tools and the section on their absence, which made 11 Stage 1
attempts end in one turn (ADR-158).

The oracle passes a tree when every `require` matches, no `forbid` does, an
exact task's added lines are all there, nothing outside the allowed regions
changed (no other file, no line the fix leaves alone; a change git calls
binary, a mode change or a deleted file is outside), and every changed file
passes the check for its kind (Go parses, JSON is valid, YAML has no tab
indentation, markdown closes its fences).
`TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` proves, for every
task, that the oracle fails the base, passes the reference fix, and fails the
fix plus one stray line, and the fix with the line before its first change
deleted.

## Receipts: `docs/bench/<date>-<stage>/`

- `run.json` — the plan: stage, tasks, models, configurations, repeats, seed,
  CLI version, mrw version, containment; under ADR-162 also `profile`, `plugin`
  (`tag`, `commit`), and with `--status-line` `status_line_sha256`, `prompt_sha256`
  (of `PromptTemplate`, the task's instruction, a blank line, the status sentence) and
  `budget_seconds`. A second run with another plan into the same directory is
  refused.
- `<task>.<config>.<model>.<repeat>.started` — written before the attempt starts.
  An attempt with this file and no receipt crashed or was stopped, and is not run
  again on resume; an interrupt to `bench` leaves the attempt in flight just so,
  and stops the stage. An excluded attempt's replacement is `<attempt>.r1`
  (`.r2`), and `bench verify` wants each one a stage is owed.
- `….receipt.json` — written once, never replaced:
  `attempt`, `stage`, `task`, `shape`, `source`, `model`, `config`, `repeat`,
  `order`, `started`, `wall_ms`, `cli_version`, `mrw_version` (B only),
  `exit_code`, `timed_out`, `over_budget` (20 minutes or 60 turns), `is_error`,
  `num_turns`, `tool_calls`, `total_cost_usd`, `usage` (`input_tokens`,
  `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`),
  `init_tools`, `init_mcp_servers`, `oracle` (`solved`, `missing`, `outside`,
  `structure`; for a SWE-bench attempt also `harness_resolved`, the harness's own
  verdict, kept when the budget makes `solved` false, and `broke_tests`, any
  PASS_TO_PASS test failing), `silent_mis_edit`, `diff` (`files`, `insertions`, `deletions`),
  `hook_refused`, `hook_reads`, `slipped_writes` (B only), `plugin` (P only),
  `status` (`done`, `not done` or `unreadable`, with `--status-line`), `exclusion`
  (`start-error` or `setup`, the registered two), `note`, `check_red_at_base`
  (the task's flag, when it carries one).
- `….transcript.jsonl`, `….stderr` — what the CLI printed.

**The claim line (ADR-162 Decision 3).** With `--status-line` every instruction
ends with one sentence, the same in every configuration: "When you have
finished, end your reply with one line, exactly `STATUS: done` if the task is
complete, or `STATUS: not done: <reason>` if it is not." The receipt's `status`
is `done` or `not done` when exactly one line of the final message has that form,
`unreadable` otherwise; no prose is classified. `bench export-paired` writes
quality-harness ADR-106's rows for one model and one pair: a `{"config": …}` line
naming the dataset, split, revision, selection rule and salt, the instance count
and the SHA-256 of the instance list, `k`, `n`, the model, the CLI, the plugin,
the scaffold, the prompt's and status sentence's SHA-256, the turn and time
limits, the scorer's blob sha (`--scorer-sha`, from ADR-106's pre-registration),
the exporter's revision and the canary's result; then one row per instance, arm
and repeat — `instance`, `arm`, `repeat`, `status`, `resolved` (the harness's
own), `brokeTests`, `limit`, `unrun` — a retry chain read from its last try, and
`unrun` only when every try was excluded.
- `canary.json` — `pass`, `cli_version`, `model`, `started`, `problems` by
  configuration (and `sentinel`), `total_cost_usd` by configuration.

**Observed on the first passing canary** (2026-10-14, CLI 2.1.299, Sonnet 5.5,
`docs/bench/canary-2026-10-14/canary.json`): the stream-json fields bench reads
are the ones the Claude Agent SDK documents (`system`/`init` with `tools` and
`mcp_servers`; `assistant` messages with `tool_use` items; `result` with
`total_cost_usd`, `usage`, `num_turns`, `is_error`), and a subscription run does
report `total_cost_usd` (A $0.0170, B $0.0373), so cost is the CLI's own figure,
not one computed from token prices (ADR-156 Decision 4). The transcripts are not
kept in the tree: they hold this machine's absolute scratch and home paths.

The init line also lists the CLI's own built-in plugins (`cc-plugin-agents-md`,
`cc-plugin-telemetry`, `cc-plugin-plugin-authoring`) and agents (`claude`,
`Explore`, `general-purpose`, `Plan`, `statusline-setup`) in both
configurations. They ship with the CLI, not from the user's setup, so they are
not a leak; the canary does not check them. What they could carry in is what
the canary does check — no CLAUDE.md text, no hook output, no MCP server, and the
task tree's own instruction files without a line that names mrw, `AGENTS.md` included.

## Stage 5: blocks and looks (ADR-162)

The pool is `bench/swe-bench/selection-adr-162.json`: the 191 SWE-bench Verified instances with 2 or more hunks
less the 40 Stages 3 and 4b used, in `sha256("adr-162:" + id)` order, with the 31 that have 5 or more as a
descriptive subset; `bench swebench-select` regenerates it byte for byte. A block is 40 of them in that order
(`--block K`; the fifth block the remaining 31), run into `docs/bench/stage-5/block-K/` with its own `run.json`
naming the block and the five looks. A model's configurations come from `--configs-for MODEL=…`, as `bench look`
printed after the last block; a block whose CLI differs from block 1's is refused, since an upgrade starts a new
stage. Three start errors in a row stop the run, exit 2, resumable, rather than spending its retries on an
outage; with `--disk-floor N --disk-path P` no attempt starts below N GiB free on P, exit 2, resumable.

`bench look docs/bench/stage-5` reads the blocks and applies the stop rule. Each model's comparisons with A (B,
and P) are read at 99% — 1 − 0.05/5, a nominal Bonferroni allocation over the five looks — the cost by the
percentile bootstrap and the solved difference by Tango's score interval, which does not collapse to [0, 0] when
no instance is discordant. A look is eligible when every scheduled attempt has a receipt (with the replacements
owed), at most 5% of the instances are excluded in either configuration, and at least 95% are paired. A
comparison closes at its first eligible PAYS or LOSES, and its configuration is not run again for that model; a
model whose comparisons are both closed runs no more blocks; a block holding a configuration the rule did not
schedule is refused. It exits 0 when the stage is complete — after the fifth look, or when no model runs on — and
1 when a block is due, naming each model's next configurations. `bench analyze` prints a Stage 5 directory in a
table of its own, each verdict with the look it closed at, and leaves it out of the 95% table.

## Commands

```
bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass \
              [--profile stage5 [--plugin-dir DIR] [--node PATH]]
bench run     --stage S --out DIR --model M --claude PATH --mrw PATH --config-dir DIR \
              --containment bypass --canary DIR/canary.json [--task ID ...] [--repeats 2] \
              [--configs A,B[,P]] [--profile stage5 [--plugin-dir DIR] [--node PATH]] \
              [--status-line] [--block K [--configs-for MODEL=A,B,P ...]] [--disk-path PATH --disk-floor GiB] \
              [--source swe-bench --dataset FILE --swebench-python PATH [--selection FILE] [--docker PATH]]
bench look    DIR
bench verify  --stage S --tasks N --repeats R DIR
bench export-paired --pair X,Y --model M --scorer-sha SHA --selection FILE --canary FILE DIR...
bench analyze [--tasks-dir DIR] [--selection FILE ...] DIR
bench swebench-select [--n 20] < instance-ids
bench swebench-select --dataset FILE --min-hunks N [--exclude FILE ...] [--salt S] [--n 20] \
              [--subset-min-hunks N] [--no-unnamed] [--name D --revision R --split S]
bench mine    --kind wide-rename|check-coupled --repo DIR --commit SHA [--n N] [--min-occurrences 15 --min-files 8] [--min-test-files 1]
```

`bench analyze` prints one row per stage, model and task source, sorted by stage,
so two stages are never pooled, with the verdict of
the bar registered in `docs/adr/BACKLOG.md` ("Pre-registered for ADR-156"),
the bootstrap intervals, the exclusions, the silent mis-edits of each
configuration, and configuration B's counted shell reads and slipped writes —
the files B changed by a means its hook missed. The hook is a heuristic, not a
security boundary (the record's Out of Scope); the slipped writes are what
catches a write that got past it. A pilot is left out wherever its receipts sit:
a receipt whose stage begins `pilot` is not counted (until the review of #416
only a directory named exactly `pilot` was skipped, and a rerun's `pilot-2/` was counted).

Where a model's and source's attempts include tasks flagged `check_red_at_base`,
a second row follows the verdict, computed without them and marked descriptive.
It is not a verdict of the registered bar: the verdict is the first row, over
every task (Zy, 2026-10-14).

Where a group's attempts include `varied-multi-site` tasks, the verdict row is
followed by one more descriptive row, computed over those tasks alone (ADR-158).
`bench run` without `--task` runs every task that is not superseded.

Where a group's attempts include `wide-rename` or `check-coupled` tasks (ADR-160's
Stage 4), one descriptive row follows per category the group holds, the
`single-line` control included; and `bench analyze --selection FILE` adds a row
over a slice's `unnamed` instances, those whose problem statement names none of
the patched files. `bench swebench-select --dataset` writes that subset, and
reads the gold patch only to count hunks and files: nothing it reads reaches a
task.

`bench analyze` reads the corpus (`--tasks-dir`, default `bench/tasks`) for the
tasks superseded since a stage ran. Where a group's attempts include one of them,
one more descriptive row is printed, computed without them and labelled as
registered after the stage ran (ADR-159): it shows how much of a published
result a task graded wrongly carries, and the verdict row stays the one computed
over every task.

`bench run --source swe-bench` runs the SWE-bench slice (ADR-156 T5;
`bench/swe-bench/selection.json`, registered in BACKLOG, "Pre-registered for
ADR-156 T5"). A task is the instance's repository at its base commit, cloned
locally and named in `bench/repos.json`, and the agent is told its problem
statement and nothing else of the dataset row. The attempt's change is graded
by the official harness (`--swebench-python`, an interpreter with `swebench`
installed, reading `--dataset` at the revision T5 recorded) in the instance's
`linux/amd64` image. An instance's attempts run one after another, so its image
is pulled once and removed after they are graded. A SWE-bench row prints
"not measured" for silent mis-edits, since the slice has no allowed regions.

**Found after Stage 3 ran (T3's pilot, ADR-160 T6).** Configuration B's hook
refused an interpreter whenever the command's text matched a write pattern, the
body of another command's heredoc included, and read `> b` in a comparison as a
redirect. In Stage 3 it refused 8 commands in 6 B attempts:
`sympy__sympy-24661` (Opus, 3), `sphinx-doc__sphinx-8265` (Opus and Sonnet),
`scikit-learn__scikit-learn-25747`, `django__django-16642` and
`matplotlib__matplotlib-25775` (Opus). All six were solved, as were their A
pairs, so no outcome moved; the refusals cost B a turn each. Replayed through
the fixed hook, the five in sympy and sphinx pass; the three where python wrote
a plan or a temporary file outside the tree are still refused, since the hook
cannot tell where an interpreter writes. Stage 3's receipts and verdict are
unchanged.
