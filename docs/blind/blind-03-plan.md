# Blind reading 03, plan: can a caller with only the binary find, read and plan?

**Written and committed before any trial ran.** It follows readings 01 and 02, both void on scorer
defects (`blind-01-void.md`, `blind-02-void.md`). The criterion is the one pre-registered in
`docs/adr/BACKLOG.md` ("From ADR-009"), unchanged. This plan is frozen at its first trial.

## What changed from reading 02, and nothing else did

**An mrw call** is a command word, as `command_words` in `scripts/blind-score.py` finds it, that is
`$MRW`, `${MRW}`, `mrw` or a path ending in `/mrw`. The parser:
- splits segments on `|`, `;`, `&`, `&&`, `||`, newlines and `$(`, only outside quotes;
- drops `#` comments;
- skips leading `VAR=value` words.

The ban check reads the same words.

How the counter was checked before this plan:
- 13 hand-written shapes, every one taken from the twelve void transcripts, all pass;
- a run over those twelve transcripts, as a check and never a re-score;
- the one defect that run found (comments swallowing the command) is fixed.

The prompt, the fixture, the answer extraction, the criterion and the verdict rule are reading 02's.

## Build, trials, scoring, verdict

As in `blind-02-plan.md`:
- v1.22.3 (`63729bd`);
- Haiku ×3 and Sonnet ×3 on fresh fixtures, dispatched together and scored after all six finish;
- a VOID run is replaced, at most 3 replacements per model;
- per model, the first rule that applies: FAIL if any non-void run misses; otherwise INCONCLUSIVE if
  fewer than 3 non-void runs; otherwise PASS;
- the bench passes only if both models PASS.

Replacements are dispatched after the first six are scored, one per VOID, and scored the same way.

## Predictions, written before the first trial

These use what readings 01 and 02 showed. They are void readings, so this is a basis for predicting,
not evidence.
1. **Sonnet PASSES.** Six of six Sonnet runs across both void readings answered 9 of 9 without a ban
   violation.
2. **Haiku PASSES if a replacement covers its VOID.** Haiku's valid runs answered 9 of 9 in about 13
   calls. About one Haiku run in three used `cat` (h2 in both readings), so replacements are likely
   to be needed.

## What voids the whole reading

- A build other than the one recorded.
- A prompt edited between trials.
- A score changed after it was produced.
- A trial whose transcript was never scored.

## Known limitations

Those of `blind-02-plan.md`, and one more: readings 01 and 02 informed the scorer and these
predictions. The prompt, tasks and criterion were not changed on their account.
