# ADR-073 Tasks

Implementation tasks for ADR-073: a file mrw cannot split into lines is not edited. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | An encoded or non-regular file is not line-edited; contract §146 | done | — | `docs/adr/ADR-073-a-file-mrw-cannot-split-is-not-edited/tasks/T1-an-encoded-file-is-not-line-edited.md` fence |
| T2 | `read` notes an encoded file | done | — | `docs/adr/ADR-073-a-file-mrw-cannot-split-is-not-edited/tasks/T2-read-notes-an-encoded-file.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-073 owns `internal/lines`, `internal/apply` and `internal/read/read.go`. `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/iter`, `internal/rooted` and the rest of `internal/read` stay byte-identical.
