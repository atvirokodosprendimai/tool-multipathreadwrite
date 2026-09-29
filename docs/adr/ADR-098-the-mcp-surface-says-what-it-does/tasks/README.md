# ADR-098 Tasks

Implementation tasks for ADR-098: the MCP surface says what it does, and an ast_grep index pages. See the
parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | An ast_grep index pages with `after`; contract §189 | done | — | `docs/adr/ADR-098-the-mcp-surface-says-what-it-does/tasks/T1-an-ast-grep-index-pages.md` fence |
| T2 | The served text says what the code does; contract §190 | done | — | `docs/adr/ADR-098-the-mcp-surface-says-what-it-does/tasks/T2-the-served-text-says-what-the-code-does.md` fence |
| T3 | README and AGENTS teach the ast_grep index's paging | done | — | `docs/adr/ADR-098-the-mcp-surface-says-what-it-does/tasks/T3-teach-the-ast-grep-page.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `after` with `ast_grep` (`afterCursor`) | T2, T3 | T1 first: T2 describes the behaviour and T3 documents it |

## Notes

- Engine go/no-go: ADR-098 owns two engine COMMENTS (`internal/apply/apply.go:38-41`,
  `internal/read/read.go:589`) and no engine statement. Every engine package stays byte-identical against
  the merge-base except comment lines in those two files; T2's fence checks exactly that.
- Contract sections §189 and §190 were free on 2026-09-29 (`sort -k2,2n` recipe: highest was §188).
