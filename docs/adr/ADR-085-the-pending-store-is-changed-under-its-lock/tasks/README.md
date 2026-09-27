# ADR-085 Tasks

Implementation tasks for ADR-085: the MCP pending store is changed under its lock. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | hold and promote take the pending lock | done | — | `docs/adr/ADR-085-the-pending-store-is-changed-under-its-lock/tasks/T1-hold-and-promote-take-the-lock.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-085 owns `internal/mcp` only. `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` and `internal/subproc` stay byte-identical.
