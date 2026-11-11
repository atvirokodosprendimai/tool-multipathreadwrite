# ADR-196 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | require urfave/cli v3.14.0, remove the guards that modelled v3.11.0, and pin the parser's promise with a property test | none | done |
| T2 | the text stops teaching the `--` escape for a padded path | T1 | done |
| T3 | on Windows, a plan file or a `--files-from` list spelled as an alias is refused (Decision 6, the Codex review of #628) | T1 | done |
| T4 | on Windows, a plan file or a `--files-from` list named as a device or an NTFS stream is refused (Decision 7, the Windows field test of v1.86.0) | T3 | done |
| T5 | past a `\\.\` or `\\?\` prefix only a drive, a share, a volume or a pipe is accepted, and a pipe is opened once (Decision 8, the Windows re-probe of v1.86.1) | T4 | done |
| T6 | past a device prefix a drive, volume, share or pipe names a file only with a component after it (Decision 9, the Windows field test of v1.86.3) | T5 | done |
| T7 | a plan or list refusal names the reason that held: an unreadable identity, inside the state directory, or nothing past the prefix (Decision 10, the Windows field test of v1.86.4) | T6 | done |

Execution order: T1–T7. T1–T3 in one pull request; T4, T5, T6 and T7 each in their own.

Status: `pending` | `partial` | `blocked` | `done`.
