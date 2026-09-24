# ADR-053 Tasks

Implementation tasks for ADR-053: Contract does not grep README for tutorial phrases. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Pin and drop README prose greps; keep AckRule and §75 | done | — | `docs/adr/ADR-053-contract-does-not-grep-readme-prose/tasks/T1-drop-readme-prose-greps.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

None.

## Notes

- Branch from `origin/main`. Do not commit. Do not PR.
- Engine go/no-go: do not touch `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`.
