# Reading 15, plan: a second model family, through a prompt delivery

**Written and committed before any trial ran.** Nothing here changes any earlier reading.

## Why a fifteenth reading

Every client so far came from one vendor — Sonnet and Haiku — and the note's limits say so. This
reading takes the same forty-five cells to a client from another family, OpenAI's `gpt-5.6-sol`
driven through the Codex CLI, so that "served size did not bend the curve" and "a plausible second
number beside mrw's did" can be told from "this vendor's clients behave so".

## Cells

Reading 4's forty-five cells — fifteen each at 2,000, 20,000 and 200,000 bytes, byte-identical to
the copies readings 8–13 used (verified before the first trial), window from line 1. So every
trial pairs by cell with readings 4 (reader, Haiku), 8 and 9 (bare tool result, Haiku), 3 (reader,
Sonnet) and 11 and 13 (tool result with the number, Haiku).

## Client and arm

**Client:** `gpt-5.6-sol` at `model_reasoning_effort="high"`, one fresh `codex exec --ephemeral`
process per trial, `--ignore-user-config`, sandbox read-only, working directory an EMPTY directory
made for the trial, so no file on disk holds the served text or the answer.

**Arm: prompt delivery.** The prompt, read from stdin, is the cell's instruction (`task.json`'s
`instruction` field, verbatim) followed by the cell's `served.txt` verbatim — mrw's rendering,
`N|` the only number on any row — and the closing sentence "Reply with the plan text and nothing
else." The client's final message (`-o`) is the plan. This is a third delivery form beside a file
reader and a Bash tool result: the served text arrives as part of the prompt, with no tool call
between the client and the bytes.

**Compliance.** The client has no tool that can reach the served text other than reading its own
prompt; whatever it runs in the empty directory finds nothing, and is recorded from the event
stream and reported, not disqualifying. A trial is void if the final message holds no parseable
plan (no `@@` header) or holds more than one hunk; a void trial is reported, never counted as a miss.
Cost is not measured: the CLI reports no per-request token counts this harness can attribute, and
the result document says so.

## Predictions, recorded before collection

1. **At the ceiling at every size: at least 14 of 15 in each tier.** The author's expectation,
   stated so it can be wrong.
2. **No miss at `target+2`** — no outer number exists in this delivery — and a miss, if any, is
   classified as a wrong-service miss (a distractor's `timeout` line) or an off-by-N on the right
   service, reported separately.
3. **Compliance at least 42 of 45.**

## What this decides

At the ceiling: a second vendor's client through a delivery with mrw's number alone is flat across
a hundredfold in served bytes, and the account "the number, not the size" gains a family. Bent:
the first client to bend without a second number is found, and served size becomes a variable for
that family through this delivery — which reopens the cap question for it. Either is accepted.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json` (it cannot: nothing is on disk).
