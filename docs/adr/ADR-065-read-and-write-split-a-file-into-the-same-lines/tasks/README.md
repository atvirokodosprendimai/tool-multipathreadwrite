# ADR-065 Tasks

Implementation tasks for ADR-065: read, write and the plan compilers split a file into the same lines. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | `internal/lines`; apply, read, `--grep` and MCP paging number lines alike; contract §120 | done | — | `docs/adr/ADR-065-read-and-write-split-a-file-into-the-same-lines/tasks/T1-one-splitter-for-read-and-write.md` fence |
| T2 | apply_patch and search_replace compile against the same lines; contract §121 | done | — | `docs/adr/ADR-065-read-and-write-split-a-file-into-the-same-lines/tasks/T2-the-plan-compilers-use-the-same-lines.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | `lines.Split` | T2 | T2 calls the package T1 creates |

## Notes

- No `§NN` row in any Tests table (ADR-052/054 lesson).
- Engine go/no-go: T1 owns `internal/lines`, `internal/apply` (`readLines` only), `internal/read` and `internal/mcp`; T2 owns `internal/ingest`. `internal/plan`, `internal/seen`, `internal/check`, `internal/state` and `internal/guide` stay byte-identical, and both fences assert it.
