# ADR-156: Whether mrw pays for itself is measured on agent runs, against a bar written first

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — "cut v1.66.0 when 155 lands, then do 156", after choosing both models, both task sources, mrw forced and the subscription only. The two choices the real runs waited on were made the same day: Zy chose "Your account, bypassPermissions" over a dedicated macOS user and over Docker, and did the one-time subscription login in the dedicated config directory (Decision 2).
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-020, ADR-118, ADR-141
**Governs:** `cmd/bench/**`, `internal/bench/**`, `bench/**`, `docs/bench/**`
**Enforced-by:** `internal/bench/oracle_test.go::TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit`
**Invalidates:** none — checked: ADR-020's harness and its pre-registration are untouched; `scripts/measure.sh` keeps its byte and call model; ADR-009's refusal of telemetry holds, because every receipt stays on this machine
**Served-path change:** none — nothing `mrw` reads, serves, refuses or applies changes; this record adds a third binary, `bench`, that runs an agent CLI as a subprocess and grades what it left behind

## Context

Nothing in this corpus says whether mrw makes an agent cheaper or more correct on real work.
- `scripts/measure.sh` models one side of it. It counts calls and input bytes for seven edit shapes, and it keeps the shapes where mrw loses (`scripts/measure.sh:27-32`). It says itself that output tokens are not measured, so its result is "an INPUT-side and round-trip result, not a total-cost one" (`scripts/measure.sh:22-25`). No agent runs in it.
- ADR-020 measures whether a plan addressed the line it meant to, with a planted target and no human judge (ADR-020 Decision 3), and it deliberately calls no model (ADR-020 Decision 1).
- The blind readings (`docs/blind/`) froze each plan before scoring.
- The 14-session survey of 2026-10-09 listed what nobody had reported: "tokens or calls saved, time per refusal, hook latency" (`docs/adr/BACKLOG.md`, "From the 14-session survey").

On 2026-10-14 Zy compared mrw with sgrep, a semantic search tool. sgrep publishes a SWE-bench Verified result: 21.3% less model spend at 70 against 68 solved out of 100. Zy asked how to measure mrw the same way, then chose:
- both Sonnet 5.5 and Opus 5.5;
- our own oracle-graded tasks and a SWE-bench Verified slice;
- mrw forced in the second configuration;
- runs on the Claude Code subscription only, with no API key.

**Audit of the class** — *a claim in this repository about what mrw saves an agent*: `mrw read --grep '(?i)saves|cheaper|fewer (calls|tokens)|token' README.md AGENTS.md` names README's "Measured" section, which quotes `scripts/measure.sh`, and AGENTS.md's "one read of every site, then one plan, then one write". Neither rests on an agent run. This record does not change either sentence; T6 changes README only as far as the bar supports.

## Existing Primitives Audit

- `scripts/measure.sh` — the deterministic model. It is kept, and T1's losing control tasks are chosen from its losing shapes so the agent runs can contradict it.
- `internal/curve`, `cmd/curve` (ADR-020) — planted ground truth, and a scorer that refuses results it did not ask for (ADR-020 Decision 2). `internal/bench` takes the same rule: an oracle grades by construction, never by opinion.
- `docs/adr/BACKLOG.md` pre-registrations (ADR-020's bar; ADR-119's at "Pre-registered for ADR-119") — the bar is written there before any harness exists, in its own commit.
- The `claude` CLI 2.1.299 (`claude --help`): `-p`, `--output-format stream-json`, `--model`, `--tools`, `--strict-mcp-config`, `--mcp-config`, `--settings`, `--setting-sources`, `--disable-slash-commands`, `--no-session-persistence`, `--no-chrome`, `--permission-mode`, `--append-system-prompt-file`, `--bare`; and the environment variable `CLAUDE_CONFIG_DIR`, which the binary reads (49 references in the Mach-O image). No model client is added: `bench` starts the CLI and reads what it prints.

## Decision

1. **Two configurations, one model at a time, one task at a time, each attempt in a fresh worktree.**
   - **A (baseline):** tools `Bash,Read,Edit,Write,Glob,Grep`. No mrw on the agent's `PATH`, no mrw MCP server, and no word about mrw in its instructions or setup. A task's own files may name mrw (the qamstack page describes it); configuration A still has nothing to run, and the canary, on a fixture task that names no tool, is where "no word" is checked.
   - **B (mrw forced):** tools `Bash,Glob` only — Read, Edit, Write, MultiEdit, NotebookEdit and Grep are not offered — with a pinned mrw build on `PATH` and `mrw instructions` appended with `--append-system-prompt-file`. A `PreToolUse` hook on Bash (`bench hook`) refuses a command that **writes** a file by other means: redirection to a file (`>&file` included), `tee`, `cp`, `mv`, `rm`, `touch`, `dd`, `truncate`, `patch`, `ln`, `install`, `sed -i`, `perl -i`, `gofmt`, `go fmt`, `git apply`/`checkout`/`restore`/`reset`/`stash`/`am`, and an interpreter writing a file, looking through variable assignments and wrappers such as `env`, `command` and `timeout` (`command -v` runs nothing and passes); quotes are read in one pass as the shell reads them, and a command substitution — bare, in double quotes, in an unquoted heredoc or in arithmetic, with quotes of its own — is judged as a command of its own, one nested too deep to read refused. A command that **reads** file contents through `cat`, `head`, `tail`, `less`, `nl`, `sed`, `awk`, `grep` or `rg` given a path passes and is **counted, not refused**, and the count is reported beside every verdict — Zy's decision, 2026-10-14, before any stage ran. It is a heuristic, so its refusals and counts are recorded per attempt. A write that slipped past it is found afterwards: every file the diff shows changed must have a content one of the attempt's own mrw ledgers recorded for it, each ledger read against the root its `root` marker names, and a file gone from the tree is a slip. A change made by other means and then read back with mrw is not caught: the ledger keeps what mrw saw, not who wrote it. Each slip is reported per attempt, and `bench analyze` sums B's slips beside every verdict. A slip check that cannot run — a ledger with no root marker, one bench cannot read, or two roots' ledgers holding different content for one changed file, which cannot say which mrw saw last — makes the attempt the registered `setup` exclusion, since bench could not grade it, never a count of zero. The hook is a measurement aid for a cooperative agent, not a security boundary: a further way past it is named under Out of Scope, not patched.
     - **A write outside the tree passes the hook** (Zy, 2026-10-14, after the uncounted pilot, "Allow outside the tree"): the pilot's hook refused `cat > /tmp/plan.txt` with a heredoc, a plan file for `mrw write PLAN` that never touched the tree, and the agent spent a call going round it. A redirection or a `tee` whose target is an absolute path landing outside the attempt's tree passes; `bench run` names the tree to the hook. A relative target is judged inside, since the Bash tool keeps a working directory of its own across calls that the hook cannot see; so is a quoted target, one with an expansion, a glob or a dot-dot (the shell resolves a dot-dot where a link before it leads), and one through a link into the tree or a dangling link. The tree is recognised by identity, so a case-folded spelling of it is the tree. The slipped-write check stays the backstop.
       An existing file outside the tree that is the same file as one in it, a hard link, is judged inside (the review of #413).
   - Neither configuration may use Task or Agent (no subagents), WebFetch or WebSearch.
2. **Isolation from this machine's own setup is proved before any measured run.** A `claude -p` here would otherwise load `~/.claude/CLAUDE.md` (which carries the mrw rule), the user's hooks (quality-harness, memoryd, agentsmemory), plugins, user-scope MCP servers (mrw among them) and auto-memory.
   - **`--bare` is rejected.** It skips hooks, plugins, auto-memory and CLAUDE.md discovery, but its help text says its auth is "strictly ANTHROPIC_API_KEY or apiKeyHelper … (OAuth and keychain are never read)", and Zy chose the subscription only.
   - **The candidate:** a dedicated, empty `CLAUDE_CONFIG_DIR` under `bench`'s state directory, with no CLAUDE.md, settings, hooks, plugins, MCP servers or memory. Zy logs in once with the subscription, then every attempt runs:
     `claude -p --output-format stream-json --verbose --model <id> --tools <configuration's list> --strict-mcp-config --mcp-config <empty> --settings <configuration's settings> --disable-slash-commands --no-session-persistence --no-chrome --permission-mode bypassPermissions`
     The CLI is started by absolute path, in a process group of its own that is stopped when it returns, so nothing it started outlives the attempt. Its environment carries `USER`, by which the CLI finds the subscription's credentials in the macOS keychain (the first real canary answered "Not logged in" without it). Its `PATH` is bench's: links to the host's `git`, `go`, `gofmt`, `node`, `npm`, `npx` and `python3` and the system directories, never the directories those tools sit in, and an attempt of configuration A is refused if `mrw` is anywhere on it. The tools past `git` and Go were added on Zy's word, 2026-10-14, after the uncounted pilot ("Keep, add node, declare it"): quality-harness's declared check (`bash scripts/selftest.sh`) runs node, and with none on the attempt's `PATH` configuration B's automatic check stopped at once with `node: command not found` (exit 127), a failure the harness made; `python3` is the host's, since the system one is older than that check's scripts expect. That check is red at the seven quality-harness tasks' pinned commit whatever the harness does — measured 2026-10-14 on this Mac against quality-harness `bbf82ff3`: in a full-history clone with the host's whole `PATH`, one test fails (`lifecycle.test.mjs`, "reported: a scratch corpus does not pull the record gates"); in the single-commit tree `bench` builds, two `orphan-sweep.test.mjs` cases that read git history fail as well; with only node added to bench's old `PATH`, nine fail, the rest from the older system python3 and a missing ruff. So those seven tasks carry `check_red_at_base`, a measured fact (T1 S6), each receipt copies it (T2 S7), and `bench analyze` prints a descriptive row without them beside each verdict (T6 S5). The config directory is resolved before use and refused if it is the user's own `~/.claude` by any spelling or link. `bypassPermissions` leaves the agent's Bash free outside the tree: Zy chose that, under their own account, on 2026-10-14, over a dedicated macOS user and over Docker; the canary's sentinel outside the tree is the check that nothing reached past it.
   - **The task tree keeps its instructions, without mrw in them, and loses its host configuration.** At any depth, `CLAUDE.md`, `CLAUDE.local.md`, `AGENTS.md` and `.cursorrules` are kept with every line that names mrw removed (the letters in any case, with no letter or digit on either side, so `mrw_read` counts and `smrw` does not), the same in both configurations. `.claude/`, `.cursor/` and `.codex/` are removed at any depth, and `.mcp.json` at the tree's root, because a project's settings, hooks and skills would load and run inside an attempt (nothing passes `--setting-sources`); a `.mcp.json` below the root is no host's configuration (`--strict-mcp-config` ignores even the root's) and is kept, since agentsmemory's tests read its plugin's `clients/claude-code/.mcp.json`. The result is committed as the attempt's base, and the oracle diffs against that commit. This replaced stripping every instruction file on Zy's word, 2026-10-14, after the uncounted pilot ("Redact, don't strip"): agentsmemory's tests read its `AGENTS.md`, so in a stripped tree configuration B's automatic check (ADR-054) ran the whole suite and failed for a reason the harness made, in configuration B only. Measured after the change in the tree `bench` materializes (2026-10-14): agentsmemory's declared check exits 0 on the unmodified tree and 0 on the prepared one. A repository's own instructions are part of its environment; only the word mrw is taken out, so configuration A still sees no word of mrw from them. This and the hook change above are fixes to the harness made before any counted run; the pilot is not counted, and the registered bar is untouched.
     - **Imports and links** (the review of #413 and its re-review): a file in the tree that a kept instruction file imports with `@path` is redacted the same way, up to five imports deep, since Claude Code loads it into the instructions too; an `@path` inside a code block or a code span is an example, not an import, and its target is left alone. An instruction file that is a link into the tree stays a link and its target is redacted; one that leads out of the tree, or into a directory preparation removes, is removed. A letter or digit of any script beside mrw makes another word (`émrw`).
   - **A canary proves it** (T2): one canary attempt per configuration. It passes only if:
     - configuration A's transcript (stream-json, `system/init` included) never contains `mrw`;
     - neither transcript contains a distinctive sentence of the user's `~/.claude/CLAUDE.md` (read at run time) or any hook's output;
     - `init` lists exactly the configuration's tools and no MCP server;
     - Zy's own session still authenticates afterwards.
   - **Stop condition:** if no configuration passes the canary with subscription auth, nothing is measured, and the record goes back to Zy.
3. **Tasks.**
   - **Own corpus, 30 tasks.** Each is pinned to a commit of one of four public repositories on this machine, listed with their URLs in `bench/repos.json`: agentsmemory and crossagentschat (Go), quality-harness (JavaScript, YAML, markdown) and the atvirokodosprendimai.github.io site (HTML). Not this repository: an agent in configuration A could build mrw from its source. Private repositories on this machine are excluded, so another reader can rerun the corpus. Three more tasks of the shape `fixture` carry their base tree in the corpus and need no repository; they are never measured: CI runs their oracles, and `canary` is the canary's task. The shapes:
     - 8 multi-site renames or signature changes across files;
     - 6 structural multi-line edits in YAML, templates or markdown;
     - 6 edits whose obvious anchor repeats in the file;
     - 4 single-site edits in large files;
     - 6 single-line, single-file edits as the control where `measure.sh` says mrw loses.

     A task is an instruction, a base commit, allowed regions per file, a reference fix and an oracle. The allowed regions are the lines the fix itself changes; a task's slack widens where lines may be inserted and never licenses changing a line the fix leaves alone. A task marked exact must leave each touched file equal to the reference result, so a rename whose call site is commented out, or kept inside a block comment, fails though every token survives. A change git sees as binary, a mode change or a deleted file is outside. The oracle must:
     - fail on the base;
     - pass on the reference fix;
     - fail on any change outside the allowed regions, the line just before the fix's first change included.

     A test proves those three properties for every task without an agent (T1).
   - **SWE-bench Verified slice, 20 tasks.** The 20 instances with the lowest `sha256("adr-156:" + instance_id)`, chosen before any run. They are graded by the official harness at the version T5 records first, one container at a time.
4. **What is measured, from what the CLI and the oracle report.**
   - **Primary:**
     - *solved* — the oracle exits 0;
     - *silent mis-edit* — the run ended reporting success while the tree differs from the base outside the allowed regions, or a touched file fails its structural check. This is the failure mrw exists to prevent. It is not measured on the SWE-bench slice, which has no allowed regions.
   - **Secondary:**
     - *cost* — the `total_cost_usd` the CLI reports in its stream-json result, computed from token counts, since the subscription bills nothing per run;
       A subscription run does report that field: the first passing canary (2026-10-14, `docs/bench/canary-2026-10-14/canary.json`) recorded $0.0170 for A and $0.0373 for B. Had it been absent or zero, cost would have been computed from the token counts at the per-token prices published on the day T3 starts.
     - input, output, cache-read and cache-write tokens;
     - tool calls;
     - wall time;
     - B's hook refusals and slipped writes;
     - the CLI version, model id and mrw version, recorded per attempt.
5. **The bar is the one registered in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-156", before `bench` or any task existed.** It gives one verdict per model and per task source — PAYS, LOSES or INCONCLUSIVE — never pooled. All three are published.
   - **The registration overlapped in one place, found while implementing it (2026-10-14):** a solved-rate interval that lies wholly below 0 with its lower bound at −5 points or above met PAYS's solved condition and LOSES's at once. **Zy decided, 2026-10-14, before any stage ran: LOSES wins.** It is written into the registration as a clarification, not a change to the bar's numbers, and `bench analyze` checks LOSES first.
   - **A descriptive row beside the verdict, never in its place (Zy, 2026-10-14, before any counted run):** where a model's and source's attempts include tasks flagged `check_red_at_base`, `bench analyze` prints a second row computed without them and marked descriptive. The verdict is the first row, over every task, exactly as registered; the bar in `docs/adr/BACKLOG.md` is not changed, and `docs/bench/RESULTS.md` publishes both rows, naming the seven tasks and the red check as a fact about quality-harness at that commit, not about mrw.
6. **Receipts stay here (ADR-009).**
   - Each attempt writes one JSON receipt under `docs/bench/<date>-<stage>/`, linked into place whole, so an interrupted write leaves no half a receipt. `bench` refuses to overwrite an earlier run's receipts, and resumes without repeating a started attempt. An interrupt to `bench` stops the stage: the attempt in flight gets no receipt, stays started and is never repeated. An attempt the registration excludes gets a replacement under the id `<attempt>.r1` (and `.r2` for a start error), every receipt kept, and a stage verifies only with each replacement it is owed. A start error is an attempt that never opened a session, or that erred before its first tool call naming authentication, a rate limit or a quota; any other error — a CLI that exits non-zero after a success result included — is an unsolved attempt with its cost counted.
   - Transcripts are kept beside their receipts. Before T3, T2 checks whether they hold source text from a non-public repository; there should be none, since private repositories are excluded.
7. **Stages, each started only on Zy's word:**
   - **Stage 1 (T3), Sonnet on our own tasks:** a pilot of 5 tasks × 2 configurations × 1 (not counted), then 30 × 2 × 2.
   - **Stage 2 (T4):** the same tasks on Opus.
   - **Stage 3 (T5):** the SWE-bench slice on both models, × 2 × 1.

   Under the subscription each run is rate-limited, so a stage may take days. Runs are sequential; `costly-runs` applies, and to Stage 3's containers above all.

It fails if an oracle grades a reference fix unsolved or a stray edit solved, if configuration A ever sees mrw, if a verdict is computed from attempts the registered rules did not admit, or if README states more than the verdict.

## Alternatives Considered

- **SWE-bench only, as sgrep did.** Rejected as the only source: it mostly measures whether the model can solve the bug, where editing is a small share of the cost, and it cannot see a silent mis-edit. It is kept as Stage 3 for comparability.
- **`--bare` with an API key.** It isolates cleanly, but Zy chose subscription only. It returns as the fallback if the canary fails, on Zy's word.
- **mrw allowed rather than forced.** More realistic, but configuration B would measure whatever mix the agent chose. Zy chose forced; the hook's counts say how forced it was.
- **Extend `scripts/measure.sh`.** Rejected: it is a deterministic model and stays one. Agent runs are a different instrument with a different failure profile.
- **Grade by a model or a human.** Rejected: ADR-020's rule is a planted answer and no rater.

## Component / Boundary Impact

A new `cmd/bench` (the runner, entry point; `run`, `canary`, `verify`, `analyze`, `hook`, `swebench-select`) and `internal/bench` (task loading, the oracle, the runner and its receipts, the hook, the canary's checks, the analysis), standard library only. `go.mod` keeps one requirement. Tasks live in `bench/tasks/`, the repositories they come from in `bench/repos.json`, results in `docs/bench/`. No engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench` CLI: `run`, `canary`, `verify`, `hook`, `swebench-select`, `analyze` | new binary | T2, T5, T6 | the operator (Zy); `hook` is called by Claude Code in configuration B |
| `bench/tasks/<id>/`, `bench/repos.json` | task, reference fix, oracle; the repositories by URL and local path | T1 | T2, T3, T4 |
| receipt JSON per attempt, `run.json`, `canary.json` | new local formats, listed in `docs/bench/README.md` | T2 | T6 |
| `docs/adr/BACKLOG.md` | the pre-registered bar | this record | T6 |
| README | a claim only as far as the verdict goes | T6 | readers |

## Inter-task Contracts

- T1 produces `bench/tasks/` and `internal/bench.Oracle`; T2 consumes them.
- T2 produces `bench run`, the receipt format and the canary; T3, T4 and T5 consume them.
- T3, T4 and T5 produce receipts; T6 consumes them.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** the first measurement of whether mrw pays for itself, with its failure published as readily as its success.
- **Negative:** subscription runs share Zy's rate limits and take days. Stage 3 needs Docker and is heavy. The task corpus is ours, so it is biased toward the shapes mrw was built for, and the control tasks are the only check on that.
- **Neutral:** one agent policy (Claude Code) and two models. The result says nothing about other hosts.

## Out of Scope

- Other agent hosts — Codex, Cursor, Claude Desktop (deferred: docs/adr/BACKLOG.md, "Pre-registered for ADR-156", other hosts)
- mrw over MCP rather than the CLI (deferred: docs/adr/BACKLOG.md, "Pre-registered for ADR-156", the MCP arm)
- Tuning mrw during a stage (permanent: boundary: the mrw build is pinned per stage; a change made between stages is named in the next stage's receipts)
- Private repositories on this machine (permanent: boundary: a corpus another reader cannot rerun is a claim, not a measurement)
- Closing every way past configuration B's write hook (permanent: boundary: `bench hook` reads shell by heuristic, as a measurement aid for a cooperative agent, never as a security boundary. Known ways past it: a program not in its table (`make`, `go run`, a script of the agent's own), shell it does not model (`eval`, `sh -c`, `bash -c`, `source`, functions, aliases, process substitution, a command after a control keyword such as `if true; then touch f; fi`), a command name spelled with quotes (`"touch" f`, `t""ouch f`), an interpreter write its pattern does not match, a second redirection written against the first with no space between (`printf x >/tmp/p>tree/f`, whose second target the scan does not see), and a `tee` operand that starts with a dash after `--` (`tee -- -f`, taken for an option). Four review rounds of PR #411 found nested and arithmetic substitution and heredoc quoting, all patched; the fifth found the control-keyword and quoted-name classes, and the review of #413 the adjacent-redirection and dash-operand classes, recorded here and not patched. `slipped_writes` — the attempt's tree judged against its own mrw ledgers after the attempt, summed by `bench analyze` beside every verdict — is the backstop that counts a write that got past. A further way past the hook found by review is added here, not patched: decided in the review of #411, 2026-10-14)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Configuration A sees mrw through the user's setup | Medium | High | the canary (Decision 2) gates every stage; a receipt carries the `init` tool and MCP list |
| A login in the dedicated config directory disturbs Zy's own session | Unknown | High | the canary checks Zy's session after; on any doubt, stop and ask |
| B's hook misses a write by other means | Medium | Medium | the hook is a heuristic by design (Out of Scope); slipped writes are found from the ledger after each attempt, reported per attempt and summed by `bench analyze` beside every verdict |
| Too few tasks to see a difference | High | Medium | INCONCLUSIVE is a registered, publishable outcome; the bar is non-inferiority, not equality |
| Subscription rate limits truncate a stage | High | Low | resumable runner; the exclusion rule re-queues a rate-limited start |
| SWE-bench x86 images are impractical on this Mac | Medium | Medium | T5 starts with a feasibility probe (one gold patch graded) and stops if it fails |
| Configuration B's automatic check is red at base in a repository, so B pays a failing check A never runs | High (quality-harness's seven tasks, measured 2026-10-14) | Medium | the tools that check needs are on the attempt's `PATH`, so it runs as it would for any agent on this machine; each such task declares `check_red_at_base` and its receipts carry it; `bench analyze` prints a descriptive row without those tasks beside the verdict, which stays over every task (Zy, 2026-10-14) |
| An agent under `bypassPermissions` with Bash touches files outside its worktree, on Zy's own machine | Low | High | Zy chose, 2026-10-14, to run attempts under their own account with `bypassPermissions` rather than as a dedicated macOS user or in Docker; the canary checks a sentinel file outside the worktree is untouched, and every attempt runs in a scratch copy |

## Rollback

Delete `cmd/bench`, `internal/bench`, `bench/` and `docs/bench/`; the BACKLOG registration stays as history. Nothing in mrw depends on them.

## Follow-ups

- Rerun on the next minor release of mrw if the verdict is PAYS or LOSES, to see whether it holds.
