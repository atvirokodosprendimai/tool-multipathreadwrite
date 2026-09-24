# ADR-064 Tasks

Implementation tasks for ADR-064: both finders apply ADR-007's exclusion rule, and a taught pipeline runs as an agent runs it. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | ast-grep applies ADR-007 in both halves; contract §116; campaign probe | done | — | `docs/adr/ADR-064-a-named-path-is-served-by-every-finder/tasks/T1-astgrep-serves-a-named-file.md` fence |
| T2 | the taught pipeline runs under an agent-shaped stdin; contract §117 | done | — | `docs/adr/ADR-064-a-named-path-is-served-by-every-finder/tasks/T2-the-taught-pipeline-runs-under-an-open-stdin.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | both finders apply ADR-007's rule | — | independent of T2 |
| T2 | taught pipeline executed under open stdin | — | independent of T1; its fence guards only the paths neither task owns |

## Notes

- No `§NN` row in any Tests table. The lock hasher can't hash a contract section, so such a row freezes the task UNPROVEN (found on ADR-052/054, 2026-09-24).
- Engine go/no-go:
  - T1 owns `internal/read/astgrep.go` and its tests.
  - `internal/read/walk.go`, `internal/guide`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` and `internal/mcp` stay byte-identical, and both fences assert it.
