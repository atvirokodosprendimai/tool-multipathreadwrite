# Getting an agent to use mrw

An agent that has mrw on its PATH does not use it on its own. It reaches for the tools it was trained on —
`cat`, `grep`, `sed -n`, its host's Edit and Write — unless its instructions say otherwise, every session.
This page says what to put in those instructions, and what has failed. Each point names the evidence behind
it; none of it is a guess about agents in general.

## 1. A standing instruction, in the file the agent always reads

Put the trigger where the host loads it on every session, not in a document the agent may open:

| Host | File |
|------|------|
| Claude Code | `CLAUDE.md` (repository) or `~/.claude/CLAUDE.md` (every repository) |
| Codex | `AGENTS.md` (repository) or `~/.codex/AGENTS.md` |
| Cursor | `.cursor/rules/mrw.mdc` with `alwaysApply: true` (repository or `~/.cursor/rules/`) |
| opencode | the plugin in `cmd/opencode/mrw-plugin` (README, "opencode") |
| any other | whatever file that host injects at session start |

**Say "always", never a threshold.** An instruction that said to use mrw for "3+ edits, 2+ files, several
ranges; below that, don't" told agents never to use it, because an agent works one edit per turn and no
single turn ever crosses the line (ADR-062, 2026-09-16). The rule that works is the binary's own first
sentence: *use mrw always: plan the activity as one read of every site, then one plan, then one write.*

**Say that it overrides the host's own advice.** The host's built-in system prompt may tell the agent to
prefer its own Edit and Write tools, or allow `sed` and heredocs for small edits. In a field test on
2026-10-22, six of seven Mac sessions named that prompt as the instruction pulling against mrw, and those
that said mrw would win said so because their instruction file claims to override the defaults; none had
edited a file yet, so that is stated intent, not observed behaviour. Say it in the trigger, in one
sentence.

**Point at the contract; do not copy it.** `mrw instructions --core` prints the eight rules that matter most
in under 300 words (ADR-141), and `mrw instructions` adds the recipes. A survey of 14 adopting sessions
(2026-10-09) found about 900 words of mrw guidance carried every session for about eight rules, and two
paragraphs that eight and five sessions had acted on zero times; a copy also goes stale with the next
release. A trigger of five short paragraphs that names the rule, the exception and where the contract lives
is enough. A working one:

```markdown
## mrw — every edit goes through it

Every edit and every new file goes through `mrw write`. Its ledger refuses a write to a line it has not
served you, so read with `mrw read` what you mean to edit, and batch: one read of every site, one plan,
one write. A write that bypasses mrw is a rule broken: say so in the same reply. The only exceptions are
files mrw cannot serve (binary, images, PDFs) and a host with no shell.

Inspection reads are free: `git show`, `git diff`, logs and files you will not edit may use any reader.

`mrw instructions --core` prints the rules that matter most; `mrw instructions` adds the recipes.
```

**Use one trigger per host, and keep them identical.** A machine that runs Claude Code, Codex and Cursor
side by side needs the same text in all three files: an agent cannot follow a rule its host never loaded.
Change them together.

## 2. A brief to a subagent carries the commands, not the word

A subagent follows its brief literally, and the brief is all it has. Survey and review subagents briefed
only "read with mrw" still used `grep` over file contents, `cat`, `printf >` and `cp` (2026-09-29). Paste
the commands into every brief:

```text
Every file you read to edit, and every edit, goes through mrw:
- read: `mrw read path:N-M path:/regexp/` — many specs in one call; outside the checkout `mrw --root DIR read …`
- search contents: `mrw read --grep P dir/`, or `rg -l P . | mrw read --files-from -` — never grep/cat/sed -n/head on a file
- narrow command output: `cmd 2>&1 | mrw read --grep P --max-lines 40 -`
- edit: one plan on standard input, `mrw write -` (`@@ path N-M replace anchor="…"`), read first
- new file: `mrw write --create path < content`; delete: `@@ path - unlink`
- `--no-check` for a write to prose or shell only; a literal `$` in a `--sub` replacement is `$$`
- never `>`, `cp`, `sed -i`, or the host's Edit/Write
Your final report lists any command that broke this.
```

Then read the subagent's transcript for slips: the last line is a request, and a fast agent breaks it.

## 3. In a sandbox, give mrw a writable state directory first

mrw keeps its read ledger outside the checkout, under `XDG_STATE_HOME` (default `~/.local/state`). A
sandbox that lets the agent write only its workspace — Codex's default, many CI runners — cannot write
there. Since v1.71.0 a content read that names its specs still serves and exits as it would, with
`mrw: nothing was recorded` on standard error, just above the kind line when it exits 1 (ADR-166); `--quiet`,
`--sub`, `--rewrite` and a read that needs the working set still exit 2. No new licence is recorded, so a write
to those lines exits 2 naming the state directory while it stays unwritable, and once it can be written is
refused as unread unless an earlier read licensed them.
Tell the agent, or its launcher, to export `XDG_STATE_HOME` to a
writable absolute directory outside the checkout for the whole session, so every read and write uses one
ledger.

## 4. A host without a shell uses the MCP server

With a shell, the CLI is the better surface: it has `--context`, `--then-sh` and every read flag, and one
command reaches any checkout with `--root`. A host with no shell runs `mrw mcp` and calls `mrw_read` and
`mrw_write` (README, "MCP"). Register it only for the hosts that need it: a user-scope registration puts the
two tools into every project on the machine, including those whose instructions say to use the shell.

## 5. Check that it is happening

An instruction is a claim about behaviour; `mrw stats` is the measurement. It counts the plans this checkout
has been given and what became of them. One repository in the 2026-10-09 survey made mrw a hard rule in its
`CLAUDE.md` and had zero recorded plans. Zero is a reason to look, not proof by itself: a failed tally
write is not reported and `mrw stats --reset` clears the counts, so check the agent's transcript too. If
`mrw stats` stays at zero while the agent edits files, the trigger is likely not reaching it.

And check the binary is current: `mrw version --check-latest` exits 1 when a newer release exists (ADR-149).
Five of five Windows repositories in the same survey ran a release four versions old with no sign of it.
