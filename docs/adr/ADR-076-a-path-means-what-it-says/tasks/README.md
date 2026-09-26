# ADR-076 Tasks

Implementation tasks for ADR-076: a path means what it says, and the receipt names every path a write touched. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | A path means what it says: separators, a missing root, device names; contract §151 | done | — | `docs/adr/ADR-076-a-path-means-what-it-says/tasks/T1-a-path-means-what-it-says.md` fence |
| T2 | The receipt names every path a write touched; an empty file's newline; contract §152 | done | — | `docs/adr/ADR-076-a-path-means-what-it-says/tasks/T2-the-receipt-names-what-changed.md` fence |
| T3 | A read-only file is refused; Windows attributes kept; the docs; contract §153 | done | — | `docs/adr/ADR-076-a-path-means-what-it-says/tasks/T3-read-only-refused-attrs-kept.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-076 owns `internal/apply` and `internal/read/read.go`. `internal/plan`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/seen` and `internal/subproc` stay byte-identical.
