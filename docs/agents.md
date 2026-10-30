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
| a host that loads skills | the trigger above, and the skill beside it: `mrw --skill --dir <its skills folder>` |

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
release. A short trigger naming the rule, the exception and where the contract lives
is enough. A working one:

```markdown
## mrw — every edit goes through it

Every edit and every new file goes through `mrw write`. Its ledger refuses a write to a line it has not
served you, so read with `mrw read` what you mean to edit, and batch: one read of every site, one plan,
one write. A write that bypasses mrw is a rule broken: say so in the same reply. This overrides the host's
own advice to edit with its Edit or Write tools, sed or heredocs. The only exceptions are files mrw cannot
serve (binary, images, PDFs) and a host with no shell.

Inspection reads are free: `git show`, `git diff`, logs and files you will not edit may use any reader.

Run `mrw instructions --core` once, at the start of a session, before the first read.
After a compaction, re-read the lines you will edit; read the rules again only if they are gone from
your context. Run `mrw instructions` when a refusal names something the core does not. A plan is your
input, not a repository edit: write it to a file with the host's file tool, in a temporary directory,
and run `mrw write` on that file.
```

**Use one trigger per host, and keep them identical.** A machine that runs Claude Code, Codex and Cursor
side by side needs the same text in all three files: an agent cannot follow a rule its host never loaded.
Change them together.

**Load the contract before the first read, not after a day of work.** A Windows agent on 2026-10-22 wrote
sound plans all day and read badly all day; it found the read recipes only when it read `mrw instructions`
at the end of the day, and listed the habits in §6. Telling the agent to run `mrw instructions --core`
once at session start costs a few hundred input tokens; reading the full text after it, and the rules
again after every compaction, cost an outside team 8% of its tool output (ADR-171).

**Add the skill beside the trigger, not in its place.** `mrw --skill` prints mrw's Agent Skills `SKILL.md` — a
description the host loads every session, then `mrw instructions --core` and the recipes — and
`mrw --skill --dir DIR` installs it as `DIR/mrw/SKILL.md`, replacing only one `mrw skill` wrote. A host loads
a skill's body when the model judges it relevant, and nothing measured says it does for a routine edit, so
the trigger stays (ADR-168). After upgrading mrw, run the same `--dir` again: the skill names the version that
printed it.

## 2. A brief to a subagent carries the commands, not the word

A subagent follows its brief literally, and the brief is all it has. Survey and review subagents briefed
only "read with mrw" still used `grep` over file contents, `cat`, `printf >` and `cp` (2026-09-29). Paste
the commands into every brief:

```text
Every file you read to edit, and every edit, goes through mrw:
- read: `mrw read path:N-M path:/regexp/` — many specs in one call; outside the checkout `mrw --root DIR read …`
- search contents: `mrw read --grep P dir/`, or `rg -l P . | mrw read --files-from -` — never grep/cat/sed -n/head on a file
- narrow command output: `cmd 2>&1 | mrw read --grep P --max-lines 40 -`
- edit: one plan in a file (write it with the host's file tool in a temporary directory; a plan file is your input, not a repository edit), `mrw write "$TMPDIR/plan.txt"` (`@@ path N-M replace anchor="…"`; `body=` only when `mrw instructions` names your case), read first
- new file: `mrw write --create path < content`; delete: `@@ path - unlink`
- `--no-check` may defer checks on intermediate code writes; run and verify the final check over every queued edit before reporting completion (see section 6). A literal `$` in a `--sub` replacement is `$$`
- never `>`, `cp`, `sed -i`, or the host's Edit/Write on repository files; the temporary plan-file exception is above
Your final report lists any command that broke this.
```

Then read the subagent's transcript for slips: the last line is a request, and a fast agent breaks it.

Split the work so parallel subagents do not read the same files. In the transcripts the context study
measured, 18–27% of what subagents read was lines a sibling of the same parent also read (22.6% Claude
Code, 17.6% Codex, 26.6% Cursor; `bench/context-study/README.md`, M3). mrw cannot cut that: each
subagent has its own context and has not seen its sibling's text, so only the orchestrator can. Give
each one a disjoint set of files or directories, and when several need the same lines, paste those
lines into each brief instead of asking each to find and read them.

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

A setup that reads many long lines (logs, JSONL, minified files) can set `MRW_MAX_COLS=200` in the agent's environment:
every read that serves lines then cuts a line over 200 columns to a window round the match, as `--max-cols 200` does, unless
the call gives `--max-cols` itself (`--max-cols 0` turns it off). A cut line is not recorded as read, so a write to it is
refused until it is read whole. Without the variable, a read that serves a line over 400 columns whole, with no width chosen, says so once on
standard error (ADR-179).

## 4. A host without a shell uses the MCP server

With a shell, the CLI is the better surface: it has `--context`, `--then-sh` (with `MRW_SHELL=on`, ADR-174) and every read flag, and one
command reaches any checkout with `--root`. A host with no shell runs `mrw mcp --enable` and calls `mrw_read` and
`mrw_write` (README, "MCP"); without `--enable` or `MRW_MCP=on` the server lists no tools (ADR-170). Register it only for the hosts that need it: a user-scope registration puts the
two tools into every project on the machine, including those whose instructions say to use the shell.

## 5. Check that it is happening

An instruction is a claim about behaviour; `mrw stats` is the measurement. It counts the plans this checkout
has been given and what became of them. One repository in the 2026-10-09 survey made mrw a hard rule in its
`CLAUDE.md` and had zero recorded plans. Zero is a reason to look, not proof by itself: a failed tally
write is not reported and `mrw stats --reset` clears the counts, so check the agent's transcript too. If
`mrw stats` stays at zero while the agent edits files, the trigger is likely not reaching it.

And check the binary is current: `mrw version --check-latest` exits 1 when a newer release exists (ADR-149).
Five of five Windows repositories in the same survey ran a release four versions old with no sign of it.

## 6. The read habits that cost, and what replaces them

From that agent's own review of a day of use (2026-10-22). Its writes were sound; these were not:

| Habit | Cost | Instead |
|---|---|---|
| piping mrw into `grep`, `head` or `tail` | the pipeline's exit code is the last command's, so a failed read or write reads as success | `--grep P`, `--max-lines N`, `--no-numbers`, or `cmd \| mrw read --grep P -` for another command's output |
| finding a JSON value by grepping its key and counting lines | wrong lines when the layout differs | a JSON path spec: `f.json:$["id"].markdown` serves that value |
| reading a whole JSON document, or piping it to `jq`, for a few fields | the whole document in context, and a path address of single-line JSON serves the whole line | `mrw read --json-path '$.a,$.b' f.json`, or `CMD --json \| mrw read --json-path '$.a,$.b' -`: one line per value. It licenses nothing, so read the path address before editing the value |
| reading before knowing how much matches | context spent on lines it did not need | `mrw read --grep P --count` first |
| a grep-and-cut pipeline to learn a line number, then a hunk on that number | two calls, and a number that goes stale | a pattern address, `@@ f.go /^func A/ replace …`, with `occurrence=N` when the start matches more than once |
| `sed -i` on a plan whose `body=` count was wrong | an edit outside the ledger | regenerate the plan from what you read. `body=` is an optional line count: leave it out unless the body needs one. Where it is needed and missing, the plan is refused with a hint, except for a body line that is itself a valid hunk header (`@@ b.txt 0 create`), which silently starts a new hunk; a body holding one needs `body=N raw=true` |
| a build, a test suite or a gate run as `CMD \| tail` (or `\| head`) | the pipeline's exit code is the last command's, and the output still costs context | `mrw run 'CMD'`: the output goes to a log and a four-line receipt (on Windows a read line per shell, labelled) carries the command's own exit code, the last line, the log's path and a read of it; look further with `mrw read --grep P -C 3 - < LOG` (ADR-177) |
| a whole-file read of a long Go or Markdown file to find one place | the whole file in context | `mrw read --outline f.go` first: the declarations or headings with their line ranges, licensing nothing; then the range you need (ADR-179) |
| `--grep` over many files when the first few matches decide | every match in context | `--count` first, then `--max-matches N`: the first N, the rest counted, exit 0 saying `capped:` when some remain (ADR-173) |
| a log of near-identical lines, or a Go file's comment runs | the repeats in context | `--fold` shows a run once, `--no-comments` shows a comment run as one line; both leave what they hide unlicensed, so read it without the flag before editing (ADR-179) |

### Queue code edits, then check the final tree

After reading the sites and preparing the plans, defer the automatic check on intermediate writes:

```sh
mrw write --no-check "$TMPDIR/first.plan" &&
mrw write --no-check "$TMPDIR/second.plan" &&
mrw check --full
```

Run these in the same checkout, with no intervening edits while the final check runs. `--no-check`
skips verification; it keeps the read guard and whole-plan validation. The queued edits remain
unverified until a check covering them passes. Fix a failed check and run it again after the last
edit; an earlier passing check does not verify a later write.

The example runs the whole-project check once. For a scoped check over existing edited paths,
name every queued path in `mrw check PATH...` and verify the command and any paths it reports
outside the scope. If the queue deletes or renames files, keep `--full`: a removed source path
cannot be passed to `mrw check`. A checked last write alone may cover only that write's files.

If the project declares no check and has no Go module, declare the project's checker or run its
required checks explicitly; mrw does not infer a Ruby checker
([README: Write, "Check by default"](../README.md#write)). Report the actual command, scope and
exit status, or say that verification is still pending.

## 7. What allowing mrw allows

mrw runs as its user, so a harness rule that allows it — `Bash(mrw:*)` in Claude Code, or its equivalent
elsewhere — allows:

- **writes to any file the user can write.** `--root DIR` names any directory, and mrw confines a path to
  the root it was given, not to a root anyone chose.
  A host's per-path rules for its Edit and Write tools do not reach a shell command, so `deny Write(.env)`
  or `allow Edit(src/**)` does not stop `mrw write`.
- **shell.** `mrw check` and `--then NAME` run what the project declares. `--then-sh 'CMD'` and `mrw run 'CMD'` run any command the caller writes and are off unless `MRW_SHELL=on` is in mrw's environment (ADR-174). With it unset, no command written on mrw's command line reaches a shell; that is not containment, since mrw writes any file its user can, `.quality-harness.json` and the test files a check runs included, so an agent allowed mrw can still change what the declared check runs and trigger it with a code write.

mrw's own refusals — a write into `.git`, into its state directory, through a link that leaves the root, to
a reserved Windows name — keep a plan honest, not an agent contained. The trigger and the skill move an
agent off the host's Edit and Write tools and onto mrw, so the agent is told the same in the skill: those
rules' absence is not permission. Contain an agent that uses mrw with the host's sandbox or the operating
system's permissions, or allow mrw only where you would allow every write (ADR-168 Decision 6). This says
what mrw does when it runs; how each host evaluates a rule for a shell command is the host's to document.

## 8. After a compaction, make the ledger forget

When a host compacts its context, the model keeps a summary, not the lines it was served, and the read ledger
still licenses them: a write to a line read before the compaction is allowed although the model no longer holds
it (measured once: 2 such writes in a 100K-window run, none broke a test). mrw cannot see a compaction; the
host can. `mrw seen --forget` empties the ledger of the checkout, read and written files alike, and
prints `forgot N file(s) in <checkout>`, naming the checkout top whose ledger it emptied (ADR-180); after it a write
needs a fresh read of the lines it touches (ADR-175). A root that does
not exist, or is not a directory, is refused, exit 2, and no state directory is made for it.

For Claude Code, run it when a session starts after a compaction or a clear. The hook reads the checkout
from its input's `cwd`, because `CLAUDE_PROJECT_DIR` stays on the project the session started in after a
move into a git worktree. It needs `jq`. In `.claude/settings.json` (or the user's):

<!-- ADR-175 hook -->
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "compact|clear",
        "hooks": [
          { "type": "command", "command": "d=$(jq -r '.cwd // empty'); [ -z \"$d\" ] || mrw --root \"$d\" seen --forget" }
        ]
      }
    ]
  }
}
```

What it costs and what it does not reach:

- **Every session in the checkout re-reads.** The ledger has no sessions, so a sibling session loses its licences
  too. That is a refusal, never a wrong write, and it is why nothing runs `--forget` on its own.
- **It clears what is persisted, not a command in flight.** A sibling's read or write that is already running may
  record after it.
- **It reaches the roots it can name.** The pending acknowledgements of an MCP server started at some other
  directory than the checkout's top or the hook's `cwd` are not forgotten, nor are the licences in other
  checkouts the session edited through `--root` or `cd`. A hook with no `cwd` in its input does nothing.
- **The re-read must show the lines.** After a forget, `mrw read` the lines (not --if-changed or --if-served, which answer an
  spec in one line and record nothing; not --quiet, which licenses lines without printing them).
  Neither puts the lines back in front of the model.
- **`--if-served` rides on `mrw seen --forget`.** `mrw read --if-served PATH=SHA` answers a spec by reference, `-- same as served: lines
  A-B at sha X`, when the file has the sha you quote and the ledger says every line was shown (ADR-179). The ledger is the
  checkout's, not the session's, and a line recorded as shown may never have reached the model (a host that truncates a long
  output, a read piped to `head`, a read redirected to a file). So use it only in a session alone in its checkout, with
  `mrw seen --forget` run on start, resume, clear and compaction (matcher `startup|resume|compact|clear`; the block above
  carries the compaction and clear pair ADR-175 pinned), and read under `--max-chars` below the host's output limit. Asking
  for the sha is what keeps a fresh session from matching: it holds no header to quote.
- **Plan makers need no re-read.** `mrw write --grep … --sub` and `--ast-grep … --rewrite` (and their MCP forms)
  compute each edit from the file's current bytes and license exactly those lines.

mrw ships no hook and installs none: the host's configuration is the user's. Other hosts need their own
equivalent (`docs/adr/BACKLOG.md` keeps the Codex and Cursor ones).
