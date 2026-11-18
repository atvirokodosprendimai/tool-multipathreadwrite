# What mrw saves

This page measures interface calls and bytes in scripted edit shapes. A planned
batch uses one read of every site and one write of the whole plan: **two calls,
for any N sites**. Actual model turns, cost and task outcomes are measured
separately by the agent benchmark linked below.

```sh
./scripts/measure.sh
```

These shape measurements describe the engine's interface overhead. The
[agent benchmark](bench/RESULTS.md), summarized in the [README](../README.md),
measures model turns, cost and task outcomes on agent runs. The call and byte
tables here do not establish those outcomes on their own.
For native wall/CPU distributions, checker modes and warning evidence, see the
[experimental observability tool and verification](bench/observability/RESULTS.md). It adds p95 and
scaling descriptions to the existing native samples and keeps command costs separate from agent outcomes.


Re-run the script rather than quoting the table. The ratios track how large
this repository's own files are. Shape D's file list is `git ls-files '*.go'`,
so the D ratio moves whenever that count does.

## Where it fits — the QAM stack

mrw is one of three tools in the **[QAM stack](https://atvirokodosprendimai.github.io/qamstack/)**:

> Three tools that make Claude's work checkable: gates that exit non-zero, memory
> that outlives the session, edits that come back with a receipt.

| | | |
|---|---|---|
| **Quality Harness** | *Gates, not vibes* | a Claude Code plugin whose gates report through exit codes rather than assertions |
| **AI Agent Memory** | *The reasoning survives* | an MCP server letting agents read and write shared memory across sessions, decisions and rejected alternatives included |
| **mrw** | *Edits with a receipt* | this: batched edits applied atomically, with a verdict per hunk |

**mrw stands alone and needs neither of them.** It is an ordinary binary;
nothing here depends on a plugin or a server, and `AGENTS.md` carries everything
an agent needs to drive it from a plain checkout.

The three overlap on one conviction, which is why they are a stack rather than a
bundle: **a report that cannot fail is not a report.**

The six-guarantee grid against git apply, Codex apply_patch, and Claude Edit is [comparison.md](comparison.md).

## Shapes A–D

Measured on this tree at **`fe51cdab`** (2026-11-18, `main` after #657, which made the
script run again: a rename had left it exiting 1 since 2026-10-20, and three of its
specs had drifted). Round trips are still **2 calls for any N.** Bytes moved because the
tree grew and because shapes A and B now serve exactly the sites they count. Shape D is
**730** Go files, not 244 (the `9cf28ee` reading of 2026-09-29).

| shape | | baseline | mrw | |
|---|---|---|---|---|
| **A.** 4 sites, 4 large files | bytes vs reading those files **whole** | 402,533 | 4,307 | **93.5× less** |
| | bytes vs a **windowed** `offset`/`limit` read | 3,499 | 4,307 | **1.2× MORE** |
| | calls, whole-file (reads + edits) | 8 | 2 | 4.0× fewer |
| | calls, windowed (search + reads + edits) | 9 | 2 | **4.5× fewer** |
| | requests if every call is batched, whole / windowed | 2 / 3 | 2 | same to 1.5× fewer |
| **B.** 2 sites, 2 mid-sized files | bytes vs whole | 47,876 | 550 | 87.0× less |
| | bytes vs windowed | 326 | 550 | 1.7× MORE |
| | calls | 4 / 5 | 2 | 2.0–2.5× fewer |
| | requests if batched | 2 / 3 | 2 | same to 1.5× fewer |
| **C.** 1 site, whole small file | bytes (window *is* the whole file) | 36,313 | 43,378 | **1.2× MORE** |
| | calls | 2 / 3 | 2 | same to 1.5× fewer |
| | requests if batched | 2 / 2 | 2 | the same |
| **D.** 1 site in **every** Go file — 730 sites, 730 files | calls (reads + edits) | 1,460 | 2 | **730.0× fewer** |
| | requests if batched | 2 / 3 | 2 | same to 1.5× fewer |
| | bytes vs whole | 4,753,254 | 66,410 | 71.6× less |
| | bytes vs windowed | 10,730 | 66,410 | **6.2× MORE** |

**Shape D is the one to read, and read it for the CALLS, not the bytes.** It is
the change every codebase gets eventually — a renamed symbol, an added build
tag, a changed import — one site in each Go file. Its `6.2× MORE` is mrw's
worst possible input by construction: 730 files at ONE line each, so a per-file
header and a per-file receipt are charged against 10,730 bytes of payload.

**Shape C is in the table on purpose.** When you need a whole file and there is
one site, mrw prints *more* than the file holds — it adds a header and a line
number per line — and saves no round trips. Keep this overhead visible while
following [ADR-062](adr/ADR-062-instructions-teach-always-and-a-plan.md)'s usage
instruction: use mrw always and plan one read, one plan, one write.

**Read the two byte rows together or neither.** `Read` takes `offset`/`limit`,
so the windowed reader is the documented interface, not a strawman. Against it
mrw costs *more* bytes. The whole-file ratio is real for the case an agent is
usually in: it does not yet know where to look. Once it knows, the byte
advantage is gone and the round trips are what is left.

The arithmetic does not depend on this repository:

| sites across files | Read + Edit | mrw | |
|---|---|---|---|
| 4 in 4 | 8 calls | 2 | 4× fewer |
| 13 in 1 | 14 calls | 2 | 7× fewer |
| N in M | M + N | **2** | — |

**Calls are not requests, and the script prints both ends.** A host can issue
several tool calls within one model turn. The *calls* rows are the end where every
call is its own request. The *requests if batched* rows are the other end: every
Read in one request and every Edit in the next, which is 2 for any N and ties mrw's 2.
A windowed read needs one request more, for the search that finds the window (not in
shape C, where the window is the whole file), and shape G one more for the
verification: 3 against mrw's 2. Fewer calls alone do not establish fewer model turns
or lower cost; the agent benchmark measures actual turns and cost.

Where real sessions sit between those ends was read once, on one machine
(`docs/bench/effect-study/`, 942 Claude Code transcripts, entries before
2026-11-18T10:00:00Z): a request that used only the host's `Read`, `Edit` and
`Write` acted on 1.3 file sites on average, and a shell call that ran mrw on 3.7,
with a third of those calls at one site. Those are sites per request and per call,
not requests per task: the study does not group calls into tasks, so it does not say
how close to either end a whole task came. The
same reading put 92% of what the token classes would cost, at assumed price
weights, in cache writes and cache reads: each request re-reads the session's
context. That says where the cost sits. It does not say whether requests or bytes
move it, because neither was varied, and the sessions compared did different work.

The method, and its biases: the Read+Edit column counts each file's **raw**
bytes, which understates it, because the real Read tool numbers every line. The
mrw column counts its **actual** output, headers and line numbers included.
Output tokens are not measured, so this is an input-side and round-trip result,
not a total-cost one.

## Shape F — the search is charged

A–D count a windowed search as **+1 call and 0 bytes**. Shape F runs a real
`rg -n` (`grep -nH` if rg is missing) on `git ls-files '*.go'` for `^package `,
adds those bytes to the matching lines, and compares that to
`mrw read --grep` on the same files. Calls stay 2 (read + write). The
windowed-only row is still printed so a byte win cannot be quoted against the
documented Read interface.

Measured in the same run as A–D above (`fe51cdab`, 2026-11-18), on the same 730
Go files, with `rg` and `ast-grep` on PATH.

| | baseline | mrw | |
|---|---|---|---|
| bytes vs whole | 4,753,254 | 66,410 | 71.6× less |
| bytes vs windowed | 10,001 | 66,410 | **6.6× MORE** |
| bytes vs rg+windowed | 43,911 | 66,410 | **1.5× MORE** |
| bytes, `--ast-grep 'package $P'` vs rg+windowed | 43,911 | 66,410 | **1.5× MORE** |
| calls, windowed (search+reads+edits) | 1,461 | 2 | **730.5× fewer** |

Charging the search **does not flip the byte comparison**. The scripted comparison still
uses fewer interface calls. Re-run `./scripts/measure.sh` rather than quoting this table
after the file list moves.

## Shape E — the overhead, held still

Shape E holds the task still and varies the span, on one 1.06 MB file of 20,000
lines of identical length and distinct content. Those ratios are a property of
the construction, not of a file list:

| span | whole file | windowed | mrw | | |
|---|---|---|---|---|---|
| 100 lines | 1,060,000 | 5,300 | 6,011 | **176.34× less** than whole | 1.13× more than windowed |
| 2,000 lines | 1,060,000 | 106,000 | 120,113 | **8.83× less** | 1.13× more |
| 20,000 lines | 1,060,000 | 1,060,000 | 1,220,054 | 1.15× more | 1.15× more |

The overhead against a windowed read is **13% to 15%** (`fe51cdab`, 2026-11-18) — it is the
line-number gutter, which is what makes a served line addressable by a later
write, and it widens by a character once the line numbers reach five digits. The *saving* is the part of the file you were never going to look at, so
it collapses as the span approaches the whole file.

## Shape G — edit, then verify (`--then`)

A–F stop at the edit. Verification adds the project's check and any later commands.
Since ADR-092, `mrw write --then NAME` (a step the project declares) and `--then-sh CMD`
(an ad-hoc step) can run them in the write's own call, each with its own verdict.
Current releases require `MRW_SHELL=on` for `--then-sh`; declared `--then` steps need no
ad-hoc shell opt-in. Shape G uses named steps: it builds a four-file Go module in a scratch
directory, declares a check (`gofmt -l`) and two steps (`go vet ./...`, a `grep -c`),
and compares the same edit and verification three ways.

Measured 2026-11-18 with the binary built from `fe51cdab`. The fixture does not
read this tree, so these numbers move only when the receipt's shape does, and by
a byte or two between runs, since the receipt carries the check's duration. The
write passes `--check`: since ADR-187 a write runs no check unless asked, and until
#657 this shape's mrw side ran two steps against the baselines' three commands.

| | baseline | mrw | |
|---|---|---|---|
| calls, separate (4 reads + 4 edits + check + 2 steps) | 11 | 2 | 5.5× fewer |
| calls, verification as one `&&` chain | 9 | 2 | 4.5× fewer |
| requests if every call is batched (reads, edits, the chain) | 3 | 2 | 1.5× fewer |
| bytes of verification output, separate | 32 | 531 | **16.6× MORE** |
| bytes of verification output, `&&` chain | 32 | 531 | **16.6× MORE** |
| verdicts named | 3 separate / 1 chain | 3 | |

**The `&&` chain is the baseline to read for calls.** It ties mrw on the
verification, one call each, so the gap it leaves is the edit side: the same
M + N as A–D. What the chain gives up is the verdict per step. It exits with one
status and does not say which step stopped it, while the receipt names every step
it ran and every step it did not.

**Bytes go the other way, and by a lot.** A passing check prints almost nothing.
The receipt names every hunk, every file written, the check command and each
step, so its length grows with the plan, not with the verification. The
baseline's column also leaves out the Edit tool's own replies, a bias against
mrw, like the raw-bytes Read column in A–D. A failing run was not measured: both
sides then print the failing command's output, and the receipt adds the steps it
did not run.

## Campaign identity

A break campaign of **57 probes** (`scripts/break-campaign.sh`) found no silent
wrong write, and every refusal names its reason. Run 2026-09-29 against v1.31.0
(`2ea8bd5`): exit codes **identical** to `docs/break/campaign-v1.30.0.txt`, probe
for probe; `docs/break/campaign-v1.31.0.txt` is the receipt. Histogram:
exit=0 ×30, exit=1 ×19, exit=2 ×8.

That identity is evidence of no *unintended* change, not that the campaign
covers every later record. The receipts live in `docs/break/`.

## Does serving more hurt?

The thesis is round trips, not bytes. The obvious objection to collapsing N
round trips into one call is that bulk delivery degrades the model. The
measurement note — instrument, results, method, limits — is
[`docs/notes/served-size-and-delivery.md`](notes/served-size-and-delivery.md).
The model × score table (readings 1–20, Sonnet / Haiku / gpt-5.6-sol) is
[docs/model-benches.md](model-benches.md).
For a capable client, 100× the bytes cost 2.4–2.5× input tokens with no
measurable loss in correct addressing; where accuracy did drop, the cause was a
second line-numbering in the delivery path, not volume.

## Windows

The reproduction scripts are `bash`. Run them under **WSL** or **Git Bash**.
The binary itself is native. See [CONTRIBUTING.md](../CONTRIBUTING.md).
