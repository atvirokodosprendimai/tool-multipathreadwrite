# ADR-213 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | on Linux and macOS a no-replace move is one `renameat2(RENAME_NOREPLACE)` / `renameatx_np(RENAME_EXCL)` through the root's directories, falling back to #744's path where the filesystem cannot; `golang.org/x/sys` added | none | pending |

Execution order: T1. One pull request, after Zy accepts the record.

Status: `pending` | `partial` | `blocked` | `done`.
