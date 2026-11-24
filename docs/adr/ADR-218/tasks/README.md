# ADR-218 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `MRW_SHOWN=off` makes `mrw read` and `mrw hook read` record a licence with no showing; nothing is detected; contract §385 | none | done |
| T2 | under the switch `mrw write` refuses a number alone on any unshown line and its refusals name remedies that need no read; the instructions and `docs/agents.md` say it; contract §386; the pilot | T1 | blocked |

Execution order: T1, T2. T2 is built, gated and reviewed; it is `blocked` on its S5: the pilot ended with every digest right and no refusal, and missed its bar by the letter of two void rules (`docs/break/code-mode-790/pilot/results.md`). The owner decides whether it releases.

Status: `pending` | `partial` | `blocked` | `done`.
