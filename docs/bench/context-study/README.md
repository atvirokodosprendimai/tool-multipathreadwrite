# context-study — would a session working-memory map pay?

Measured 2026-10-22 (Vilnius) over the agent sessions on one Mac: Claude Code, Codex and the Cursor
IDE. The idea under test: mrw keeps short-lived handles to the spans it served, scoped to an epoch that
resets at context compaction, shared with subagents, and answers a re-read with a delta. Three questions
decide whether that pays:

- **M1** — of what the model writes, how much could a "no-retype" handle ever replace?
- **M2** — of what reads serve, how much is a re-serve inside one epoch (a delta read could answer it)
  versus across a compaction (the model had likely lost it)?
- **M3** — of what subagents are served, how much had the parent already been served (handles handed
  down could replace it), and how much did a sibling read first?

## Answer

Each number below is a share of all served text for that host. On Codex, read inference covers only
about 4 read-like commands in 10, so its M2 and M3 numbers are lower bounds.

| | Claude Code | Codex | Cursor IDE |
|---|---|---|---|
| **M1** no-retype ceiling (write bodies + retyped old text), share of reported output | **12.2%** (main) | **3.3%** | n/a: output not recorded (24.2% + 2.6% of *visible* output) |
| M1 reasoning, share of reported output | 64% (main, residual) | 48% (recorded) | n/a |
| M1 commands / prose, share of reported output | 13.0% / 5.5% (main) | 31.7% / 6.0% | 26.7% / 14.9% of visible |
| **M2** within-epoch re-serves (delta-read savable) | **3.8%** ≈ 1.20 M tokens | **2.5%** ≈ 0.15 M | **9.9%** ≈ 3.46 M |
| M2 cross-epoch re-serves (after a compaction) | 1.5% | 5.1% | 0.5% |
| **M3** subagent lines the parent already held, share of subagent served text | **5.4%** ≈ 1.03 M tokens | 13.2% (0.9 pp outside forks) | **4.1%** ≈ 0.54 M |
| M3 sibling overlap, share of subagent served text | **20.9%** | 17.3% | **25.3%** |

In plain words:

- **Output.** A perfect no-retype mechanism could touch at most about one output token in eight on
  Claude Code (main sessions), one in thirty on Codex, and about a quarter of the visibly typed output
  on Cursor. The rest is reasoning, commands and prose, which no handle replaces. Reasoning is the
  largest part everywhere it is measured: 64% on Claude (a residual, so an upper bound) and 48% on
  Codex (recorded). This is the same ceiling copy-study's `from=` lived under. copy-study measured
  0.026% for exact copies; this ceiling also counts every typed byte of new content.
- **Delta reads.** Re-reads inside one epoch are a small share of input: 3.8% on Claude, 2.5% on
  Codex, 9.9% on Cursor. Most re-reading happens before any compaction (Claude main within/cross =
  1.85 M / 1.91 M chars). Codex re-reads after compactions more than inside an epoch. 40 Claude, 50
  Codex and 44 Cursor sessions had a mid-session compaction, and 32, 29 and 35 of them re-read a line
  they had read before it. Re-serves are concentrated: the top tenth of sessions holds 66% (Claude),
  79% (Codex) and 82% (Cursor) of within-epoch re-serves.
- **Subagents.** 4–5% of what subagents read on Claude and Cursor are lines their parent had already been
  served; on Codex almost all of the 13% is inside **forked** subagents (1,355,315 of 1,448,961 chars),
  which already carry the parent's history in context: redundancy within one context. Fresh Codex
  spawns: 93,646 of 756,823 served chars (12.4%), from 180 subagents. The larger number is **sibling
  overlap**, 17–25% on every host: parallel subagents of one parent read the same lines. Neither is
  reachable by handing out handles: a subagent has its own context and has not seen its parent's or a
  sibling's text, so a pointer gives it nothing to reason over. What these numbers measure is duplicated
  reading across agents, which only the orchestrator can cut (split the work so readers do not overlap,
  or brief a subagent with the lines it needs), or a provider's prompt cache can make cheaper.

All token figures are chars/4 estimates unless the row says "recorded".

## Corpus

| | Claude Code | Codex | Cursor IDE |
|---|---|---|---|
| source | `~/.claude/projects/**/*.jsonl` over 10 KB | `~/.codex/sessions`, `~/.codex/archived_sessions` | `globalStorage/state.vscdb` (25 GB, read-only, immutable) |
| main sessions / subagents | 1,307 / 870 | 151 / 344 (164 forks, 98 fresh, 82 guardian reviews) | 610 / 1,148 composers |
| session trees | 1,307 | 151 | 984 (331 composers had no bubbles) |
| bytes read | 3.57 GB | 0.67 GB compressed (490 `.zst`, 5 `.jsonl`) | bubbles of 1,758 composers |
| subagent → parent linked | 672 by `toolUseId`, the rest to the session's main | 344 of 344 | 1,105 of 1,148 (43 parents absent) |
| tool calls | 125,287 | 28,677 | 711,012 (after dropping repeats; see below) |
| output tokens recorded | 80.8 M (main 72.8 M, coverage 99%) | 15.8 M, of which 7.7 M reasoning | 4.7 M on 7% of composers: unusable |
| mid-session compactions | 195 in 40 sessions (1 in a subagent) | 249 in 50 sessions | 104 in 44 composers |
| runtime | 65 s | 10 s | 124 s |

The run was one process under `nice -n 19`, streaming, with load near 15 on 10 cores; the runtimes
are wall clock under that contention. opencode has no data on this Mac (`~/.local/share/opencode` is
absent).

## Reconciling with copy-study's 5.28%

| | served chars | redundant chars | share |
|---|---|---|---|
| copy-study as published (its corpus, 2,188 transcripts) | 124,614,135 | 6,579,729 | 5.28% |
| copy-study's own scanner on today's corpus (`copy_study_scanner_today`) | 123,110,873 | 6,584,456 | 5.35% |
| this scanner, strict (copy-study's sources and rule) | 123,110,873 | 6,584,137 | 5.35% |
| this scanner, extended (adds inferred plain reads) | 127,026,954 | 6,734,648 | 5.30% |

- Published versus today: the corpus is live. 2,177 transcripts are over 10 KB now, against 2,188 then.
- Today's copy-study run versus this scanner's strict pool: the served text is identical. This scanner
  finds 319 fewer redundant chars because it also counts `cat > F <<EOF` and apply_patch as writes to
  F, which empty that file's pool. copy-study keeps those heredocs only as `--create` sources.
- Strict split: 4,670,844 within-epoch plus 1,913,293 cross-epoch makes 6,584,137.
- Extended adds 3.9 M served chars from plain `sed`/`cat`/`nl` reads. Their redundancy is lower,
  so the share falls slightly.

## Method

`scan.py` normalises each host's transcripts into one shape and then measures every host the same
way. The shape is typed output by category, plus an ordered stream of events: a line served (file,
line number, text), a write to a file, and a compaction. `scan.py` loads `../copy-study/scan.py` by
path and reuses it unchanged:

- the Read and mrw numbered-output parsers (`parse_read_result`, `parse_mrw_output`)
- the write-unit parsers (`write_units`, `bash_writes`, `parse_plan`) and the heredoc patterns
- path identity (`Session.fid`: two paths are the same file when equal, or when one is a `/`-suffix of
  the other)
- `chars`, `tokens` and `Reservoir`

The redundancy rule is also copy-study's (`Tracker` subclasses its `Session`). A served line is
redundant when the session was already served the same text at the same file and line, with no own
write to that file in between.

### The hosts

| host | messages and tool calls | output tokens | compaction | parent link |
|---|---|---|---|---|
| Claude Code | `assistant` content blocks (`text`, `thinking`, `tool_use`); `user` `tool_result` | `message.usage.output_tokens`, the last per message id (copy-study's rule) | `type:"system", subtype:"compact_boundary"` (195 records, each followed by an `isCompactSummary` user record; no `type:"summary"` records exist) | `agent-*.meta.json` `toolUseId` → the transcript that issued that tool call; otherwise the session's main transcript (workflow agents under `subagents/workflows/`) |
| Codex | `response_item`: `message` (assistant `output_text`), `reasoning` (summary text; the content is encrypted), `function_call` / `custom_tool_call` and their `*_output`, paired by `call_id` | `event_msg` `token_count` `info.total_token_usage`: `output_tokens` and `reasoning_output_tokens`. The last value minus what the first record carried in (`total − last_token_usage`), because 88 forks and resumes start with the parent's running total | `type:"compacted"` (with `replacement_history`) and `response_item` `compaction`. A record before the session's first tool call seeds a fork or resume and is not counted: 201 of 450 | `session_meta.payload.parent_thread_id` (or `source.subagent.thread_spawn.parent_thread_id`) |
| Cursor IDE | `composerData:<id>` gives the bubble order (`fullConversationHeadersOnly`). `bubbleId:<composer>:<bubble>`: `type` 2 is the assistant, with `text`, `thinking.text` and `toolFormerData` (`name`, `rawArgs`, `params`, `result`) | `tokenCount.outputTokens`: non-zero on 7% of composers and below the visible text, so not used | a bubble carrying `conversationSummary`, at its position. `cachedConversationSummary` (247 bubbles) is a summary prepared ahead and is not counted | `composerHeaders.value.subagentInfo.parentComposerId` |

Each Codex rollout is decompressed as a stream through the `zstd` CLI (`zstd -dc`) when that CLI is on
PATH, and through the `zstandard` module otherwise. With neither, each `.zst` file is counted as
`unreadable`. This run used the zstd CLI.

The Cursor database is opened with `mode=ro&immutable=1`, so the scanner takes no lock and writes
nothing. It reads only `composerHeaders`, the `composerData:` keys and a range scan of each composer's
`bubbleId:` keys. Bubbles outside the composer's conversation headers are dropped (834,174 of them).
On the 120 largest composers, 64% of those bubbles repeat a tool call that is in the conversation,
34% carry no tool call, and 2.7% are unique tool calls, most likely from reverted branches. A tool
call repeated inside the conversation counts once (9 cases).

### What counts as a served line

- **strict** — copy-study's sources only: numbered `Read` output and mrw read output (`==>` headers
  and `N|` lines), on any host. This is the pool the 5.28% was measured on.
- **ext** (extended) — strict, plus lines whose numbers a plain read command implies:
  - a single `sed -n 'A,Bp' F`, `cat F`, `head -N F`, or `nl -ba F` / `cat -n F` (optionally piped
    into `| sed -n 'A,Bp'`);
  - chains of `sed -n` or `nl -ba | sed -n` ranges joined by `&&`, `;` or newlines. Numbered chains
    are split by their line numbers. Plain chains count only when the output has exactly the number
    of lines the ranges ask for;
  - Codex JS cells whose output has one item per `exec_command` call;
  - Cursor `read_file` / `read_file_v2` results, numbered from `params.offset`. The offset is 1-based:
    on 120 composers, whole-file and offset reads of the same file agreed 12,024 times when the offset
    was read as 1-based and 451 times when it was read as 0-based.

### Output categories (M1)

Visible output is counted in characters, then estimated in tokens at chars/4:

- `prose` — assistant text to the user.
- `thinking_visible` — whatever thinking text the transcript kept. On Claude it is usually empty,
  with only a signature. Codex keeps the reasoning summary. Cursor keeps `thinking.text`.
- `write_new` — new file content typed into a tool call: `Edit`/`MultiEdit` `new_string`, `Write`
  `content`, mrw plan and `--create` bodies (copy-study's write units), heredoc bodies that feed
  `cat >`, `tee` or mrw, apply_patch `+` lines, and Cursor `edit_file_v2` `streamingContent`,
  `search_replace` `new_string` and `write` `contents`.
- `write_old` — existing text retyped to locate an edit: `Edit` `old_string`, apply_patch `-` and
  context lines, and Cursor `search_replace` `old_string`. A handle to a served span could replace
  these too, so the no-retype ceiling is `write_new + write_old`.
- `commands` — the rest of shell, search and read tool calls: Bash command text minus write bodies,
  Grep/Glob patterns, Read arguments, Codex `exec` cells and Cursor terminal commands.
- `subagent_prompt` — Agent, Task and `task_v2` input.
- `other_tool` — everything else typed into a tool call (todo lists, messages, MCP arguments), plus
  the JSON around the arguments.
- `reasoning` — Codex records it (`reasoning_output_tokens`, counted inside `output_tokens`). Claude
  transcripts keep thinking redacted, so for Claude it is a **residual**: reported output minus
  visible chars/4. Code and JSON tokenise denser than 4 chars per token, so the visible estimate is
  low and the residual is an upper bound on reasoning.

### M2 epochs

A compaction starts a new epoch. A redundant re-serve is classed by where the earlier serve of the
same line falls:

- **within-epoch** — the earlier serve came after the last compaction. The model still held the
  line, so a delta read could have answered "unchanged".
- **cross-epoch** — a compaction came between the two serves. The model had likely lost the line,
  so the re-read was needed, though a handle that survived in the summary could have pointed at it.

### M3 subagents

M3 looks at each line a subagent was served for the first time within that subagent:

- **Parent savable** — the parent had been served the same file, line and text before the subagent
  started, and the parent did not write to that file between that serve and the subagent's start.
- **Sibling overlap** — another subagent of the same parent was served the same line earlier in time.

Path identity is shared across the whole tree. Codex forks (`forked_from_id`) start with the parent's
history in context, so their overlap is reported separately.

## Limits

- **Claude subagent output tokens are unusable.** Their recorded `output_tokens` (7.9 M) fall below
  their visible text (9.7 M at chars/4), and 563 of 870 transcripts come out negative. M1 for Claude
  therefore uses main sessions. Main sessions show no negative residuals.
- **Codex read coverage is partial.** Of 4,901 read-like Codex commands, 2,119 got line numbers.
  Another 734 multi-call JS cells had outputs that could not be paired with their calls. Reads that
  are chained with `printf` separators or `|| true`, and reads of several files in one `sed`, are not
  inferred. Codex M2 and M3 are lower bounds. On Claude, Bash reads are rare: 7,767 read-like
  commands, beside the Read tool and mrw.
- **Cursor `read_file_v2` contents are sometimes absent.** 771 of 5,898 reads in a 50-composer sample
  had empty contents, and a read that failed (`Error: …`) serves nothing.
- **Cursor compaction is inferred.** Treating a `conversationSummary` bubble as the compaction point
  was not checked against Cursor's own code.
- Codex `replacement_history`, which carries the parent's history into a fork, is not parsed. Lines
  a fork holds from its parent are therefore not in its own pool, and its first re-read of them looks
  new to M2. M3 reports them under `parent_savable_chars_in_forked`.
- "Served" counts every numbered line in every tool result, once per serve. The same text staying in
  context across later turns is a caching question, not a re-read.
- Writes the scanner cannot see (`sed -i`, formatters, `git checkout`, `--sub` writes) do not empty
  the pool, as in copy-study.
- Every chars/4 estimate is approximate.

**Left out.**

- The Cursor CLI: 7 `store.db` files, about 1 MB in all. Messages are content-addressed blobs whose
  order is held in a protobuf root, which is more effort than 1 MB of data justifies.
- opencode, which has no data on this Mac.
- Windows transcripts, which this Mac does not have. The next section is for peers on Windows.

## Run it on another machine

The scanner is standard-library Python 3.11+, uses `pathlib` throughout, and runs no shell commands
apart from the optional `zstd`. It imports `../copy-study/scan.py`, so run it from a checkout of
this repository (or copy both `docs/bench/copy-study/scan.py` and `docs/bench/context-study/scan.py`,
keeping that layout):

```sh
python3 -I docs/bench/context-study/scan.py --out docs/bench/context-study/results.json
```

On Windows: `py -3 -I docs\bench\context-study\scan.py --out results.json`. The defaults find
`%USERPROFILE%\.claude\projects`, `%USERPROFILE%\.codex\sessions` and
`%APPDATA%\Cursor\User\globalStorage\state.vscdb`. Codex `.zst` files need `zstd` on PATH
(`winget install zstd`) or `pip install zstandard`. Without either, they are counted as `unreadable`
and the rest still runs. Use `--hosts claude,codex` to skip a host. If the Cursor database is too
large, use `--cursor-sample 0.25`, which takes a fraction of session trees chosen by a hash of the
root composer id.

**Share only `results.json`.** It holds aggregates only: no paths, file contents, project names or
session ids. `--examples` prints sampled cases to stderr for checking by eye. They carry no paths or
contents either, but keep them local.

## Self-test

`python3 -I docs/bench/context-study/test_scan.py` builds one synthetic session tree per adapter, with
known answers. Each tree has a re-read inside an epoch, a compaction, a re-read across it, a write
that ends redundancy, and subagents that re-read their parent's lines and a sibling's lines. The
Claude tree also covers a sed read and an Edit. The Codex tree covers a `.zst` subagent, a chain, a
two-call JS cell and a fork's carried-in usage. The Cursor tree is a SQLite database with an offset
read and a summary bubble. 32 checks. Breaking the Claude compaction matcher and the Codex JS-cell
matcher together turned 7 of 28 checks red. Breaking the chain splitter turned 2 of 32 red.

## Files

- `scan.py` — the scanner. `python3 -I scan.py --help` lists the flags.
- `test_scan.py` — the self-test.
- `results.json` — this run's aggregates per host, with `main`, `subagent` and `all` keys and a
  distribution block.
