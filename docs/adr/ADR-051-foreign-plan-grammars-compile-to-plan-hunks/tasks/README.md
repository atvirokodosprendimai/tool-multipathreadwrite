# ADR-051 Tasks

Implementation tasks for ADR-051: Foreign plan grammars compile to plan hunks. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated. `adr-lint` fails when the README lists a task with no file or omits
an existing task file; wave/order drift against Depends-on + Consumes edges is caught by
`adr-lint` (cycles too — the wave table must be a valid topological leveling of the task
DAG); Covers-column drift is caught at review. Regenerate rather than hand-edit.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Compile apply_patch to plan hunks; an unread sibling writes nothing | done | — | `go test ./internal/ingest/` |
| T2 | `--format=apply_patch` on write; contract §82 | done | — | `go test ./cmd/mrw/` + `grep '^# 82\. '` |

Status: `pending` | `partial` | `blocked` | `done`.

- `pending` — not started, or started and carrying no evidence yet.
- `partial` — genuinely part-done: some of the work has landed and some has not. It is a
  status with OBLIGATIONS, not a softer `pending`: everything its landed evidence claims is
  checked exactly as hard as for a `done` task, so a partial task with a passing Acceptance
  fence still owes a killed mutant. What it does not owe is a `done` row's exit-0 evidence,
  because it claims no completion.
- `blocked` — waiting on something outside this repository. Say what, in the task's
  `**Blocked-on:**` header, as an event a later reader can check HAS HAPPENED — ideally a
  command that exits 0 once it has, e.g. `git merge-base --is-ancestor <sha> master`. Where
  only someone with other access can confirm it, say who.
- `done` — finished, with tool-written acceptance and mutation evidence to match.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `ingest.CompileApplyPatch` (T1) | T2 | T1 before T2 |

## Notes

- First slice is `apply_patch` only. Aider SEARCH/REPLACE, MCP `format`, Delete File / Move, and ast-grep stay in BACKLOG.
- 4096 stays. 019 A stands. Do not stream. Do not parse target syntax.
