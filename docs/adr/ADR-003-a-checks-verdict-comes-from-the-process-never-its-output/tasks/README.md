# ADR-003 Tasks

Implementation tasks for ADR-003: A check's verdict comes from the process,
never its output. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

**This ADR is a retrofit.** Both tasks describe behaviour shipped in `f0e12a9`
and corrected in `6655113`, released as v0.0.1 and v0.0.2. They are `done` as of
2026-08-31: each carries a tool-written exit-0 Verification Log entry and a
killed mutant.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The check runs scoped to what changed, and its verdict is the process's real exit code | done | — | `go test ./internal/check/ -v … && go test ./internal/check/` |
| T2 | Four exit statuses, each meaning a different next move | done | — | `go build -o bin/mrw ./cmd/mrw && ./scripts/contract.sh … && go test ./cmd/mrw/` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `check.Config`, `check.Run`, `check.Result`, `check.Result.OK` | T2 | T1 before T2 — T2 maps a result T1 defines |

## Notes

- **T2's statuses are asserted by a shell script, not by Go tests, and that is a
  real limitation stated rather than hidden.** An exit status is a property of
  the process; a Go test calling `rootCommand().Run` in-process sees a returned
  error, not the status a shell observes. `scripts/contract.sh` is therefore the
  only place all four are checked as statuses, and its fence asserts that the
  adversarial row is PRESENT as well as passing — a contract script that quietly
  stopped running that case would otherwise still exit 0.
- The exit-3 semantics were shipped wrong once and corrected in `6655113`: a
  check that could not RUN also exited 3, under a definition saying a check
  "ran and did not pass". That correction is why the parent ADR lists it as a
  rejected alternative rather than leaving the history invisible.
