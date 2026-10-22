# ADR-168 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `mrw skill` and `mrw --skill` print the skill | none | done |
| T2 | `mrw skill --dir DIR` installs through the write engine, and replaces only its own | T1 | done |
| T3 | onboarding names the skill | T1, T2 | done |
| T4 | the binary's skill is the one skill | T1, T2 | done |
| T5 | what allowing mrw allows is said to the caller and the agent | T1, T4 | pending |

Execution order: T1, T2, then T3 and T4, then T5. The record merges in the same pull request as all five (ADR-168, Implementation).

Status: `pending` | `partial` | `blocked` | `done`.
