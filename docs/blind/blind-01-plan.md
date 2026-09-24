# Blind reading 01, plan: can a caller with only the binary find, read and plan?

**Written and committed before any trial ran.** The criterion was pre-registered in
`docs/adr/BACKLOG.md` ("From ADR-009", the blind-agent bench) on 2026-09-24, before the harness
existed. This plan fixes how that criterion is measured. It is frozen at its first trial, not at
its commit.

## Why

`mrw instructions` is the whole contract a caller with only the binary gets (ADR-037, ADR-062,
ADR-063). ADR-062 and ADR-063 both claim it is enough to use mrw well, and nothing has measured
that. A one-off on 2026-09-24 (one Haiku run, 9 of 9) prompted this bench. It predates the
criterion and is not a reading.

## The build under test

`mrw` v1.22.3, the tag cut from `63729bd`, installed at `~/.local/bin/mrw` and printing
`v1.22.3 (63729bd)`. If `mrw version` prints anything else when the first trial starts, the reading
does not start until the build is recorded here.

## Trials

- Six trials: three on Haiku and three on Sonnet, one fresh subagent each (the Agent tool's `haiku`
  and `sonnet` models, general-purpose type).
- Each trial gets its own fixture from `scripts/blind-agent.sh DIR`, a fresh `DIR`, and receives
  `DIR/prompt.txt` verbatim as its whole prompt.
- The nine tasks, the rules and the answer shape are in the prompt the script writes. The answer key
  is in `DIR/answer-key.json`.
- All six are dispatched together and scored after all six finish, so no score can shape a later
  prompt.

## Scoring

`scripts/blind-score.py DIR TRANSCRIPT`, per trial, on the subagent's JSONL transcript:

- **Compliance comes from the transcript's tool calls, not from the agent's report.** Any banned
  command, banned tool or `--help` makes the run VOID. A transcript the scorer cannot read also makes
  it VOID. Reading 12 in `docs/curve/` voided for exactly that. On 2026-09-24 this session read three
  subagent transcripts from their output files, which is why this plan relies on it.
- **An mrw call** is one invocation of the binary in a Bash command, counted from the transcript.
- **Task scores:** t1–t7 against the key; t8 and t9 against both the reported exit codes AND the
  files on disk afterwards.
- **A run MEETS the criterion** at ≥ 8 of 9 correct with ≤ 20 mrw calls.

## Verdict (the pre-registered rule, restated, not changed)

- A VOID run is replaced by a fresh trial, at most 3 replacements per model.
- Per model, the first rule that applies:
  - FAIL if any non-void run misses the criterion;
  - otherwise INCONCLUSIVE if it has fewer than 3 non-void runs after its replacements;
  - otherwise PASS.
- The bench passes only if both models PASS.

## Predictions, written before the first trial

1. Haiku PASSES. The 2026-09-24 one-off answered 9 of 9 on the same tasks with an older prompt, but under this bench's ban it would have been VOID: it wrote its plan files with `cat > FILE <<'EOF'`. The scorer caught that when it was first run on that transcript. The prompt now says in so many words that cat is banned even for writing a plan. That is a prompt change made before any trial, not a change to the pre-registered criterion.
2. Sonnet PASSES.
3. The task most likely to be missed is t6 (a comma list in one spec) or t1, if a run forgets
   `--exclude build`, since `--grep` does not read `.gitignore`.

## What voids the whole reading

- A build other than the one recorded above.
- A prompt edited between trials.
- A score changed after it was produced.
- A trial whose transcript was never scored.

## Known limitations

- Two models, one fixture family, three runs each. That is enough to fail the claim, and not enough
  to put a rate on it.
- Subagents run inside this session's host, so this is not a Desktop or a Codex population.
