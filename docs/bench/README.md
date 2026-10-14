# bench — does mrw pay for itself on agent runs (ADR-156)

`bench` runs an agent CLI over a fixed corpus of editing tasks, once in each of
two configurations, and grades what each run leaves behind. It calls no model
itself: it starts `claude -p` as a subprocess, reads the stream-json it prints,
and judges the tree with an oracle that knows the answer by construction.

**Zy chose, on 2026-10-14, to run attempts under his own account with
`bypassPermissions`** — over a dedicated macOS user and over Docker — and logged
in once in the dedicated config directory (ADR-156 Decision 2). `bench run` and
`bench canary` still refuse without `--containment bypass` and `--config-dir`,
and `bench run` refuses without a passing canary for the same CLI version.

## Configurations

| | Tools | mrw | Hook |
|---|---|---|---|
| A | Bash, Read, Edit, Write, Glob, Grep | not on PATH | none |
| B | Bash, Glob | a pinned build on PATH, `mrw instructions` appended | `bench hook --tree`: refuses writes by other means, command substitutions included, except a redirection or `tee` to an absolute path outside the attempt's tree (Zy, 2026-10-14, after the pilot); counts reads, not refuses them (Zy, 2026-10-14), and `bench analyze` reports the count beside each verdict |

Both run with `--strict-mcp-config` and an empty MCP config, a settings file
written per attempt, `--disable-slash-commands`, `--no-session-persistence`,
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

## The corpus: `bench/tasks/<id>/`

- `task.json`:
  - `id` — the directory's name;
  - `shape` — `multi-site`, `structural`, `repeated-anchor`, `large-file`, `single-line`, or `fixture` (never measured);
  - `repo` — a name in `bench/repos.json`, or `fixture`;
  - `commit` — the full sha the base tree is taken from;
  - `instruction` — what the agent is told; it names no tool;
  - `slack` — how many lines either side of the fix's own insertion point new lines may also go (default 0); it never licenses changing a line the fix leaves alone;
  - `exact` — each touched file must equal the reference result, so a call site commented out or kept in a block comment fails though its tokens survive (the renames);
  - `check_red_at_base` — why the repository's own declared check fails at the pinned commit in the tree bench builds, so configuration B's automatic check is red whatever the agent does: a measured fact, carried by the seven quality-harness tasks (2026-10-14) and absent elsewhere;
  - `require`, `forbid` — `{file, match}` regular expressions over the finished file.
- `reference.patch` — the fix, as `git diff` writes it. Its changed lines, not its context lines, are the allowed regions.
- `base/` — a fixture's whole base tree.

`bench/repos.json` maps each repository to its public URL and the local path
its clone is expected at (`~/` is the home directory).

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
  CLI version, mrw version, containment. A second run with another plan into the
  same directory is refused.
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
  `structure`), `silent_mis_edit`, `diff` (`files`, `insertions`, `deletions`),
  `hook_refused`, `hook_reads`, `slipped_writes` (B only), `exclusion`
  (`start-error` or `setup`, the registered two), `note`, `check_red_at_base`
  (the task's flag, when it carries one).
- `….transcript.jsonl`, `….stderr` — what the CLI printed.
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

## Commands

```
bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass
bench run     --stage S --out DIR --model M --claude PATH --mrw PATH --config-dir DIR \
              --containment bypass --canary DIR/canary.json [--task ID ...] [--repeats 2]
bench verify  --stage S --tasks N --repeats R DIR
bench analyze DIR
bench swebench-select [--n 20] < instance-ids
```

`bench analyze` prints one row per model and task source with the verdict of
the bar registered in `docs/adr/BACKLOG.md` ("Pre-registered for ADR-156"),
the bootstrap intervals, the exclusions, the silent mis-edits of each
configuration, and configuration B's counted shell reads and slipped writes —
the files B changed by a means its hook missed. The hook is a heuristic, not a
security boundary (the record's Out of Scope); the slipped writes are what
catches a write that got past it. A pilot directory is left out.

Where a model's and source's attempts include tasks flagged `check_red_at_base`,
a second row follows the verdict, computed without them and marked descriptive.
It is not a verdict of the registered bar: the verdict is the first row, over
every task (Zy, 2026-10-14).
