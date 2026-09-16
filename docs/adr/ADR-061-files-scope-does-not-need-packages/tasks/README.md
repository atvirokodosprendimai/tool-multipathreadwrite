# ADR-061 Tasks

Implementation tasks for ADR-061: `{files}` still scopes when `packages()` cannot map. See the parent ADR for the decision.

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
| T1 | `{files}`-only scoped_check runs when packages() is empty; contract §113 | done | — | `docs/adr/ADR-061-files-scope-does-not-need-packages/tasks/T1-files-scope-when-unmapped.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | `{files}`-only scopes when unmapped | — | sole task |

## Notes

- Branch from `origin/main`.
- Engine go/no-go: this record owns `internal/check.command`. `packages()` stays byte-identical.
- Do not put §113 in the Tests table (ADR-054 first-red lock: a contract section is UNPROVEN forever).
