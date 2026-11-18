# effect-study: four ideas from a research reading, measured before anything is built

Four ideas came out of a reading of 2026 papers on agent edit formats, billed cost, compaction and
concurrent writers (2026-11-18). Each got a measurement and a bar in `docs/adr/BACKLOG.md`
("Pre-registered for four ideas from the 2026-11-18 research reading"), committed and pushed before
`scan.py` ran on anything but its synthetic fixtures. This folder is the scanner, its self-test and the
numbers. Nothing here changes mrw.

## Answer

Measured 2026-11-18 over every Claude Code transcript on one Mac with an entry before
2026-11-18T10:00:00Z: 965 transcripts, 97,605 requests, 16,100 `mrw write` calls, 199 compactions. Two
runs, identical output. The second column leaves out this repository's own sessions, which provoke
refusals on purpose (786 transcripts, 73,694 requests).

| | all | without this repository | bar | verdict |
|---|---|---|---|---|
| **E1** cache write + cache read, share of priced cost | **92.2%** | 92.0% | at least 60% | met |
| E1 output, share of priced cost | 7.8% | 8.0% | | |
| E1 sites per `mrw` call: mean / median / 90th | 4.26 / 2 / 9 | 4.23 / 2 / 9 | | |
| E1 sites per request with host tools only: mean / median / 90th | 1.30 / 1 / 2 | 1.30 / 1 / 2 | | |
| **E2** different neighbouring lines with the same 2-hex tag | **0.33%** | (this repository's files) | at most 1.0% | met |
| E2 identical neighbouring lines | 0.20% | | | |
| E2 anchor text, share of recorded output tokens | 0.16% | 0.15% | at least 1.0% | missed |
| E2 anchor misses per anchored hunk | **2.52%** | 1.79% | at least 1.0% | met |
| E2 single-line `replace` with no anchor, share of hunks | **22.6%** | 21.9% | at least 20% | met |
| **E3** host Edit/Write share, 30 calls before / after a compaction | 4.72% / 4.90% | 5.81% / 5.94% | +5 points | missed |
| E3 compactions paired | 159 | 120 | at least 20 | |
| **E4** "changed since" refusals with no cause found, per `mrw write` | **0.14%** (23) | 0.03% (3) | at least 0.5% | missed |
| E4 refusals in all: own shell command / own subagent / nothing found | 55: 21 / 11 / 23 | 16: 10 / 3 / 3 | | |

In plain words:

- **E1, met.** On this corpus 92% of what the token classes would cost is cache traffic: 74% cache
  reads, 18% cache writes. Output is 8% and fresh input is nothing. What a tool returns is paid for once
  as a cache write and then again, at a tenth, on every later request of the session, so the number of
  requests is the larger lever and bytes served the smaller. One `mrw` call acts on 4.3 sites on
  average; a request that used only the host's `Read`, `Edit` and `Write` acts on 1.3. That comparison
  is observational: the sessions, tasks and models differ.
- **E2, met on its second arm.** A 2-hex tag would be a cheap anchor only in the sense of safety, not
  of output: anchor text is 0.16% of output tokens. What meets the bar is exposure: 2.5% of anchored
  hunks are refused for an anchor that is not in the addressed line, and 22.6% of hunks are single-line
  replaces that carry no anchor, where the same mistake would land. A tag differs between two
  different neighbouring lines 99.67% of the time.
- **E3, missed.** The host's own edit tools are 4.7% of file-changing calls before a compaction and
  4.9% after. The rules do not slip here; the instructions file is reloaded after a compaction.
- **E4, missed.** 23 refusals in 16,100 writes had no cause in the session's own transcript, and 20 of
  them are in this repository, where several sessions and agents share the checkout.

What follows from the bars: E1 earns a requests column in `scripts/measure.sh` and a sentence in
`docs/measure.md`; E2 earns a draft record for Zy; E3 and E4 close.

## Method

`scan.py` reads every `*.jsonl` under `~/.claude/projects`, subagent transcripts included, and counts
an entry by its own timestamp. It prints aggregates only.

- **E1.** Token classes come from each assistant message's `usage`, once per message id (the same
  usage is repeated on every line of a message). Priced cost weights them input 1, output 5, cache
  write 1.25 (5-minute) and 2 (1-hour), cache read 0.1. The weights are an assumption about price
  ratios; the raw classes are in `results.json`. Sites: for an `mrw read`, the `@@` range lines in its
  answer; for an `mrw write`, the hunk count on its summary line; for the host, `Read`, `Edit`,
  `Write`, `MultiEdit` and `NotebookEdit` calls in one message.
- **E2 (a).** Over `git ls-files` of the repository named by `--repo`, files with a NUL in the first
  8 KiB left out: each pair of neighbouring lines is identical, or different with the first 2 (or 3)
  hex digits of their SHA-1 equal, or neither.
- **E2 (b).** Hunk headers are counted in the text of every shell command that runs `mrw … write` and
  in the content of every `Write` call. An anchor miss is a result line holding
  `anchor "…" not in line N`.
- **E3.** File-changing calls are the host's `Edit`, `Write`, `MultiEdit`, `NotebookEdit` and a shell
  command that runs `mrw … write`. For each compaction after the session's first `mrw write`, the 30
  such calls before it and the 30 after, not crossing another compaction; five or more on each side.
- **E4.** For each `PATH changed since mrw last saw it`, the scanner walks back to the last mrw answer
  that showed or wrote PATH and looks at what the session did since: an `Edit`/`Write` of that path, an
  `Agent` call, or a shell command that can rewrite files (a formatter, a git command that moves the
  tree, `sed -i`, `adr-verify`, `mv`, `cp`, `tee`, a Python or npm script).

## Limits

- One machine, one user, one host. Codex and Cursor transcripts are not read.
- Transcripts carry no success label. Cost per solved task is what the stage runs under `docs/bench/`
  measure; E1 does not replace them and says nothing about whether fewer requests solve as many tasks.
- E1's weights ignore model, tier and batch pricing. Sites per request compares different work.
- E2's anchor-miss rate counts every miss, including a stale memory of a line that moved, not only an
  address that is off by a few lines; a tag catches both, an anchor already catches both. The 22.6% is
  exposure, not a measured rate of wrong single-line replaces: nothing here observes one landing. A
  plan written to a file by a shell heredoc that does not also run `mrw … write` is not counted.
- E2 (a) reads this repository only. Code with more repeated lines collides more on identical
  neighbours, which no tag or anchor separates.
- E3 cannot see a rule broken in prose or a refusal; it sees which tool changed a file.
- E4's "nothing found" is a ceiling for another writer: it includes causes the transcript does not
  show, such as an editor or a hook. "Own shell" may be a command that touched other files.

## Run it

```sh
python3 -I docs/bench/effect-study/test_scan.py                 # 30 checks on synthetic transcripts
python3 -I docs/bench/effect-study/scan.py --until 2026-11-18T10:00:00Z --repo . > results.json
python3 -I docs/bench/effect-study/scan.py --until 2026-11-18T10:00:00Z --skip tool-multipathreadwrite
```

Give `--until` a time in the past: a live session keeps appending to its transcript.

## Files

- `scan.py`: the scanner. `test_scan.py`: its self-test.
- `results.json`, `results-without-this-repository.json`: the two runs above.
