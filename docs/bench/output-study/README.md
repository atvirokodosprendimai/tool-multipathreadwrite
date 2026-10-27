# Output study: where tool output goes, and what each candidate would cut

Two read-only scripts over the Claude Code transcripts on one Mac, written 2026-10-27 for ADR-177, ADR-178
and ADR-179. Neither writes anything; each prints one JSON summary on stdout, naming its window.

```sh
python3 -I docs/bench/output-study/measure_output.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z > out.json
python3 -I docs/bench/output-study/measure_repeat.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z > repeat.json
```

A tool result is counted when its transcript line's own `timestamp` falls in `[since, until)`, so a later run
over the same window counts the same results whatever the files' mtimes. `--glob` names other transcripts.

- `measure_output.py` buckets every tool result by tool (and, for Bash, by the command's shape: `mrw read`,
  `mrw write`, `mrw run`, a command running more than one of them (`mrw mixed`, attributed to none),
  cat/grep/rg/sed, tests and gates, git, gh, other) and counts what each candidate could have cut: lines over
  400 columns cut at 200; JSON documents of 1,000 characters or more; runs of identical and identical-up-to-digits
  lines, with mrw's `NNN| ` prefix stripped and blank lines never counted; whole-file reads over 150 lines; Bash
  outputs over 4 KB, with and without the reads among them; commands piped into `head`/`tail`; the two mrw
  stderr notices counted at first, and any notice line repeated inside one result. It also counts what the
  records add, for the re-measure: `mrw run` calls and reads of its `mrw-runs/` logs, `--json-path` calls,
  `--terse` writes, `-- same as served` answers, and `mrw read --grep` output.
- `measure_repeat.py` counts, within each session, tool results of 500 characters or more that repeat an earlier
  result byte for byte, and mrw content blocks (path, sha, served text; a block counts only if it holds a numbered
  line) served again unchanged. An answer by reference is counted apart (`reference_chars`), never as a repeat.

## The registered baseline (2026-10-27)

`baseline-2026-10-27.json` and `repeat-baseline-2026-10-27.json` are the two scripts' output for the window above:
129 main-session transcripts (107 with an mrw call), 100,931,328 characters of tool output.

| bucket | characters | calls | share of all |
|---|---|---|---|
| `mrw read` | 57,276,199 | 19,075 | 56.7% |
| `mrw read` and `mrw write` in one command | 5,782,400 | 6,943 | 5.7% |
| cat / grep / rg / sed through Bash | 17,694,558 | 17,695 | 17.5% |
| lines over 400 columns in `mrw read`, beyond 200 | 4,549,698 | 1,939 | 4.5% |
| `mrw read --grep` output | 16,373,298 | 7,525 | 16.2% |
| Bash outputs over 4 KB, not mrw | 7,810,154 | 952 | 7.7% |
| … of which not reads | 1,544,742 | 142 | 1.5% |
| JSON documents of 1 KB or more, any tool | 6,190,444 | 1,020 | 6.1% |
| … of which through `mrw read` | 71,678 | | 0.07% |
| whole-file reads over 150 lines (Read 1,566,745 + `mrw read` 2,622,244) | 4,188,989 | 219 | 4.2% |
| foldable runs, identical up to digits | 1,138,774 | | 1.1% |
| foldable runs, identical | 173,151 | | 0.17% |
| repeated identical mrw content blocks (measure_repeat) | 223,463 | 240 | 0.22% |
| the two mrw stderr notices counted | 100,280 | 243 | 0.10% |
| notice lines repeated inside one result | 28,763 | 149 | 0.03% |

Also: 2,080 `mrw read` calls already passed `--max-cols`; 11,478 of 23,850 non-mrw Bash calls were piped into
`head` or `tail`; `mrw write` averaged 606 characters per call (5,165 calls).

**The first run** (2026-10-27, earlier versions of these scripts, files chosen by mtime) reported figures the
records were first written with: 155 transcripts, 109.6M characters; Bash outputs over 4 KB 10.2M in 1,251 calls,
the reads included; repeated identical mrw blocks 0.52M, blocks with no served line included. The fixed-window
re-run above replaces them; its method is the one registered.

## What these numbers do not say

- **Main sessions only.** By default the scripts read `~/.claude/projects/*/*.jsonl`; subagent transcripts live
  in subdirectories and are not scanned.
- **One machine, one owner, mostly this repository's own development.** Not a sample of mrw's users.
- **Every saving is an upper bound.** It assumes the feature is used on every output it could apply to, and that
  the agent needed none of what it would have cut.
- **JSON pays only if agents route JSON through mrw.** 72K of the 6.2M arrived through `mrw read`; the motivating
  case is a peer session's `task-state --json` loop of about 60-70 KB per call, run through Bash.
- **The cat/grep/rg/sed bucket is adoption, not a feature gap.** mrw already reads and greps; that 17.5% is
  agents not using it, which `docs/agents.md` addresses.
- **A command classified `mrw mixed`** ran a read and a write (or a run) at once; its output is not split.
- **Shares, not totals, carry the comparison.** The workload differs from one 30 days to the next.

The pre-registered re-measure after the release is in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-177,
ADR-178 and ADR-179".
