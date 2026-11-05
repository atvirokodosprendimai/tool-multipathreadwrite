# ADR-173 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | a read whose only unserved ranges are the caller's own `--max-lines` or `--max-matches` cap exits 0 and says `capped:`; the exit-1 expectations migrate; every place that states the exit says so | none | done |

Execution order: T1. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
