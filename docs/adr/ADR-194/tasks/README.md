# ADR-194 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | replay the Stage 8 transcripts and price the narrow licence against the pre-registered bar | none | done |
| T2 | a write records the caller's carried licence plus the lines it wrote | T1 | blocked |

Execution order: T1, then T2 on a GO verdict. Two pull requests.

T2 is blocked: T1 said INCONCLUSIVE (W=238 R=0 U=137), and a re-run placing `--terse` hunks from plan text cannot reach GO on this corpus (at least 52 of the U are in attempts with no `--terse` write, against a GO bound of 11). Zy, 2026-11-04: stop at T1. The whole-file licence of ADR-005 §4 stays in force until a new pre-registration says otherwise.

Status: `pending` | `partial` | `blocked` | `done`.
