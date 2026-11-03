# ADR-193: Claude hosts are taught to edit in search_replace blocks, if a paired pilot says it pays

**Status:** Proposed
**Date:** 2026-11-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-051, ADR-141, ADR-156, ADR-162, ADR-168, ADR-171, ADR-190, ADR-192
**Governs:** `internal/bench/**`, `cmd/bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestConfigurationSIsBPlusTheEditParagraph`
**Invalidates:** none — checked: without `--profile stage8` every configuration's arguments, tools, prompt and refusal text stay as ADR-162, ADR-163 and ADR-192 left them, so Stages 1 to 7 stay reproducible; `mrw instructions` and `mrw instructions --core` without `--host` stay byte-identical (T5's test and Invariants), so ADR-141's bound and ADR-192 T1's invariant "B's prompt under stage7 is `mrw instructions` alone" hold
**Served-path change:** None until T5, and T5 lands only on a SHIP verdict: then `mrw instructions --host claude` (and `--core --host claude`) print today's text with the measured paragraph after it, and the Claude Code trigger in `docs/agents.md` names the flag; without `--host` nothing a caller sees changes.

## Context

**The direction (Zy, 2026-11-03, *"niiice, lets go, i think it is good direction"*).** mrw is the guarantee layer —
a plan applied whole or not at all, a per-line read licence, a verdict per edit, paths that mean what they say — and
it should reach each model in the edit format that model was trained on. mrw already compiles two foreign formats into
its native plan (ADR-051): `--format=search_replace` (Aider SEARCH/REPLACE blocks) and `--format=apply_patch` (Codex
`*** Begin Patch`). Since ADR-190 (`1f01dcdc`, in v1.83.0) a SEARCH block that quotes every line it replaces, exactly
and matching once, needs no prior read, and a multi-line block whose SEARCH and REPLACE end in the same kept line meets
ADR-052's line-after rule by that quote.

**What agents are taught today.** `mrw instructions --core` (`internal/guide/guide.go`, `Core()`, 298 words on
v1.83.0, measured 2026-11-03 with `wc -w`) teaches only the native `@@ PATH N-M replace anchor=` hunk (rule 4). The full
`mrw instructions` (`CLI()`, 4,294 words) names `--format=search_replace` once, as the form for a file seen "with your
host's own tools" or after a compaction, and does not name apply_patch. Bench's configuration B appends the full text
(`internal/bench/runner.go:1161`); `docs/agents.md` §1 tells real hosts to run `--core`.

**External evidence.**
- Cursor, "Continually improving our agent harness", 2026-04-30, https://cursor.com/blog/continually-improving-our-agent-harness
  (read at the primary source 2026-11-03): each model is given the edit tool format it had in training — OpenAI models
  a patch format, Anthropic models string replacement — and an unfamiliar format costs extra reasoning tokens and
  produces more mistakes. A vendor's report about its own harness, not a controlled study.
- arXiv 2607.10569 (Yang, Yu, Desell, 2026-07-12): the same tool-surface change cost Claude Code +14.4% (not
  significant) and made Codex significantly cheaper, with pass rates tied. A tool's effect is agent-specific, so a
  change for Claude is measured on Claude.
- arXiv 2607.12161 (Weinberger, Hozez, v5 2026-08-12): token reduction is not cost reduction (r = 0.15); judge cost per
  success. That is why measure (iii) below is cost per success and (ii) is only one of two ways to pass.
- SWE-Edit (arXiv 2604.26102, read second-hand): 93.4% edit success for native str_replace. Its denominator and harness
  differ from bench's, so it is context, not a baseline.
- memoryd `tool-multipathreadwrite` `56caf49e` (2026-10-30, a Codex session): "investigate a native-looking editing
  interface backed by existing MRW validation/atomicity".

**Bench evidence (Stage 5, ADR-162: mrw v1.68.0, Claude Code 2.1.302, 191 SWE-bench Verified instances).**
- **The registered result** (`docs/bench/stage-5/looks/look-5.txt`): B against A is INCONCLUSIVE on both models — Opus
  5.5 cost −19.4% [−25.5, −12.7], solved −0.5 pt [−6.7, +5.5]; Sonnet 5.5 cost −7.0% [−12.1, −1.8], solved −2.6 pt
  [−8.7, +2.8]; 99%, five looks.
- **Post-hoc, not registered** (the coordinator's analysis of 2026-11-03): cost per success on the both-solved subset,
  B against A, Opus −23.9% [−28.8, −18.8] n=158, Sonnet −9.8% [−17.0, −2.4] n=125 (percentile bootstrap over
  instances, 4,000 resamples, one run per arm). Context for the size of effect a guide change might move; never a
  verdict.
- **B's write calls, counted 2026-11-03 over Stage 5's stored B transcripts** (`~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5/block-*/`,
  191 per model) with the draft of the rule T2 freezes (Decision 4): Opus 547 decided calls, 87 refused (15.9%: 76 a
  plan that did not parse, `plan has N error(s)`; 11 a validation refusal, `nothing was written`), 6 undecided, 17
  blocked by bench's own Bash hook; 66 of 191 attempts with a refusal. Sonnet 349 decided, 96 refused (27.5%: 80
  parse, 16 validation), 17 undecided, 11 blocked; 70 of 191 attempts with a refusal. **No B attempt on either model
  used `--format=search_replace` or `--format=apply_patch`.** Output tokens per landed write call: Opus 1,614, Sonnet
  1,299. The brief's figures for the same transcripts, 32 of 578 (5.5%) Opus and 28 of 384 (7.3%) Sonnet, came from a
  rule nobody recorded; they are not reproduced here, and this record cites only the counts its own rule produces.
  The larger bucket — plans that did not parse (`body=` counts, header shape) — is exactly what a SEARCH/REPLACE block
  has no syntax for.
- **ADR-192 T3's lesson** (`docs/bench/stage-7/pilot.md`, 2026-11-03): a permissive sentence ("you may also read files
  with the Read tool") moved 0 of 4 attempts; a directive one moved 4 of 4. A variant's wording must be directive.

**Hosts' environments.** Claude Code sets `CLAUDECODE=1` (and `CLAUDE_CODE_ENTRYPOINT=cli`) in the processes its tools
start: observed 2026-11-03 in this session's environment. Nothing in this repository records what Codex sets
(`git grep CODEX_HOME` finds only a fixture patch, `bench/tasks/am-wide-claude-kit/reference.patch:529`).

## Existing Primitives Audit

- **Reuse:** `ingest.CompileSearchReplace` and ADR-190's quote licence, unchanged; no engine package changes.
- **Reuse:** ADR-192's mechanism for a configuration that is B plus fixed text after `mrw instructions`
  (`HSentence`, `internal/bench/profile.go`; the append at `internal/bench/runner.go:1166`; its SHA-256 in `run.json`,
  refused when a block names another, `runner.go:348`), its canary, and its uncounted pilot shape (ADR-192 T3).
- **Reuse:** `scripts/bench_mechanism.py`'s `mrw_calls` and `result_text`, imported, not copied, by the new counting
  script. Checked 2026-11-03: `mrw_calls` reads a SEARCH/REPLACE document written by a quoted heredoc to `$TMPDIR`
  then `mrw write --format=search_replace FILE`, or piped to `mrw write --format=search_replace -`, as one `write`;
  and bench's Bash hook built from `8c9bdef8` (`bench hook --tree`) passed both shapes and an unquoted heredoc, while
  it refused `echo x > f.txt` and `echo x >> REPLACE` in the tree. The `<<<<<<<` and `>>>>>>>` marker lines inside
  a heredoc are text to both. T1 and T2 pin this with tests and T3's preflight meets the real CLI.
- **Reuse:** bench's solved rule (the oracle passed, within budget, with no agent error) and its exclusion of setup
  failures.
- **Not reused:** `bench look` and `bench analyze`'s sequential verdict (ADR-156's bar at 99% over five looks): a pilot
  of 18 instances has no looks; its rule is Decision 5's, computed by the frozen script.
- **Class audit — the places that tell a Claude host how to edit with mrw.** `git grep -lE 'guide\.(Core|CLI|Shared|Skill)\(' -- '*.go' ':!*_test.go'`
  run 2026-11-03 lists three files, the surfaces that print the guide: `cmd/mrw/main.go` (`instructionsCmd`, `mrw
  instructions` and `--core`), `cmd/mrw/skill168.go` (`mrw skill`, whose `guide.Skill` embeds `Core()`) and
  `internal/mcp/instructions.go` (the MCP handshake, `guide.Shared`); the prose copies are `docs/agents.md` §1, `AGENTS.md`, `README.md`
  and the memoryd `craft` skill mirror. T5 changes the first, behind a flag, and `docs/agents.md` §1's Claude Code
  trigger. Left out on purpose: `mrw skill` (deferred: Out of Scope), the MCP handshake (a host with a shell drives the
  CLI), and AGENTS.md's and README's per-flag prose, which T5 extends by one line each.

## Decision

1. **Configuration S is B plus one directive paragraph after `mrw instructions`.** Tools `Bash, Glob` (and `Skill`, as
   the profile offers every configuration), as B. The prompt is `mrw instructions` exactly as B gets it, a newline, then
   this paragraph, registered now:

   > In this session, make every edit to an existing file as a SEARCH/REPLACE block and apply it with mrw write
   > --format=search_replace FILE. A block is the file's path on its own line, then <<<<<<< SEARCH, the file's current
   > lines exactly as they are, =======, the lines that replace them, and >>>>>>> REPLACE; one FILE may hold many blocks
   > for many files. SEARCH must match exactly once. End SEARCH and REPLACE with the same unchanged line, and the block
   > needs no mrw read of the lines it quotes. Use a native @@ hunk only for an edit by line number, and mrw write
   > --grep P --sub R for one change at many sites.

   Its SHA-256 is in `run.json` (`edit_paragraph_sha256`) and a block naming another is refused. T3 may change it once,
   before any counted attempt, when its smoke shows no SEARCH/REPLACE write on either model, and never after. New files
   stay `mrw write --create`, as the instructions say.
   - **Honest about the fit.** Claude's trained edit tool is a JSON call, `str_replace` with `old_str` and `new_str`.
     Aider's block is not that syntax; its semantics are the same — an exact, unique match of the old text, replaced by
     the new. S tests whether the semantics carry the benefit through a text syntax; it does not test a JSON front end
     (Alternatives).
   - **Why the closer clause is in the paragraph.** Without it a multi-line block on an unread file is refused by
     ADR-052's line-after rule (v1.83.0 prints "end SEARCH and REPLACE in the same line, which keeps it and quotes
     it"), and refusals would move from `body=` to that rule; the pilot would then measure the wording's gap, not the
     format.
   - **What the native plan keeps.** Line-addressed edits, generated plans (`--sub`, `--rewrite`, `from=`) and
     many-site edits stay native; the paragraph says so.
2. **Profile `stage8`: stage5 plus configuration S, which runs only there.** Under `stage8` a list names only B and S,
   every model that runs S runs B, and B's arguments, tools, prompt, settings and refusal text equal stage5's. A, H, P
   and the side arms are not run: the question is S against B.
3. **The Claude variant is selected by an explicit flag, `mrw instructions --host claude`, not by the environment.**
   - **(a) By environment (`CLAUDECODE=1`) — rejected.** Bench builds B's prompt with `exec.Command(o.Mrw,
     "instructions")` (`runner.go:1161`), which inherits bench's environment, and bench is launched from Claude Code
     sessions: this one carries `CLAUDECODE=1`. An environment-switched `instructions` would silently turn every B into
     S and break ADR-192 T1's invariant. It would also make the output depend on an input the caller does not see, and
     nothing here records what Codex sets.
   - **(c) Only `docs/agents.md` and the skill — rejected.** The paragraph would be a prose copy no test holds to the
     binary, the drift ADR-141 exists to prevent.
   - **(b) The flag — chosen.** Deterministic for a given input, testable by the same tests that hold `Core()` (ADR-141
     Decision 2: every flag the text names is a flag of the binary), and absent unless asked for. `--host` takes only
     `claude`; any other value is a usage error, exit 2, naming the values it takes. `--host claude` prints `CLI()`, a
     newline and the paragraph T3 froze — byte for byte the prompt S ran — and with `--core` prints `Core()`, a newline
     and the same paragraph. ADR-141's 300-word bound stays on `Core()`; the host form is that plus the paragraph.
   - **Not measured: the `--core` form.** The pilot appends to the full text, as B does; `docs/agents.md` tells hosts to
     run `--core`. Shipping the same paragraph after `Core()` is an extrapolation, named in Risks; Zy may instead ship
     the full form only.
   - `mrw skill`, which embeds `Core()`, takes no `--host` here (Out of Scope).
4. **Counting is a frozen script, `scripts/bench_edits.py`, committed before the first counted call.** Per transcript:
   - **A write call** is one Bash tool call whose command tokenizes (`bench_mechanism.mrw_calls`) to at least one `mrw
     write` that is not `--dry-run`, counted once per tool call however many writes it chains.
   - It is **blocked** when its result holds `PreToolUse:Bash hook error` (bench's own hook refused it); else **landed**
     when the result holds `— applied` or `PARTIALLY APPLIED`; else **refused** when the result holds `nothing was
     written` (any case), `plan has N error`, `search_replace: `, `apply_patch: ` or `flag provided but not defined`;
     else **undecided**. Blocked and undecided are reported and left out of the rate.
   - **A SEARCH/REPLACE write call** is a write call one of whose writes carries `--format=search_replace` or `--format
     search_replace`.
   - Output tokens and cost come from the receipt (`usage.output_tokens`, `total_cost_usd`), solved from bench's rule.
5. **The pilot, pre-registered here, before any code or run.**
   - **Fixed:** Claude Code 2.1.317 (`~/.local/share/mrw-bench/claude-2.1.317`), mrw v1.83.0's release asset (as ADR-192
     T3 ran it, checksum against `SHA256SUMS.txt`), `claude-opus-5-5` and `claude-sonnet-5-5`, one attempt per arm per
     instance per model, bench's default 20-minute budget.
   - **Instances:** `bench/swe-bench/selection.json` in file order, from the third on — the first two were ADR-192 T3's
     pilot, and T3 here uses them for an uncounted smoke — which is 18: `pytest-dev__pytest-5631`, `django__django-16642`,
     `sphinx-doc__sphinx-8265`, `scikit-learn__scikit-learn-25747`, `matplotlib__matplotlib-23314`, `sympy__sympy-18199`,
     `sympy__sympy-24661`, `django__django-11532`, `django__django-14011`, `scikit-learn__scikit-learn-12585`,
     `matplotlib__matplotlib-25775`, `django__django-14434`, `django__django-16116`, `django__django-11555`,
     `astropy__astropy-14539`, `django__django-11451`, `django__django-14404`, `django__django-13807`. None is in the
     191-instance pool (`selection-adr-162.json`; checked 2026-11-03, 0 overlap). ADR-156's Stage 3 ran them on an older
     mrw; that does not bear on a paired comparison run now.
   - **Exclusions:** an attempt bench excludes as a setup failure is run once more; excluded again, its instance drops
     from both arms for that model. Timed-out and over-budget attempts count, unsolved, and their write calls count.
   - **Measures per model and arm,** over the counted attempts: (i) **refused-write rate**, refused over landed plus
     refused, summed over attempts; (ii) **output tokens per landed write**, output tokens summed over attempts over
     landed write calls summed; (iii) **cost per success on the both-solved subset**, S's summed cost over B's summed
     cost minus 1 on the instances both solved, with a percentile bootstrap 95% interval over those instances (4,000
     resamples, seed 193). Secondary, reported, never gating beyond (U) and (a): solved count, turns, attempts with a
     refusal, undecided and blocked calls.
   - **Rule per model.** (U) **UNTESTED** when fewer than half of the counted S attempts, rounded up (9 of 18, 8 of
     16), make at least one SEARCH/REPLACE write call. Otherwise **PASS** when all of: (a) S solved at least B solved
     minus 2; (b) S's (i) at most two thirds of B's, **or** S's (ii) at most 0.85 of B's; (c) (iii)'s point estimate at
     most +10%. Otherwise **FAIL**.
   - **The verdict:** **SHIP** when both models PASS; otherwise **NO SHIP**, and the counts are recorded and nothing in
     mrw's guide changes. A mixed result is NO SHIP: a larger run is Zy's call (Out of Scope).
   - **Prediction, written now:** uptake holds on both models (the paragraph is directive); (i) falls, since the parse
     bucket is most of B's refusals and SEARCH/REPLACE has no `body=` or header to get wrong, while exact-match misses
     are its own new failure; (ii) rises or holds, since a block carries the old lines as well as the new where a
     native hunk carries an anchor; (iii) within ±10%; solved within the margin. So the likely route to PASS is (b) by
     (i), and SHIP is plausible but not expected with confidence.
   - **What it cannot tell.** About 52 decided write calls per arm on Opus and 33 on Sonnet (Stage 5's B rates, 2.9 and
     1.8 calls per attempt): at 16% and 27% refused, a binomial standard error near 5 and 8 points, so a drop to two
     thirds is about one standard error on Opus. A solving margin of 2 in 18 cannot see a loss smaller than about 11
     points. The both-solved subsets will hold roughly 8 to 14 instances, so (iii)'s interval will span tens of percent;
     only its point estimate gates, which is a direction, not a verdict. A SHIP from this pilot is evidence enough to
     change a flag-selected text; it is not evidence for a README claim.
6. **Price, said first.** Stage 5's B per attempt (measured 2026-11-03 over its 191 receipts per model): Opus $0.200 and
   68 s, Sonnet $0.053 and 25 s. The counted pilot is 18 × 2 arms × 2 models = 72 attempts, about $9.1 and 56 minutes of
   attempt time; T3's smoke (B and S, 8 attempts) and canaries add about $1.5. If the counted pilot's spend passes $15,
   it stops and goes to Zy.

What would make it FAIL:
- an S whose init tools are not B's, whose prompt is not `mrw instructions`, a newline and the frozen paragraph, or whose
  paragraph's sha differs from T3's;
- a B under stage8 that differs from stage5's B in arguments, tools, prompt or refusal text;
- a counting rule, threshold, instance list or exclusion rule changed after the first counted call;
- an S reported PASS on a model whose uptake was below 9;
- a T5 that ships text other than the paragraph S ran, or changes `mrw instructions` without `--host`.

## Alternatives Considered

- **A native JSON `str_replace` front end** (an MCP tool or a `--format=str_replace` taking `{path, old_str, new_str}`).
  Closest to Claude's trained call. Not chosen now: it is a new surface with its own parser and receipts, and bench
  drives Claude Code through Bash, where a JSON document is no closer to the trained call than a text block; if S
  passes, it is the follow-up that would close the syntax gap (Out of Scope).
- **Change the core's rule 4 first, then measure.** Rejected: the guide is every adopter's served path, and ADR-192
  set the precedent of measuring before changing it.
- **Select by environment.** Rejected in Decision 3: it would change B in bench.
- **Run under stage7 beside A, B and H.** Rejected: `stage7Problem` holds stage7 to A, B and H (`runner.go`), and its
  Stage 7 is a paid sequential run whose arms must not move.
- **A larger pilot (the 20 of `selection-adr-160.json` too).** Rejected for now: about $10 more for an answer that is
  still a direction; it is the follow-up if the pilot is mixed.
- **Teach apply_patch to Claude as well.** Rejected: apply_patch is Codex's trained format, not Claude's.

## Component / Boundary Impact

None in mrw's engine. T1 and T2 are internal to bench (`internal/bench`, `cmd/bench`, `scripts/bench_edits.py`); T5,
only on SHIP, adds a flag to `mrw instructions` (`cmd/mrw`) and a paragraph to `internal/guide`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run` / `bench canary` `--profile stage8 --configs B,S` | new profile and configuration | T1 | T3, T4 |
| `run.json` `edit_paragraph_sha256` | new field, refused if it changes between blocks | T1 | T3, T4 |
| `scripts/bench_edits.py count` / `verdict` | the frozen counting and decision rule | T2 | T4 |
| `docs/bench/stage-8/pilot.md` `paragraph:` and `paragraph sha256:` | the frozen paragraph | T3 | T4, T5 |
| `docs/bench/stage-8/RESULTS.md` `verdict:` | SHIP or NO SHIP | T4 | T5 |
| `mrw instructions --host claude` | new flag (only on SHIP) | T5 | Claude Code hosts, `docs/agents.md` §1 |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| configuration S and profile `stage8` | T1 | T3, T4 | No — new names |
| `bench_edits.py` and its rule | T2 | T4 | No — new script |
| the frozen paragraph and its sha | T3 | T4, T5 | No |
| the verdict line | T4 | T5 | No |

## Implementation

See `docs/adr/ADR-193/tasks/README.md`.

## Consequences

- **Positive:** the first measurement of whether the edit format, not only the read rule, costs a Claude agent turns,
  on real repositories, for about $10.
- **Positive:** the refused-write count gets a recorded rule; Stage 5's reference row and every later stage can be
  counted the same way.
- **Negative:** 18 instances give a direction; a SHIP ships a flag-selected text on that direction.
- **Negative:** another paid run on the shared machine while ADR-192's Stage 7 is pending (T4's Awaiting-decision).
- **Neutral:** if the agent writes blocks for a file it never read, ADR-190's quote check is what stands between it
  and a wrong edit; the pilot's undecided and refused counts show whether that check fires.

## Out of Scope

- Codex steering toward `--format=apply_patch` (deferred: docs/adr/BACKLOG.md "From ADR-193")
- Cursor (permanent: boundary: bench drives Claude Code only, so no pilot can measure a Cursor variant; ADR-192 stopped at the same host line)
- A native JSON `str_replace` front end (deferred: docs/adr/BACKLOG.md "From ADR-193")
- Hashline-style per-line hashes in reads and edits (deferred: docs/adr/BACKLOG.md "From ADR-193")
- A safety and long-session benchmark, its own record (deferred: docs/adr/BACKLOG.md "From ADR-193")
- `mrw skill --host` (deferred: docs/adr/BACKLOG.md "From ADR-193")
- A larger confirmation run after a mixed or SHIP pilot (deferred: docs/adr/BACKLOG.md "From ADR-193")
- The MCP handshake text (permanent: boundary: a host with a shell drives the CLI, and MCP is for hosts without one, ADR-170)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The paragraph does not move the agent to SEARCH/REPLACE | Med | High | T3's smoke first, one change allowed before counting; (U) closes UNTESTED |
| Blocks on unread files apply a wrong edit | Low | High | ADR-190's exact, unique quote, re-checked under the lock; the oracle grades each attempt |
| Refusals move to ADR-052's line-after rule | Med | Med | the paragraph teaches the kept closer; the refused bucket is reported by marker |
| Bench's hook or the counting script misreads a SEARCH/REPLACE command | Low | High | checked 2026-11-03 (Existing Primitives Audit); T1's and T2's tests and T3's preflight |
| The counting rule misreads a hidden receipt (`> /dev/null`, `--quiet`) | Med | Low | such a call is undecided, reported, and left out of the rate |
| The `--core` host form ships unmeasured | Certain if SHIP | Med | named in Decision 3; Zy may ship the full form only |
| 18 instances mislead | Med | Med | Decision 5 says what it cannot detect; NO SHIP on a mixed result |

## Rollback

None — measurement until T5: `stage8` and S are opt-in names. T5, if it lands, is one flag and one paragraph; removing
them leaves `mrw instructions` as v1.83.0 prints it.

## Follow-ups

- [ ] On SHIP: Codex's apply_patch variant, with a way to measure it (BACKLOG "From ADR-193").
