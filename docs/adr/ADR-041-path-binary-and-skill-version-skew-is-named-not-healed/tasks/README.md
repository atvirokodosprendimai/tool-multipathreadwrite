# ADR-041 Tasks

Implementation tasks for ADR-041: PATH binary and skill version skew is named, not healed. See the parent ADR for the decision.

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

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Keep the record. Do not write a version-negotiation protocol. | pending | — | `grep` the Accepted record |

Status: `pending` | `partial` | `blocked` | `done`.

- `pending` — not started, or started and carrying no evidence yet.
- `partial` — genuinely part-done: some of the work has landed and some has not.
- `blocked` — waiting on something outside this repository.
- `done` — finished, with tool-written acceptance and mutation evidence to match.

## Contract Coupling

None.

## Notes

- *"good, accepted all"* records this leftover. It does not mean implement an engine dream.
