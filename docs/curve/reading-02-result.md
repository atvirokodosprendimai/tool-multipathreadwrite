# Reading 2, result: the curve is flat, and the one failure mode is address transcription

**Collected 2026-09-04 under `reading-02-plan.md`, which was written before any trial ran.**
Forty-five trials, one client population (fresh Sonnet subagents), the read arm only. The search
arm's 18 observations from reading 1 stand as collected and are not pooled with these.

## What was run

Every trial got one directory holding `task.json` and `served.txt` — the bytes mrw would serve —
and nothing else. The client was restricted to reading and writing files, with no search tool of
any kind, and had to author a one-hunk plan naming the line to change. Compliance was checked from
each trial's own transcript afterwards, filtered on the read arm's prompt marker rather than on the
cell id, because both arms use the same ids.

**All 45 trials were compliant: zero search-tool calls.**

That compliance was achieved against a competing instruction, which is worth stating because it was
not designed in. Partway through, the harness injected a reminder into trial clients telling them to
prefer `Bash` for reading and editing files. `Bash` is one of the four tools this arm forbids and one
of the four the compliance check counts, so a client that had followed it would have voided its own
trial. Three clients said in their transcripts that they had received the reminder and were
disregarding it in favour of the task's explicit restriction; the reminder's own text is not
preserved in a transcript, so how many of the 45 received it cannot be counted. What can be counted
is the outcome: **no trial called `Bash`, or any other search tool, at all.**

## The curve

| Served bytes | early | middle | late |
|---|---|---|---|
| 2,000 | 5/5 | 5/5 | 4/5 |
| 20,000 | 4/5 | 5/5 | 5/5 |
| 200,000 | 5/5 | 5/5 | 4/5 |

Forty-two hits in forty-five. Five repeats give a 95% Wilson interval of roughly 0.57 to 1.00 on a
5/5 cell, so these cells cannot be told apart and this reading resolves no difference between 2 KB
and 200 KB. **A flat curve was pre-registered as an acceptable answer, and this is that answer.**

The honest statement is narrow: at this difficulty, with a target that carries a unique service
name, a hundredfold increase in served bytes did not measurably change whether the client addressed
the right line.

## The three misses land two lines below the target, every time

Every miss addressed a line exactly **two below** the planted one, at three different file sizes and
two different positions:

| Trial | Planted line | Addressed |
|---|---|---|
| 2000-late-3 | 39 | 41 |
| 20000-early-1 | 76 | 78 |
| 200000-late-5 | 2917 | 2919 |

In all three the replacement text was correct. The client found the right service block and the
right setting; it got the address wrong, and it got it wrong by the same amount every time.

`served.txt` opens with two lines that carry no line number — the `==>` header naming the file and
the `@@` line naming the served range — and then renders each line as `   N| text`. Counting rows in
that rendering instead of reading the `N` printed in front of each one produces an address exactly
two too high. That is the most economical explanation of the data.

**It is not, however, a demonstrated mechanism, and this reading cannot make it one.** Every cell
serves the whole file as `@@ 1-N`, so a target's row in the rendering and its line number plus two
are the same integer in all forty-five trials. Any account that adds two predicts what was observed
equally well. The three transcripts do not settle it either: each client reported its answer as a
bare number with no reasoning recorded.

**The experiment that would discriminate**: a cell whose served window does not begin at line 1.
With a window of `@@ 500-900`, a row count and a line-number-plus-two differ by 498, and the two
accounts stop predicting the same answer. None of the forty-five cells has such a window, and adding
one is a generator change, so it belongs to a later reading rather than to this one.

What the data does support without qualification is the shape of the failure: **retrieval did not
degrade with served size; addressing did, and the error was constant.**

## Why this matters more than the curve

Run against each cell's own fixture, the client's plan applies and the receipt says so. For
`2000-late-3`:

```
ok   services.conf 41 replace  -1 +1
wrote services.conf  45L -> 45L  sha fe18f9ce
1 hunk(s), 1 file(s), 0 failed — applied
```

Line 41 was a comment. It now reads `timeout = 45`, and the setting the caller meant to change is
untouched two lines above. That is the failure the tool exists to prevent, arriving through a green
receipt. All three misses behave this way: exit 0, one hunk `ok`, the wrong line rewritten.

The same three addresses with an anchor guard are refused, all three at exit 1, nothing written:

```
FAIL services.conf 41 replace (plan line 1): anchor "timeout = 30" not in line 41: # each can an registry service block the and without
1 hunk(s), 1 file(s), 1 failed — NOTHING WRITTEN
```

Three of forty-five plans authored by an unaided client would have written the wrong line silently.
All three were run both ways against the real fixture, and `anchor=` caught all three.

## Cost

Two numbers per trial, both taken from the client's own request records with duplicate records
collapsed by message id. **Spend** is input plus cache-creation plus output summed over requests.
**Peak** is the largest single request's input, which is the figure the harness reports as a
subagent's own token total: for `200000-early-5` this method gives 138,142 against the harness's
138,234.

| Served bytes | median spend | vs 2 KB | median peak | vs 2 KB |
|---|---|---|---|---|
| 2,000 | 48,680 | 1.00× | 72,176 | 1.00× |
| 20,000 | 54,412 | 1.12× | 77,972 | 1.08× |
| 200,000 | 118,879 | 2.44× | 142,577 | 1.98× |

A hundredfold increase in served bytes costs about 2.4× the spend and 2.0× the peak context, and
buys no measurable loss of accuracy. Cost grows far sublinearly in served bytes, because the fixed
cost of a session dominates until the window is very large.

## What this reading does not say

- It does not say a curve cannot bend. The target here carries a unique service name, so a client
  that reads at all can find it. A fixture whose target is identified by a relational property
  rather than a name would be a harder test, and building one is a generator change in ADR-020.
- It does not say the search arm behaves the same way. Reading 1 was voided because its clients
  searched instead of reading, which is itself the finding that tool-using clients never process
  the served window at all.
- Five repeats per cell resolve only large effects. A flat result here means "no large effect at
  this difficulty", not "no effect".
