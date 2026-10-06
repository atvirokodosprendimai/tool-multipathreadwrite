# ADR-126 Tasks

Implementation tasks for ADR-126: a check stopped before it started is a timeout, exit 3. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a check stopped before it started by its deadline exits 3 | done | none | `docs/adr/ADR-126-a-check-stopped-before-it-started-is-a-timeout/tasks/T1-timeout.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-126 owns `internal/check`.
- No contract section: see the record's Wiring.
