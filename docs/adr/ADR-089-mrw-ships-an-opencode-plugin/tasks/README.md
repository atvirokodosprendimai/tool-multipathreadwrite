# ADR-089 Tasks

Implementation tasks for ADR-089: mrw ships an opencode plugin. See the parent ADR for the decision.

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
| T1 | The plugin works, and CI proves it | done | — | `docs/adr/ADR-089-mrw-ships-an-opencode-plugin/tasks/T1-the-plugin-works.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No Go package changes; every engine directory stays byte-identical.
