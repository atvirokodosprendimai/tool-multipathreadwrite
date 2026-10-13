# ADR-147 Tasks

Implementation tasks for ADR-147: every language the tree runs is analysed, not only Go. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The tree passes every new analyser | done | — | `docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T1-the-tree-passes.md` fence |
| T2 | The gate runs them and proves each can fail | done | — | `docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T2-the-gate-runs-them.md` fence |
| T3 | CI installs them and the docs name them | pending | — | `docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T3-ci-and-docs.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | a tree every new analyser passes | T2 | T1 before T2 |
| T2 | the new `static.sh` steps | T3 | T2 before T3 |

## Notes

- Engine go/no-go: ADR-147 owns nothing under `internal/` or `cmd/mrw/`; every task's fence checks both stay byte-identical against the merge-base and that `go.mod` keeps one requirement.
- The record and its tasks land in one pull request: fence-prose reads pending tasks' fences too, so T3's prose greps are red until T3 is done.
