# ADR-178 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `--json-path` prints the selected values and licenses no line | none | done |
| T2 | the instructions teach `--json-path`, and that it licenses nothing | T1 | done |

Execution order: T1, T2. One pull request.

**Branch: shared with ADR-179, apart from ADR-177.** Both this record and ADR-179 change `internal/read` and hold `internal/check` byte-identical; ADR-177 changes `internal/check` and holds `internal/read` byte-identical. ADR-177 ships on its own branch from `main`.

Status: `pending` | `partial` | `blocked` | `done`.
