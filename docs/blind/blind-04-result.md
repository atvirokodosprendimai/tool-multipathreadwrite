# Blind reading 04: FAIL — on the call budget, with every answer right

**Both models FAIL the criterion pre-registered in `docs/adr/BACKLOG.md` ("From ADR-009"), so the
bench fails.** The criterion is at least 8 of 9 correct and at most 20 mrw calls. Every non-void run
answered 8 or 9 of 9 correctly. One Haiku run and one Sonnet run took **21** calls. Under the
verdict rule, one non-void miss fails a model.

**What changed, and the one thing that did.** This is reading 03's bench (`blind-03-result.md`,
PASS) with its confound removed. Each trial was a headless `claude -p --safe-mode` session started
under `/tmp`, so no CLAUDE.md, AGENTS.md, skill, hook or MCP server that teaches mrw was in its
context. Prompt, fixture, scorer and criterion are unchanged. The plan is `blind-04-plan.md`,
committed and pushed at `dd4655f` before any trial ran. Build v1.24.0 (`7da61c1`), as `mrw version`
printed before dispatch (`blind-04-scores/mrw-version.txt`). Each trial's prompt was piped from its
own `prompt.txt`, unedited.

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
efficiently.** Removing the repository's documentation roughly doubled Sonnet's calls and left
correctness where it was. That is prediction 3's case, and it answers the question reading 03 left
open.

## Where the calls went: `body=` is taught without its place

The largest single cost is visible in the transcripts. `mrw instructions` says what `body=` means
("a line count, not a character count", `body=@path`, "empty is `body=0`") but never shows it on an
`@@` header. Three of the seven trials put it on a plan line of its own at least once:

| Trial | `body=` on its own line | on a header |
|---|---|---|
| h2 | 7 | 0 |
| s2 | 3 | 4 |
| s3 | 2 | 0 |
| others | 0 | 0–2 |

On its own line, `body=1` is body text, so mrw wrote it into the file. s2's task 8 took ten
commands, six of them after that write, noticing and undoing a `docs/meta.yaml` that now held a line reading
`body=1`. Reading 03's agents had AGENTS.md in context, which shows `@@ new.txt 0 create body=0`
on the header, and none of them did this.

A second, smaller cost: task 1's `--grep` attempts. s1 opened with `mrw -C "$F" --grep …` before
the subcommand, and s2 ran the grep three times.

**Predictions.**
1. Sonnet PASSES: **wrong**. It met the accuracy bar and missed the call bar by one.
2. Haiku PASS after replacements: **wrong**. Only one VOID this time, and a MISS instead.
3. Any drop is the context removed: **the case that occurred.**

## What this FAIL does and does not show

- It shows the call criterion is sensitive to a teaching gap the instructions can close. The misses
  are one call over, and the dominant waste traces to one sentence the instructions lack.
- It does not show the instructions teach anything wrong. Every answer was right.
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
