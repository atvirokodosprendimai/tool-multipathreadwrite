# Blind reading 04: FAIL — on the call budget, with every run within the accuracy bar

**Both models FAIL the criterion pre-registered in `docs/adr/BACKLOG.md` ("From ADR-009"), so the
bench fails.** The criterion is at least 8 of 9 correct and at most 20 mrw calls. Every non-void run
answered 8 or 9 of 9 correctly. One Haiku run and one Sonnet run took **21** calls. Under the
verdict rule, one non-void miss fails a model.

**What changed.** This is reading 03's bench (`blind-03-result.md`, PASS) with its confound
removed. Each trial was a headless `claude -p --safe-mode` session started under `/tmp`, so no
CLAUDE.md, AGENTS.md, skill, hook or MCP server that teaches mrw was in its context. Fixture,
scorer and criterion are unchanged.

**The prompt is not byte-identical to reading 03's**, and the first version of this document said
it was (corrected after the Codex review of #216). The prompt pastes `mrw instructions` verbatim
(`scripts/blind-agent.sh:98`), and between v1.22.3 and v1.24.0 ADR-066 changed two of its
sentences. "If any hunk fails, nothing is written" became "if any hunk fails validation, nothing is
written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED)", and the `why` line
was narrowed the same way. No task exercises a failed commit, but this reading compares two
**conditions**, context and build together, not context alone.

The plan is `blind-04-plan.md`, committed and pushed at `dd4655f` before any trial ran. Build
v1.24.0 (`7da61c1`), as `mrw version` printed before dispatch
(`blind-04-scores/mrw-version.txt`). Each trial's prompt was piped from its own `prompt.txt`,
unedited.

## Scores (verbatim, `blind-04-scores/`)

| Trial | Model | Verdict | Correct | mrw calls | Note |
|---|---|---|---|---|---|
| h1 | Haiku | MEETS | 8 | 16 | |
| h2 | Haiku | VOID | — | 29 | banned: `head`, `ls` |
| h3 | Haiku | MISS | 9 | 21 | one call over |
| s1 | Sonnet | MEETS | 9 | 14 | |
| s2 | Sonnet | MISS | 9 | 21 | one call over |
| s3 | Sonnet | MEETS | 9 | 14 | |
| h4 | Haiku | MEETS | 9 | 18 | replacement for h2 |

Per model, first rule that applies: Haiku has a non-void MISS (h3), so it FAILs; so does Sonnet
(s2). h4 was dispatched because the plan requires one replacement per VOID. It could not change
Haiku's verdict and was scored anyway.

## Against reading 03

| | reading 03 (repo context) | reading 04 (none) |
|---|---|---|
| Sonnet mrw calls | 7, 8, 8 | 14, 14, 21 |
| Haiku mrw calls (non-void) | 15, 16, 14 | 16, 21, 18 |
| correct, every non-void run | 8–9 of 9 | 8–9 of 9 |
| VOID | 3 of 6 Haiku (`cat << 'EOF'`) | 1 of 4 Haiku (`head`, `ls`) |

**The binary's instructions alone are enough to get every task right, and not enough to get there
efficiently.** Under this reading's conditions, Sonnet's calls roughly doubled and correctness did
not move. The removed documentation is the likeliest cause, and the transcripts point at it (below).
The build difference cannot be ruled out by this reading alone.

## Where the calls went: `body=` is taught without its place

The largest single cost is visible in the transcripts. `mrw instructions` says what `body=` means
("a line count, not a character count", `body=@path`, "empty is `body=0`") but never shows it on an
`@@` header. **Four of the seven trials put `body=` on a plan line of its own** at least once: three
as `body=N`, and h4 as `body=` written straight before the content. Counted over every Bash command
in the reduced transcripts:

| Trial | `body=` on its own line | of which `body=N` | on a header |
|---|---|---|---|
| h1 | 0 | 0 | 3 |
| h2 | 7 | 7 | 0 |
| h3 | 0 | 0 | 0 |
| h4 | 4 | 0 | 0 |
| s1 | 0 | 0 | 0 |
| s2 | 3 | 3 | 4 |
| s3 | 2 | 2 | 0 |

On its own line, `body=1` is body text, so mrw wrote it into the file. s2's task 8 took ten Bash
calls (eleven mrw invocations). Six of those calls (seven invocations) came after that write, noticing
and undoing a `docs/meta.yaml` that now held a line reading `body=1`. Reading 03's agents had
AGENTS.md in context, which shows `@@ new.txt 0 create body=0` on the header, and none of its nine
trials put `body=` in a command at all.

A second, smaller cost: task 1's `--grep` attempts. s1 opened with `mrw -C "$F" --grep …` before
the subcommand, and s2 ran the grep three times.

**Predictions.**
1. Sonnet PASSES: **wrong**. It met the accuracy bar and missed the call bar by one.
2. Haiku PASS after replacements: **wrong**. Only one VOID this time, and a MISS instead.
3. Any drop is the context removed: **what the transcripts point at**, with the build difference
   named above as the one thing this reading cannot separate from it.

## What this FAIL does and does not show

- It shows the call criterion is sensitive to a teaching gap the instructions can close. The misses
  are one call over, and the dominant waste traces to one sentence the instructions lack.
- It does not show the instructions teach anything wrong. Every non-void run met the accuracy bar.
- The `--safe-mode` isolation rests on Claude Code's description and on a self-report probe, as the
  plan says. A headless session is not a subagent in other ways too.

## Follow-up

`mrw instructions` should show `body=` on the header, in one worked plan line. That is a change to
the served instructions (ADR-062/063 territory), so it gets a record rather than an edit. Recorded
in `docs/adr/BACKLOG.md`. A reading 05 on the build that ships it re-runs this exact plan.

## Evidence

- `blind-04-scores/<trial>.json`: the scorer's output, verbatim.
- `blind-04-replay/<trial>/`: `answer-key.json`, the scored `tree/docs/` files, and
  `transcript.jsonl` reduced to the assistant's text and tool calls. Session metadata is dropped;
  the scorer reads only these. Every score was recomputed from its kit, byte-identical.

**The scorer's known gaps did not touch this reading.** BACKLOG lists shapes the scorer misreads
(backslash-newline, heredocs, `command cat`, `cat` inside `$(…)`, `env mrw`). None appears in any
reading-04 transcript, checked 2026-09-24 over the committed replay kits, so no verdict here
depends on them.
