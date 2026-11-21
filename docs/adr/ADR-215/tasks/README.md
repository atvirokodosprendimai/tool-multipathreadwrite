# ADR-215 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | every op refuses a rooted plan path at one place, exit 1, naming the root-relative spelling; the header splitter reads the first token as a path | none | pending |
| T2 | the instructions say it; contract §381 drives the built binary | T1 | pending |

Execution order: T1, T2. One pull request each; T1 waits for the owner's choice in the record's Follow-ups.

Status: `pending` | `partial` | `blocked` | `done`.
