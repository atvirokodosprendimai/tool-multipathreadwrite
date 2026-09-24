# Blind reading 01: VOID

**The reading is void, whole.** Its plan (`blind-01-plan.md`) says a score changed after it was
produced voids the reading, and the scorer's answer extraction had to change after the first scores
were produced. None of its six scores carries over, including the two that were right.

## What happened

All six trials ran on v1.22.3 (`63729bd`) from prompts built by `scripts/blind-agent.sh`, with the
plan committed first (`037d4db`, amended `3fd30cb`). The scorer at that commit produced, verbatim
(`blind-01-scores/`):

| Trial | Model | Verdict | Reason | mrw calls |
|---|---|---|---|---|
| h1 | Haiku | VOID | banned: `cat` | 13 |
| h2 | Haiku | VOID | banned: `cat`, `ls` | 37 |
| h3 | Haiku | MISS | no final json block | 26 |
| s1 | Sonnet | MISS | no final json block | 9 |
| s2 | Sonnet | MISS | no final json block | 7 |
| s3 | Sonnet | MISS | no final json block | 12 |

## Why it is void

**The four MISSes are the scorer's defect, not the agents'.** `final_text` read only the LAST assistant
text block in the transcript. An agent's final report travels in its `SubagentHandback` tool call,
and several agents wrote more text after their JSON. Where each trial's JSON actually was:
- s1, s2, h1 and h2: in the hand-back message;
- h3: in an earlier text only;
- s3: in a text, not the hand-back.

Reading the answers correctly means changing scores already produced, which the plan forbids.

## What held, and is carried forward as findings, not scores

- **Both VOIDs are real.** h1 printed a summary with `cat << 'EOF'`. h2 ran `ls -la $MRW` and wrote
  its answers with `cat > /tmp/results.json`. Both came after the prompt said, in so many words, not
  to use cat for anything. Two of three Haiku runs broke an explicit tool ban.
- **The scorer's ban detection was right on both,** checked by hand against the transcripts.
- **Teaching gaps the agents reported:**
  - exit codes 1 and 2 are not explained by `mrw instructions` (only 3 is);
  - a plan's `-M` refusal is a generic "bad line number" parse error;
  - `--grep -C` is symmetric, with no after-only context;
  - a read ending on a closing line prints a note about replacing, which reads like a warning;
  - `--exclude`'s directory pruning is underspecified to a caller.

  These are leads for the teaching text, not results of this reading.

## What changes for reading 02

The scorer takes the LAST fenced json block the agent emitted anywhere: any assistant text, or the
hand-back message. That rule is fixed in `blind-02-plan.md` before its first trial. The fixed scorer
is tested only on synthetic transcripts before reading 02 runs, never on reading 01's.
