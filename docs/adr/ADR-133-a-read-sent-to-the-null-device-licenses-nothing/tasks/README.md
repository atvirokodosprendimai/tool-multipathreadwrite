# ADR-133 Tasks

Implementation tasks for ADR-133: a read sent to the null device licenses nothing. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a CLI read whose stdout is the null device records nothing and says so | done | none | `docs/adr/ADR-133-a-read-sent-to-the-null-device-licenses-nothing/tasks/T1-null.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches no engine package: `cmd/mrw` only.
- Contract section §234.
