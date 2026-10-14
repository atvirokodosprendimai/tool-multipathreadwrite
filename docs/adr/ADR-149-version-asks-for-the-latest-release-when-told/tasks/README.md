# ADR-149 Tasks

Implementation tasks for ADR-149: `mrw version --check-latest` asks for the latest release, and nothing else does. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | `mrw version --check-latest`, the one network path | done | none | `docs/adr/ADR-149-version-asks-for-the-latest-release-when-told/tasks/T1-check-latest.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches no engine package: `cmd/mrw` only.
- Contract section §253.
