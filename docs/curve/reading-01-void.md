# Reading 1 is VOID, and why — plus the finding it did produce

**Collected: 18 of 45 trials. Outcome: 18 hits, 0 misses, 0 refusals. Correct-address rate 100% at
every size and every position measured.** That number is not evidence about served size, and this
file exists so nobody reads it as though it were.

## What went wrong: the client never read the served window

The reading plan gave each client a directory holding `task.json` and `served.txt`, and told it to
read the served text completely. What the clients actually did — two of them said so unprompted in
their own reports — was **grep for the service name and jump straight to the block**:

> "Grep for the unique string `svc-gdamy` in served.txt located exactly one match; a targeted read of
> the surrounding lines confirmed the block."

The instruction names the target service, the generator gives every service a unique name, and any
client with a search tool therefore localises in one call at a cost independent of file size. The
independent variable was never manipulated. A 200,000-byte cell and a 2,000-byte cell are the same
task when you can grep, and the flat 100% is the signature of that, not of a model reading a large
window without loss.

The pre-registration named this threat in one form — *"a planted target is easy to find if it is a
unique string, and then the task measures string matching rather than reading"* — and answered it by
requiring near-identical distractors. Three near-identical distractors do defeat a human skimmer and
a model reading prose. They do not defeat `grep`, because the distractors differ in exactly the token
the instruction supplies.

## The token counts confirm it, and show the population was not even homogeneous

Per-trial client cost, 200,000-byte cells:

| Trial | Subagent tokens | Reading strategy implied |
|---|---|---|
| middle-1 | 142,648 | read the window |
| early-5 | 147,876 | read the window |
| late-4 | 91,191 | grep, self-reported |
| early-1 | 64,645 | grep |
| early-2 | 64,871 | grep |

Same cell, same size, and a factor of 2.3 in tokens between clients that read and clients that
searched. **The cell is a mixture of two different tasks**, so even its refusal count would not mean
one thing. Cells cannot be pooled across strategies, and nothing in the design recorded which
strategy a trial used.

## What this reading DOES establish, stated as narrowly as it deserves

**For a client with a search tool over a file on disk, localisation accuracy is unaffected by served
size across 2 KB to 200 KB, because the served size is not what the client processes.** 18 of 18.
That is a true statement about tool-using agents and it is not the question ADR-020 was built to
answer, which is what a model does with bytes *in its context*.

It is also the more interesting half of the pair for `MaxResultChars`: if real callers search rather
than read, then the cap governs a cost nobody pays in accuracy — and the honest way to find out is to
measure both arms, which is what reading 2 does.

## Reading 2, and what changes

The plan file for reading 2 states it. In short: the served text must reach the model as **context**,
not as a file it can index. Two arms, same fixtures, same criterion:

- **read arm** — the client receives the served text inline and has no search tool. This is the
  pre-registered question.
- **search arm** — the client gets the file and its tools, as here. This is reading 1 repeated on
  purpose, as the comparison.

The 18 trials collected here are kept as the search arm's first 18 observations. They are not
re-scored, not re-run, and not counted toward the read arm.

## The rule this cost, and it was already written down

`.claude/rules/testing.md`, first line: *"A green test that never reached the defect is the failure
mode here."* An eval is a test. Eighteen green trials that never reached the variable are the same
failure, and the tell was in the cost column before it was in anyone's report.
