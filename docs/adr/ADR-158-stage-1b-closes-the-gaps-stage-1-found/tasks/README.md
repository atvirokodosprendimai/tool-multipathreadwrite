# ADR-158 tasks

| # | Tasks | Gate |
|---|-------|------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2, and Zy's word |

| ID | Goal | Status | Depends-on | Acceptance |
|----|------|--------|------------|------------|
| T1 | The host's shell, git and Python setup stay out, and declared sections are dropped | done | none | `T1-isolation-and-section-drop.md` fence |
| T2 | The corpus revised, ten varied multi-site tasks, and a verdict per stage | done | T1 | `T2-corpus-and-per-stage-verdicts.md` fence |
| T3 | Stage 1b on Sonnet 5.5, a pilot first | done | T2 | `T3-stage-1b-sonnet.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No contract row: nothing on mrw's served path changes.
- The registration is in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-158", committed with this record and before any Stage 1b run.
- Stage 1b ran on 2026-10-15 on this Mac, bench at edb65629, CLI 2.1.300, mrw v1.66.0 (05ec3f8): canary passed; the pilot (5 tasks × 2 × 1, `pilot/`) solved 10/10 and its transcripts were read; then 160 attempts, 0 excluded, receipts in `docs/bench/stage-1b-sonnet/`. `bench analyze` as registered: **INCONCLUSIVE**, cost (B−A)/A −0.004 [−0.050, +0.042], solved B−A −0.013 [−0.037, +0.000], silent mis-edits A 0 / B 0, B shell reads 59, slipped writes 0. Descriptive, not the verdict: the ten `varied-multi-site` tasks, cost −0.075 [−0.161, +0.010], solved equal (40/40); without the tasks red at base, cost +0.019 [−0.029, +0.068]. B's hook refused 4 commands over 3 attempts (`sed -i` twice, a `python3` write, a command substitution), each the registered rule; every one of those attempts went on to solve.
- **A defect in a Stage 1 task's reference fix, found by reading the unsolved attempts (2026-10-15).** The only unsolved attempts of Stage 1b are three of `cac-rename-post-send`'s (A 1 of 4, B 2 of 4), and they are the whole of the solved difference. The package holds two things named `postSend`: the handler method `(h *handler) postSend` in `send.go`, which the instruction asks to rename, and an unrelated package-level test helper `func postSend(t, url, body)` in `send_test.go`. The reference fix renames both; each unsolved attempt renamed the method and its references, left the helper, and said why — a separate function, not a reference to the method. That reading is defensible, and the oracle refuses it. Stage 1 graded the same task A 2/2, B 0/2. Under the registration a task is superseded after the pilot only for a defect in its base or its reference fix, and never for what an attempt did; this defect is in the reference fix but was found through attempts, after the stage ran. The verdict is published as registered. Zy chose, the same day, to supersede the task and print a descriptive row without it: ADR-159.
