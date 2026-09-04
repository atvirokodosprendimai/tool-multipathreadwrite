# Reading 2 — two arms, fixed before collection

Reading 1 is void (`reading-01-void.md`): the clients grepped, so served size was never manipulated.
The pre-registration in `docs/adr/BACKLOG.md` is unchanged and is not re-derived here. What this file
fixes is the instantiation, which is what failed.

## The change that matters

**The served text must reach the model as CONTEXT, not as a file it can index.** A client that can
search localises at a cost independent of size, which is what happened, and no amount of distractors
fixes it because the distractors differ in exactly the token the instruction supplies.

Two arms, the same 45 fixtures and seeds, so the arms are comparable trial by trial:

| Arm | Client may use | Answers |
|---|---|---|
| **read** | `Read` only. Grep, Glob and Bash are forbidden by the prompt. | The pre-registered question: does a model's next edit get worse as more is served *into its context*? |
| **search** | Its full toolset, as reading 1 | What a tool-using agent actually does, and the honest comparison |

## Compliance is verified, not trusted

Every trial's subagent transcript is grepped for search-tool invocations after the fact. A read-arm
trial whose transcript contains a `Grep`, `Glob` or `Bash` call is **void and reported as void** — it
is not re-run to a better number, and it is not counted in either arm. The count of voided trials is
published beside the cells.

This is the part reading 1 lacked: it asked the client to read completely and had no way to know that
it had not.

## Cells, repeats, seeds

Unchanged from reading 1, so the arms and the voided reading share fixtures: 2,000 / 20,000 / 200,000
bytes × early / middle / late × 5 seeds = **45 trials per arm**. Distractors held at 3.

The search arm already holds 18 of its 45 from reading 1. Those stand as collected. The remaining 27
are run only if the read arm shows an effect worth comparing against; if the read arm is flat too,
18 observations are enough to say the search arm was flat and the cost of 27 more buys nothing.

## What is scored, and what would void the reading

Scoring is unchanged: `curve score` applies the plan and the changed line number is the measurement.

The reading is void if the fixtures, the cells, this plan or the harness change after the first trial,
or if a client reaches `answer.json`. As before, that is a re-run under a new plan file, not an edit
to this one.

## Known limitation, stated now rather than found later

Five repeats give wide intervals. This reading can detect a large effect and cannot resolve a small
one, and a flat result means "no large effect at this difficulty", which is not the same as "no
effect". The tally prints the interval so the claim cannot be read stronger than the data.

## Verifying compliance: the trial id is not a key

Both arms run against the same cell ids, so a transcript found by searching for `200000-early-2`
matches the read-arm run **and** the search-arm run of that cell. Matching on the id alone reported
three of the first six read-arm trials as non-compliant; every one of those hits was the search arm's
transcript for the same cell.

Compliance is therefore read from transcripts filtered on the read arm's own marker — the words
`TOOL RESTRICTION`, which appear only in the read arm's prompt — and only then counted for
`Grep`, `Glob`, `Bash` and `WebSearch` calls. With that filter the first nine read-arm trials all
returned `search=0`.
