# ADR-215 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | every op refuses a rooted plan path in one preflight, exit 1, naming a root-relative spelling only when it is clear; a header whose first token begins with `/` and has no second `/` keeps exit 2 and gains a sentence | none | done |
| T2 | the instructions say it; contract §381 drives the built binary | T1 | pending |

Execution order: T1, T2. One pull request each; Zy accepted the Decision as recorded on 2026-11-21.

Status: `pending` | `partial` | `blocked` | `done`.
