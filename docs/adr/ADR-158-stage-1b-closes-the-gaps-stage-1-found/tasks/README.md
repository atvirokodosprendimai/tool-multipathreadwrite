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
| T3 | Stage 1b on Sonnet 5.5, a pilot first | pending | T2 | `T3-stage-1b-sonnet.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No contract row: nothing on mrw's served path changes.
- The registration is in `docs/adr/BACKLOG.md`, "Pre-registered for ADR-158", committed with this record and before any Stage 1b run.
