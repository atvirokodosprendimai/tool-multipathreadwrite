# Reading 1 of the served-size curve — the parameters, fixed before collection

ADR-020 built the harness and `docs/adr/BACKLOG.md` carries the pre-registration: **correct-address
rate against served bytes, stratified by target position, refusals reported separately, and a flat
curve accepted as the answer.** None of that is re-derived here. What this file fixes is the part
the pre-registration left to the reading — the cells, the repeat count, the client, and how the
client is kept from the answer — and it is committed BEFORE the first trial runs, for the same
reason the criterion was.

## Cells, fixed

| Parameter | Value | Why this and not another |
|---|---|---|
| Served size | 2,000 · 20,000 · 200,000 bytes | Two orders of magnitude. The top is `MaxResultChars`, the number the reading exists to justify or refute; the bottom is a window a reader cannot get lost in. |
| Target position | early · middle · late | Pre-registered as strata, not nuisance. |
| Distractors | 3, held constant | Pre-registered: distractor count is a second IV and is not this experiment. |
| Repeats | 5 per cell | Fixed in advance. 3 × 3 × 5 = **45 trials per client**. |
| Seeds | 1..5 per (size, position) | Recorded so the fixtures are reproducible; the same seeds are reused for every client, so clients see identical trees. |

Five repeats give a wide Wilson interval by construction. That is stated here rather than discovered
later: this reading can show a large effect and cannot resolve a small one, and the tally prints the
interval so nobody has to take a point estimate on trust.

## The client, and what it may see

One trial is one **fresh subagent** with no memory of any other trial. It is given exactly two
things: the instruction from `manifest.json`, and `served.txt` — mrw's own rendering of what a read
would serve. It writes back a plan.

It is given **no path into the cell directory**, so `answer.json` is not merely undisclosed, it is
unreachable. The trial directory is split before any client runs: a client view holding the
instruction and the served text, and everything else out of reach.

Client 1 is a Claude Code subagent on **Sonnet**. A second client is a separate decision made after
reading 1, not a promise made here.

## What is scored

`curve score` applies the plan and diffs, so the changed line number is the measurement (ADR-020
Decision 3). Four outcomes: `hit`, `miss`, `refused_parse`, `refused_apply`. Refusals are reported
beside the cell and excluded from the correct-address denominator, per the pre-registration.

## What would make this reading void

- A client that reaches `answer.json`, or any file outside its view.
- A trial whose result does not echo its `trial_id` and `served_bytes` — the scorer refuses it, and a
  refused-to-score trial is reported as missing data, never silently re-run to a better number.
- Any change to the harness, the fixtures or these parameters after the first trial. If something is
  wrong with the design, the reading is abandoned and re-run whole under a new plan file, and this
  one stays in the tree saying so.
