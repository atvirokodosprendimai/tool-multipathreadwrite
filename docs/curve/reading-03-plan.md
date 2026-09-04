# Reading 3, plan: the same curve against a fixture that can be failed

**Written and committed before any cell of this reading was generated and before any trial ran.**
Reading 2's result stands as collected; nothing here changes it or reinterprets it.

## Why there is a third reading

Reading 2 returned 42 correct addresses in 45 trials, flat from 2,000 to 200,000 served bytes, with
every client verified to have read the served window whole. That is a **ceiling**, not a curve. Its
target carried a unique service name, so a client that read at all could find it, and the record said
so at the time rather than after.

ADR-020-T2 and T3 built the fixture that removes that: the instruction names no service, and the
target is the one block whose retry budget differs from every other. T3 then drew both budgets from
the trial seed, because T2 had left the odd value constant at `retries = 5` across every seed — a
signature a client could carry between cells. **This reading is the first one whose task can be
failed**, and therefore the first whose curve has room to bend.

## The criterion, which is not re-derived here

The pre-registration in `docs/adr/BACKLOG.md` governs, unchanged: correct-address rate against served
bytes, stratified by target position, refusals reported separately, **and a flat curve accepted as the
answer**. It also requires the measurement across models; this reading, like reading 2, is one client
population, so it does not discharge that and does not claim to.

## Cells, fixed before collection

3 sizes (2,000 / 20,000 / 200,000 served bytes) × 3 positions (early / middle / late) × 5 seeds =
**45 trials**, distractors held at 3, selector `odd-retries`. The sizes, positions, seeds and
distractor count are exactly reading 2's, so the two readings differ in the SELECTOR and in nothing
else. That is the whole design: one manipulated variable between readings.

## Arm

**Read arm only**, the same restriction reading 2's read arm used: the client may use a file reader
and a file writer and no search tool of any kind. Keeping the arm identical is what makes the two
readings comparable; adding a search arm here would change two things at once.

Compliance is verified per trial from its own transcript, not from the instruction:

- **zero** calls to any search tool, and
- **full coverage** of `served.txt`, computed by unioning each read call's offset and limit and
  comparing against the served row count.

A trial failing either is reported as void, not as a miss. Transcripts are matched on the arm's own
prompt marker, never on the cell id, because reading 2 found that both arms share cell ids and
matching by id reported three compliant trials as violations.

## What is scored

Unchanged from reading 2: `curve score` applies the plan and the changed line number is the
measurement. Outcomes are `hit`, `miss`, `refused_parse`, `refused_apply`. The 45 score files and the
`curve tally` over them are committed with the result, so the table can be recomputed.

## Predictions, recorded before collection

Stated so the result cannot be read as whatever it turns out to be:

1. **The overall rate will be lower than reading 2's 42/45.** If it is not, the relational fixture is
   not meaningfully harder and T2/T3 bought nothing measurable — a real and publishable outcome.
2. **If a curve exists, it bends downward with served bytes**, because the property cannot be
   evaluated without obtaining every block's value and more bytes means more to hold.
3. **A flat curve remains an acceptable answer** and would then be a stronger one than reading 2's,
   because it would be flat on a task that can be failed.

## What would void this reading

The fixtures, the cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`; or a client using a search tool. As before that is a re-run under a new plan file, not
an edit to this one.

## Known limitations, stated now

Five repeats per cell resolve only large effects, and one client population does not discharge the
pre-registration's across-models requirement. The budgets are small integers, so a determined client
could enumerate the seven values rather than compare blocks; T3 records that this removes the
constant, not the alphabet.
