# Reading 19 is VOID, whole — and the reason it is whole is worth more than the number it cost

**Collected 2026-09-05 under `reading-19-plan.md`, committed before any trial ran. Thirty trials
collected. No rate is published, and no score is committed.**

The first version of this document published the 2 KB stratum as 15 of 15 and voided only the 20 KB
stratum. That was wrong, and the Codex review of PR #115 caught it. This document replaces it.

## What went wrong

Two deviations, both the author's, both in the second half of the run:

1. **The harness changed after the first trial.** After `20000-middle-3` reported the file's last
   line correctly by number and words but dropped its leading `# `, the coverage instruction in the
   final six dispatches gained the phrase "copied exactly, including any leading `#`".

2. **A failed trial was retried, and the plan does not authorize it.** `20000-middle-5`'s first
   client ended without writing either file. A second was dispatched. Retrying only non-answers
   filters the surviving set: answers are never retried, so the survivors are not a sample of what
   the arm produces.

## Why the whole reading voids, and not just the stratum

`reading-19-plan.md:152` says, without qualification:

> The cells, this plan or **the harness changing after the first trial**; a client reaching
> `answer.json`; a `coverage.json` not strictly newer than its `result.json`; a cell failing the
> three-field identity check.

It voids **this reading**. It does not say "the affected stratum", and it does not say "unless the
data predates the change".

The argument for keeping the 2 KB stratum was that it is untainted in fact: its fifteen trials all
ran before either deviation, and its fifteen prompts were verified byte-identical apart from the cell
name. That is true, and it is not enough. **The rule was written before the results were seen; the
exception was invented after.** Reading 12 was voided on compliance that could not be checked, and
reading 18 on a delivery defect — both when their numbers were ambiguous or bad. Discovering a
stratum-level exception in the one case where the number came out well is the exact asymmetry a
pre-registration exists to prevent, and a corpus that allows it once cannot be read strictly
anywhere.

So the pre-registration binds as written. Thirty trials are void.

## What the void trials showed, as a statement about what to expect and NOT as a rate

Recorded so reading 20 knows what it is walking into. None of this is a measurement, and none of it
is committed:

- Every trial that produced an answer addressed the planted line, at both sizes.
- `next_read_sends` was 0 in every trial at both sizes, consistent with 2,000 and 20,000 bytes
  sitting far below both mrw's cap and the truncation reading 18 found.
- One trial in thirty produced no answer at all.
- One trial reported the file's last line without its comment marker.

A reader who wants to know whether the MCP delivery arm is at the ceiling must wait for reading 20.
ADR-020's item is **not** discharged by this reading at any size.

## What this cost, and what it bought

It cost thirty trials and the first number this corpus would have had for the shipped delivery path.

It bought three rules that reading 20 pre-registers instead of discovering:

1. The `last_line_text` comparison and the prompt wording that feeds it are **both** fixed before the
   first trial, so a transcription slip cannot tempt a mid-run clarification.
2. A client that writes no `result.json` is a **`no_answer`** — recorded beside hits and misses,
   never folded in, and **never retried**.
3. The **coverage reports are committed** alongside the scores. Reading 19's first result document
   published "compliant 15 of 15" and "zero paged reads" while its provenance section claimed every
   number came from the committed scores; the scores carry neither. Secondary numbers need committed
   machine-readable evidence, exactly as the primary does. That gap was the Codex review's second
   finding and it is closed by construction rather than by care.

## Provenance

Thirty scores and thirty compliance reports exist and are **not committed**, because they are not
measurements. The plan they ran under is `docs/curve/reading-19-plan.md`, unedited since it was
committed at `4132bd8`. The cells were the fifteen 2 KB and fifteen 20 KB cells verified against
`docs/curve/reading-13-scores/` on `trial_id`, `served_bytes` and the planted line; their sha256s are
in the plan.
