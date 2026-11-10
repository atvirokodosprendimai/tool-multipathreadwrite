# ADR-196 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | require urfave/cli v3.14.0, remove the guards that modelled v3.11.0, and pin the parser's promise with a property test | none | done |
| T2 | the text stops teaching the `--` escape for a padded path | T1 | done |
| T3 | on Windows, a plan file or a `--files-from` list spelled as an alias is refused (Decision 6, the Codex review of #628) | T1 | done |
| T4 | on Windows, a plan file or a `--files-from` list named as a device or an NTFS stream is refused (Decision 7, the Windows field test of v1.86.0) | T3 | done |

Execution order: T1, T2, T3, T4. T1–T3 in one pull request; T4 in its own.

Status: `pending` | `partial` | `blocked` | `done`.
