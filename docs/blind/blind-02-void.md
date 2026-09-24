# Blind reading 02: VOID

**The reading is void, whole.** Its plan says a score changed after it was produced voids the
reading, and its mrw-call counts, which decide two verdicts, had to change after they were produced.

## What was produced (verbatim, `blind-02-scores/`)

| Trial | Model | Verdict | Correct | mrw calls (as counted) |
|---|---|---|---|---|
| h1 | Haiku | MISS | 9 | 26 |
| h2 | Haiku | VOID (banned: `cat`) | — | 27 |
| h3 | Haiku | MISS | 9 | 26 |
| s1 | Sonnet | MEETS | 9 | 3 |
| s2 | Sonnet | MEETS | 9 | 6 |
| s3 | Sonnet | MEETS | 9 | 9 |

## Why it is void

The scorer counted mrw calls with a regex over the raw command. It got them wrong both ways:

- **It counted assignments.** Haiku opened every command with `export MRW=/…/mrw`, and the regex
  counted each one as a call. h1 and h3 made 13 calls each, not 26, so their MISS verdicts were
  wrong: 9 of 9 correct in 13 calls meets the criterion.
- **It missed quoted calls.** Sonnet called the binary as `"$MRW" …`, and the regex needed
  whitespace before the name. s1's 3 and s3's 9 were undercounts, so the MEETS verdicts happened
  not to depend on them.

Reading 01's void was the answer channel. This one is the call count. Both are defects in the
scorer, which was tested on synthetic command strings only, never on the shapes agents write.

## What held

- Answer extraction was right. Every non-void run answered all nine tasks correctly: the per-task
  scores above, which the counting defect did not touch.
- h2's VOID is real: it used `cat`.

## What changes for reading 03

`command_words` in `scripts/blind-score.py` is a quote-aware parser. It splits on `|`, `;`, `&`,
`&&`, `||`, newlines and `$(` only outside quotes, drops `#` comments, and skips leading `VAR=value`
words. An mrw call is a command word that is `$MRW`, `${MRW}`, `mrw` or a path ending in `/mrw`, and
the ban check reads the same words.

It was checked against every invocation shape in the twelve void transcripts of readings 01 and 02,
as a check and not a re-score. The one comment case (`# Task 1:` lines) was found and fixed that
way. Where it disagreed with a hand-written recount, the recount was wrong: it counted an
assignment-only `MRW=/…/mrw` line. That check is described in `blind-03-plan.md`.
