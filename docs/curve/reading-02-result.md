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

## The three misses have one cause, and it is not retrieval

Every miss addressed a line exactly **two below** the planted one, at three different file sizes and
two different positions:

| Trial | Planted line | Addressed |
|---|---|---|
| 2000-late-3 | 39 | 41 |
| 20000-early-1 | 76 | 78 |
| 200000-late-5 | 2917 | 2919 |

In all three the replacement text was correct. The client found the right service block and the
right setting; it got the address wrong.

`served.txt` opens with two lines that carry no line number — the `==>` header naming the file and
the `@@` line naming the served range — and then renders each line as `   N| text`. In all three
misses, the number the client wrote is the target's **row** in that rendering, not the `N` printed
in front of it. Two header lines, an error of exactly two, three times out of three.

So retrieval did not degrade. Transcription did, and the two header lines are the whole of it.

## Why this matters more than the curve

Under mrw with no guard, that plan applies and the receipt says so:

```
ok   services.conf 41 replace  -1 +1
wrote services.conf  45L -> 45L  sha fe18f9ce
1 hunk(s), 1 file(s), 0 failed — applied
```

Line 41 was a comment. It now reads `timeout = 45`, and the setting the caller meant to change is
untouched two lines above. That is the failure the tool exists to prevent, arriving through a green
receipt.

The same address with an anchor guard is refused instead:

```
FAIL services.conf 41 replace (plan line 1): anchor "timeout = 30" not in line 41: # each can an registry service block the and without
1 hunk(s), 1 file(s), 1 failed — NOTHING WRITTEN
```

Three of forty-five plans authored by an unaided client would have written the wrong line silently,
and `anchor=` catches every one of them by construction, because the guard is checked against the
content the client actually read.

## Cost

Token cost per trial, summed over each subagent's requests as input plus cache-creation plus output.
This accounting is about 1.7× the figure the harness reports as a subagent's own total, so read the
ratios rather than the absolute numbers.

| Served bytes | median | min | max |
|---|---|---|---|
| 2,000 | 93,935 | 91,402 | 130,288 |
| 20,000 | 110,972 | 100,752 | 151,753 |
| 200,000 | 239,696 | 234,959 | 329,432 |

A hundredfold increase in served bytes costs about 2.6× the tokens and buys no measurable loss of
accuracy. Serving a large window in one read is not where the money goes; the fixed cost of a
session dominates until the served bytes are very large.

## What this reading does not say

- It does not say a curve cannot bend. The target here carries a unique service name, so a client
  that reads at all can find it. A fixture whose target is identified by a relational property
  rather than a name would be a harder test, and building one is a generator change in ADR-020.
- It does not say the search arm behaves the same way. Reading 1 was voided because its clients
  searched instead of reading, which is itself the finding that tool-using clients never process
  the served window at all.
- Five repeats per cell resolve only large effects. A flat result here means "no large effect at
  this difficulty", not "no effect".
