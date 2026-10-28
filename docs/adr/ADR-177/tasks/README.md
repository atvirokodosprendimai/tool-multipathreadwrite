# ADR-177 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `mrw run 'CMD'` keeps the output in a log and prints one receipt | none | done |
| T2 | the instructions teach `mrw run`, its exit codes and its trust | T1 | done |

Execution order: T1, T2. One pull request.

**Branch: its own, from `main`.** T1 changes `internal/check` and holds `internal/read` byte-identical; ADR-178 and ADR-179 change `internal/read` and hold `internal/check` byte-identical. On one branch every fence of one record would fail on the other's work, so ADR-177 ships alone and ADR-178 with ADR-179 share another.

Status: `pending` | `partial` | `blocked` | `done`.
