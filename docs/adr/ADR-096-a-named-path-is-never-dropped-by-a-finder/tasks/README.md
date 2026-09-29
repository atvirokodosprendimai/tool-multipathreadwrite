# ADR-096 Tasks

Implementation tasks for ADR-096: a path the caller names is never dropped by a finder. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The walk refuses a named link to a directory, by name; contract §184 | done | — | `docs/adr/ADR-096-a-named-path-is-never-dropped-by-a-finder/tasks/T1-the-walk-refuses-a-named-directory-link.md` fence |
| T2 | ast-grep searches only the named paths the judge accepts; §184's ast-grep rows | done | — | `docs/adr/ADR-096-a-named-path-is-never-dropped-by-a-finder/tasks/T2-ast-grep-searches-only-judged-named-paths.md` fence |
| T3 | `iter add` names the real reason a path cannot be statted; contract §185 | done | — | `docs/adr/ADR-096-a-named-path-is-never-dropped-by-a-finder/tasks/T3-iter-add-names-the-real-reason.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `judgeNamed` | T2 | T1 before T2 |
| T1 | contract §184 | T2 | T1 opens the section; T2 adds its ast-grep rows |

## Notes

- Engine go/no-go: ADR-096 owns `internal/read/walk.go` (T1) and `internal/read/astgrep.go` (T2).
  `internal/apply`, `plan`, `seen`, `check`, `state`, `iter`, `rooted`, `lines` and `subproc` stay
  byte-identical against the branch's merge-base with `origin/main` in every task; `go.mod` keeps one
  requirement.
- Contract sections §184–§185 were allocated to this record on 2026-09-29, while ADR-093 to ADR-095
  were drafted beside it; do not renumber from the file's tail.
- Symlink cases skip where a link cannot be created; `scripts/contract.sh` is Linux-only, so §184 and
  §185 run where links always can.
