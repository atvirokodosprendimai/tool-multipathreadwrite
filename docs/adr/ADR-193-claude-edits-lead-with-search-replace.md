# ADR-193: Claude hosts are taught to edit in search_replace blocks, if a paired pilot says it pays

**Status:** Accepted
**Accepted:** 2026-11-03 by Zy — answered *"accepted."* to the draft summary and *"go, merge when clean"* after the Codex challenge. This version answers a Codex xhigh challenge of `ebf4d780` (6 findings: one P1, five P2) and an adr-challenger review of the same commit (18 findings: 3 high, the rest medium and low), each checked against the source, every one that held taken in
**Date:** 2026-11-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-051, ADR-141, ADR-156, ADR-162, ADR-168, ADR-171, ADR-190, ADR-192
**Governs:** `internal/bench/**`, `cmd/bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/runner_test.go::TestConfigurationSIsBPlusTheEditParagraph`
**Invalidates:** none — checked: without `--profile stage8` every configuration's arguments, tools, prompt and refusal text stay as ADR-162, ADR-163 and ADR-192 left them, so Stages 1 to 7 stay reproducible; `mrw instructions` and `mrw instructions --core` without `--host` print what the same build prints today (T5's test and Invariants), so ADR-141's bound and ADR-192 T1's invariant "B's prompt under stage7 is `mrw instructions` alone" hold
**Served-path change:** None until T5, and T5 lands only on a SHIP verdict: then `mrw instructions --host claude` prints the full text with the paragraph S ran after it, and `docs/agents.md` names it as an option for Claude hosts; the standing trigger, the core, the skill and every output without `--host` stay as they are.

## Context

**The direction (Zy, 2026-11-03, *"niiice, lets go, i think it is good direction"*).** mrw is the guarantee layer —
a plan applied whole or not at all, a per-line read licence, a verdict per edit, paths that mean what they say — and
it should reach each model in the edit format that model was trained on. mrw already compiles two foreign formats into
its native plan (ADR-051): `--format=search_replace` (Aider SEARCH/REPLACE blocks) and `--format=apply_patch` (Codex
`*** Begin Patch`). Since ADR-190 (`1f01dcdc`, in v1.83.0) a SEARCH block that quotes every line it replaces, exactly
and matching once, needs no prior read, and a multi-line block whose SEARCH and REPLACE end in the same kept line meets
ADR-052's line-after rule by that quote.

**What agents are taught today.** `mrw instructions --core` (`internal/guide/guide.go`, `Core()`, 298 words on v1.83.0
by `strings.Fields`, the measure `internal/guide/core141_test.go` bounds at 300) teaches only the native
`@@ PATH N-M replace anchor=` hunk (rule 4), and its rule 2 says a line mrw never served is refused. The full
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
  success. That is why the cost gate (c) below is spend per success over every paired instance.
- SWE-Edit (arXiv 2604.26102, read second-hand): 93.4% edit success for native str_replace. Its denominator and harness
  differ from bench's, so it is context, not a baseline.
- memoryd `tool-multipathreadwrite` `56caf49e` (2026-10-30, a Codex session): "investigate a native-looking editing
  interface backed by existing MRW validation/atomicity".

**Bench evidence, Stage 5 (ADR-162: mrw v1.68.0, before ADR-171 (v1.75.1) made `body=` optional; Claude Code 2.1.302;
191 SWE-bench Verified instances).**
- **The registered result** (`docs/bench/stage-5/looks/look-5.txt`): B against A is INCONCLUSIVE on both models — Opus
  5.5 cost −19.4% [−25.5, −12.7], solved −0.5 pt [−6.7, +5.5]; Sonnet 5.5 cost −7.0% [−12.1, −1.8], solved −2.6 pt
  [−8.7, +2.8]; 99%, five looks.
- **Post-hoc, not registered** (the coordinator's analysis of 2026-11-03): cost per success on the both-solved subset,
  B against A, Opus −23.9% [−28.8, −18.8] n=158, Sonnet −9.8% [−17.0, −2.4] n=125 (percentile bootstrap over
  instances, 4,000 resamples, one run per arm). Context for the size of effect a guide change might move; never a
  verdict.
- **B's write calls on v1.68.0, before ADR-171,** counted 2026-11-03 over Stage 5's stored B transcripts
  (`~/.local/share/mrw-bench/runs/2026-10-17-s5/stage-5/block-*/`, 191 per model) with the rule Decision 4 registers:
  Opus 555 decided calls, 94 refused (16.9%), 3 undecided, 17 blocked by bench's own Bash hook, 73 of 191 attempts
  with a refusal, 1,611 output tokens per landed write call; Sonnet 359 decided, 106 refused (29.5%), 9 undecided, 11
  blocked, 74 of 191 attempts, 1,299. Most refusals were plans that did not parse, and 143 of the 156 parse refusals
  cite `body=` (the challenger's count, checked by the coordinator) — a count v1.68.0's guide still asked for and
  v1.75.1 made optional. The first draft's rule gave 547/87 and 349/96; the final rule also counts `mrw -C DIR write`
  and `mrw --root DIR write`, counts a call holding any refusal as refused, and knows more refusal markers. The
  brief's figures, 32 of 578 (5.5%) and 28 of 384 (7.3%), came from a rule nobody recorded and are not reproduced.
  No B attempt used `--format=search_replace` or `--format=apply_patch`, and none was asked to: v1.68.0's guide did
  not mention either.
- **On v1.83.0** (Claude Code 2.1.317), counted 2026-11-03 by the coordinator over `~/.local/share/mrw-bench/runs/stage7/block-1/`
  and `stage7-t3/pilot*`: 16 B and H attempts, 31 `mrw write` calls, 6 refused — 3 "has not been read" (H attempts
  editing an existing file with no read), 1 "needs a served line after 81 (the closer sits below the body)", 1 an
  insert-after with an empty body, 1 "body= asked for 1 more line". B alone: 5 calls, none refused. Small and
  descriptive only.
- **So the premise is not `body=` any more.** On the release the pilot runs, a refusal is mostly an unread line or a
  closer below the body — the two refusals a SEARCH/REPLACE block that quotes its lines and ends on its kept closer
  avoids under ADR-190 — and B may refuse too rarely for a refusal rate to say anything (Decision 5 says what then).
- **ADR-192 T3's lesson** (`docs/bench/stage-7/pilot.md`, 2026-11-03): a permissive sentence ("you may also read files
  with the Read tool") moved 0 of 4 attempts; a directive one moved 4 of 4. A variant's wording must be directive.
- **Stage 7 is stopped.** Zy, 2026-11-03: *"too much time spent on it right now to prove nothing"*. The pilot has the
  machine's bench time to itself.

**Hosts' environments.** Claude Code sets `CLAUDECODE=1` (and `CLAUDE_CODE_ENTRYPOINT=cli`) in the processes its tools
start: observed 2026-11-03 in this session's environment. Nothing in this repository records what Codex sets
(`git grep CODEX_HOME` finds only a fixture patch, `bench/tasks/am-wide-claude-kit/reference.patch:529`).

## Existing Primitives Audit

- **Reuse:** `ingest.CompileSearchReplace` and ADR-190's quote licence, unchanged; no engine package changes.
- **Reuse:** ADR-192's mechanism for a configuration that is B plus fixed text after `mrw instructions`
  (`HSentence`, `internal/bench/profile.go`; the append at `internal/bench/runner.go:1166`; its SHA-256 in `run.json`,
  refused when a block names another, `runner.go:348`), its canary, and its uncounted pilot shape (ADR-192 T3).
- **Reuse:** `scripts/bench_mechanism.py`'s `mrw_calls` and `result_text`, imported, not copied, by the new counting
  script, and frozen: it is ADR-192's counting script at `3a970837`, and both scripts' SHA-256 are recorded and checked
  (T2, T4). Checked 2026-11-03: `mrw_calls` reads a SEARCH/REPLACE document written by a quoted heredoc to `$TMPDIR`
  then `mrw write --format=search_replace FILE`, or piped to `mrw write --format=search_replace -`, as one `write`;
  bench's Bash hook built from `8c9bdef8` (`bench hook --tree`) passed both shapes and an unquoted heredoc, while it
  refused `echo x > f.txt` and `echo x >> REPLACE` in the tree. Its heredoc pattern does swallow a `printf` that holds
  `<<<<<<< SEARCH` (the challenger's finding): the new script counts such an untokenized command holding `mrw write` as
  undecided, without editing `bench_mechanism.py`.
- **Reuse:** bench's solved rule (`internal/bench/analyze.go:311`: the oracle solved, not over budget, no agent
  error), its exclusions (`runner.go:71-73`) and its replacement walk (`replaced`, `runner.go:450`: a start error is
  replaced at most twice, a setup failure once), and `bench verify`, which checks a stage directory's planned receipts
  and the replacements each excluded one is owed (`runner.go:583`).
- **Not reused:** `bench look` and `bench analyze`'s sequential verdict (ADR-156's bar at 99% over five looks): a pilot
  of 18 instances has no looks; its rule is Decision 5's, computed by the frozen script.
- **Class audit — the places that tell an agent how to edit with mrw.** Command, run 2026-11-03:
  `git grep -lE 'instructions --core|replace anchor=|format=search_replace' -- '*.md' '*.ts' '*.go' ':!*_test.go' ':!docs/adr/**' ':!docs/bench/**' ':!docs/break/**'`
  — 14 files. Governed by T5, on SHIP only: `internal/guide/guide.go` (a new `ClaudeEdits()`; `CLI()` and `Core()`
  unchanged), `cmd/mrw/main.go` (`instructionsCmd`'s `--host`), `docs/agents.md` (one optional line for Claude hosts,
  outside §1's trigger), `AGENTS.md` and `README.md` (one line each beside `mrw instructions`). Left out, and why:
  `.claude/skills/mrw/SKILL.md` and `cmd/mrw/skill168.go` (the skill and `mrw skill`, whose `Skill()` embeds `Core()`),
  `BESTPRACTICES.md`, and the `--skill` pointer in `cmd/mrw/main.go`'s help — wiring the paragraph into the triggers
  and the skill is Zy's decision after SHIP (Out of Scope); `cmd/opencode/mrw-plugin/src/index.ts` (opencode, not a
  Claude Code host); `.claude/agents/go-implementer.md` (this repository's own agent brief, the owner's
  instructions); `internal/mcp/instructions.go` (the MCP handshake: a host with a shell drives the CLI);
  `internal/ingest/applypatch.go`, `internal/plan/plan.go` and `internal/quote/quote.go` (they parse or name these
  forms; they teach nothing).

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
   before any counted attempt, when its smoke shows S making no SEARCH/REPLACE write call on at least one of the two
   models, and never after. New files stay `mrw write --create`, as the instructions say.
   - **Honest about the fit.** Claude's trained edit tool is a JSON call, `str_replace` with `old_str` and `new_str`.
     Aider's block is not that syntax; its semantics are the same — an exact, unique match of the old text, replaced by
     the new. S tests whether the semantics carry the benefit through a text syntax; it does not test a JSON front end
     (Alternatives).
   - **Why the closer clause is in the paragraph.** Without it a multi-line block on an unread file is refused by
     ADR-052's line-after rule (v1.83.0 prints "end SEARCH and REPLACE in the same line, which keeps it and quotes
     it"), and the pilot would measure the wording's gap, not the format.
   - **What the native plan keeps.** Line-addressed edits, generated plans (`--sub`, `--rewrite`, `from=`) and
     many-site edits stay native; the paragraph says so.
2. **Profile `stage8`: stage5 plus configuration S, which runs only there.** Under `stage8` a list names only B and S,
   every model that runs S runs B (T3's smoke runs both), and B's arguments, tools, prompt, settings and refusal text
   equal stage5's. A, H, P and the side arms are not run: the question is S against B. The pilot has no
   `exclusions.json`, so no attempt is ruled an infrastructure exclusion.
3. **The Claude variant is selected by an explicit flag, `mrw instructions --host claude`, not by the environment, and
   only the measured form ships.**
   - **(a) By environment (`CLAUDECODE=1`) — rejected.** The reason is that it is a hidden input: the same command
     would print different text depending on a variable the caller does not see or pass. One consequence is measured:
     bench builds B's prompt with `exec.Command(o.Mrw, "instructions")` (`runner.go:1161`), which inherits bench's
     environment, and this session, from which bench is run, carries `CLAUDECODE=1`. Nothing here records what Codex
     sets either.
   - **(c) Only `docs/agents.md` and the skill — rejected.** The paragraph would be a prose copy no test holds to the
     binary, the drift ADR-141 exists to prevent.
   - **(b) The flag — chosen.** Deterministic for a given input, testable as `Core()` is (ADR-141 Decision 2: every
     flag the text names is a flag of the binary), and absent unless asked for. `--host` takes only `claude`; any other
     value is a usage error, exit 2, naming the value it takes. `--host claude` prints `CLI()`, a newline and the
     paragraph S ran, byte for byte the prompt S had.
   - **Zy's surface decision (2026-11-03, defaulted with Zy told): only the measured surface ships.** The pilot
     appends to the full text, as B does, so `--core --host claude` is a usage error, exit 2, until its own
     measurement (deferred: BACKLOG). The core form would also clash with the core's rule 2 ("writing a line mrw never
     served you is refused") beside a paragraph that says a quoting block needs no read (Risks).
   - **What T5 does not touch.** The standing trigger in `docs/agents.md` §1, the skill (`mrw skill`,
     `.claude/skills/mrw/SKILL.md`) and the owner's global instructions stay as they are; `docs/agents.md` gains one
     line naming the flag as an option for Claude hosts. Wiring the paragraph into the triggers and the skill is a
     separate decision for Zy after SHIP (BACKLOG).
4. **Counting is a frozen script, `scripts/bench_edits.py`, committed with `bench_mechanism.py`'s hash before the
   first counted call.** Per transcript:
   - **A write call** is one Bash tool call whose command tokenizes (`bench_mechanism.mrw_calls`) to at least one mrw
     invocation that, after the global flags `-C DIR`, `--root DIR` and `--root=DIR`, is `write` without `--dry-run`;
     counted once per tool call however many writes it chains. A command that does not tokenize but holds `mrw` and
     `write` is an **untokenized** write call.
   - **Outcome, in this order.** **Blocked** when the result holds `PreToolUse:Bash hook error` (bench's own hook);
     else **refused** when the result holds any refusal marker — `nothing was written` (any case), `plan has N
     error`, `search_replace: `, `apply_patch: `, `flag provided but not defined`, `cannot hold a line break`, `is not
     relative to the root`, `plan is empty` — so a call that chains a refused write and a landed one is refused; else
     **landed** when it holds `— applied` or `PARTIALLY APPLIED`; else **refused** when it holds a line beginning
     `mrw: `; else **undecided**. Untokenized calls are undecided. Blocked calls are left out of every measure;
     undecided ones count against coverage.
   - **An attributed SEARCH/REPLACE write call** holds exactly one write, that write carries `--format=search_replace`
     or `--format search_replace`, and its outcome is landed or refused: mrw's own verdict is in the result. A call
     whose write never ran (`false && mrw write --format=search_replace F`) prints no verdict and is not attributed;
     a call holding two writes is not attributed to either. Landed attributed calls are reported beside the gate.
   - Output tokens and cost come from the receipt (`usage.output_tokens`, `total_cost_usd`); an attempt whose
     transcript has no final `result` line has those measures **missing**, never zero. Solved is bench's rule.
   - Each counted transcript's SHA-256 is recorded beside its counts.
5. **The pilot, pre-registered here, before any code or run.**
   - **Fixed:** Claude Code 2.1.317 (`~/.local/share/mrw-bench/claude-2.1.317`), mrw v1.83.0's release asset (as ADR-192
     T3 ran it, checksum against `SHA256SUMS.txt`), `claude-opus-5-5` and `claude-sonnet-5-5`, one attempt per arm per
     instance per model (repeat 1), bench's default 20-minute budget, profile `stage8`, stage name `pilot-8`, one
     output directory per model.
   - **Instances:** `bench/swe-bench/selection.json` in file order, from the third on — the first two were ADR-192 T3's
     pilot, and T3 here uses them for an uncounted smoke — which is 18: `pytest-dev__pytest-5631`, `django__django-16642`,
     `sphinx-doc__sphinx-8265`, `scikit-learn__scikit-learn-25747`, `matplotlib__matplotlib-23314`, `sympy__sympy-18199`,
     `sympy__sympy-24661`, `django__django-11532`, `django__django-14011`, `scikit-learn__scikit-learn-12585`,
     `matplotlib__matplotlib-25775`, `django__django-14434`, `django__django-16116`, `django__django-11555`,
     `astropy__astropy-14539`, `django__django-11451`, `django__django-14404`, `django__django-13807`. None is in the
     191-instance pool (`selection-adr-162.json`; checked 2026-11-03, 0 overlap). ADR-156's Stage 3 ran them on an older
     mrw; that does not bear on a paired comparison run now. The verdict refuses any other list, order, model, arm or
     repeat.
   - **Exclusions: the final-receipt walk.** For each instance, model and arm the counted receipt is the last of
     `<id>`, `<id>.r1`, `<id>.r2` that bench's `replaced` owes: a start error is replaced up to twice, a setup failure
     once, and a replacement that is not excluded ends the walk. A walk that ends excluded (a start error after both
     replacements, a setup failure after its one) drops that instance from both arms for that model. A receipt
     excluded as infrastructure, a reason bench does not know, a started attempt with no receipt, or a missing
     replacement stops the pilot: the verdict refuses and it goes to Zy. More than 2 dropped instances on a model also
     stop it. Timed-out and over-budget attempts count, unsolved. Replacements are not counted toward the 72.
   - **Measures per model and arm,** over the counted attempts of the paired instances: (i) **refused-write rate**,
     refused over landed plus refused, summed over attempts; (ii) **output tokens per attempt**; (iii) **spend per
     success**, summed cost over solved count; (g) **landed write calls per attempt**; (O) **coverage**, decided
     (landed plus refused) over decided plus undecided write calls. A missing output-token count or cost is imputed
     against S: an S attempt takes the largest value any counted attempt of that model recorded, a B attempt the
     smallest. Descriptive, never gating: cost per success on the both-solved subset with a percentile bootstrap 95%
     interval (4,000 resamples, seed 193; none when the subset is empty), output tokens per landed write call, turns,
     attempts with a refusal, undecided, blocked and untokenized calls, and landed attributed SEARCH/REPLACE calls.
   - **Rule per model.** **UNTESTED** when (U) fewer than half, rounded up, of the counted S attempts (9 of 18, 8 of 16)
     make at least one attributed SEARCH/REPLACE write call, or (O) either arm's coverage is below 90% or it has no
     decided call. Otherwise **PASS** when all of: (a) S solved at least B solved minus 2; (b) **either** B refused at
     least 3 calls and S's (i) is at most two thirds of B's, **or** S's (ii) is at most 0.85 of B's; (g) S's landed
     write calls per attempt are at most 1.2 times B's, and both arms landed at least one; (c) S's (iii) is at most
     1.10 times B's, with neither arm at zero solved. Otherwise **FAIL**. A measure that cannot be computed fails its
     clause: zero B refusals leaves (b) only its second branch, an empty both-solved subset changes nothing, zero solved
     fails (c).
   - **The verdict:** **SHIP** when both models PASS; otherwise **NO SHIP**, and the counts are recorded and nothing in
     mrw's guide changes. A mixed result is NO SHIP: a larger run is Zy's call (Out of Scope).
   - **How often this rule says SHIP with no effect at all.** Resampling, 2026-11-03: two disjoint sets of 18 attempts
     from Stage 5's 191 B attempts per model, one taken as B and one as S, 20,000 times, seed 193, uptake and coverage
     taken to hold. The frozen script (T2's `null`, `random.Random(193)`, `sample(range(191), 36)` per model, first 18
     as B and last 18 as S; `docs/bench/stage-8/reference.md`) gives PASS 22.4% on Opus and 24.9% on Sonnet, so a false
     SHIP (both) is 5.6%; with (i) unavailable (B refusing fewer than 3), 4.0%; seeds 1 to 6 give 5.4–6.0% and 3.8–4.2%.
     The draft's own simulation gave 24.8%, 24.5%, 6.1% and 4.1%. No single reading of the rule reproduces all four: the
     frozen rule (a missing output or cost imputed, as this Decision says) gives 22.4%, 24.9%, 5.6% and 4.0%, and reading a
     missing value as the receipt holds it (zero tokens, no cost) gives 25.1%, 24.9%, 6.2% and 4.5%; the draft's sampler is not recorded.
     The rule is unchanged; the frozen script's figures are the record's. The first draft's rule passed (b) 30% and 36%
     of the time (the challenger's resampling); dropping (ii) to output per attempt, adding (g), and gating cost over all
     paired instances brought it here. Under that null one model's PASS means little; SHIP needs both.
   - **Prediction, written now:** uptake holds on both models (the paragraph is directive). B on v1.83.0 refuses
     rarely, so (i) is likely unavailable or small, and the route to PASS is (ii); a block carries the old lines as
     well as the new where a native hunk carries an anchor, so (ii) more likely holds or rises than falls by 15%. The
     refusals that remain — unread lines, a closer below the body — are what a quoting block avoids, so S's refused
     count should be no higher than B's. Expected: **NO SHIP** at least as likely as SHIP.
   - **What it cannot tell.** On v1.83.0 a B attempt made about 2 write calls and refused few, so a refusal rate is
     close to noise at 18 attempts. A solving margin of 2 in 18 cannot see a loss smaller than about 11 points. Spend
     per success over 18 instances moves by a single expensive failure. A SHIP from this pilot is evidence enough to
     add a flag-selected text; it is not evidence for a README claim.
6. **Price, said first.** Stage 5's B per attempt (measured 2026-11-03 over its 191 receipts per model): Opus $0.200 and
   68 s, Sonnet $0.053 and 25 s. The counted pilot is 18 × 2 arms × 2 models = 72 attempts, about $9.1 and 56 minutes of
   attempt time; T3's smoke (B and S, 8 attempts) and canaries add about $1.5. If the counted pilot's spend passes $15,
   it stops and goes to Zy.

What would make it FAIL:
- an S whose init tools are not B's, whose prompt is not `mrw instructions`, a newline and the frozen paragraph, or whose
  paragraph's sha differs from T3's;
- a B under stage8 that differs from stage5's B in arguments, tools, prompt or refusal text;
- a counting rule, threshold, instance list, exclusion rule, or either script's hash changed after the first counted call;
- an S reported PASS on a model whose uptake or coverage failed;
- a T5 that ships text other than the paragraph S ran, ships `--core --host claude`, or changes any output without
  `--host`.

## Alternatives Considered

- **A native JSON `str_replace` front end** (an MCP tool or a `--format=str_replace` taking `{path, old_str, new_str}`).
  Closest to Claude's trained call. Not chosen now: it is a new surface with its own parser and receipts, and bench
  drives Claude Code through Bash, where a JSON document is no closer to the trained call than a text block; if S
  passes, it is the follow-up that would close the syntax gap (Out of Scope).
- **Change the core's rule 4 first, then measure.** Rejected: the guide is every adopter's served path, and ADR-192
  set the precedent of measuring before changing it.
- **Select by environment.** Rejected in Decision 3: a hidden input.
- **Ship `--core --host claude` too.** Rejected by Zy's surface decision: unmeasured, and it clashes with core rule 2.
- **Gate output per landed write call, or cost on the both-solved subset.** The first draft's rule. Rejected after
  review: output per landed call rewards splitting an edit into more calls (the challenger), and the both-solved
  subset leaves out the spend on failures, so S could solve 2 fewer and cost 125% more per success and still pass
  (Codex P1). Both are kept as descriptive.
- **Run under stage7 beside A, B and H.** Rejected: `stage7Problem` holds stage7 to A, B and H (`runner.go`), and
  Stage 7 is stopped.
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
| `scripts/bench_edits.py count` / `verdict` / `null` | the frozen counting and decision rule | T2 | T4 |
| `docs/bench/stage-8/pilot.md` `paragraph:` and `paragraph sha256:` | the frozen paragraph | T3 | T4, T5 |
| `docs/bench/stage-8/verdict.txt` `verdict:` | SHIP or NO SHIP | T4 | T5 |
| `mrw instructions --host claude` | new flag (only on SHIP) | T5 | Claude Code hosts that choose it |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| configuration S and profile `stage8` | T1 | T3, T4 | No — new names |
| `bench_edits.py` and its rule, with both scripts' hashes | T2 | T3, T4 | No — new script |
| the frozen paragraph and its sha | T3 | T4, T5 | No |
| the verdict line | T4 | T5 | No |

## Implementation

See `docs/adr/ADR-193/tasks/README.md`.

## Consequences

- **Positive:** the first measurement of whether the edit format, not only the read rule, costs a Claude agent turns,
  on real repositories, for about $10.
- **Positive:** the refused-write count gets a recorded rule with its false-SHIP rate stated; later stages can count
  the same way.
- **Negative:** 18 instances give a direction; a SHIP ships a flag-selected text on that direction.
- **Neutral:** if the agent writes blocks for a file it never read, ADR-190's quote check is what stands between it
  and a wrong edit; the pilot's refused counts show whether that check fires.

## Out of Scope

- Codex steering toward `--format=apply_patch` (deferred: docs/adr/BACKLOG.md "From ADR-193")
- Cursor (permanent: boundary: bench drives Claude Code only, so no pilot can measure a Cursor variant; ADR-192 stopped at the same host line)
- A native JSON `str_replace` front end (deferred: docs/adr/BACKLOG.md "From ADR-193")
- Hashline-style per-line hashes in reads and edits (deferred: docs/adr/BACKLOG.md "From ADR-193")
- A safety and long-session benchmark, its own record (deferred: docs/adr/BACKLOG.md "From ADR-193")
- `mrw instructions --core --host claude` (deferred: docs/adr/BACKLOG.md "From ADR-193")
- Wiring the paragraph into the standing triggers and the skill after a SHIP (deferred: docs/adr/BACKLOG.md "From ADR-193")
- A larger confirmation run after a mixed or SHIP pilot (deferred: docs/adr/BACKLOG.md "From ADR-193")
- The MCP handshake text (permanent: boundary: a host with a shell drives the CLI, and MCP is for hosts without one, ADR-170)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The paragraph does not move the agent to SEARCH/REPLACE | Med | High | T3's smoke first, one change allowed before counting; (U) closes UNTESTED |
| Blocks on unread files apply a wrong edit | Low | High | ADR-190's exact, unique quote, re-checked under the lock; the oracle grades each attempt |
| Refusals move to ADR-052's line-after rule | Med | Med | the paragraph teaches the kept closer; refusals are counted by marker |
| Bench's hook or the counting script misreads a SEARCH/REPLACE command | Low | High | checked 2026-11-03 (Existing Primitives Audit); an untokenized command is undecided and counts against coverage; T1's and T2's tests and T3's preflight |
| Shell packaging games the measures (more calls, chained writes) | Med | Med | (g) bounds landed calls per attempt; a call with any refusal is refused; uptake needs mrw's own verdict for one write |
| A false SHIP with no real effect | Low (5.6% by resampling, T2's frozen script) | Med | both models must PASS; the rate is recorded in `docs/bench/stage-8/reference.md` |
| The core form would clash with core rule 2 | — | — | not shipped (Decision 3); its own measurement first |
| 18 instances mislead | Med | Med | Decision 5 says what it cannot detect; NO SHIP on a mixed result |

## Rollback

None — measurement until T5: `stage8` and S are opt-in names. T5, if it lands, is one flag and one paragraph; removing
them leaves `mrw instructions` as v1.83.0 prints it.

## Follow-ups

- [ ] On SHIP: Zy decides the triggers and the skill; Codex's apply_patch variant needs a way to measure it (BACKLOG "From ADR-193").
