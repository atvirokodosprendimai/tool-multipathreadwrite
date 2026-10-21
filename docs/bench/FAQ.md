# The SWE-bench result, question by question

The questions a careful reader asks about "Claude Code with mrw costs less and finishes sooner", with the answer and where to check it. Numbers are Stage 5's (ADR-162, 2026-10-17 to 2026-10-21) unless a stage is named. The README's top carries the tables; this page is for the reader who wants to shoot at them.

**What is mrw, in one sentence?**
A Go CLI (and MCP server) that reads many ranges across many files and applies many edits across them in one call, returns a verdict for every edit, and refuses to edit a line it has not served you. MIT, one binary, no network call except `mrw version --check-latest`.

**What exactly was compared?**
A is Claude Code 2.1.302 with its own file tools: Bash, Read, Edit, Write, Glob, Grep. B is the same CLI with Bash and Glob only, mrw v1.68.0 on PATH, and `mrw instructions` appended to the system prompt. Same task prompt, same model, same budget (1,200 s), same machine, one attempt per task per arm. So the comparison is "Claude Code's file tools" against "mrw as the file tool". `Configs` in `internal/bench/runner.go`; the prompt's SHA-256 in every block's `run.json`.

**So you took tools away from B. Isn't the saving just fewer tools?**
B lost Read, Edit, Write and Grep and gained mrw; that is the substitution being measured. B could still read through the shell (`cat`, `grep`), and did: 394 shell reads on Opus and 392 on Sonnet across 191 attempts each, counted and never refused (the "B shell reads" column in RESULTS.md). B made no write outside mrw ("slipped writes": 0 in every stage). We did not run a "Claude Code minus tools, without mrw" arm; if the saving came from having fewer tools, B would be expected to solve less, and it solved the same.

**It's cheaper. Why is the verdict INCONCLUSIVE and not PAYS?**
The bar was registered before any run (BACKLOG.md, "Pre-registered for ADR-156"): PAYS needs the mean cost down 10% or more with the interval wholly below zero, and the solved-rate difference's lower bound at −5 points or better. Opus met the cost half (−19.4% [−25.5%, −12.7%]) and missed the solved half (lower bound −6.7). Sonnet missed both (−7.0%, lower bound −8.7). 191 tasks give a solved-rate interval about ±7 points wide; showing non-inferiority at −5 would take roughly twice the tasks. We report the verdict the bar gives, not the one we would like.

**Why multi-hunk tasks? That is cherry-picking for mrw.**
The pool rule was registered before the run: the 191 ids with the lowest `sha256("adr-162:" + id)` among SWE-bench Verified instances whose gold patch has at least two hunks, minus the ids earlier stages had used (`bench/swe-bench/selection-adr-162.json`, with the rule in the file). Multi-hunk is where mrw's one-call mechanism should matter, and the pre-registration says so. Stage 3, 20 random Verified instances with no hunk filter, also came out cheaper (−5.3% Opus, −6.1% Sonnet, n = 20, INCONCLUSIVE).

**Show me the pre-registration.**
ADR-162 and its BACKLOG entry merged as commit `1fc00bad` on 2026-10-17 at 08:42 UTC; the instance list as `8b435f8f` at 10:55 UTC; the first counted attempt started at 13:21 UTC (`started` in its receipt). The five looks, the stop rule, the 99% level and the exclusion rule are in that commit.

**Who graded the patches?**
The official SWE-bench harness (swebench 5.0.2) with its Docker images, run locally on every attempt. Each receipt carries the diff the agent produced and `oracle.harness_resolved`. Nothing was graded by a model or by us.

**Could the agent run the tests?**
No. The checkouts the agent worked in had the code but not the test dependencies (asgiref, numpy, pytest and so on), so neither arm could check its own work; the harness graded afterwards. This holds for A and B alike, and it is the main limit of this stage: it measures bug-fixing without a feedback loop. Stage 6 (ADR-163) runs each attempt in a container where the repository's tests run.

**Opus solving 167 of 191 looks high.**
It is the single-attempt, no-test-run rate of Claude Code with Opus 5.5 on a multi-hunk subset; it is not a leaderboard number and we make no claim about it. The result is the paired difference, 166 against 167.

**What is "cost"?**
`total_cost_usd` as Claude Code reports it per attempt, at October 2026 prices, cache tokens included. Stage 5's A and B attempts together cost $109; every stage's A and B together, $227. The raw token counts (input, output, cache read, cache write) are in every receipt.

**Why did Sonnet's total tokens go up?**
Cache reads rose from 72k to 82k per attempt (mrw's instructions and its batched reads add context), while output fell 24% and cache writes fell. Cache reads are the cheapest tokens; the output that fell is priced far above them, so cost fell 7% while the token count rose 11%. Opus fell on every kind.

**Is "fewer turns" a real saving or an artefact?**
A turn is one model call (`num_turns` from Claude Code's result). Fewer turns is where the cost saving comes from: each call re-reads the context. Opus 14.6 to 11.5, Sonnet 7.8 to 6.8.

**Wall time fell 40%. Is that real?**
It fell with the turns, but it was measured on one shared Apple Silicon machine, the SWE-bench images under amd64 emulation, beside other work, and it was not pre-registered. Treat it as descriptive. No verdict reads it.

**What was excluded, and who decided?**
The registered admission rule (BACKLOG.md, "Pre-registered for ADR-156") allows two exclusions only: an attempt that died before its first tool call with an authentication, rate-limit or CLI start error, re-queued at most twice; and an attempt bench could not set up or grade. One Sonnet attempt and one Opus attempt were excluded that way and re-queued (the `.r1` receipts); every excluded receipt stays on disk. One more Opus A attempt (pytest-10051) was ended by the machine going to sleep; Zy ruled it infrastructure before the next look, and the ruling with the `pmset` log is in `docs/bench/stage-5/exclusions.json`, which is why Opus counts 190 pairs. An agent that gives up or runs past the 1,200 s budget is unsolved with its cost counted; two Opus attempts (django-16032, sympy-16597) were killed at the budget before the CLI reported any cost, so they count as unsolved and the cost mean has nothing of theirs to include, as `look-5.txt` says. Nothing else was dropped.

**One run per task. Isn't that noise?**
Yes, one attempt per task per arm; the pairing by task and 191 tasks are what give the interval its width. The intervals are 99% because the stage was read at five looks (ADR-162, Decision 5).

**Does this hold for other agents or models?**
Not measured. mrw runs under any shell, any MCP host and opencode; the numbers are for Claude Code 2.1.302 with Opus 5.5 and Sonnet 5.5 only.

**Earlier stages were worse. Why?**
They are all published (RESULTS.md). Stage 4, 24 of this project's own tasks on v1.66.0, LOSES: +17.1%, driven by wide renames at +33%. ADR-161 changed what mrw teaches in reply (recipes first), and the rerun on v1.68.0 came out INCONCLUSIVE, cost within ±3%. The own-task stages have 24 to 40 tasks and nearly every task solved in both arms, so they could never show much.

**RESULTS.md has a third configuration, P, that LOSES badly. What is it?**
P is Claude Code with our quality-harness plugin, measured in the same stage: cost +49% on Opus and +70% on Sonnet, solved unchanged, closed at the first look. The plugin is built for multi-step, recorded work with tests running; Stage 5 had neither, and its Stop hook kept attempts going in checkouts that could not run a test. It is a result about that plugin in that setting, and Stage 6 measures it where tests run.

**How do I reproduce the table?**
From a checkout, the jq command in the README's Evidence section regenerates the Stage 5 rows from the receipts; `go run ./cmd/bench analyze --selection bench/swe-bench/selection-adr-160.json --selection bench/swe-bench/selection-adr-162.json docs/bench` recomputes every verdict and interval. A full rerun needs Claude Code, Docker, the SWE-bench images (about 4.5 GB each), four days on one machine and about $110 of API spend; `bench run` and ADR-162 describe it.

**Who are you?**
Atviro kodo sprendimai, a small open-source company in Lithuania. mrw is new and has few users; that is why the evidence is laid out rather than asserted.
