# ADR-058 Tasks

Implementation tasks for ADR-058: structural find shells out to ast-grep on the read path.

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `--ast-grep` / `ast_grep` share one primitive | done | F-5, F-11, F-12, F-13, F-15, UC2-S1, UC2-S2, UC2-S3 | 7 named `--- PASS:` lines, gofmt + vet, engine go/no-go |
| T2 | contract §107–§109; BACKLOG shipped; teach | done | — | `# 107.` `# 108.` `# 109.`; BACKLOG shipped line; README/AGENTS |
| T3 | a hanging ast-grep times out | done | — | `TestAHangingAstGrepTimesOut`; `# 111.` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `read.AstGrep` + CLI `--ast-grep` | T2, T3 | T1 before T2 and T3 |
| T2 | contract §107–§109; BACKLOG shipped; teach | — | after T1 |
| T3 | hanging ast-grep times out | — | after T1 |
