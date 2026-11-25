# The ADR-218 pilot: results, 2026-11-24

Run once, serially, 19:41 to 19:52 Vilnius time, by `run.sh` under the protocol in `README.md`
(sha256 `7cf286f8…7a41`, the amended freeze; `fixture.py` `a78a112e…d97b`, both as filed in memoryd).
Claude Code 2.1.331 in Code Mode, `--model claude-fable-5-2`, auto mode, macOS; mrw `dev (c0e9f38, dirty)`,
the ADR-218 worktree's build, first on `PATH` in both arms. One uncounted control run of task 1 proved the
runner first, and one uncounted switched session with another prompt proved that the hook log holds the
read's note. No run was repeated. The hook logs hold local paths and are not kept here; they were in the
session's scratch folder.

## Verdict

**By the letter of the protocol the bar is missed: clause 1 fails.** Nine of the fifteen switched runs
are void under the rule "a host `Write` call voids a run", and five under "no `mrw read` printed the note"
(one run under both; thirteen of fifteen void by one rule or the other). Both rules were written badly, and
that is the protocol's defect, not the runs': the outcome below is reported beside the letter, not in place
of it. Whether the switch releases on this evidence is the owner's decision, as the protocol says.

**What the logs show, read by hand:**

- **All 30 runs, both arms, ended with the task's digest**, in 11 to 66 seconds.
- **No switched run had a refused write.** Zero of 15. Clauses 2 and 3 hold with nothing to count.
- **Every switched run of tasks 3, 4 and 5 landed a `search_replace` block** for its multi-line edit
  (clause 4 holds), where every control run of those tasks landed a native plan. The model read the note
  the switched read prints, which the script returned to it with the read's output, and chose the block
  form before any refusal. So the pilot shows that the note steers, and shows nothing about recovery from
  a refusal: none happened.
- **The host `Write` calls wrote plan files into the session's own scratch folder**, never under `pkg/`:
  0 of 23 such calls in the 30 runs named a path in the fixture. That is the form mrw's own instructions
  give for a plan (recipe 5). The void rule should have said "a host `Write` under `pkg/`".
- **The "no switched read" voids are the logger's cut.** In those five runs the read's output passed 4,000
  characters (four files in one read, or a read chained after `mrw instructions --core`) and the note, which
  comes last, fell outside the kept text. The hook process's own environment held `MRW_SHOWN=off` on every
  event of all 15 switched runs.
- **The summariser counted `mrw write --help` as a landed native write** (its help text holds the word
  `applied`). The table below is recounted without those rows; the summariser's own lines follow it.
- **The one refusal of the pilot was in the control arm**: task 2, run 1, a `--grep` pattern the shell
  left with a doubled backslash, a usage error, followed by a landing.

## Recounted, per run

`!` marks a refused write. `host Write` counts calls, all outside `pkg/`.

| Arm | Task | Run | Digest | mrw writes | host Write | note in log |
|---|---|---|---|---|---|---|
| switched | 1 | 1 | digest-ok | native | 1 | yes |
| switched | 1 | 2 | digest-ok | native | 1 | yes |
| switched | 1 | 3 | digest-ok | native | 0 | yes |
| switched | 2 | 1 | digest-ok | computed | 0 | cut |
| switched | 2 | 2 | digest-ok | computed | 0 | cut |
| switched | 2 | 3 | digest-ok | computed | 0 | cut |
| switched | 3 | 1 | digest-ok | block | 1 | yes |
| switched | 3 | 2 | digest-ok | block | 1 | yes |
| switched | 3 | 3 | digest-ok | block | 2 | yes |
| switched | 4 | 1 | digest-ok | block | 0 | yes |
| switched | 4 | 2 | digest-ok | block | 1 | cut |
| switched | 4 | 3 | digest-ok | block | 0 | cut |
| switched | 5 | 1 | digest-ok | native, block | 2 | yes |
| switched | 5 | 2 | digest-ok | native, block | 2 | yes |
| switched | 5 | 3 | digest-ok | native, block | 2 | yes |
| control | 1 | 1 | digest-ok | native | 0 | n/a |
| control | 1 | 2 | digest-ok | native | 1 | n/a |
| control | 1 | 3 | digest-ok | native | 1 | n/a |
| control | 2 | 1 | digest-ok | computed!, computed | 0 | n/a |
| control | 2 | 2 | digest-ok | computed | 0 | n/a |
| control | 2 | 3 | digest-ok | native | 1 | n/a |
| control | 3 | 1 | digest-ok | native | 1 | n/a |
| control | 3 | 2 | digest-ok | native | 1 | n/a |
| control | 3 | 3 | digest-ok | native | 0 | n/a |
| control | 4 | 1 | digest-ok | native | 0 | n/a |
| control | 4 | 2 | digest-ok | native | 1 | n/a |
| control | 4 | 3 | digest-ok | native | 0 | n/a |
| control | 5 | 1 | digest-ok | native, native | 2 | n/a |
| control | 5 | 2 | digest-ok | native, native | 2 | n/a |
| control | 5 | 3 | digest-ok | native, native | 0 | n/a |

## The summariser's own lines, as printed

Kept as they were written, defects included (`void=` and the `--help` rows).

```
switched task=1 run=1 digest-ok secs=14 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
switched task=1 run=2 digest-ok secs=13 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
switched task=1 run=3 digest-ok secs=11 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
switched task=2 run=1 digest-ok secs=16 writes=2 forms=native,computed landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=1 void=no-switched-read
switched task=2 run=2 digest-ok secs=12 writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no-switched-read
switched task=2 run=3 digest-ok secs=17 writes=2 forms=native,computed landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=1 void=no-switched-read
switched task=3 run=1 digest-ok secs=29 writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=host-Write
switched task=3 run=2 digest-ok secs=22 writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=host-Write
switched task=3 run=3 digest-ok secs=28 writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=host-Write,host-Write
switched task=4 run=1 digest-ok secs=29 writes=3 forms=native,native,block landed=3 refused=0 kinds=landed,landed,landed three-in-a-row=no hand-landings=3 void=no
switched task=4 run=2 digest-ok secs=23 writes=1 forms=block landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write,no-switched-read
switched task=4 run=3 digest-ok secs=22 writes=2 forms=native,block landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no-switched-read
switched task=5 run=1 digest-ok secs=25 writes=3 forms=native,native,block landed=3 refused=0 kinds=landed,landed,landed three-in-a-row=no hand-landings=3 void=host-Write,host-Write
switched task=5 run=2 digest-ok secs=25 writes=3 forms=native,native,block landed=3 refused=0 kinds=landed,landed,landed three-in-a-row=no hand-landings=3 void=host-Write,host-Write
switched task=5 run=3 digest-ok secs=66 writes=3 forms=native,native,block landed=3 refused=0 kinds=landed,landed,landed three-in-a-row=no hand-landings=3 void=host-Write,host-Write
control task=1 run=1 digest-ok secs=12 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=1 run=2 digest-ok secs=13 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=1 run=3 digest-ok secs=13 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=2 run=1 digest-ok secs=21 writes=2 forms=computed,computed landed=1 refused=1 kinds=other,landed three-in-a-row=no hand-landings=0 void=no
control task=2 run=2 digest-ok secs=15 writes=1 forms=computed landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=0 void=no
control task=2 run=3 digest-ok secs=18 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=3 run=1 digest-ok secs=12 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=3 run=2 digest-ok secs=14 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=3 run=3 digest-ok secs=16 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=4 run=1 digest-ok secs=16 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=4 run=2 digest-ok secs=16 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=host-Write
control task=4 run=3 digest-ok secs=12 writes=1 forms=native landed=1 refused=0 kinds=landed three-in-a-row=no hand-landings=1 void=no
control task=5 run=1 digest-ok secs=19 writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=host-Write,host-Write
control task=5 run=2 digest-ok secs=17 writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=host-Write,host-Write
control task=5 run=3 digest-ok secs=21 writes=2 forms=native,native landed=2 refused=0 kinds=landed,landed three-in-a-row=no hand-landings=2 void=no
```

## What this does and does not support

- It supports: with `MRW_SHOWN=off`, one model in one host did five ordinary edits, three times each,
  with no refused write, by taking the read's note at its word.
- It does not support: that a model recovers from the refusals (none occurred); anything about a script
  that hides mrw's output from the model, where the note would not arrive either; any other model, host or
  version; that no read text reached the model, which the pilot never measured.
- The protocol's two void rules and the summariser's `applied` match are defects found by running it.
  They are left as frozen in `README.md` and `summarise.py`, and named here.
