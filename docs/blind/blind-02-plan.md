# Blind reading 02, plan: can a caller with only the binary find, read and plan?

**Written and committed before any trial ran.** It replaces blind reading 01, which is void whole
(`blind-01-void.md`): its scorer read the wrong channel for the agents' answers. The criterion is the
one pre-registered in `docs/adr/BACKLOG.md` ("From ADR-009", the blind-agent bench), unchanged. This
plan is frozen at its first trial, not at its commit.

## What changed from reading 01, and nothing else did

1. **Scoring reads the last fenced json block the agent emitted anywhere:** any assistant text, or the
   `SubagentHandback` message. It is tested on synthetic transcripts only, never on reading 01's.
2. **The prompt adds one sentence:** "Every command you run is recorded. One banned command voids the
   whole run, including one used only to print, save or format your answer. Give your answer as
   text." In reading 01, two of three Haiku runs used `cat` to print or save their answers after
   being told not to.
3. **Fresh fixtures, fresh agents.** No trial or transcript of reading 01 is reused.

## The build under test

`mrw` v1.22.3, the tag cut from `63729bd`, installed at `~/.local/bin/mrw` and printing
`v1.22.3 (63729bd)`. If `mrw version` prints anything else when the first trial starts, the reading
does not start.

## Trials, scoring and verdict

As in `blind-01-plan.md`, with the scoring change above:
- six trials (Haiku ×3, Sonnet ×3), each on a fresh `scripts/blind-agent.sh DIR`, its `prompt.txt`
  passed verbatim;
- all dispatched together, and scored only after all six finish;
- compliance comes from the transcript's tool calls;
- a run MEETS the criterion at ≥ 8 of 9 correct with ≤ 20 mrw calls;
- a VOID run is replaced, at most 3 replacements per model;
- per model, the first rule that applies: FAIL if any non-void run misses; otherwise INCONCLUSIVE if
  fewer than 3 non-void runs; otherwise PASS;
- the bench passes only if both models PASS.

## Predictions, written before the first trial

1. **Sonnet PASSES.** In reading 01 no Sonnet run broke the ban, and each of its reports held a
   complete answer. That is an observation from a void reading, stated as the reason for this
   prediction, not as evidence.
2. **Haiku is the risk, on compliance rather than on mrw.** If its runs keep breaking the ban,
   replacements run out and Haiku comes out INCONCLUSIVE, which says nothing about whether the
   instructions teach mrw.
3. **mrw calls:** Sonnet well under 20. Haiku is the one that could exceed 20; h2 in reading 01 made
   37.

## What voids the whole reading

- A build other than the one recorded above.
- A prompt edited between trials.
- A score changed after it was produced.
- A trial whose transcript was never scored.

## Known limitations

As in `blind-01-plan.md`:
- two models, one fixture family, three runs each;
- subagents run in this session's host;
- instructions-only is enforced at the tool level, not the context level, because subagents
  probably load this repository's CLAUDE.md and AGENTS.md.
