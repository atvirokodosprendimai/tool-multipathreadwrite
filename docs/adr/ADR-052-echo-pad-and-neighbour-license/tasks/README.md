# ADR-052 Tasks

Implementation tasks for ADR-052: Echo pad is opt-in; a multi-line replace needs a served line after End. See the parent ADR for the decision.

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
|----|-------|--------|--------|------------|
| T1 | Neighbour license on multi-line replace; contract §87 | done | — | `docs/adr/ADR-052-echo-pad-and-neighbour-license/tasks/T1-neighbour-license-and-contract-86.md` fence |
| T2 | `--echo-pad` / `echo_pad`; contract §88 | done | — | `docs/adr/ADR-052-echo-pad-and-neighbour-license/tasks/T2-echo-pad-and-contract-87.md` fence |
| T3 | Teach the two arms; close 048 / BACKLOG | done | — | `docs/adr/ADR-052-echo-pad-and-neighbour-license/tasks/T3-teach-and-close-048.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

`blocked` here means the word `done`, not the work: all three shipped in v1.15.0 (`31422d8`)
and their fences pass. adr-lint 2.98.1 refuses `done` because no first-red row was ever recorded
for these tasks, and ADR-050's cutover (2026-09-13, the day they were accepted) makes that lock
mandatory; F-1 forbids a later red supplying it. Two things WERE repairable and were repaired on
2026-09-13: T1 and T2 now carry a killed mutant each (the neighbour licence disabled; `echoPad`
returning nothing) and exit-0 rows under their current digests — evidence the lifecycle rule
required and the original execute skipped. Same shape as ADR-054; see its README.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | neighbour license in `Apply` (T1) | T2, T3 | T1 before T2 |
| T2 | `HunkResult.Echo` / `--echo-pad` (T2) | T3 | T2 before T3 |

## Notes

- Branch from `origin/main`. Do not merge #166.
- Engine go/no-go: `internal/apply` is owned by this record.
