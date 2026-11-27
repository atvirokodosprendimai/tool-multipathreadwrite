# ADR-214 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | a write keeps a marked file's mark at byte zero (the apply join), takes it off a line that moved, and a read names it in one `-- note:` line | none | done |
| T2 | the words, and an executed probe of the built binary: contract §380 and the fence's own probe | T1 | done |

Execution order: T1, T2. One pull request each; Zy accepted option B on 2026-11-21.

Status: `pending` | `partial` | `blocked` | `done`.
