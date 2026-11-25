# The ADR-218 pilot, second run: protocol frozen 2026-11-25, before any of its sessions

The first pilot (`../pilot/`, 2026-11-24) missed its bar by the letter, on two void rules and a counting
rule that its own results file names as defects of the protocol. The owner asked for it to be run again
(Zy, 2026-11-25: *"fix red, run pilot"*). This file is the protocol of that second run. It, `hooklog.py`,
`summarise.py` and `run.sh` beside it were committed, and their sha256 filed in memoryd (space
`tool-multipathreadwrite`), before any session of this run started. `results.md`, beside this file, is
written afterwards.

## What is the same

Everything in `../pilot/README.md` that is not named under "What changed" holds here unchanged, and the
runner reads it from that file, which is not edited: the five prompts word for word, `../pilot/fixture.py`
and its base digest, the five task digests, the setup (one trusted scratch folder, the switched arm's
`"env": {"MRW_SHOWN": "off"}`, `Bash(mrw:*)`, a new state home per run, the reset), three runs per task
per arm, 30 in all, serially, one counted attempt each, the 5-minute end, and **the bar's four clauses,
word for word**.

## What changed, and which way each change leans

These were written after the first pilot's data was seen. The first relaxes a rule; that is said here
and not claimed otherwise. The others change how a count is taken, not what is counted.

1. **A host file tool voids a run only when it writes under `pkg/`.** A `PostToolUse` row for `Edit`,
   `Write` or `NotebookEdit` voids the run when its `file_path` (or `notebook_path`), resolved against
   the session's working directory with links followed, is inside the folder's `pkg/`. Before, any such
   call voided the run; the first pilot's 23 `Write` calls all wrote plan files outside `pkg/`, which is
   the form mrw's own instructions give for a plan. **This is a relaxation, made after seeing data.**
   The `Bash` rule for other writers is unchanged, pattern included.
2. **Every match is taken on the whole output, before the logger cuts it.** The logger records, for each
   `PostToolUse` row of `Bash`, whether the output (both streams) holds: the switched read's note
   (`recorded as licensed, not shown`), a landing (below), `NOTHING WRITTEN`, `no recorded showing`,
   `needs a served line after`. The summariser reads those flags. Before, the summariser searched text
   the logger had cut at 4,000 characters, which lost the note in five runs and could as well lose a
   receipt's last line.
3. **A landing is the receipt's own last line, not the word `applied`.** A write landed when its output
   holds a line that begins with a count of hunks and ends `— applied` (`N hunk(s), N file(s), N failed,
   … — applied`), or, under `--json`, `"mutation": "complete"`; and does not hold `NOTHING WRITTEN`.
   Before, the word `applied` anywhere counted, and `mrw write --help` holds it.
4. **`mrw write --help` is not a write.** A row is a write when at least one `mrw write` in its command
   carries neither `--help`, `-h` nor `--dry-run` before the next `;`, `&` or `|`. Before, `--dry-run`
   anywhere in the command excluded the row and `--help` did not.
5. **The host is the version installed on the day,** recorded per run by the runner (`claude --version`),
   not assumed. The first pilot ran 2.1.331.
6. **The build is the branch's committed head,** not a dirty tree; the runner records `mrw version` per run.

## Proving the counter, without rehearsing a task

Before the first counted run: `summarise.py` is run over the first pilot's 30 hook logs, and must give
that pilot's hand recount (`../pilot/results.md`, "Recounted, per run") for forms, landings and refusals
in every run, the cut notes apart; and one uncounted control-arm run of task 1 proves the runner. After
that only the runner's mechanics may change. No task is run in the switched arm before its counted run.

## What a met bar would and would not say

As before: one model, one host version, five ordinary edits. If, as in the first pilot, no switched write
is refused, clauses 2 and 3 hold with nothing to count, and the run says the read's note steers the model
to the block form; it says nothing about recovery from a refusal, and nothing about a script that hides
mrw's output from the model. A missed bar means the switch does not release as built.
