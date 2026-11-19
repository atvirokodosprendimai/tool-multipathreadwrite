# ADR-204 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | every landed write keeps what it replaced, in the checkout's state, within the limits | none | pending |
| T2 | `mrw undo` restores the newest kept write whole or not at all, with `--dry-run`, `--list`, `--forget` and `--json` | T1 | pending |

Execution order: T1, then T2. One pull request each.

Status: `pending` | `partial` | `blocked` | `done`.
