# ADR-114 Tasks

Implementation tasks for ADR-114: a file is edited and renamed in one plan. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the engine edits and renames a file in one plan | done | none | `docs/adr/ADR-114-a-file-is-edited-and-renamed-in-one-plan/tasks/T1-engine.md` fence |
| T2 | apply_patch compiles Move to with hunks | done | T1 | `docs/adr/ADR-114-a-file-is-edited-and-renamed-in-one-plan/tasks/T2-apply-patch.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-114 owns `internal/apply`; every other engine package stays byte-identical.
- Contract section §214 (T2). One pull request: fence-prose checks pending task fences too.
