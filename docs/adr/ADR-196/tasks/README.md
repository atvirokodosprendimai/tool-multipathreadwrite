# ADR-196 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | require urfave/cli v3.14.0, remove the guards that modelled v3.11.0, and pin the parser's promise with a property test | none | done |
| T2 | the text stops teaching the `--` escape for a padded path | T1 | done |

Execution order: T1, T2. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
