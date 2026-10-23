# ADR-169 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `mrw read --if-changed PATH=SHA` answers an unchanged file without its text | none | done |
| T2 | `mrw_read` takes `if_changed`, and an unchanged answer is not an error | T1 | done |
| T3 | the guard is taught, the scanners know its line, and its adoption bar is registered | T1, T2 | done |

Execution order: T1, T2, T3. The record merges in the same pull request as all three.

Status: `pending` | `partial` | `blocked` | `done`.
