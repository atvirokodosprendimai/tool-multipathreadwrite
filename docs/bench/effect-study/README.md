# effect-study: four ideas from a research reading, measured before anything is built

Four ideas came out of a reading of 2026 papers on agent edit formats, billed cost, compaction and
concurrent writers (2026-11-18). Each got a measurement and a bar in `docs/adr/BACKLOG.md`
("Pre-registered for four ideas from the 2026-11-18 research reading"), committed and pushed before
`scan.py` ran on anything but its synthetic fixtures. This folder is the scanner, its self-test and the
numbers. Nothing here changes mrw.

## Answer

Measured 2026-11-18 over the Claude Code transcripts on one Mac, counting entries before
2026-11-18T10:00:00Z: 934 transcripts, 97,605 requests, 15,516 shell calls that run `mrw write`, 199
compactions. Two runs, identical output. The second column leaves out this repository's own sessions,
which provoke refusals on purpose (762 transcripts, 73,694 requests).

| | all | without this repository | bar | verdict |
|---|---|---|---|---|
| **E1** cache write + cache read, share of priced cost | **92.2%** | 92.0% | at least 60% | met |
| E1 output, share of priced cost | 7.8% | 8.0% | | |
| E1 sites per shell call that runs `mrw`: mean / median / 90th | 4.25 / 2 / 9 | 4.22 / 2 / 9 | | |
| E1 sites per request that used `mrw`: mean / median / 90th | 4.36 / 2 / 9 | 4.34 / 2 / 9 | | |
| E1 sites per request with host tools only: mean / median / 90th | 1.30 / 1 / 2 | 1.30 / 1 / 2 | | |
| **E2** lines with the same 2-hex tag as a different neighbour | **0.65%** | (this repository's files) | at most 1.0% | met |
| E2 lines with an identical neighbour | 0.41% | | | |
| E2 anchor text, share of recorded output tokens | 0.16% | 0.16% | at least 1.0% | missed |
| E2 anchor-miss lines per anchored hunk typed (distinct per session) | **2.28%** (2.13%) | 1.78% (1.71%) | at least 1.0% | met |
| E2 single-line `replace` with no anchor, share of hunks typed | **23.1%** | 22.7% | at least 20% | met |
| **E3** host Edit/Write share, 30 calls before / after a compaction | 5.32% / 5.85% | 6.52% / 7.41% | +5 points | missed |
| E3 compactions paired | 168 | 124 | at least 20 | |
| **E4** "changed since" refusals with no cause found, per `mrw write` call | **0.08%** (12) | 0.01% (1) | at least 0.5% | missed |
| E4 every such refusal not after the session's own Edit, per `mrw write` call | 0.34% (52) | 0.14% (16) | | |

In plain words:

- **E1, met.** On this corpus 92% of what the token classes would cost, at the assumed weights, is
  cache traffic: 74% cache reads, 18% cache writes. Output is 8%. That is where the cost sits; it does
  not by itself say whether requests or bytes move it, since a session's cache traffic grows with both
  the number of requests and the text already in context, and neither was varied here. A shell call
  that runs `mrw` acts on 4.3 sites on average; a request that used only the host's `Read`, `Edit` and
  `Write` acts on 1.3. That comparison is observational: the sessions, tasks and models differ.
- **E2, met on its second arm.** Anchor text is 0.16% of output tokens, so a tag would save almost no
  output. What meets the bar is two counts side by side: anchor-miss lines are 2.3% of the anchored
  hunks typed, and 23.1% of hunks typed are single-line replaces with no anchor. The first is a raw
  ratio of two counts, not the share of anchored hunks that were refused: a plan sent twice counts its
  miss twice (2.1% with each distinct miss counted once per session). On this repository's files,
  0.65% of lines share a 2-hex tag with a different neighbouring line.
- **E3, missed.** The host's own edit tools are 5.3% of file-changing calls in the 30 before a
  compaction and 5.8% in the 30 after: half a point, against a bar of five. This measures which tool
  changed a file, not rule-following in general.
- **E4, missed.** 12 refusals in 15,516 write calls had no candidate cause in the session's own
  transcript. Counting every refusal that did not follow the session's own `Edit` of that path, the
  most this could be, gives 52, or 0.34%, still under the bar.

What follows from the bars: E1 earns a requests column in `scripts/measure.sh` and a sentence in
`docs/measure.md`; E2 earns a draft record for Zy; E3 and E4 close.

## Method

`scan.py` reads every `*.jsonl` under `~/.claude/projects`, subagent transcripts included, and counts
an entry by its own timestamp; an entry with none takes the last one seen in its transcript. A
transcript with no entry in the window is not counted. It prints aggregates only.

- **A shell call runs `mrw read` or `mrw write`** when `mrw` stands at a command position (the start,
  or after `;`, `&`, `|`, `(`, `$(` or a newline, with `VAR=value` words allowed before it) and the
  first word after its `--root`/`-C` flag is that subcommand. `mrw read --grep write` is a read, and
  `echo "mrw write"` is neither.
- **E1.** Token classes come from each assistant message's `usage`, once per message id. Priced cost
  weights them input 1, output 5, cache write 1.25 (5-minute) and 2 (1-hour), cache read 0.1. The
  weights are an assumption about price ratios; the raw classes are in `results.json`. Sites: the `@@`
  range lines in the answer of a shell call that runs `mrw read`, plus the hunk counts on the summary
  lines of one that runs `mrw write` (a call that runs both is one call with both); for the host,
  `Read`, `Edit`, `Write`, `MultiEdit` and `NotebookEdit` calls in one message. A request that used
  `mrw` counts its host sites too.
- **E2 (a).** Over `git ls-files` of the repository named by `--repo`, files with a NUL in the first
  8 KiB left out: the share of lines with an identical line directly above or below, and the share
  whose first 2 (or 3) hex digits of SHA-1 equal those of a different line directly above or below.
- **E2 (b).** Hunk headers are counted in the text of every shell call that runs `mrw write` and in
  the content of every `Write` call, with a quoted path and `anchor="…"`, `anchor='…'` or an unquoted
  anchor up to the next option. A single-line replace is `N`, `$` or `N-N`. An anchor miss is a result
  line holding `anchor "…" not in line N`.
- **E3.** File-changing calls are the host's `Edit`, `Write`, `MultiEdit`, `NotebookEdit` and a shell
  call that runs `mrw write`. For each compaction after the session's first `mrw write`: the 30 such
  calls before it and the 30 after, whatever else lies between; five or more on each side.
- **E4.** For each `PATH changed since mrw last saw it`, the scanner walks back to the last mrw answer
  that served PATH (a header with a line count) or wrote it (a receipt row), starts at the shell call
  that produced that answer, and looks at what the session did since. An `Edit`/`Write` of that path is
  a cause. A shell command that can rewrite files (a formatter, a git command that moves the tree,
  `sed -i`, `adr-verify`, `mv`, `cp`, `tee`, a Python or npm script) or an `Agent` call is a candidate
  only: nothing ties it to PATH.

## Limits

- One machine, one user, one host. Codex and Cursor transcripts are not read.
- Transcripts carry no success label. Cost per solved task is what the stage runs under `docs/bench/`
  measure; E1 does not replace them and says nothing about whether fewer requests solve as many tasks.
- E1's weights ignore model, tier and batch pricing. Sites per request compares different work.
- E2's miss ratio divides result lines by hunks typed; they are not the same population. A miss may be
  a stale memory of a line that moved and not an address that is off by a few lines. The 23.1% is
  exposure, not a measured rate of wrong single-line replaces: nothing here observes one landing. A
  plan written by a heredoc to a file in a shell call that does not also run `mrw write` is not
  counted, nor is one built by a script.
- E2 (a) reads this repository only. Code with more repeated lines has more identical neighbours,
  which no tag or anchor separates.
- E3 cannot see a rule broken in prose or a refusal; it sees which tool changed a file.
- E4's classes are a look-back, not a proof. "Nothing found" misses causes the transcript does not
  show (an editor, a hook), and a candidate may be a command or an agent that never touched the path,
  hiding another writer. A `--json` receipt and a `--quiet` write are not recognised as the last time
  mrw wrote a path. The second E4 row is the figure that does not depend on the candidates.

## Run it

```sh
python3 -I docs/bench/effect-study/test_scan.py                 # 36 checks on synthetic transcripts
python3 -I docs/bench/effect-study/scan.py --until 2026-11-18T10:00:00Z --repo . > results.json
python3 -I docs/bench/effect-study/scan.py --until 2026-11-18T10:00:00Z --skip tool-multipathreadwrite
```

Give `--until` a time in the past: a live session keeps appending to its transcript.

## Files

- `scan.py`: the scanner. `test_scan.py`: its self-test.
- `results.json`, `results-without-this-repository.json`: the two runs above.
