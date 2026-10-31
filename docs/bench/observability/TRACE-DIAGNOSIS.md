# Trace diagnosis — turns, tokens and cost, 2026-10-30

**Fewer turns can save money, but only when the work removed costs more than the work added.**
The saved traces contain both a win and a counterexample. This post-hoc diagnosis uses the ten
[grouped](GROUPED-RESULTS.md) and four earlier [smoke](CLAUDE-RESULTS.md) task cells, kept separate.
All fourteen have verified final checks and exact-reference solutions. No benchmark model calls
were added and no original observation, registration, guidance or engine behavior was changed.

[Calculated evidence](trace-diagnosis.json) contains per-message input usage, per-tool sizes and
fingerprints, and reconciled cost components. The [offline analysis source](trace-diagnosis.py.txt)
checks every raw transcript hash against its original published record before calculating.
Raw transcript text remains outside the public checkout; exported sizes count the original text.

## The assumption fails in a successful, shorter run

On the grouped structural task:

| Measure | Native | MRW |
|---|---:|---:|
| CLI turns | 4 | 3 |
| Tool calls | 3 | 2 |
| Final reported output tokens | 771 | 617 |
| Cache-read input tokens | 28,825 | 38,079 |
| Cache-creation input tokens | 2,854 | 3,371 |
| CLI-estimated cost | $0.0220245 | $0.0234739 |

MRW used fewer turns and 20.0% fewer output tokens, yet cost 6.6% more. Output saved $0.0015400,
while input categories added $0.0029894, giving a net $0.0014494 increase. Both solutions passed.
The amount and kind of tokens processed per turn matter as well as the number of turns.

A second measurement issue appears in the smoke pilot: native's single-line task reports six CLI
turns but contains four unique emitted assistant message IDs; its multisite task reports five turns
and also contains four messages. Grouped native/MRW counts happen to match message counts.
Thus `num_turns` is retained as the CLI counter, not interpreted as a universal API-request count.
Emitted message IDs also do not enumerate hidden transport retries or every billable HTTP request.

## Where the grouped run's additional cost is recorded

| Token category | Native tokens | MRW tokens | Native cost | MRW cost | MRW minus native |
|---|---:|---:|---:|---:|---:|
| Other input | 32 | 34 | $0.0000640 | $0.0000680 | $0.0000040 |
| Output | 2,835 | 2,906 | $0.0283500 | $0.0290600 | $0.0007100 |
| Cache reads | 112,883 | 218,496 | $0.0112883 | $0.0218496 | $0.0105613 |
| Cache creation | 14,744 | 16,933 | $0.0589760 | $0.0677320 | $0.0087560 |
| Total | — | — | $0.0986783 | $0.1187096 | $0.0200313 |

**96.5% of the additional estimated cost is in input categories**, including 52.7% in cache reads
and 43.7% in cache creation. Only 3.5% is additional output cost. This is accounting attribution,
not proof that a particular instruction or receipt caused those amounts.

For these receipts, cost is reproduced by summing tokens times these effective USD-per-million
coefficients: other input 2, output 10, cache reads 0.1, cache creation 4. All recorded creation
is one-hour cache creation. These coefficients reconcile all fourteen CLI estimates within
$0.000000000001; they are not an independently verified current provider tariff or an invoice.
No tool-time charge is added to this token-value calculation. Runtime latency is a separate outcome.

The final result's usage is authoritative. Duplicate streamed fragments are grouped by message ID.
Their input counters sum exactly to every final input/cache total. Their output counters do not:
for example, structural native frames sum to 347 while final output usage is 771. The diagnosis
therefore does not assign billed output tokens or dollars to individual text/tool fragments.

## What changed in the actual workflow

1. **The MRW treatment started with substantially more context.** Native's first requests contain
   6,985–7,124 input/cache tokens; MRW's contain 12,591–12,727. Matched differences are 5,602–5,606
   tokens before any tool result arrives. First-request cache-read prefixes are 5,807 versus 11,412
   in every grouped pair. The full appended MRW guide reconstructs to 23,187 characters / 23,222
   UTF-8 bytes from the unchanged pinned binary, SHA256
   `a76dbd46ae65184aae51853b372c174f845a99fac6e2ad45c3a632a9fb7830f7`, matching all five stored
   appended-file hashes. Tool exposure/settings also differ, so the net context difference cannot
   be assigned solely to the guide without an ablation. This prefix was already warm: first-message
   cache creation totals are almost equal, 6,171 native versus 6,169 MRW. Additional creation is
   accumulated later; it is not all the initial cost of loading the guide.
   The pilot preregistered full-guide injection for every fresh job. Current
   [adoption guidance](../../agents.md#1-a-standing-instruction-in-the-file-the-agent-always-reads)
   instead says core once, full guide on demand (lines 51–53). The pinned core is 1,598 characters.
   We carried the heavier existing benchmark setup forward; this was not a test of the lightest
   documented startup. That limits conclusions about mandatory MRW overhead. A lighter setup's
   savings and correctness remain unmeasured; the collected loss is retained.
2. **Native batched too.** Its five editing commands use loops with sed or Python. MRW has five
   actual write invocations after plan-generating loops. Transport batching supplies no large call
   advantage here. Across all Bash commands, native generated 1,939 command characters and MRW 1,672;
   this run does not support attributing its loss to larger emitted plan commands.
3. **Two MRW jobs added a round, while one removed a round.** Configuration first searched timeout
   values, then read two complete files to distinguish active and legacy sections. Native inspected
   that structure in its initial call. On Markdown, MRW wrote and then ran the required checker in a
   separate call; native combined editing and checking. Structural native used an additional diff
   inspection, so MRW saved one round there. Net: native 16 versus MRW 17 CLI turns.
4. **Tool-result text grew modestly, with larger edit receipts.** Inspection-result characters total
   10,584 native versus 10,975 MRW. Editing/checking results total 2,973 versus 5,134, including MRW's
   separate Markdown check. Combined tool results are 13,557 versus 16,109 characters (+18.8%).
   These character counts are not token counts or causal cost estimates; mixed shell commands are
   not internally separated. Receipt/header/line-number text is part of the treatment.
5. **The older refusal/retry pattern is absent here.** MRW's twelve tool results have no explicit
   error flag and every final result reports zero permission denials. There are no refused MRW
   writes or model-issued `mrw instructions` commands in the inspected Bash text. Instructions were
   appended once per fresh task instead. Native's one flagged tool result is `diff` exiting 1 for
   expected differences, followed by a successful edit, not a failed edit or permission refusal.
6. **Checks were preserved in both arms.** Native ran the declared checker in each of five editing
   commands. MRW ran it automatically for four writes and explicitly for Markdown. Its four visible
   check receipts total 108 ms. Each arm also received five external observer checks, excluded from
   CLI wall and token usage. The observer is not the source of the reported CLI-wall difference;
   the traces do not isolate model latency from all tool/client overhead.

## Why the earlier smoke pilot won

| Measure, two smoke tasks | Native | MRW |
|---|---:|---:|
| CLI turns | 11 | 6 |
| Unique emitted assistant messages | 8 | 6 |
| Tool calls | 9 | 4 |
| Output tokens | 1,083 | 696 |
| Cache-read tokens | 58,474 | 74,130 |
| Cache-creation tokens | 5,622 | 4,351 |
| CLI-estimated task cost | $0.0391974 | $0.0318010 |

Native separated discovery, metadata inspection, editing and checking into more tool calls; some
calls shared a message, explaining why turn and message counts differ. MRW combined a grep/sub
edit and its declared check. Output savings ($0.0038700) plus cache-creation savings ($0.0050840)
exceeded its extra cache-read cost ($0.0015656), producing an 18.9% task-cost reduction. Both smoke
pairs ran A then B, with warm caches and one attempt each; these are observations, not a causal
proof or a prediction for other tasks. Setup costs remain in the original reports, outside these
explicitly task-only comparisons. The two experiments are not pooled.

## Next experiments supported by this diagnosis

- A bounded short-instructions versus full-guide comparison within the MRW arm, preserving the same
  tools, safeguards, work and final check. Measure context, refusals, accepted quality and total
  usage; assume no saving until observed. This tests the large fixed-context hypothesis.
- A focused workflow comparison that combines the Markdown write and required check, and supplies
  the configuration's active/legacy structure in the initial read. Preserve checking and licensing.
  This tests two observed extra rounds independently of the instruction-size question.
- Evaluate representative stale/multifile mistakes separately if safety is the proposed benefit.
  These fourteen successful fixtures did not measure harmful edits prevented or recovery savings.

No ablation, runtime optimization, rule relaxation or broader savings claim follows from this
analysis. Fewer turns is a useful intermediate measure when context/output/check work are comparable;
accepted task cost and quality remain the outcomes.

## Reproduce and validate

From this report directory, run the saved source with the private raw roots and this directory:

```sh
python3 trace-diagnosis.py.txt GROUPED_RAW_ROOT SMOKE_RAW_ROOT .
```

The source accepts exactly these three paths and emits JSON without modifying the inputs. It verifies
all fourteen original transcript fingerprints, final usage/cost equality, one-to-one tool/result IDs,
input sums and cost reconciliation. Phase labels are manually inspected labels for these specific
commands, not a general shell parser. Unknown/mismatched data is rejected rather than filled with zero.

Validation passed: all fourteen raw hashes and receipt accounts reconcile; a fresh offline replay
reproduces the saved JSON exactly. Four negative controls reject a changed fingerprint, unknown
usage, an inconsistent cost, and a missing tool result. Scoped ruff F/E9, the repository prose gate
and whitespace validation pass. This is an artifact analysis; no full build or regression suite ran.
