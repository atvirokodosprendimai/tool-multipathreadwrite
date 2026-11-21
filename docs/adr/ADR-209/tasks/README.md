# ADR-209 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | every existing content target is hard-linked aside before the first rename, and a commit failure puts every content file that has an aside back, newest first; a file no aside could be made for is left written and named | none | partial (S4, the measured cost, is Zy's) |

Execution order: T1. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
