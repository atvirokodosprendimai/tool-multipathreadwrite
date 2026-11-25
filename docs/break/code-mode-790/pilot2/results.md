# The ADR-218 pilot, second run: results, 2026-11-25

Run once, serially, 09:05 to 09:17 Vilnius time, by `run.sh` under the protocol in `README.md` beside this
file, frozen at commit `148a1d64` with its sha256 filed in memoryd before any session started. Claude Code
2.1.332 in Code Mode, `--model claude-fable-5-2`, auto mode, macOS; mrw `dev (148a1d6)`, the committed head's
build, first on `PATH` in both arms. Before the counted runs, `summarise.py` over the first pilot's 30 logs
gave that pilot's hand recount in every run (forms, landings, refusals), the five cut notes apart, and one
uncounted control run of task 1 proved the runner. No run was repeated. The hook logs hold local paths and
are not kept here; they were in the session's scratch folder.

## Verdict

**The bar is met, all four clauses, by the summariser's own lines below.**

1. All 15 switched runs are valid and end with the task's digest.
2. Over the nine switched runs of tasks 3, 4 and 5 the median number of refused writes is 0.
3. No run holds three refusals in a row; no run holds one.
4. Each of those nine runs landed a `search_replace` block (task 5 a native write, then a block).

The control arm's 15 runs also all ended with the digest, with no refused write; its multi-line edits were
native plans.

## Checked by hand against the logs

- **No write was refused in either arm**: no `Bash` row's output held `NOTHING WRITTEN`. Nine rows held
  `no recorded showing`; all nine are `mrw instructions`, whose text holds the phrase, and none is a write.
- **The note reached every switched run**: each of the 15 has an `mrw read` whose whole output held it, and
  the hook's own environment held `MRW_SHOWN=off` on every event of those runs, and no value in the control arm.
- **Host `Write` calls: 16 in the 30 runs, none under `pkg/`** (plan files in the session's scratch folder).
  Under the first pilot's rule 12 runs would be void, 8 of them switched. This is the rule the protocol
  relaxed after seeing data, and the verdict rests on that relaxation.
- **Rows not counted as writes**: nine `mrw write --help` and one `--dry-run`, all in the switched arm.

## What this does and does not support

- It supports: with `MRW_SHOWN=off`, one model in one host version did five ordinary edits, three times
  each, with no refused write, and every multi-line edit landed as a block. The logs show the note in the
  output of a `Bash` call and the block landing after it; they do not show what the script returned to the
  model, so "the note steers" is an inference from the control arm landing native plans for the same tasks.
- It does not support: that a model recovers from the refusals, since again none occurred; anything about
  a script that hides mrw's output from the model, where the note would not arrive either; any other
  model, host or version; that no read text reached the model, which no pilot here measured.
- The first pilot ran Claude Code 2.1.331 and this one 2.1.332, on consecutive days; the two are not one sample.

## Found by review after the run: where the counter is wrong, and that no such input occurred

A review of the frozen files (Codex, 2026-11-25) showed inputs the counter gets wrong. The files stay as
frozen; each was then looked for by hand in all 162 `Bash` rows and 16 `Write` rows of the 30 logs. Every
command was split at `;`, `&` and `|` outside quotes, heredoc bodies set aside, and the program of each part
listed. The whole list: `mrw` 131 (`instructions` 40, `read` 45, `write` 46: the 36 counted, nine `--help`,
one `--dry-run`), `git` 48 (`diff` 43, `status` 5), `echo` 14, `grep` 10, `printf` 10, `cat` 5, `head` 1,
`mkdir` 1. No row runs `cp`, `mv`, `tee`, `sed`, `perl`, Python or any other program. With plan text and
scratch paths set aside, every argument form those programs were given, with its count: `git diff` (22),
`git diff --stat` (14), `git diff -U0` (5), `git diff PATH` (1) and `git diff -- PATH` (1) with one fixture
path, `git status --short` (5); `grep '^[+-]'` (4), `grep '^[+-][^+-]'` (2), `grep -n scale` with four fixture
paths (2), `grep -n -i -B3 -A12 PATTERN` (1) and `grep -n -i -A8 format=search_replace` (1), the last two on
the output of `mrw instructions`; `head -60` (1); `echo ----` (6), `echo exit=…` (6), `echo exit …` (2);
`printf %s` of plan text piped into `mrw write` (6) or redirected to a scratch file (4); `cat >` a scratch
file from a heredoc (5); `mkdir -p` a scratch folder (1). No `--output`, `-o` or other option that names a
file to write.

| The counter's defect | In the 30 logs |
|---|---|
| a non-mrw writer chained after a counted write is not checked (`mrw write p; cp /tmp/n pkg/n.txt`, `git diff --output=pkg/n.txt`), and the digest reads only `pkg/*.go` | no fixture file was written outside mrw: by the argument list above, the only commands that name a file to write are the redirects of `cat` and `printf` and the one `mkdir`, and every such target, like every host `Write` target, is in the session's own scratch folder, outside the fixture folder (git's own bookkeeping under `.git`, which `git status` may refresh, is not a fixture file) |
| `python3 --version` alone would void a run | no row runs Python |
| `mrw --root=. write` is not counted; a quoted `'--help'` is counted | neither spelling occurs |
| a quoted `|`, `;` or `&` before `--dry-run` splits the command | does not occur; the one dry run is excluded |
| the form is read from the whole row, so a help row chained with a write could mislabel it, and `search_replace` anywhere in the command (a plan file's name) makes an `apply_patch` or native write a block | no row holds two `mrw write`; recounted from each write's own options: switched, 9 with `--format=search_replace` (one in each run of tasks 3 to 5), 3 with `--grep … --sub`, 6 with a plan path or `-` alone; control, 3 with `--grep … --sub`, 15 with a plan path or `/dev/stdin` alone; no `apply_patch` anywhere. The same forms as the summariser's lines |
| a command cut at 4,000 characters could lose a flag | no command was cut |
| `PKG/` passes the `pkg/` test on a case-folding volume | no target path holds `pkg` in any case |
| a landing line or `"mutation": "complete"` printed by another command in the row (`mrw write --bogus; echo '… — applied'`), or a refusal phrase from `mrw instructions` in the same row as a write | none: no row holds `--json` or the word `mutation`; in each of the 36 counted rows the kept output holds exactly one landing line, the line above it is a `wrote` or `created` line, no output was cut, and nothing else in the row could have printed either line: 27 of the 36 rows are one `mrw write` alone, 6 are `printf PLAN | mrw write …; echo exit=…`, 2 are `printf PLAN | mrw write …`, and 1 is `mrw write …; echo "exit $?"; git diff`, whose diff lines begin with `+`, `-` or a space; no counted row holds an `mrw read`, `mrw instructions` or `cat`; no row holds both `instructions` and a write |

Two sentences of the frozen `README.md` are loose and are corrected here, not there: the first pilot missed
on clause 1 alone, by its two void rules, not by a counting rule; and "`mrw write --help` is not a write"
changes which rows are writes, not only how a count is taken.

## The summariser's lines, as printed

```
switched task=1 run=1 digest-ok secs=16 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=1 run=2 digest-ok secs=44 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=1 run=3 digest-ok secs=16 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=2 run=1 digest-ok secs=17 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
switched task=2 run=2 digest-ok secs=22 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
switched task=2 run=3 digest-ok secs=19 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
switched task=3 run=1 digest-ok secs=23 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=3 run=2 digest-ok secs=35 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=3 run=3 digest-ok secs=26 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=4 run=1 digest-ok secs=34 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=4 run=2 digest-ok secs=32 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=4 run=3 digest-ok secs=32 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=5 run=1 digest-ok secs=30 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
switched task=5 run=2 digest-ok secs=38 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
switched task=5 run=3 digest-ok secs=36 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
control task=1 run=1 digest-ok secs=17 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=1 run=2 digest-ok secs=18 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=1 run=3 digest-ok secs=16 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=2 run=1 digest-ok secs=17 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
control task=2 run=2 digest-ok secs=17 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
control task=2 run=3 digest-ok secs=24 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
control task=3 run=1 digest-ok secs=16 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=3 run=2 digest-ok secs=16 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=3 run=3 digest-ok secs=18 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=4 run=1 digest-ok secs=17 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=4 run=2 digest-ok secs=18 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=4 run=3 digest-ok secs=18 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=5 run=1 digest-ok secs=22 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
control task=5 run=2 digest-ok secs=21 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
control task=5 run=3 digest-ok secs=24 host=2.1.332 (Claude Code) mrw=dev (148a1d6) writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
```
