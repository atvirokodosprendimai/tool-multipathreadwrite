# Reading 20, plan: the MCP delivery arm at 2 KB and 20 KB, under rules that cover what voided reading 19

**Written and committed before any trial ran.** Reading 19 is void, whole; `reading-19-void.md`
records why, and none of its thirty scores carries over.

**This plan has been amended TWICE, BEFORE its first trial, and has run none.** A plan is frozen at
its first trial, not at its commit, so both amendments are legitimate and both are recorded here
rather than left to a reader to reconstruct. First, on 2026-09-05, it covered the 20 KB stratum
alone, because reading 19's first result document published its 2 KB stratum; the Codex review of
PR #115 established that reading 19 voids whole, so both sizes need re-running. Second, on
2026-09-06, a follow-up review found that rule 3 promised every published number came from committed
data while prediction 4 pre-registered a cost comparison the plan itself calls unreproducible; cost
is now named as the one explicit exception. Nothing else has been touched.

## Why a twentieth reading

The MCP delivery arm — mrw's served text arriving through `mrw_read`, the tool mrw ships — is still
the one delivery this corpus has never measured. ADR-020 has carried it open since reading 8. Three
readings have now failed to produce a number for it:

- **Reading 12** voided: compliance could not be verified, because a subagent's transcript is not
  readable by the session that spawned it.
- **Reading 18** voided at 200 KB with cause, and the cause is a defect rather than a rate: mrw pages
  correctly and the host truncates the page before the model sees it, while the ledger records the
  page whole. That is in `docs/adr/BACKLOG.md` under ADR-023 and is not re-attempted here.
- **Reading 19** voided whole on two deviations of its author's, both prevented below.

This reading runs the two sizes where nothing pages and nothing is truncated.

## The build under test

The registered `mrw mcp` server running the binary built at `d6c62e7`, not rebuilt, whose engine is
byte-identical to `v1.1.0` (`git diff v1.1.0 d6c62e7 -- internal cmd` is empty). If that server has
been restarted onto a different build when this reading runs, **the reading does not start** until
the build is recorded here.

## Cells

Reading 4's fifteen 2,000-byte and fifteen 20,000-byte cells — 3 positions × 5 seeds, distractors 3,
selector `odd-retries`, window from line 1 — the cells of readings 9, 13 and 19. Each is regenerated
and verified against `docs/curve/reading-13-scores/*.score.json` on `trial_id`, `served_bytes` and
the planted line. Any mismatch stops the reading.

| cell | trial_id | served | target | sha256 of served.txt |
|---|---|---|---|---|
| `2000-early-1` | 315c39e714ac | 2009 | 10 | `0cd14384a3d3dbc50a952446a3bc27cc812c936a427bef32ac46188843c70122` |
| `2000-early-2` | 7e6efa11c03f | 2028 | 10 | `a3828bdf1ade51bed4b105622d30a2dc50190fdc9b0067edea51e934645b6925` |
| `2000-early-3` | 4c1e258b7f81 | 2047 | 10 | `089381a061a52b70bb3105f3f3a73d8d026f1c91bbfede3360f98a433aa0b027` |
| `2000-early-4` | f09d43fae76a | 2014 | 10 | `fcfe0423cbaa177365f1609a8f0c2be04ca65fb4e0898e76c3f09752b6f7c7e6` |
| `2000-early-5` | 26050bb07eb5 | 2029 | 10 | `8216c9cad09ec9f4e0c582e9cc4dbedf0de22a2a1811795691d43080035cee56` |
| `2000-middle-1` | 5a655a39d9f3 | 2009 | 20 | `f153446cb368159bcec77cfcc6e2f586fe3b86461cd379cbcb51ed50f621f626` |
| `2000-middle-2` | cbfe3454b13d | 2028 | 20 | `da783800d071647c3433b2d3aaa9d407c3c0cbc065c682d6b26fc6f1af499913` |
| `2000-middle-3` | f3a4cd6ad5c4 | 2047 | 20 | `29d599819ff0ef4546d60d0236c358c5463e7cc19b472cadd1f5fee76b9993ad` |
| `2000-middle-4` | bfd54c3667ef | 2014 | 20 | `630e5000a357b54357529f83914360eaa4eab4751533c5a5c30796f5403ca067` |
| `2000-middle-5` | 091f04e5a113 | 2029 | 20 | `fd54ab22cdda0e8464a037d44565d423314cc1acb04a84fb1331e6df335c85ef` |
| `2000-late-1` | 0ec02568ccc3 | 2009 | 40 | `fc5354ab75c5f8b674a8462aac3a9a7c9d229e9f60f8612a431db10f26f59ab0` |
| `2000-late-2` | a480952c9a39 | 2028 | 40 | `3c42d2aa929cf9adabd8c0971990eaea099d2f644b144bf99d820fec3c77604d` |
| `2000-late-3` | 75df68d71da9 | 2047 | 39 | `4743a0c10323135c9add3f254396e49b77ab72fddd4b3cae3e4cc04a221674ed` |
| `2000-late-4` | 06b192d6afdf | 2014 | 38 | `6963df240abfcbd2cd163bb12d8993c369d398b4cf96ac647b8aa14969d37cbc` |
| `2000-late-5` | 92201d56db11 | 2029 | 40 | `6c6933b0dcf12883e1c5d3479b3c25720fc3ab8e4a235a33cb099f741de897ed` |
| `20000-early-1` | 377185e0ed50 | 20032 | 76 | `ee5877f234ee39c81530c871acfcdbba508788969ca1cbc9b11e97b10a46b2ce` |
| `20000-early-2` | 67347acc9b7b | 20002 | 76 | `42bb1190e98a9cea547ddaf5da2858fc0c8b26e772b72ba846ee3913c040b004` |
| `20000-early-3` | 5db2cf5965c4 | 20033 | 75 | `3e1577b60a0528f3ba3194dcb2fc48ba84fe6762c0005dc89b771a60a70a39f7` |
| `20000-early-4` | ede6f69b5701 | 20019 | 75 | `1f8db107bc1eef7ea0e55bad6819540e941cc17088823f27b7d812c332293bbe` |
| `20000-early-5` | 06e47a5a5be2 | 20036 | 76 | `812860e6368c239093aeefcd02e6482b99906db5e012b4b692821a7b9380c2f0` |
| `20000-middle-1` | 760cae286e4c | 20032 | 152 | `15321b50aafcfa0c87f4b4e0abde6936bb72722902db5881238f5b6dff2b1af1` |
| `20000-middle-2` | faabba4471ff | 20002 | 152 | `284957abbadc3f3fc1a559eb7d6117f72161b6c76610eb99c38b9816f618e290` |
| `20000-middle-3` | 5c4fefd9b6d8 | 20033 | 150 | `d11f88fa2b994da7fe541661002d9f25b19e1b297a374a50d827a7b4e67dc315` |
| `20000-middle-4` | 5a96c73dd0f9 | 20019 | 150 | `50970c60a0f3f035b306902eda538166a1e5eefa229ad0b710357832553ead37` |
| `20000-middle-5` | 4fa6f91e2018 | 20036 | 152 | `73d7794e9f644b574a3f1ddf949edc9d5839f3bf2ed26f111098406d11d902e7` |
| `20000-late-1` | 7e98fc490953 | 20032 | 302 | `dfcb47724bcf0252fb454e8cb6667a875c1c586e966c7fffb639d03e1568965f` |
| `20000-late-2` | 8869d328f3f2 | 20002 | 304 | `99c9efd256f364f91121b08c38510752fcd5ec8e61e01f82ed72c89d29093abb` |
| `20000-late-3` | cd18c2d87f00 | 20033 | 300 | `8f1ee98083ec8cd04b1e39b05e8dd7c2041c257d514ccc603e9a3e4a388fe4e4` |
| `20000-late-4` | 21f77219d76f | 20019 | 300 | `8f447c1c7c834792b1c001d4f445171eb5bed2a37e04be742702eef6d1d72c18` |
| `20000-late-5` | d001b57fc89c | 20036 | 304 | `a13a7a5f37b8685af2f5ac99702b5ec8ab32c6e7206acbe9ecdab9fd37a06dfc` |

**All thirty cells were answered once already, under reading 19's void run.** They are re-run with
fresh clients; no void score carries over, and the result reports the two runs' agreement as an
observation, never pooling them.

Each `answer.json` is moved out of the repository root after generation and moved back only for
`curve score`, so while any client runs the ground truth is not addressable by `mrw_read`.

## Client and arm

A fresh Haiku subagent per trial (`claude-haiku-4-5`); `mcp__mrw__mrw_read` and `Write` and nothing
else. The listed calls:

1. `mrw_read` with specs `["tmp/curve/<cell>/task.json"]` — the instruction, `trial_id`,
   `served_bytes`.
2. `mrw_read` with specs `["tmp/curve/<cell>/tree/services.conf"]`, a bare path. At both sizes this
   returns the whole file in one result; the prompt still says a page is to be followed rather than
   treated as a failure, and any trial that meets one is reported.
3. `Write` of `result.json` — `{"trial_id","served_bytes","plan"}`, the plan addressing
   **`services.conf`**, the name `task.json` gives in its `file` field and not the path passed to
   `mrw_read`, using **mrw's gutter number** and not the row's position.
4. `Write` of `coverage.json`, second.

`ToolSearch` with `select:mcp__mrw__mrw_read`, the reply channel, and any `am_*` call after the
writes are harness protocol, not trial behaviour; **the writes are the answer** and a subagent's
final report is not read.

**The prompt text is fixed by this plan and is identical for all thirty trials but for the cell
name**, and that is verified by diffing the thirty dispatched prompt strings before the result is
written.

## The three rules this reading exists to add

Each closes something a previous reading discovered mid-run.

1. **`last_line_text` is compared after stripping a leading comment marker, AND the prompt says
   "copied exactly, including any leading `#`".** Both halves are fixed here, before the first trial.
   Reading 19 fixed the second half after a trial failed on the first, which is what voided it.

2. **A client that writes no `result.json` is a `no_answer`: recorded beside hits and misses, never
   folded into them, and NEVER retried.** Its stratum is reported at the n it actually reached.
   Reading 4's rule — "a void trial is never counted as a miss" — extended to the case where nothing
   comes back. Retrying only non-answers filters the surviving set, and disclosure does not fix a
   bias mechanism.

3. **The coverage reports are committed alongside the scores**, in
   `docs/curve/reading-20-coverage/`, one JSON per trial. Reading 19's first result document
   published "compliant 15 of 15" and "zero paged reads" while its provenance claimed every number
   came from the committed scores — which carry neither. **Every RATE and COUNT this reading
   publishes — the correct-address rate, the compliance count, the paged-read count and the
   `no_answer` count — is computed from committed data**: the scores in
   `docs/curve/reading-20-scores/` and the coverage files beside them. The compliance verdict per
   trial is derived from the committed coverage file and the cell, not asserted in prose.

   **The one exception is cost, and it is named here rather than left to be noticed.** Token spend
   comes from request records this corpus has never committed, at any reading. It is therefore an
   OBSERVATION reported beside the result and **never a published number**: prediction 4 below is a
   sanity check on the arm's shape, and if the records are unavailable when this reading is written
   the prediction is reported as unevaluated rather than estimated. No claim in the result rests on
   it.

## Predictions, recorded before collection

1. **At least 27 of 30, and the author expects 30 of 30.** Reading 9 was 15 of 15 at both sizes on
   these cells through a bare tool result; readings 13 and 4, whose arms lay a second plausible line
   number beside mrw's, were 14 and 15 of 15 at 2 KB and 13 and 12 of 15 at 20 KB. This arm has no
   second number. Stated so it can be wrong: this author's expectation was wrong in readings 10, 11
   and 13, and readings 12, 18 and 19 all ended somewhere their plans did not foresee.
2. **There is no `T + 2` to predict.** Any miss is a third account, reported with its offset from the
   target and what sits at the addressed line.
3. **Compliance at least 27 of 30**, computed from the committed coverage files.
4. **Cost within 20% of reading 9's medians on these cells — 31,909 at 2 KB and 36,444 at 20 KB** —
   plus the coverage write. This is the named exception in rule 3: an observation from uncommitted
   request records, reported and not published, and unevaluated if the records are unavailable.
5. **Zero paged reads.** Both sizes sit far below mrw's cap.
6. **At most one `no_answer` in thirty.** Reading 19's void run saw exactly one. More than one is a
   fact about running this arm and is reported as its own number.

## What this decides

A two-size point for the delivery mrw actually ships, below the truncation boundary reading 18 found.
With reading 18 it completes ADR-020's item in two parts: what the arm does where it fits in one
result, and why it cannot be measured where it does not. It does not touch 200 KB.

## What would void this reading

The cells, this plan or the harness changing after the first trial — including the prompt text, which
this plan fixes; a client reaching `answer.json`; a `coverage.json` not strictly newer than its
`result.json`; a cell failing the three-field identity check; **any retry of any trial**. A void is a
re-run under a new plan file, as readings 12, 18 and 19 were, and **it voids this reading whole** —
there is no stratum-level void, which is the exception reading 19 invented after seeing its results.

## Known limitations, stated now

Two sizes, one client, one fixture family, five repeats per cell. Thirty trials at the ceiling cannot
distinguish 30 of 30 from 28 of 30. All thirty cells have been seen once by a void run. The in-band
coverage report is the client's own account of what it was served: committing it makes the number
reproducible, not truthful, and the mtime rule constrains order but not honesty. Cost comes from
request records that are not committed and is reported rather than reproducible.
