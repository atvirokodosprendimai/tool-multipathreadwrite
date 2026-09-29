# ADR-097 Tasks

Implementation tasks for ADR-097: a flag named for a subcommand names the subcommand. See the parent
ADR for the decision.

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
|----|-------|--------|--------|------------|
| T1 | The usage error names the subcommand a flag was spelled as; contract §188 | done | — | `docs/adr/ADR-097-a-flag-named-for-a-subcommand-names-the-subcommand/tasks/T1-the-usage-error-names-the-subcommand.md` fence |
| T2 | README.md and AGENTS.md say what `mrw --instructions` answers | done | — | `docs/adr/ADR-097-a-flag-named-for-a-subcommand-names-the-subcommand/tasks/T2-teach-the-named-refusal.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the named refusal (`subcommandForFlag`) | T2 | T1 before T2: T2 documents what T1 ships, and its fence runs T1's test |

## Notes

- Engine go/no-go: ADR-097 owns no engine package. `internal/read`, `apply`, `plan`, `seen`,
  `check`, `state`, `iter`, `rooted`, `lines` and `subproc` stay byte-identical against the
  branch's merge-base with `origin/main`; `go.mod` keeps one requirement.
- Contract section §188 was allocated to this record on 2026-09-29 by the coordinator, while ADR-093
  to ADR-096 were drafted beside it (§184–§185 are ADR-096's); do not renumber from the file's tail.
