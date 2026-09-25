# Blind reading 04, plan: can a caller with only the binary find, read and plan — with nothing else in its context?

**Written and committed before any trial ran.** It follows reading 03 (`blind-03-result.md`), which
PASSED on Haiku and Sonnet but named its own confound: every trial was a subagent of a session in
this repository, so it probably loaded this repository's CLAUDE.md and AGENTS.md and the user's
global CLAUDE.md, all of which teach mrw at length. This reading removes that confound, and changes
nothing else. The criterion is the one pre-registered in `docs/adr/BACKLOG.md` ("From ADR-009"),
unchanged. This plan is frozen at its first trial.

## What changed from reading 03, and nothing else did

**Where a trial runs.** Each trial is a headless Claude Code session, not a subagent:

```sh
cd <fixture dir under /tmp>        # outside any git repository
claude -p --safe-mode --model <haiku|sonnet> --allowedTools Bash \
  --output-format stream-json --verbose < prompt.txt > transcript.jsonl
```

- `--safe-mode` disables CLAUDE.md, skills, installed plugins, hooks, MCP servers and custom agents
  (Claude Code 2.1.281's own `--help`). A probe on 2026-09-24 in `/tmp` with this flag reported no
  CLAUDE.md or project instructions, no MCP tools, only built-in skills, and no knowledge of mrw.
  That is the session's self-report, not proof; it is why the prompt is also recorded (below).
- `--allowedTools Bash`: a headless session cannot ask for approval, so the one tool the prompt
  allows is pre-approved and every other tool is denied. A denied call still appears in the
  transcript as a `tool_use`, so the ban check sees an attempt exactly as it did in reading 03.
- The fixture directory is created fresh under `/tmp` by `scripts/blind-agent.sh`; no ancestor
  directory is a repository that documents mrw.

The prompt, the fixture, the answer extraction, the scorer (`scripts/blind-score.py`, unchanged
since reading 03), the criterion and the verdict rule are reading 03's.

## Build, trials, scoring, verdict

- v1.24.0 (`7da61c1`), the installed `~/.local/bin/mrw`; `mrw version` is recorded before dispatch.
  The CLI surface is v1.23.0's plus nothing: ADR-067 changed only the MCP server. `mrw instructions`
  is whatever that build prints, pasted into `prompt.txt` by `blind-agent.sh`.
- Haiku ×3 and Sonnet ×3 on fresh fixtures, dispatched together and scored after all six finish.
- A VOID run is replaced, at most 3 replacements per model; replacements are dispatched after the
  first six are scored, one per VOID, and scored the same way.
- Per model, the first rule that applies: FAIL if any non-void run misses; otherwise INCONCLUSIVE if
  fewer than 3 non-void runs; otherwise PASS. The bench passes only if both models PASS.
- Each trial's `prompt.txt` is compared byte for byte with what was piped to `claude -p`.
- Evidence: a replay kit per trial (`answer-key.json`, `tree/`, `transcript.jsonl`) under
  `blind-04-replay/`, as reading 03 did, so every score recomputes from the tree.

## Predictions, written before the first trial

1. **Sonnet PASSES.** All three Sonnet runs in reading 03 answered 9 of 9 in 7–8 calls with no ban
   violation, and so did all six in the void readings 01 and 02 (a basis for predicting, not
   evidence). The instructions it needs are all in the prompt.
2. **Haiku is the uncertain one, and the likeliest outcome is PASS only after replacements.** In
   reading 03, three of its six runs were VOID for `cat << 'EOF'` and all three allowed replacements
   were needed; its valid runs answered 8–9 of 9 in 14–16 calls (h2 got task 9 wrong and still met the
   criterion at 8 of 9). Without the repository's documentation, INCONCLUSIVE (too many VOIDs) or FAIL (a run below 8 of 9) are both live.
3. **If either model does worse than in reading 03, the difference is the context this reading
   removed** — the only variable changed — and that is the finding, whatever the verdict.

## What voids the whole reading

- A build other than the one recorded.
- A prompt edited between trials, or one that differs from its `prompt.txt`.
- A score changed after it was produced.
- A trial whose transcript was never scored.
- A trial started without `--safe-mode`, or from a directory inside a repository.

## Known limitations

Those of `blind-03-plan.md`, and:
- `--safe-mode` is Claude Code's own description of what it disables; this reading does not audit
  the system prompt beyond the probe named above.
- A headless session and a subagent differ in more than context (session setup, tool-permission
  path). The comparison with reading 03 is a comparison of conditions, not of context alone.
