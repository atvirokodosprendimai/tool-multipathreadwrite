# ADR-175 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `mrw seen --forget` empties this checkout's ledger, never a newer mrw's | none | done |
| T2 | `docs/agents.md` gives the compaction hook and the re-read rule; README and AGENTS.md name the flag | T1 | partial |

Execution order: T1, T2. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
