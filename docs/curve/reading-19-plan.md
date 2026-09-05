# Reading 19, plan: the MCP delivery arm at the sizes where it can be measured

**Written and committed before any trial ran.** Readings 12 and 18 are void and stay void; nothing
here revives their scores.

## Why a nineteenth reading

Reading 18 established that at 200 KB this arm cannot be measured on this host: mrw serves ADR-014's
first page correctly and the host truncates it before the model sees it, so the client is shown a
fraction of the file and told nothing about which fraction. That is a defect, recorded in
`docs/adr/BACKLOG.md` under ADR-023, and it is not a curve question.

But it is a defect of the 200 KB case specifically — of a served rendering large enough to page. At
2,000 and 20,000 bytes mrw's render is far below both its own `MaxResultChars` of 200,000 and the
point at which this host truncated (one result put roughly 58,860 characters through). At those
sizes the whole served text arrives in one result, unpaged and uncut, and **the arm ADR-020 has
carried open since reading 8 can finally be run.**

Readings 9 and 13 have committed numbers on exactly these thirty cells, for two other deliveries.
That is what makes this reading worth running rather than merely possible.

## The build under test

Unchanged from readings 12 and 18: the registered `mrw mcp` server running the binary built at
`d6c62e7`, not rebuilt, whose engine is byte-identical to `v1.1.0` (`git diff v1.1.0 d6c62e7 --
internal cmd` is empty).

## Cells

Reading 4's fifteen 2,000-byte and fifteen 20,000-byte cells — 3 positions × 5 seeds, distractors 3,
selector `odd-retries`, window from line 1 — the cells of readings 9 and 13. Each was regenerated
into `tmp/curve/` and verified against `docs/curve/reading-13-scores/*.score.json` on three fields:
`trial_id`, `served_bytes` and the planted line. Any mismatch stops the reading; none mismatched.

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

Each cell's `answer.json` is moved out of the repository root after generation and moved back only
for `curve score`, so while any client runs the ground truth is not addressable by `mrw_read`. "A
client reaching `answer.json`" stays a void condition.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`), as in readings 4, 8 to 11, 13
and 18.

**Arm:** `mcp__mrw__mrw_read` and `Write`, and nothing else. The listed calls:

1. `mrw_read` with specs `["tmp/curve/<cell>/task.json"]` — the instruction, `trial_id` and
   `served_bytes`.
2. `mrw_read` with specs `["tmp/curve/<cell>/tree/services.conf"]`, a bare path. **At these sizes it
   returns the whole file in one result: no page, no `isError`, no `next_read`.** The prompt still
   says what to do if a page appears, so that a page is followed rather than mistaken for a failure,
   and any trial that meets one is reported (prediction 5).
3. `Write` of `result.json` — `{"trial_id","served_bytes","plan"}`, the plan addressing
   **`services.conf`**, the name `task.json` carries in its `file` field and not the path passed to
   `mrw_read`, and using **mrw's gutter number**, not the row's position. Reading 12's pilot
   established that a client given no other reader copies the header's root-relative path into its
   hunk and scores `refused_apply`.
4. `Write` of `coverage.json`, second — below.

**Named as harness protocol, not trial behaviour:** `ToolSearch` with `select:mcp__mrw__mrw_read`,
because the MCP tools are deferred; the reply channel; any `am_*` call after the writes; and **the
writes are the answer** — a subagent's final report is not read.

## Compliance, in band, and the clause reading 18 earned

As in reading 18, and for the same reason — a subagent's transcript is not readable by the session
that spawned it. `coverage.json` carries:

```json
{"trial_id": "...",
 "last_line_number": N, "last_line_text": "...",
 "service_block_lines": [N, N, N, N],
 "next_read_sends": K,
 "other_tools_used": []}
```

`curve score` never reads it, and `curve.Result` ignores unknown fields, so this costs no engine
change. The order is load-bearing and is checked: `result.json` first, `coverage.json` second, and
**any trial whose `coverage.json` is not strictly newer than its `result.json` is void** — enumerating
the blocks sits close enough to the task that a client asked for it first would be taught the answer.

**The clause reading 18 earned:** at 200 KB an incomplete `service_block_lines` was a *delivery*
failure, and that is what made the reading unmeasurable. **Here it is a client failure.** Nothing
truncates at these sizes, so a client that does not report every block did not read what it was
given, and that is a compliance result about the client — which is exactly the inversion that makes
the small-size arm interpretable where the large-size one was not.

## Predictions, recorded before collection

1. **At least 27 of 30, and the author expects 30 of 30.** Reading 9 put this client at the ceiling
   on exactly these cells — 15 of 15 at both sizes — through a delivery whose only number was the
   served text's own, and this delivery's only number is mrw's gutter, which is the address the
   answer needs. Stated so it can be wrong; the author's expectation was wrong in readings 10, 11
   and 13, and readings 12 and 18 both ended somewhere their plans did not foresee.
2. **There is no `T + 2` to predict.** No row index is laid beside mrw's gutter in this arm, so the
   offset that explains every miss in readings 4, 11 and 13 does not exist here. **Any miss is a
   third account and is reported as one**, with its offset from the target and what sits at the
   addressed line.
3. **Compliance at least 27 of 30** by the in-band report: every block, the true last line, no
   forbidden tool, and the two writes in order.
4. **Cost within 20% of reading 9's medians on these cells — 31,909 at 2 KB and 36,444 at 20 KB** —
   plus the coverage write.
5. **Zero paged reads.** Both sizes are far below mrw's cap, so no trial should meet a `next_read`.
   Any that does is reported: it would mean the page boundary sits lower than reading 18's
   observation implies.

## What this decides

If the ceiling holds, the delivery mrw actually ships is, at these sizes, as good as the best
delivery measured — and the flat curve of readings 8 and 9 describes the shipped path and not only a
harness. Together with reading 18 the statement becomes: **this arm is at the ceiling where it fits
in one result, and unmeasurable where it does not, because of the truncation in BACKLOG.** That is a
complete answer to ADR-020's open item, in two parts, and it is more useful than a single rate.

If it does not hold, the miss belongs to mrw's own delivery at a size nothing truncates, which would
be the first such finding in this corpus and puts the served format in scope for a record.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching `answer.json`;
a `coverage.json` not strictly newer than its `result.json`; a cell failing the three-field identity
check. A void is a re-run under a new plan file, as readings 12 and 18 were.

## Known limitations, stated now

Two sizes, one client, one fixture family, five repeats per cell. **This reading says nothing about
200 KB** — reading 18 says what there is to say there, and it is a defect rather than a rate, so
these thirty trials must not be read as a curve extending to the cap. The in-band coverage report is
a client's own account of what it was served, which is weaker than a transcript: it can be filled in
wrongly, and the mtime rule constrains order but not truthfulness. Cost comes from request records
that are not committed.
