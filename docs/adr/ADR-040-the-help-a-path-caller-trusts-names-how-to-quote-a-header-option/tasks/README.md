# ADR-040 Tasks

Implementation tasks for ADR-040: The help a PATH caller trusts names how to quote a header option. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated. `adr-lint` fails when the README lists a task with no file or omits
an existing task file; wave/order drift against Depends-on + Consumes edges is caught by
`adr-lint` (cycles too — the wave table must be a valid topological leveling of the task
DAG); Covers-column drift is caught at review. Regenerate rather than hand-edit.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | none |

T1 is the floor and the first observable slice. T2, T3 and T4 are independent of T1 and of
each other. *"good, accepted all"* armed every fork.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `write --help` and `CLI()` name how to quote a header option | done | — | `grep` §79 + `go test` of the three new/extended tests |
| T2 | Unquoted `anchor=` consumes until the next `key=` | done | — | `go test` of the parse fixture + §80 |
| T3 | `mrw version` prints `versionString()` | done | — | `go test` of the subcommand + AGENTS.md `#73` gate |
| T4 | Single-quoted `anchor=` parses | done | — | `go test` of the single-quote fixture + §80 |

Status: `pending` | `partial` | `blocked` | `done`.

- `pending` — not started, or started and carrying no evidence yet.
- `partial` — genuinely part-done: some of the work has landed and some has not.
- `blocked` — waiting on something outside this repository.
- `done` — finished, with tool-written acceptance and mutation evidence to match.

## Contract Coupling

None.

## Notes

- *"good, accepted all"* (2026-09-12) Accepts every fork: teach, parse, version, single quotes, both global `-C` and `--root`.
- *"I NAME THEM ALL"* ranked the inventory; it did not Accept.
- `# 79.` `# 80.` `# 81.` are the contract sections this record reserved.
