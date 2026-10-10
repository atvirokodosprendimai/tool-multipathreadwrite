# ADR-145 Tasks

Implementation tasks for ADR-145: `version` and `instructions` touch no state. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | The install check migrates nothing | done | none | `docs/adr/ADR-145-version-and-instructions-touch-no-state/tasks/T1-no-migration.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `cmd/mrw`.
- Contract section §249.
