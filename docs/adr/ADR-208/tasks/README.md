# ADR-208 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | a multi-line replace on a whole written-shaped record needs the line after shown since the write, plan makers excepted | none | done |
| T2 | (Proposed) a `search_replace`-only write is recorded written-shaped, and on any written-shaped record a multi-line replace needs the line after shown at the current sha; no format change | T1 | pending |

Execution order: T1, then T2. One pull request each.

Status: `pending` | `partial` | `blocked` | `done`.
