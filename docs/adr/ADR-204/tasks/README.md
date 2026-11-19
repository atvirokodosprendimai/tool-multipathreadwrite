# ADR-204 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | on unix, a landed write of edits and creates of regular files keeps what it replaced, for that one write; `mrw undo` puts it back or refuses before anything moves | none | blocked |

Execution order: T1. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
