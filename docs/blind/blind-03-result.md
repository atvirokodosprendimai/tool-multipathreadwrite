# Blind reading 03: PASS

**Both models PASS the criterion pre-registered in `docs/adr/BACKLOG.md` ("From ADR-009"), so the
bench passes.** A fresh agent whose prompt gave it only `mrw instructions` and a tree it had never
seen answered the nine read/plan tasks within the limits: at least 8 of 9 correct, at most 20 mrw
calls. It did so in every run that kept the tool ban, on Haiku and on Sonnet.

**What this PASS does not show.** The prompt restricts the tools, not what is in the agent's
context. `blind-01-plan.md` names this confound, and readings 02 and 03 inherit it. Each trial
was a subagent started from a session in this repository, so it probably also loaded this
repository's CLAUDE.md and AGENTS.md and the user's global CLAUDE.md, all of which teach mrw at
length. All three Sonnet runs reported that a SubagentStart hook asked them to search team memory,
and that they skipped it. So the PASS shows an agent can do these tasks with the binary and the
instructions. It does not show that the binary's instructions alone are enough. Measuring that
needs a trial started outside any repository that documents mrw.

The plan is `blind-03-plan.md`, committed at `716609c` before any trial ran. Build v1.22.3 (`63729bd`), as
`mrw version` printed before dispatch. Every trial's prompt was compared byte for byte with its
`prompt.txt` after dispatch, and all nine were identical apart from the fixture path.

## Scores (verbatim, `blind-03-scores/`)

| Trial | Model | Verdict | Correct | mrw calls | Note |
|---|---|---|---|---|---|
| h1 | Haiku | VOID | — | 15 | banned: `cat << 'EOF'` to print its findings |
| h2 | Haiku | MEETS | 8 | 15 | t9: after the `-2` refusal it deleted lines 1-2 with `1-2 delete` |
| h3 | Haiku | VOID | — | 17 | banned: `cat << 'EOF'` to print its results |
| s1 | Sonnet | MEETS | 9 | 8 | |
| s2 | Sonnet | MEETS | 9 | 8 | |
| s3 | Sonnet | MEETS | 9 | 7 | |
| h4 | Haiku | MEETS | 9 | 16 | replacement for h1 |
| h5 | Haiku | VOID | — | 15 | replacement for h3; banned: `cat << 'EOF'`, its answers were 9 of 9 |
| h6 | Haiku | MEETS | 9 | 14 | replacement for h5, the third and last Haiku replacement allowed |

- **Haiku: PASS.** Non-void runs: h2, h4 and h6, all MEETS. Three replacements were used, which is
  the limit.
- **Sonnet: PASS.** Non-void runs: s1, s2 and s3, all MEETS. No replacement was needed.

The six first trials were dispatched together and scored after all six finished. The replacements
went out one per VOID, after that, as the plan says.

## Predictions, against what happened

1. **Sonnet PASSES.** It did: 9 of 9 in 7 or 8 calls, three runs out of three.
2. **Haiku PASSES if a replacement covers its VOID.** It did, but only just. Three of Haiku's six runs
   were VOID, each for `cat`, so it needed every replacement the plan allowed. A fourth `cat` would
   have made Haiku INCONCLUSIVE.

## How the scores were checked

Every verdict was checked against its transcript by hand, and no score was changed.
- **Call counts:** recounted from each Bash call for s2, s3, h4 and h6, and all four matched. The s2
  agent's own report said 10 calls (7 reads, 3 writes). The transcript shows 8, and so does the
  scorer.
- **VOIDs:** all three are real. Each is a `cat` heredoc written after the prompt said "Do NOT use
  cat for anything".
- **h2's t9 miss is real.** The task asks for a write plan to be tried. h2 tried it, got exit 2, then
  deleted lines 1-2 of `docs/notes.txt` with a line-number plan. The key requires that file to be
  unchanged (`blind-score.py`, the t9 rule).

## Known limitations, found in this reading

`command_words` has gaps that `blind-03-plan.md` does not list. This reading found two:
- **Backslash continuations.** It turns an unquoted newline into a separator even after a backslash,
  so the first word of each continued line is read as a command.
- **Heredoc bodies.** It reads each line of a heredoc body as a command.

The Codex review of PR #206 reproduced more of them on synthetic transcripts:
- `command cat f`, and a `cat` inside `"$(…)"`, score MEETS instead of VOID;
- `env mrw read f` is not counted as a call;
- a banned word printed inside a double-quoted argument after `;` scores VOID;
- a final JSON fence that is not an object is skipped, so an earlier one wins;
- a task answer of the wrong type crashes the scorer instead of scoring a miss.

None of them changed a verdict here, and this was checked on all nine transcripts:
- **Continuations:** joining them before parsing changes no ban and no call count.
- **Heredocs:** only h1, h3 and h5 contain one, and each wraps a real `cat`, which voids the run by
  itself.
- **Other banned words:** outside single-quoted literals, every `grep` is mrw's `--grep` flag. Every
  `find` is in a `#` comment. The only backticks are in comments and in h5's heredoc.
- **Other shapes:** no transcript uses `env`, `command`, `$(`, `eval` or `xargs`.
- **Answers:** every JSON fence is an object, and all nine transcripts scored with exit 0.

The gaps are recorded in `docs/adr/BACKLOG.md` for the next reading. The scorer is left as this
reading ran it, because changing it now would change the instrument after its scores were produced.

## Replay

`blind-03-replay/<trial>/` holds what the scorer reads for each of the nine trials:
- `answer-key.json`;
- the final `tree/docs/meta.yaml` and `tree/docs/notes.txt`;
- `transcript.jsonl`, the full subagent transcript reduced to the assistant's text and tool calls.

Tool results and everything else are dropped, and nothing kept is edited. Run the unchanged scorer on
each kit and it reproduces the committed score exactly:

    for t in h1 h2 h3 h4 h5 h6 s1 s2 s3; do d=docs/blind/blind-03-replay/$t
      python3 scripts/blind-score.py $d $d/transcript.jsonl | diff - docs/blind/blind-03-scores/$t.json; done

The audit above can be rerun on the same files. Readings 01 and 02 committed only their scores.

## Teaching leads the agents reported

These are what the agents said the instructions left unclear. They are leads, not results.
- **The `-M` refusal on a write.** `@@ f -2 delete` fails as `bad line number ""`, exit 2. That is a
  parse error that does not name the form. s1 and s3 reported it, and reading 01 found it too.
- **Exit codes.** `mrw instructions` explains exit 3 and the read's exit 1, but not a write's exit 1
  (a hunk failed) or exit 2 (usage). s2 reported that it had to learn them from the output.
- **`--exclude` on a directory.** All three Sonnet runs avoided `--exclude`. They grepped the whole
  tree and dropped `vendor/` and `build/` by hand, because the text does not say whether a bare
  directory name prunes the subtree. It does: Haiku runs h4, h5 and h6 used `--exclude vendor
  --exclude build` and got it right.
- **Pattern matching.** One Sonnet run noted that `/regexp/` is a substring match unless anchored,
  which is why `/status:/` matched two lines.

Re-scored with the ADR-070 scorer (2026-09-25): all nine trials in `blind-03-replay/`, old scorer
(`main` at `c43095d`) against the fixed one. Every verdict, correct count and mrw call count is
unchanged, so this reading's PASS stands.
