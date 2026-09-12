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
| 3 | T3 | T2 |
| 4 | T4 | T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Compile apply_patch to plan hunks; an unread sibling writes nothing | done | F-1, F-2, F-3, F-4, F-5, F-6, F-7, F-8, F-9, F-10, F-11, F-12, F-13, F-14, F-15, F-16, F-17, F-18, UC1-S2, UC2-S1, UC2-S2 | `go test ./internal/ingest/` |
| T2 | `--format=apply_patch` on write; contract §82 | done | F-19, F-20, F-21, F-22, F-23, UC1-S1, UC3-S1, UC3-S2 | `go test ./cmd/mrw/` + `grep '^# 82\. '` |
| T3 | `mrw_write` grows `format`; contract §83 | done | F-24, F-25, F-27, UC4-S1, UC4-S2, UC4-S3 | `go test ./internal/mcp/` + `grep '^# 83\. '` |
| T4 | `--format=search_replace` on write and MCP; contract §84 | done | F-26, UC5-S1, UC5-S2, UC5-S3 | `go test ./internal/ingest/ ./cmd/mrw/ ./internal/mcp/` + `grep '^# 84\. '` |

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
| T1 | `ingest.CompileApplyPatch` (T1) | T2, T3 | T1 before T2 and T3 |
| T2 | `write --format=apply_patch` (T2) | T3 | T2 before T3 |
| T3 | `mrw_write.format` (T3) | T4 | T3 before T4 |

## Notes

- MCP `format` on existing `mrw_write` shipped with F-27. SEARCH/REPLACE is `--format=search_replace` (F-26). Delete File / Move and ast-grep stay in BACKLOG.
- 4096 stays. 019 A stands. Do not stream. Do not parse target syntax.
