# ADR-156 Tasks

Implementation tasks for ADR-156: whether mrw pays for itself is measured on agent runs, against a bar written first. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Why together |
|------|-------|--------------|
| 1 | T1 | the corpus and its oracles, hermetic |
| 2 | T2 | the runner and the isolation canary consume T1 |
| 3 | T3 | Stage 1, Sonnet — on Zy's word |
| 4 | T4, T5 | Stage 2 (Opus) and Stage 3 (SWE-bench) — each on Zy's word, run one at a time |
| 5 | T6 | analysis and README |

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the task corpus and an oracle per task, proved without an agent | done | none | `T1-task-corpus-and-oracles.md` fence |
| T2 | the runner, its receipts, configuration B's hook and the isolation canary | done | T1 | `T2-runner-and-isolation-canary.md` fence |
| T3 | Stage 1: our own tasks on Sonnet 5.5 | pending | T2 | `T3-stage-1-sonnet.md` fence |
| T4 | Stage 2: our own tasks on Opus 5.5 | pending | T3 | `T4-stage-2-opus.md` fence |
| T5 | Stage 3: the SWE-bench Verified slice | partial | T2 | `T5-stage-3-swe-bench-slice.md` fence |
| T6 | analysis against the registered bar, and README | partial | T3, T4, T5 | `T6-analysis-and-readme.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- T3, T4 and T5 each start only on Zy's word, recorded in their Verification Log as a human-observed line.
- No contract row: nothing on mrw's served path changes.
- T2's S5 passed on 2026-10-14 at 968ae417, Zy's own account with `bypassPermissions` and the one-time login in the dedicated config directory: the receipt is `docs/bench/canary-2026-10-14/canary.json`. Stage 1 (T3) now waits only for Zy's word.
- T5's selection (S1) and T6's analysis (S1, S2) were built with T2, before any run, so the analysis is fixed before there is data to fit it to; both stay partial until their stages run.
- The uncounted Stage 1 pilot (2026-10-14, 5 tasks × 2 configurations on Sonnet) found two harness defects, both fixed before any counted run on Zy's word: instruction files are redacted of mrw instead of stripped (T1 S5), and the hook passes a write outside the tree (T2 S6). The canary and the pilot run again after this change; the registered bar is untouched.
- The pilot's third harness gap, decided by Zy on 2026-10-14 ("Keep, add node, declare it"): the attempt's `PATH` had no node, so quality-harness's declared check stopped at once; `node`, `npm`, `npx` and `python3` are now linked (T2 S7). That check is red at the pinned commit even so, measured, so its seven tasks carry `check_red_at_base` (T1 S6) and `bench analyze` prints a descriptive row without them beside each verdict (T6 S5). The registered bar is untouched.
