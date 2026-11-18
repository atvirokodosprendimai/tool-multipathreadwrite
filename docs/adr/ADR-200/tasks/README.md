# ADR-200 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | apply marks numbered hunks sent to a file whose ledger record is whole and written-shaped, a landing is counted once in `Land`, and `mrw stats` prints the count beside its bar | none | partial |

Execution order: T1. One pull request.

T1 is `partial` and not `done` on purpose: its code, receipts and mutants are complete on the branch, and the record is still Proposed. It becomes `done` when Zy accepts the record.

Status: `pending` | `partial` | `blocked` | `done`.
