# ADR-057 tasks

Execution order: T1 → T2 → T3.

| Task | Scope | Status |
|------|-------|--------|
| T1 | parse `unlink`/`rename`; apply stage/restore; `seen.Drop` | done |
| T2 | `apply_patch` Delete File / Move to compile | done |
| T3 | contract §98–§99; teach; BACKLOG | done |

Status: `pending` | `partial` | `blocked` | `done`.

Contract rows are cited in Ordered Steps and Acceptance, not in the Tests table
(the first-red hasher takes only `func TestX` rows).
