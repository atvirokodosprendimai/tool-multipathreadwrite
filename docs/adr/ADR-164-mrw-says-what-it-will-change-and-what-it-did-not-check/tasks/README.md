# ADR-164 Tasks

Implementation tasks for ADR-164: mrw says what it will change, what a guard does not cover, and what its check left out. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T5 | -v and -o on read get a pointer | done | none | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T5-an-inverted-grep-says-there-is-none.md` fence |
| T1 | a refusal says sha= does not stand in for anchor= | done | none | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T1-sha-is-not-an-anchor.md` fence |
| T6 | a refusal names its real cause | pending | T1 | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T6-a-refusal-names-its-real-cause.md` fence |
| T4 | a check that places some files runs those and names the rest | pending | none | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T4-partial-map-runs-scoped.md` fence |
| T2 | a plan maker's dry run shows each change | pending | T1, T4 | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T2-dry-run-shows-each-change.md` fence |
| T3 | the state-directory hint reaches every surface | pending | T2 | `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T3-mrw-mcp-names-its-state-directory.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Execution order is the table's. T2 edits `internal/apply` after T1 and `internal/mcp` after T4; T3 edits `internal/mcp` after T2.
- The record merges after PR #460 (its BACKLOG section, `cmd/mrw/statehint.go`).
- Engine packages `internal/apply` (T1, T2), `internal/check` (T4), `internal/read` (T3) and `internal/plan` (T6) change; this record owns each change.
- Contract sections §273 (T1), §274 (T2), §275 (T3), §276 (T4), §277 (T5), §278 (T6); §272 is PR #460's.
- A planned test is named without backticks until it exists: `internal/adversarial/record_test.go` checks every backticked name in a Tests table against its file. The executor moves it into the table, backticked, when it is written.
