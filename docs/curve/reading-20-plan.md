# Reading 20, plan: the 20 KB stratum reading 19 voided, re-run under rules that cover what went wrong

**Written and committed before any trial ran.** Reading 19's 2 KB stratum stands as collected;
nothing here changes it, and its fifteen scores are not re-used.

## Why a twentieth reading

Reading 19 measured the MCP delivery arm at two sizes and could keep only one. Its 2 KB stratum is
15 of 15 and committed. Its 20 KB stratum is void on two deviations by that reading's author: the
coverage instruction gained a clarifying phrase after the first trial, and a client that wrote
nothing was retried without the plan allowing it. Neither is a fact about the client, and both are
cheap to prevent. This reading re-runs that stratum with them prevented.

## The build under test

Unchanged from readings 12, 18 and 19: the registered `mrw mcp` server running the binary built at
`d6c62e7`, not rebuilt, whose engine is byte-identical to `v1.1.0` (`git diff v1.1.0 d6c62e7 --
internal cmd` is empty). If that server has been restarted onto a different build when this reading
runs, the reading does not start until the build is recorded here.

## Cells

Reading 4's fifteen 20,000-byte cells — 3 positions × 5 seeds, distractors 3, selector
`odd-retries`, window from line 1 — regenerated and verified against
`docs/curve/reading-13-scores/*.score.json` on `trial_id`, `served_bytes` and the planted line. Any
mismatch stops the reading.

| cell | trial_id | served | target | sha256 of served.txt |
|---|---|---|---|---|
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

**These cells were answered once already, under reading 19's void stratum.** They are re-run with
fresh clients; no void score carries over, and the result reports the two runs' agreement as an
observation, never pooling them.

Each `answer.json` is moved out of the repository root after generation and moved back only for
`curve score`.

## Client and arm

Unchanged from reading 19: a fresh Haiku subagent per trial (`claude-haiku-4-5`);
`mcp__mrw__mrw_read` and `Write` and nothing else; `task.json` read first, then the fixture by bare
path; `result.json` written first with the plan addressing `services.conf` by mrw's gutter number;
`coverage.json` written second. `ToolSearch`, the reply channel and any `am_*` call after the writes
are harness protocol, and the writes are the answer.

**The prompt text is fixed by this plan and is identical for all fifteen trials but for the cell
name.** It is verified so before the result is written, by diffing the fifteen dispatched prompt
strings against each other — reading 19 did that check for its surviving stratum and it is cheap.

## The two rules this reading exists to add

1. **`last_line_text` is compared after stripping a leading comment marker.** Reading 19's
   `20000-middle-3` gave the right line number and the right words without the leading `# `, and was
   scored non-compliant for it. The clause exists to show the client reached the file's end, and a
   dropped marker does not bear on that. The prompt also says "copied exactly, including any leading
   `#`" — **both halves are fixed here, before the first trial**, which is what reading 19 failed to
   do.

2. **A client that writes no `result.json` is a NO-ANSWER, and is never retried.** It is recorded as
   `no_answer`, reported beside hits and misses and never folded into them, and its stratum is
   reported at the n it actually reached. Reading 4's rule — "a void trial is never counted as a
   miss" — extended to the case where nothing at all comes back. Retrying only failures filters the
   surviving set, which is why reading 19's retry voided its stratum rather than merely being
   disclosed.

## Predictions, recorded before collection

1. **At least 13 of 15, and the author expects 15 of 15.** Reading 9 was 15 of 15 on these cells
   through a bare tool result and reading 19 was 15 of 15 at 2 KB through this arm; reading 13's
   numbered arm was 13 of 15 here and reading 4's file reader 12 of 15, both with a second number
   this arm does not have. Stated so it can be wrong.
2. **No `T + 2` exists in this arm.** Any miss is a third account, reported with its offset from the
   target and what sits at the addressed line.
3. **Compliance at least 13 of 15**, by the in-band report under rule 1 above.
4. **Cost within 20% of reading 9's median of 36,444** at this size, plus the coverage write.
5. **Zero paged reads.** 20,000 bytes is far below mrw's cap; reading 19's void trials saw
   `next_read_sends` 0 throughout, and any page here would contradict that.
6. **No `no_answer` trials.** Reading 19 saw exactly one in fifteen at this size. If more than one
   appears, that is a fact about running this arm at this size and is reported as its own number.

## What this decides

A clean 20 KB point for the MCP delivery arm, which with reading 19's 2 KB point gives the arm a
two-size shape below the truncation boundary. It does not touch 200 KB: reading 18 says what there
is to say there, and it is the defect in `docs/adr/BACKLOG.md`, not a curve point.

## What would void this reading

The cells, this plan or the harness changing after the first trial — including the prompt text, which
this plan fixes; a client reaching `answer.json`; a `coverage.json` not strictly newer than its
`result.json`; a cell failing the three-field identity check; any retry of any trial.

## Known limitations, stated now

One size, one client, one fixture family, five repeats per cell. Fifteen trials at the ceiling cannot
distinguish 15 of 15 from 14 of 15. The cells have been seen once by a void run. The in-band coverage
report is the client's own account, weaker than a transcript. Cost comes from request records that
are not committed.
