# ADR-204 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | a landed write keeps what it replaced, for that one write, in the checkout's state; the previous copy goes first | none | blocked |
| T2 | `mrw undo` puts back the last write or refuses before anything moves, with `--dry-run`, `--forget` and `--json` | T1 | blocked |

Execution order: T1, then T2, in one pull request.

Status: `pending` | `partial` | `blocked` | `done`.
