# The ADR-218 pilot: frozen 2026-11-24, before any code of the record existed

This file and `fixture.py` are the pilot's inputs. Both were written, and their sha256 filed in memoryd
(space `tool-multipathreadwrite`), before the first line of ADR-218's implementation and before any
session ran a task. Nothing below changes after the first counted run; `results.md`, beside this file,
is written afterwards and holds what happened.

Amended once, 2026-11-24, after the code was written and before any session ran a task, on a reviewer's
finding that the bar could be met without the refusals being met: the bar's clauses 2 and 4 now read
tasks 3 to 5 only, and the counting rules say what they left open. Every change makes the bar harder
to meet or a count less open to reading. The new sha256 of this file was filed in memoryd then.

The pilot measures friction: whether a model in Claude Code's Code Mode, with `MRW_SHOWN=off`, still
gets ordinary edits done, and what the refusals cost it. It does not show what text reached the model.

## Setup

- **Host:** Claude Code 2.1.331, started interactively with `CLAUDE_CODE_EXPERIMENTAL_CODE_MODE=1` and
  `--model claude-fable-5-2`, in auto mode, one fresh session per run, in one already trusted scratch
  folder. The account's own user-level instructions and hooks are loaded, as in any session there.
- **mrw:** the build of the ADR-218 branch, first on `PATH` for the session, in both arms. The runner
  records `mrw version` per run.
- **Switched arm:** the folder's `.claude/settings.json` holds `"env": {"MRW_SHOWN": "off"}`.
  **Control arm:** the same file without that key. Nothing else differs.
- **Permissions:** `Bash(mrw:*)` is allowed; nothing else is added to what auto mode permits.
- **Fixture:** `python3 -I fixture.py make DIR` writes 12 files of 60 lines under `pkg/`. Base tree
  digest (`fixture.py sha DIR`): `950e7eeb98470b820de9aecf8f1caf80be6ca56877da09d42a6d88ce10dc97d7`.
- **Reset before every run:** the folder's `pkg/` is removed and made again, the result is committed
  in the folder's own git repository, and the session gets a state home (`XDG_STATE_HOME`) that did not exist.
- **A logging hook** on `PreToolUse` and `PostToolUse` writes every payload, strings cut at 4,000
  characters, to one file per run.

## The five prompts, word for word

Each is sent as the session's first and only message. Runs: three per task per arm, 30 in all, serially.

1. `Use mrw for every read and every edit; run mrw instructions first if you do not know it. In pkg/m03.go change the constant Limit03 from 30 to 45. Change nothing else. Say DONE when it is written.`
2. `Use mrw for every read and every edit; run mrw instructions first if you do not know it. In pkg/m05.go, pkg/m06.go, pkg/m07.go and pkg/m08.go the function scale is called twice in each file. Rename each of those eight calls to rescale. Change nothing else. Say DONE when it is written.`
3. `Use mrw for every read and every edit; run mrw instructions first if you do not know it. In pkg/m02.go replace the five lines inside func Clamp02, everything between its opening line and its closing brace, with exactly two lines, each indented four spaces: "y := x * 2" and "return y". Change nothing else. Say DONE when it is written.`
4. `Use mrw for every read and every edit; run mrw instructions first if you do not know it. Make three changes with one mrw write. In pkg/m09.go replace the five lines inside func Clamp09 with the two lines "y := x * 2" and "return y". In pkg/m09.go replace the five lines inside func Sum09 with the two lines "total := len(xs)" and "return total". In pkg/m10.go replace the five lines inside func Clamp10 with the two lines "y := x * 2" and "return y". Every new line is indented four spaces. Change nothing else. Say DONE when it is written.`
5. `Use mrw for every read and every edit; run mrw instructions first if you do not know it. Make two separate mrw writes to pkg/m11.go, the second only after the first has applied. First change the constant Limit11 from 110 to 99. Then replace the five lines inside func Clamp11 with the two lines "y := x * 2" and "return y", each indented four spaces. Change nothing else. Say DONE when it is written.`

## What each task must leave

`fixture.py sha` of the folder after the run:

| Task | Digest |
|------|--------|
| 1 | `fbe92dbf67c544a9888c80a2481010dd4bd28293d8698821723e24097e75a5d2` |
| 2 | `d90c044b8d28d74581ab5457802179f0bc6a94d3b6fe19de1b73c84ff6deb730` |
| 3 | `0c22fe529d4833e2390eaee2ef6fb40f6bf5be2a3832350305f0fe997d3e56d4` |
| 4 | `8d06fed2e3f88a0c8f617c7ef43b99d8169c68e79937508c29704eea7532a0fa` |
| 5 | `24079b7122321fdded5f482751b9ba9dc85ac371337b6f4738482c2cff874c01` |

## Counting

Read from the run's hook log, never from the session's own account of itself.

- **A write** is a `PostToolUse` row for `Bash` whose command runs `mrw` with the subcommand `write` and
  no `--dry-run`. One row is one write, however many commands it chains. It **landed** when its output,
  both streams, holds the lower-case word `applied` and not `NOTHING WRITTEN`; anything else is
  **refused**, a usage error and `PARTIALLY APPLIED` included. A refusal's **kind** is the first of these
  that fits: `number` when the output holds `no recorded showing`; `line-after` when it holds
  `needs a served line after`; `other`.
- **Three in a row** is three consecutive writes of one run refused with the same kind; a landing between
  them ends the row.
- **A write's form:** `computed` with `--sub` or `--rewrite`; `block` with `--format=search_replace`;
  `patch` with `--format=apply_patch`; `native` otherwise.
- **A run ends** when the session stops working, or at 5 minutes, whichever is first. The digest is taken then.
- **A run is void, and counts as a miss,** when its log holds a host `Edit`, `Write` or `NotebookEdit`
  call, or a `Bash` command other than `mrw` that writes under `pkg/` (a redirect, `tee`, `sed -i`,
  `perl -i`, `python`, `cp`, `mv`, `git checkout`, `git restore`, `git apply`); and, in the switched arm,
  when no `mrw read` in it printed `recorded as licensed, not shown`.
- **One counted attempt per task, run and arm.** A run in which the session did not start, or the host
  reported an API error before the first tool call, is run again once and the event is noted. Every
  other outcome counts as it fell.
- **No rehearsal of a task in the switched arm.** The runner is proved on one uncounted control-arm run
  of task 1; after it only the runner's mechanics may change.

## The bar, for the switched arm

1. All 15 runs are valid and end with the task's digest.
2. Over the nine runs of tasks 3, 4 and 5, the ones a computed plan cannot do, the median number of
   refused writes per run is at most 1.
3. No run holds three refusals of the same kind in a row.
4. Each of those nine runs lands at least one `native` or `block` write: a pilot in which the model did
   not write those edits by hand has not met the refusals, and is reported as not exercised, which is a miss.

A missed bar means ADR-218's switch does not release as built; what replaces it is a new decision.
The control arm's 15 runs are reported beside these, by the same rules, and are not part of the bar.
One model and one Claude Code version; nothing is claimed beyond them.
