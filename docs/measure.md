# What mrw saves

**The unit is agent turns, not seconds.** mrw does not make anything faster to
execute, and no figure here is a wall-clock figure. What it removes is *steps*
— the read-edit-read-edit round trips an agent spends to change N places, each
one a full model turn. **Two calls, for any N.** That is the whole product.

```sh
./scripts/measure.sh
```

This is the only benchmark in the repository. There are no `go test -bench`
benchmarks, on purpose: a CPU figure for a tool that spends its life blocked on
a file read would measure nothing anyone is paying.

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

Measured on this tree at **`adb1b5d`** (2026-09-26, the v1.27.1 code plus docs: `measure.sh`
built its own binary from the tree). Round trips are still **2 calls for any N.**
Bytes moved because the tree grew. Shape D is **207** Go files, not 132 (the
v1.24.0 reading of 2026-09-24).

| shape | | baseline | mrw | |
|---|---|---|---|---|
| **A.** 4 sites, 4 large files | bytes vs reading those files **whole** | 232,203 | 4,159 | **55.8× less** |
| | bytes vs a **windowed** `offset`/`limit` read | 3,008 | 4,159 | **1.4× MORE** |
| | calls, whole-file (reads + edits) | 8 | 2 | 4.0× fewer |
| | calls, windowed (search + reads + edits) | 9 | 2 | **4.5× fewer** |
| **B.** 2 sites, 2 mid-sized files | bytes vs whole | 25,143 | 904 | 27.8× less |
| | bytes vs windowed | 470 | 904 | 1.9× MORE |
| | calls | 4 / 5 | 2 | 2.0–2.5× fewer |
| **C.** 1 site, whole small file | bytes (window *is* the whole file) | 15,990 | 19,406 | **1.2× MORE** |
| | calls | 2 / 3 | 2 | same to 1.5× fewer |
| **D.** 1 site in **every** Go file — 207 sites, 207 files | calls (reads + edits) | 414 | 2 | **207.0× fewer** |
| | bytes vs whole | 1,783,339 | 18,681 | 95.5× less |
| | bytes vs windowed | 2,929 | 18,681 | **6.4× MORE** |

**Shape D is the one to read, and read it for the CALLS, not the bytes.** It is
the change every codebase gets eventually — a renamed symbol, an added build
tag, a changed import — one site in each Go file. Its `6.4× MORE` is mrw's
worst possible input by construction: 207 files at ONE line each, so a per-file
header and a per-file receipt are charged against 2,929 bytes of payload.

**Shape C is in the table on purpose.** When you need a whole file and there is
one site, mrw prints *more* than the file holds — it adds a header and a line
number per line — and saves no round trips. Use Read + Edit there.

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

Each of those calls is a **full model turn**. That is the cost mrw removes, and
it is why the floor is 2 rather than "fewer".

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

Measured in the same run as A–D above (`adb1b5d`, 2026-09-26), on the same 207
Go files. `ast-grep` was not on PATH; that arm is skipped, not failed.

| | baseline | mrw | |
|---|---|---|---|
| bytes vs whole | 1,783,339 | 18,681 | 95.5× less |
| bytes vs windowed | 2,929 | 18,681 | **6.4× MORE** |
| bytes vs rg+windowed | 12,766 | 18,681 | **1.5× MORE** |
| calls, windowed (search+reads+edits) | 415 | 2 | **207.5× fewer** |

Charging the search **does not flip the byte comparison**. The win is still
turns. Re-run `./scripts/measure.sh` rather than quoting this table after the
file list moves.

## Shape E — the overhead, held still

Shape E holds the task still and varies the span, on one 1.06 MB file of 20,000
lines of identical length and distinct content. Those ratios are a property of
the construction, not of a file list:

| span | whole file | windowed | mrw | | |
|---|---|---|---|---|---|
| 100 lines | 1,060,000 | 5,300 | 6,121 | **173.17× less** than whole | 1.15× more than windowed |
| 2,000 lines | 1,060,000 | 106,000 | 120,124 | **8.82× less** | 1.13× more |
| 20,000 lines | 1,060,000 | 1,060,000 | 1,200,054 | 1.13× more | 1.13× more |

The overhead against a windowed read is **flat at ~13%** — it is the
line-number gutter, which is what makes a served line addressable by a later
write. The *saving* is the part of the file you were never going to look at, so
it collapses as the span approaches the whole file.

## Campaign identity

A break campaign of **57 probes** (`scripts/break-campaign.sh`) found no silent
wrong write, and every refusal names its reason. Run 2026-09-26 against v1.27.1
(`6d35d58`): exit codes **identical** to `docs/break/campaign-v1.27.0.txt` and to
v1.26.0, probe for probe; `docs/break/campaign-v1.27.1.txt` is the receipt. Histogram:
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
