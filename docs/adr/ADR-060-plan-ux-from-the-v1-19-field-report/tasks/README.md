# ADR-060 tasks

Execution order: T1 → T2 → T3 → T4 → T5.

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 leftover `body=` + dry-run parsed | none |
| 2 | T2 read neighbour hint | none |
| 3 | T3 unquoted `anchor=` with `"` refused | none |
| 4 | T4 `body=@path` | T1 |
| 5 | T5 check last-error line; teach | T1, T2, T3, T4 |

| Task | Scope | Status |
|------|-------|--------|
| T1 | leftover names declared vs extra; `--dry-run` prints `parsed:`; §100 | done |
| T2 | `read` neighbour hint; §101 | done |
| T3 | unquoted `anchor=` containing `"` refused; §102 | done |
| T4 | `body=@path`; §103 | done |
| T5 | `check last:` above log path; teach; BACKLOG; §104 | done |

Status: `pending` | `partial` | `blocked` | `done`.

Contract rows are cited in Ordered Steps and Acceptance, not in the Tests table
(the first-red hasher takes only `func TestX` rows).

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | leftover extra count | T4 may mention `body=@path` in the same message | T1 before T4 |
| T4 | `LoadBodyFiles` | CLI/MCP in T4 | — |
| none | T2, T3, T5 | — | independent of each other; T5 teach last |

## Notes

- Engine go/no-go: this record owns `internal/plan` and `internal/read`. `internal/apply`, `internal/seen`, `internal/check`, `internal/state` stay byte-identical against the merge base. `go.mod` declares exactly one `require`.
- T1 and T3 both edit `parseHeader` / Parse; sequential on purpose.
