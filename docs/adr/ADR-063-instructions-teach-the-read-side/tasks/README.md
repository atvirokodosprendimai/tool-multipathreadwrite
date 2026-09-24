# ADR-063 Tasks

Implementation tasks for ADR-063: `mrw instructions` teaches the read side. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | Read section on CLI(); every read flag named; help summaries; skill description; §115 | done | — | `docs/adr/ADR-063-instructions-teach-the-read-side/tasks/T1-read-side-on-cli.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | read section on CLI(); every read flag named | — | sole task |

## Notes

- Branch `adr-063-instructions-teach-the-read-side`.
- Engine go/no-go: this record owns `internal/guide`, three `Usage` strings in `cmd/mrw`, and the skill description. `internal/mcp` and the engine packages stay byte-identical; the fence asserts it.
- Do not put §115 in the Tests table (ADR-054 first-red lock: a contract section is UNPROVEN forever).
