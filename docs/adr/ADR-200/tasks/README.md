# ADR-200 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | apply marks numbered hunks sent to a file whose ledger record is whole and written-shaped, a landing is counted once in `Land`, and `mrw stats` prints the count beside its bar | none | done |
| T2 | rule 2 of the core, recipe 7 and a new recipe 21 say that a written file's line numbers have moved, and show the edit that needs no number | T1 | done |

Execution order: T1, then T2. A pull request each.

T1 was held at `partial` while the record was Proposed, with its code, receipts and mutants complete on the branch; Zy accepted the record on 2026-11-18 and it is `done`.

Status: `pending` | `partial` | `blocked` | `done`.
