# ADR-201 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | the seven state reads open without blocking, each with its decided outcome; a failed read is not saved over; a lock error names its file; the built binary is not held by a FIFO at a marker | none | done |

Execution order: T1. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
