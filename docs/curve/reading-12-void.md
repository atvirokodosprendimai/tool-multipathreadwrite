# Reading 12 is VOID, and why — plus the three findings it did produce

**Collected: 5 of 15 trials, under `reading-12-plan.md` at `f6d6b8d`. Outcome: 0 hits, 5 misses,
0 refusals.** That 0 of 5 is **not** a miss rate for the MCP delivery arm, and this file exists so
nobody reads it as though it were.

## What went wrong: compliance cannot be verified in this harness

Every reading since reading 2 has verified compliance per trial from that trial's own transcript —
which calls the client made, whether it covered the served window, whether it reached for a tool the
arm forbids. `reading-12-plan.md` requires exactly that, and lists a harness change after the first
trial as a void condition.

In this session the clients are subagents, and **a subagent's transcript is not persisted anywhere
the parent can read it**: the parent session's own `.jsonl` records no sidechain (`isSidechain` false
on all 426 records at the time of checking, and no subagent tool call in it), no per-agent transcript
appears under `~/.claude/projects/`, and the task directory holds only the parent's own background
shell. A follow-up question sent to a finished trial agent returned no report either.

So there is no way, under this plan, to tell a compliant trial from a non-compliant one. Every trial
run under `f6d6b8d` is uncheckable by the plan's own standard, and the remaining ten were not run.

## Why the 5 of 5 is uninterpretable rather than a finding

The five misses are unusually coherent, and that is exactly what makes them unusable. In each of the
five `early` cells the client addressed the **last** `[service …]` block in the file, not the odd
one:

| cell | target | addressed | what sits at the addressed line |
|---|---|---|---|
| `200000-early-1` | 725 | 2898 | the last block, `retries = 5` |
| `200000-early-2` | 726 | 2904 | the last block |
| `200000-early-3` | 726 | 2901 | the last block |
| `200000-early-4` | 726 | 2903 | the last block |
| `200000-early-5` | 730 | 2917 | the last block |

The fixtures are correct and the task is well posed. For these seeds `retryPair` drew the common
budget `retries = 5` and the odd budget `retries = 3`, so three distractors carry `retries = 5` and
the target — the one service whose retry budget differs from every other's — carries `retries = 3`.
Readings 8 and 9 solved these same cells 15 of 15 through a Bash delivery, so the cells are solvable
by this client population.

Two accounts fit the table, and **this reading cannot separate them**:

- **A finding about the shipped path:** the client is served the whole file through `mrw_read` and
  still answers from the end of it.
- **A finding about the harness:** the client retained only the tail of a 200,000-character page and
  answered from what it still had.

Compliance is the evidence that would tell those apart — coverage of the served window is precisely
what distinguishes them — and it is the evidence this harness cannot produce. Publishing 0 of 5 as a
rate would assert the first account while holding no evidence against the second.

## The three findings, which stand

They were bought before the trials, from a pilot cell scored nowhere, and they do not depend on
compliance.

1. **ADR-023's fix is live on this host and build.** A narrow `mrw_read` against the running server
   (`mrw mcp` from `d6c62e7`, whose engine is byte-identical to `v1.1.0` — `git diff v1.1.0 d6c62e7
   -- internal cmd` is empty) returned the served lines as its first content block and the receipt as
   its second. The envelope of issue #109 no longer stands in for the text. **The MCP arm is
   unblocked**; what blocks it now is compliance, not delivery.

2. **Every 200,000-byte cell pages.** The served rendering is 200,013 to 200,039 characters against
   `mcp.MaxResultChars` of 200,000, so a bare-path `mrw_read` returns ADR-014's first page —
   `isError: true` plus a `next_read`. The cut falls in the file's last line, far past the furthest
   target (2,917 of about 3,619 lines), so paging cannot itself move the primary variable. Any arm
   over these cells must tell the client that an `isError` page is not a failure.

3. **A `mrw_read` spec is root-relative; a plan's path is tree-relative, and nothing says so.** The
   served text arrives under a header reading `==> tmp/curve/<cell>/tree/services.conf`, while the
   plan is applied against a copy of the fixture tree where the file is `services.conf`. The pilot
   client, given no other reader, copied the header's path into its hunk and scored `refused_apply`
   — "does not exist" — having addressed a line it chose freely. No earlier arm could meet this:
   readings 6 to 14 served the text with `sed` from `served.txt` and never showed the client a path
   that looked authoritative. `reading-12-plan.md` names the file for this reason, and any successor
   must keep doing so.

A fourth, checked while writing this and worth recording because the next plan rests on it: **mrw's
gutter number is the address a plan needs.** In `200000-early-1` the target is line 725 and the
served text's row 727 reads `  725| timeout = 30` — the row index and the gutter differ by the two
header rows, and it is the gutter, not the row, that the plan must carry.

## What replaces it

`reading-18-plan.md`: the same arm, with compliance moved **in band**. What a transcript would have
proved is carried in the client's own `result.json`, which `curve score` tolerates because
`curve.Result` is unmarshalled without `DisallowUnknownFields` and ignores fields it does not name —
so this costs no engine change. A client answering from the tail of a page cannot fill in the file's
last line, the retries value of every block, and the number of `next_read` sends it made.

The fifteen cells are kept. They were verified against `docs/curve/reading-11-scores/` on three
fields each — `trial_id`, `served_bytes` and the planted line — and their `served.txt` sha256s are
recorded in `reading-18-plan.md`.

## Provenance

The five scores are **not** committed, because they are not measurements. The plan they ran under is
`docs/curve/reading-12-plan.md` at `f6d6b8d`, unedited since; this file is the result document for
it, and reading 12 is closed.
