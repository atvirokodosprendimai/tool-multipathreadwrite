# ADR-056 tasks

Execution order: T1 → T2 → T3. T1 and T2 are independent in code; T3 teaches both.

| Task | Scope | Status |
|------|-------|--------|
| T1 | `pattern` on both JSON receipts; MCP schema; contract §95 | done |
| T2 | `Result.StrictWouldRefuse`; the `pricing` counters; stats block; contract §96 | done |
| T3 | Teach; correct the BACKLOG pre-registration's data source | done |

Status: `pending` | `partial` | `blocked` | `done`.

Contract rows are cited in Ordered Steps and Acceptance, not in the Tests table
(the first-red hasher takes only `func TestX` rows). Every assertion a mutant
could need is written BEFORE the first red row — ADR-055 T2 is `blocked` on the
word for adding one afterwards.
