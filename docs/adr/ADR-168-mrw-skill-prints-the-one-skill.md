# ADR-168: `mrw skill` prints the one mrw skill, from the binary

**Status:** Accepted
**Accepted:** 2026-10-22 by Zy — "so i want no penging, left for later, no followups. this adr must solve these. if it is - then I accept", after the record was revised to defer nothing: its Out of Scope carries no deferral, its Follow-ups are none, and the memoryd refresh is T4 S4. Earlier answers: "Print + --dir", "Replace both", "Version stamp only", "Both work".
**Amended:** 2026-10-22 by Zy — asked *"hmm what about security? would this tool allow agents to bypass restricitons?"*, then *"add, yes"* to adding what allowing mrw allows as T5 of this record (Decision 6), so it is not left for later.
**Amended:** 2026-10-22, from the chaos test Zy asked of every session (five Mac sessions and two Windows sessions replied; three Windows sessions could not clone, refused by their own classifiers): Decisions 1 and 2 now stamp the body's sha-256 and refuse a body a person edited, say "larger" for a file over the bound, read a byte order mark as CRLF is read, name byte differences on "unchanged", re-judge a run that loses a race to another, refuse a rollback by an older release, refuse a second `--dir`, quote a control character in a path, and give the engine's refusals back with the caller's path; a misspelt `--skill` names the spelling that works.
**Date:** 2026-10-22
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-037, ADR-041, ADR-062, ADR-063, ADR-066, ADR-071, ADR-077, ADR-081, ADR-089, ADR-097, ADR-125, ADR-141, ADR-143, ADR-145, ADR-161, ADR-167, docs/agents.md
**Governs:** `internal/guide/guide.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`, `docs/agents.md`, `.claude/skills/mrw/SKILL.md`, `.claude/skills/release/SKILL.md`
**Enforced-by:** None — until T1 lands, which adds `internal/guide/skill168_test.go::TestTheSkillIsAValidAgentSkill`
**Invalidates:** ADR-097, narrowed for one token: a bare `--skill` in the subcommand position is read as the subcommand `skill` instead of refused with a pointer, because `mrw --help` tells agents to type it (Decision 4); every other spelling at the root (`-skill`, `--skill=x`, a padded `--skill `) is refused, exit 2, with a sentence naming the bare spelling instead of ADR-097's "is not a flag", which would deny the spelling the help teaches; `--skill` after a subcommand keeps the ordinary refusal. ADR-141 Out of Scope "Shortening `mrw instructions`, AGENTS.md or the skill", narrowed: the skill stops being a hand-maintained mirror and becomes the output of `mrw skill`; `mrw instructions` and AGENTS.md keep that disposition. The sync rule in `.claude/skills/mrw/SKILL.md` and `.claude/skills/release/SKILL.md` §7 ("AGENTS.md is authored; the skill mirrors it"), and the premise of `cmd/mrw/agentsdoc_test.go`'s comment that the centralised skill mirrors AGENTS.md, are replaced by T4. ADR-041 holds: skew is named (the stamp), not healed.
**Served-path change:** `mrw skill`, or `mrw --skill`, prints an Agent Skills `SKILL.md` — frontmatter, then `mrw instructions --core`, the recipes and a pointer to `mrw instructions` — and `--dir DIR` writes it to `DIR/mrw/SKILL.md` through the write engine; `mrw --help` and a bare `mrw` print `AGENTS: READ THE SKILL FIRST: mrw --skill` as their first line.

## Context

Zy, 2026-10-22: *"got a new idea `mrw --skill` to give out skill ready with frontmatter — how to use this effectively in any coding agent which supports skills"*, then *"plan the skill — i think this is good addition.. need to figure out where then instructions goes? or is it `mrw --help` that should point to `mrw --skill`. how to onboard it."* Mindaugas, 2026-10-22, pasted by Zy: *"install this tool … ./mrw --help … READ SKILL WITH --skill … capitals mean a lot generally … attention"*.

**Hosts load skills in three stages.** The Agent Skills format (url https://agentskills.io/specification, read 2026-10-22) puts `name` and `description` (at most 1024 characters) in every session, about 100 tokens; loads the body, recommended under 5,000 tokens and 500 lines, only when the model decides the skill applies; and loads anything further on demand. `name` is lowercase letters, digits and single hyphens, at most 64 characters, and must equal the skill's folder name; `metadata` is a map of strings. Skill directories present on this Mac on 2026-10-22, each holding other skills: `~/.claude/skills`, `~/.codex/skills`, `~/.cursor/skills`, `~/.agents/skills` (Codex CLI 0.163.0, Cursor agent 2026.10.14). Which directory each host reads is the host's to say; this record names none as a promise.

**The class this record governs:** *a standing text in this repository that tells an agent how to use mrw and can drift from the binary.* Enumerated 2026-10-22 with

```
git ls-files '.claude/*.md' '.claude/**/*.md' '*.md' docs/agents.md internal/guide/guide.go internal/mcp/instructions.go 'cmd/opencode/mrw-plugin/src/*.ts' \
  | grep -E '^(\.claude/|[^/]+\.md$|docs/agents\.md|internal/|cmd/)' \
  | xargs grep -l -E 'mrw instructions|load_skill|update_skill|`mrw (read|write)|mrw_read|mrw_write'
```

— 14 files, each with its disposition here:

| Member | Disposition |
|--------|-------------|
| `internal/guide/guide.go` | the source; T1 adds `Skill`, T3 one sentence |
| `.claude/skills/mrw/SKILL.md` | becomes `mrw skill`'s output (T4) |
| `.claude/skills/release/SKILL.md` | §7 refreshes from `mrw skill` (T4) |
| `AGENTS.md`, `README.md`, `docs/agents.md` | name the skill beside the trigger (T1, T3) |
| `internal/mcp/instructions.go` | unchanged: an MCP host is taught by the handshake (ADR-037) |
| `CLAUDE.md` | unchanged: it points at AGENTS.md and teaches no format |
| `.claude/agents/{adr-challenger,go-implementer,go-reviewer,scout}.md` | unchanged: a subagent brief carries the commands, not a skill (`docs/agents.md` §2) |
| `BESTPRACTICES.md` | unchanged: ADR-161 T4 owns its recipes |
| `cmd/opencode/mrw-plugin/src/index.ts` | unchanged: ADR-089 owns how opencode is told |

Outside the tree are two more: memoryd's `craft` skill `mrw`, at v64 on 2026-10-22, a hand mirror of AGENTS.md "Using mrw" that `.claude/skills/release/SKILL.md` §7 tells the releaser to refresh by hand; and the global triggers on Zy's machines. The in-repo `.claude/skills/mrw/SKILL.md` (129 lines) is a pointer to the memoryd mirror plus repository facts. It records two failures of the hand-mirror arrangement: on 2026-09-03 it pointed at a central skill that did not exist; on 2026-09-04 that skill, once created, had fallen behind the source and taught a rule ADR-013 had made false.

**What exists to hand an agent.** `mrw instructions --core` (ADR-141): eight rules, nine lines, 299 words (measured 2026-10-22), held to the binary's flags and to the full text by test. `guide.Recipes()` (ADR-161, ADR-167): ten recipes. Together they are what a skill body should be; `docs/agents.md` §1 already says to point at the contract, not copy it. What is missing is the package — the frontmatter a skill host reads — and a way to put the file where the host looks.

**What a skill cannot do.** A skill is chosen by the model from its description; a standing instruction (`CLAUDE.md`, `AGENTS.md`, a Cursor rule) is loaded every session. `docs/agents.md` §1 records what has worked: the standing trigger, "always", the override sentence. Nothing measured here says a skill fires for routine edits. So the skill is added beside the trigger, and the record claims nothing more for it.

**What allowing mrw allows, and who has not been told.** mrw runs as its user. A harness rule that allows it (`Bash(mrw:*)` or its equivalent) allows writes to any file the user can write — `--root DIR` names any directory, and mrw confines a path to the root it was given, not to a root anyone chose — and shell through `--then-sh`, `mrw check` and `--then`. A host's per-path rules for its own Edit and Write tools do not reach a shell command. Only the shell half is written down (README:342, AGENTS.md:571); README, `docs/agents.md` and the skill say nothing of the write half. This record makes it matter more: the skill and the trigger tell an agent to edit through mrw in place of the host's Edit and Write tools, moving it from a tool with per-path prompts to a command whose permission is all or nothing. `--dir` itself adds no capability: it writes only mrw's own text, where `mrw --root ~/.claude/skills write` could already write any.

**Gates that already read the skill file.** Three accepted fences and tests read `.claude/skills/mrw/SKILL.md` and must stay green after T4: `cmd/mrw/agentsdoc_test.go::TestReadmeAndSkillNameInstructions` (a backticked `mrw instructions`); ADR-063 T1's fence (`^description: >-`, a folded block indented two spaces, naming `--grep`, under 1024 bytes); ADR-161 T4's fence (the literal `mrw write --grep '\bOld\b' --sub 'New' .`, which `Recipes()` carries). And `TestEverySubcommandReachesTheAgentFacingGuide` (`cmd/mrw/agentsdoc_test.go:36-55`) requires every subcommand to appear backticked in AGENTS.md, so `skill` cannot be registered without it.

## Existing Primitives Audit

- **`guide.Core()`** (ADR-141) and **`guide.Recipes()`** (ADR-161) — the body, used as they are; nothing is copied.
- **`instructionsCmd`** (`cmd/mrw/main.go:541`) — the shape of a teaching subcommand. The `Before` hook's two verb lists (`cmd/mrw/main.go:169` and `:175`) are where `version` and `instructions` skip state (ADR-145); `skill` joins both.
- **The write engine** (`writer.Prepare` and its landing, `internal/writer/flow.go:93`, over `internal/apply`) — taken for `--dir`. It already resolves links and junctions (ADR-071), refuses reserved names (ADR-081), `.git` (ADR-143) and mrw's own state (ADR-077), reports a target held without delete sharing (ADR-125), keeps a replaced file's mode and creates with 0644 (`internal/apply/apply.go:2374`), creates missing parent directories and names them, and reports what reached disk when a commit fails (ADR-066). A second writer for one file would need every one of those again; both challengers listed them as gaps of the first draft.
- **`sha=` and `Force`** — the licence a whole-file replace needs: `--dir` reads the existing `SKILL.md` itself, after an `Lstat`, judges its marker, and replaces it with `sha=` set to the bytes it judged and `Force` standing for that read; the engine refuses the replace if the file changed in between (`internal/apply/apply.go:1407`, `TestPreconditions`'s "stale sha" case).
- **ADR-097's refusal** (`subcommandForFlag`, `cmd/mrw/main.go:301-311`) — stays for every spelling but the one Decision 1 accepts.
- **The argument rewrites in `main`** (`liftLoneDash`, then `refusePaddedFlagValues`, `cmd/mrw/main.go:364-377`) — the place the `--skill` rewrite goes.
- **`cli.Command.CustomRootCommandHelpTemplate`** (urfave/cli v3.11.0, `command.go:102`) — a template prefixed with the agents line makes it the first line of `--help` and of a bare `mrw`.
- **The version string** printed by `mrw version` — the stamp.

## Decision

1. **`mrw skill` prints `guide.Skill(version)` to standard output**, exit 0; it takes no arguments (exit 2 otherwise) and, like `version` and `instructions`, touches no state (ADR-145). **`mrw --skill` is the same command**: in `main`, before every other argument check, the first argument that is neither a root flag nor a root flag's value is replaced by `skill` when it is exactly `--skill`. So `mrw --skill`, `mrw -C d --skill` and `mrw --skill --dir DIR` behave as `mrw skill …`; `mrw --root --skill instructions` is unchanged (`--skill` is `--root`'s value there, as the parser reads it); `-skill`, `--skill=x` and a padded `--skill ` are refused, exit 2, naming the bare spelling. ADR-097 (its Alternatives) and ADR-145 (its Alternatives) rejected predicates on `os.Args` because they must name every spelling the parser takes; this one names one exact token and leaves every other spelling to the parser's refusal, so it does not need to. The text is:
   - YAML frontmatter: `name: mrw`; `description: >-`, a folded block indented two spaces, at most 1024 bytes, that names the trigger — every file edit and new file, and reading what will be edited — what mrw does in one sentence, `--grep` for finding sites, that it overrides the host's own edit advice, and that it applies only while `mrw` is on PATH — otherwise the host's own tools are used and mrw is not installed; `metadata:` with `mrw-version: "<version>"`, `generated-by: "mrw skill"` and `content-sha256: "<hex>"`, the sha-256 of the whole file with that one line taken out — description and body alike — all double-quoted strings.
   - The body: `guide.Core()`, then `guide.Recipes()`, then one closing paragraph — `` `mrw instructions` `` (backticked) prints the whole contract; this file was printed by mrw `<version>`, and when `mrw version` prints another, `mrw skill --dir <the directory holding this mrw folder>` replaces it.
2. **`mrw skill --dir DIR` writes the same text to `DIR/mrw/SKILL.md` through the write engine**, as `mrw --root DIR write` would apply one hunk: a `create` when the file is absent (the engine creates `DIR/mrw`), or a whole-file replace when it is present. `DIR` is taken as given, relative to the working directory; it must be an existing directory (exit 2 otherwise, so `~/.claude/skils` is not created). Before a replace, `--dir` reads the file and refuses it, exit 1 and nothing written, unless its frontmatter's `metadata` carries `generated-by: "mrw skill"` — a skill a person wrote is never overwritten; the marker in the body or a comment does not count. `DIR/mrw` or `SKILL.md` that is a link or a junction is refused by `--dir` itself, exit 1, because the engine follows a link that stays inside its root. When the bytes are already equal nothing is written. Every other refusal is the engine's, with its exit code and reason: a link that leaves `DIR`, a reserved name, a path through `.git`, mrw's own state directory, a target held on Windows, a read-only file. One line on standard output says what happened: `created`, `replaced` with the old and new `mrw-version`, or `unchanged`; when the ledger cannot record a landed write, that line is printed and the exit is 2. A `--dir` write is a write: it migrates a legacy `./.mrw/` first, names a `DIR` ledger written by a newer mrw, and keeps the engine's state for `DIR` — its lock, ledger and tally — as any `mrw --root DIR write` does; plain `mrw skill` keeps none.
   The chaos tests of 2026-10-22 and their re-runs added the rest of what a person's file needs: a stamped file that no longer matches its `content-sha256` — a body line, an appended line, or the trigger `description` edited — was edited after `mrw skill` wrote it and is refused, exit 1, never replaced (the digest catches an honest edit, not a deliberate forgery: a person who recomputes it has taken the file as theirs to lose); a file over 1 MiB is refused as larger than any skill mrw writes; a leading byte order mark is read as CRLF and a missing final newline are, `unchanged` names which of them the file has, and a replace keeps the byte order mark as it keeps the line endings; an empty `SKILL.md` is the engine's probe of a racing run's new name (ADR-105) and is judged again, and a run whose create or replace is refused because another `--dir` changed the file first judges again too, at most five times, so runs that agree all exit 0; a skill stamped by a newer release (both versions `vX.Y.Z`, compared numerically, a pre-release counting as older) is refused, exit 1, so a stale binary on PATH cannot roll it back; a replace at the same version says the content differed, not "X → X"; `--dir` given twice is refused, exit 2; a path holding a control character is printed quoted; and a refusal the engine makes is given back on standard error with the folder as the caller spelled it and no temporary file's name — only the engine's exact relative names (`mrw/SKILL.md`, `mrw/.mrw-…`) are rewritten, with the separator the engine wrote there, so an absolute path that holds `mrw/` (a DIR such as `/tmp/host mrw` included) is left as it is; a landing that left a file behind is reported with its leftovers and its own exit, never retried as a race; and `CRLF` is said to be kept only for a file all in CRLF, since a replace of mixed endings writes LF (the Codex review of 6860a48). On Windows a `DIR` with a component ending in a dot or a space (`d1.`, `d1 `) is refused, exit 2: Win32 reads it as the name without them, so it would name another folder (ADR-071's rule, applied to the folder `--dir` is given). A replace names a byte order mark, CRLF or a missing final newline it kept, as `unchanged` does. A file another process holds on Windows is the target's cause, exit 1 (ADR-125), worded as the engine words one. A hard link at `SKILL.md` is replaced by a new file, so the other name keeps its old text, as every mrw write does. A SKILL.md written by this record's draft before `content-sha256` existed (PR #507's first head, never released) is not the shape and is refused; no released mrw wrote one, so there is nothing to migrate.
3. **The binary's skill is the one mrw skill.** `.claude/skills/mrw/SKILL.md` becomes `mrw skill`'s output: once, by hand, its repository facts that AGENTS.md lacks move to AGENTS.md and the old file is removed with `mrw write`; then `mrw skill --dir .claude/skills` writes it. From then on a test holds it equal to `guide.Skill(v)`, where `v` is the file's own `mrw-version`. Release §7 replaces "update the skill by hand, mirror it" with: refresh memoryd's `craft` skill `mrw` from the tagged binary's `mrw skill`, and regenerate the in-repo file with `--dir`. The memoryd skill is refreshed the first time from the merged build in T4, not left for the next release.
4. **Onboarding names it where a caller looks first.** `mrw --help` and a bare `mrw` print `AGENTS: READ THE SKILL FIRST: mrw --skill` as their first line, then a blank line, through `CustomRootCommandHelpTemplate`; the root description adds how to install it (`mrw --skill --dir DIR`) and `mrw instructions --core` for the rules alone. The help grows by at most four lines over the 25 measured on 2026-10-22. `mrw instructions` ends with one sentence naming `mrw skill` (not `--skill`, which is a rewrite and not a declared flag). README "Tell your agent to use it", the `docs/agents.md` §1 table, AGENTS.md §5 and its "none of these commands writes to your checkout" paragraph say: keep the standing trigger, add the skill, and `--dir` writes where it is told.
5. **The skill is added beside the standing trigger, never in its place.** No text this record changes says a skill alone is enough, and the trigger text in `docs/agents.md` §1 is unchanged; the global triggers that copy it therefore need no change either.
6. **What allowing mrw allows is said where the caller and the agent look.** README "Tell your agent to use it" and a new `docs/agents.md` section, "What allowing mrw allows", say: mrw runs as its user; allowing it allows writes to any file the user can write under any `--root`, and shell through `--then-sh`, `mrw check` and `--then`; a host's per-path rules for its Edit and Write tools do not reach it; mrw's own refusals (`.git`, its state, a link that leaves the root) keep a plan honest, not an agent contained; contain an agent with the host's sandbox or the operating system's permissions, or allow mrw only where every write would be allowed. The skill's closing paragraph carries one sentence for the agent: a host's per-path rules for its Edit and Write tools do not reach mrw, so their absence is not permission, and mrw is not a way to reach a path the host refused.

## Alternatives Considered

- **`mrw skill` only, with `mrw --skill` refused by ADR-097 naming the subcommand** — rejected by Zy (2026-10-22, "Both work"): the help line tells an agent to type `--skill`, and a refusal of what the help said costs that agent a turn.
- **`--skill` only, no subcommand** — rejected: every other teaching surface is a subcommand (`instructions`, `version`), and `--dir` reads as the subcommand's flag.
- **`mrw instructions --skill`** — rejected: `--dir` is a write, and `instructions` is a read-only command whose flags change only what it prints.
- **A writer of its own for `--dir`** (the first draft: temp file and rename in `DIR/mrw`, its own refusals) — withdrawn after both challengers (2026-10-22) listed what it lacked: links and junctions under `DIR`, reserved names, `.git`, mrw's state, held files on Windows, file modes, cleanup reporting. Each is an accepted decision the engine already keeps.
- **Print only, with `mrw skill > DIR/mrw/SKILL.md` in the README** — rejected by Zy (2026-10-22, "Print + --dir"): a redirect overwrites a skill a person wrote, is not atomic, does not make `DIR/mrw`, and says nothing about what it replaced.
- **Detect the hosts and install into all of them (`--install`)** — rejected by Zy: mrw would guess directories that differ per host and per version, and write where nobody named.
- **`mrw skill --check DIR`, exit 1 when stale** — rejected by Zy ("version stamp only"): the stamp and the closing paragraph name the skew (ADR-041), and `--dir` heals it when asked.
- **Copy the full contract into the skill, or into a `references/` file beside it** — rejected: a copy of `mrw instructions` is the mirror that fell behind; the body points at the binary for the rest.
- **Keep three skills (binary, in-repo, memoryd)** — rejected by Zy ("replace both").
- **Generate the description from `Core()`** — rejected: the description is the trigger, a different job from a rule, written once and bounded by test.

## Component / Boundary Impact

`internal/guide` gains `Skill`; `cmd/mrw` gains the `skill` command, the `--skill` rewrite and the help template. `--dir` drives the existing write and read engines through their public entry points and changes neither: no engine package (`internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`) changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `guide.Skill(version)` | new function | T1 | `cmd/mrw`, T4's test |
| `mrw skill`, `mrw --skill` | new subcommand, and its root spelling | T1 | CLI callers, skill hosts |
| `mrw skill --dir DIR` | new flag; writes `DIR/mrw/SKILL.md` through the engine | T2 | CLI callers |
| `mrw --help`, bare `mrw` | first line for agents | T3 | every caller |
| `mrw instructions` | one sentence at its end naming `mrw skill` | T3 | CLI callers |
| `scripts/contract.sh` | §293 | T1, T2 | CI Linux |
| AGENTS.md | the `mrw skill` bullet (T1); the skill beside the trigger (T3) | T1, T3 | every agent |
| README, `docs/agents.md` | the skill beside the trigger | T3 | readers |
| `.claude/skills/mrw/SKILL.md`, `.claude/skills/release/SKILL.md`, memoryd `craft` skill `mrw` | generated; the release step | T4 | Claude Code here; other repositories; releasers |
| README, `docs/agents.md` "What allowing mrw allows", the skill's closing sentence | what allowing mrw allows | T5 | callers who configure a harness; agents |

No receipt key changes (`docs/receipts.txt`): `mrw skill` has no `--json`.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `guide.Skill(version)` | T1 | T2, T3, T4 | no |
| `mrw skill --dir` | T2 | T3, T4 | no |

## Implementation

See `docs/adr/ADR-168/tasks/README.md`: T1, then T2, then T3 and T4, then T5. **The record merges in the same pull request as its tasks**, as ADR-167 did (a455fded, #503): `internal/adversarial/record_test.go::TestEveryTestNamedByARecordExists` requires every test a task names to exist, so the record alone turns the suite red.

## Consequences

- **Positive:** one skill, printed by the binary it describes, so it cannot teach a flag the binary lacks; installing and upgrading are one command per host, through the engine's guards; the in-repo and memoryd copies stop being hand mirrors.
- **Negative:** the memoryd skill shrinks from the AGENTS.md section to the core and recipes; a session in another repository that wants the rest runs `mrw instructions`. An installed copy still lags its binary until someone runs `--dir`; the stamp says so. A `--dir` write leaves mrw state for `DIR` in mrw's state base, as any `--root` write does.
- **Neutral:** the standing trigger stays the instruction; the skill is an addition to it.

## Out of Scope

- Saying which directory each host reads skills from (permanent: boundary: the host owns that and changes it; README names the hosts measured with the date, not a promise)
- Replacing the standing trigger with the skill (permanent: boundary: Decision 5 — nothing measured says a skill fires for routine edits, and this record claims nothing it has not measured)
- The opencode plugin's own injection (permanent: boundary: ADR-089 owns how opencode is told)
- Editing the global triggers on Zy's machines (`~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, `~/.cursor/rules/mrw.mdc`) (permanent: boundary: Decision 5 leaves the trigger text unchanged, so they need no edit)
- An MCP form of the skill (permanent: boundary: an MCP host is told by the handshake, ADR-037's 4,096-byte document)
- Skill `scripts/` or `assets/` (permanent: boundary: the body points at the binary; a script beside it would be a second mrw)
- The four agent definitions and `BESTPRACTICES.md` (permanent: boundary: a brief carries the commands, `docs/agents.md` §2; ADR-161 T4 owns those recipes)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the description breaks a host's parser or bound, so the skill is skipped silently | Low | the skill never loads | `TestTheSkillIsAValidAgentSkill` checks name, folder, 1024-byte bound, folded form, string metadata, quoting of a stamp such as `dev (18ebc2f)` |
| `--dir` overwrites a skill a person wrote | Low | lost work | the frontmatter marker check before any replace; `TestSkillDirInstallsAndReplacesOnlyItsOwn` |
| `--dir` escapes `DIR` through a link or junction, or writes into `.git` or mrw's state | Low | a write where nobody asked | the engine's `Resolve` and refusals (ADR-071, ADR-077, ADR-143); T2 tests a link and a `.git` target |
| the `--skill` rewrite eats a root flag's value or a later argument | Low | a command means something else | the argument matrix in `TestSkillCommandPrintsTheSkill`; the rewrite skips root-flag values |
| a link swapped in under `DIR` between `--dir`'s check and the write | Low (a racing writer in the caller's own skills folder) | the skill lands where the link leads, inside `DIR` | accepted as a boundary: the engine follows a link that stays inside its root by design (ADR-071 confines it to `DIR`), as `mrw --root DIR write` does; `--dir` refuses a link it can see, and `sha=` refuses a changed file |
| a host never selects the skill for routine edits | Medium | the skill adds nothing | the trigger stays the instruction (Decision 5); nothing relies on the skill firing |
| the in-repo file and the binary disagree | Low | Claude Code here reads a stale skill | `TestTheRepositorySkillIsTheBinarysSkill` |
| T4 breaks a gate that reads the skill file | Medium | a done task goes red | Decision 1 keeps the backticked `mrw instructions`, `description: >-` with `--grep`, and the recipe-1 literal; T4 runs those three gates |

## Rollback

Revert T1–T4 in one revert of the merged PR: the subcommand, the rewrite and the help line go; files `--dir` wrote stay where the caller put them and are plain text; restore the in-repo SKILL.md and release §7 from git, and memoryd's `craft` skill `mrw` from its previous version (`restore_skill`). No mrw state changes shape.

## Follow-ups

- None.
