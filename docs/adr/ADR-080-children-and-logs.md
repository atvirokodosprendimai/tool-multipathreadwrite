# ADR-080: A check and a finder leave nothing running and name what they kept

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), and answered *"Reap always (Recommended)"* for a check's leftover processes
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-008, ADR-072, ADR-074, docs/adr/BACKLOG.md
**Governs:** `internal/subproc/**`, `internal/check/check.go`, `internal/check/reap080_unix_test.go`, `internal/check/logs080_test.go`, `internal/read/astgrep.go`, `cmd/mrw/main.go`, `cmd/mrw/checklog080_test.go`, `cmd/mrw/astgrep_reap080_unix_test.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`, `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree.md`
**Enforced-by:** `internal/subproc/reap080_unix_test.go::TestAGrandchildOfACleanExitIsReaped`
**Invalidates:** ADR-072 Decision 4's scope *"kills the group on cancel"*, which becomes "after every exit"
**Served-path change:** after every check and every `--ast-grep` run, on unix, the child's process group is killed, so a background process it left behind does not outlive mrw; a check cancelled before its process starts reports `interrupted` and a write exits 3 (was "could not start", exit 2, "declare one"); a timed-out or interrupted check names the log it keeps, and one that never started keeps none; each check run removes `mrw-check-*.log` files older than 7 days from the temp directory and the receipt says how many (`pruned_logs` in JSON).

## Context

**What was observed.**

- The waiver on #232: a wrapper that exits 0 and leaves a background grandchild — its stdio
  redirected, so no pipe holds `Wait` — returned at once, and the grandchild outlived mrw. `exec.Cmd`
  calls `Cancel` only on a deadline or a cancel. A check that passed the same way left its
  background process running. M chose to reap always (2026-09-26).
- The review of #229: a signal that lands after the check's handler is installed and before its
  process starts reported "could not start: context canceled", exit 2, with advice to declare a
  check — which one was.
- BACKLOG inventory (2026-09-24): a failing or truncated check keeps its log, since the report points
  at it, and nothing bounded how many accumulated — 3,103 on one machine. A timed-out or interrupted
  check kept its log and named nowhere to find it.

## Existing Primitives Audit

- `subproc.Command` starts a child in its own process group (unix) and kills the group on cancel.
- `check.Run` writes the check's output to `os.CreateTemp("", "mrw-check-*.log")` and removes it on a
  pass that withheld nothing.

## Decision

1. `subproc.Run` and `subproc.Output` wait, then kill whatever is left of the child's process group.
   The check and ast-grep use them. Elsewhere there is no group; the wait bound still applies.
2. A check whose process never started because its context was cancelled reports `check.Interrupted`;
   the write exits 3 with "interrupted before it started", and `mrw check` does too.
3. A kept log is named on every path that keeps one; a check that never started removes its empty log.
4. `check.Run` removes this program's logs older than `LogRetention` (7 days) from the temp directory
   before writing its own, and reports the count (ADR-008: a delete says what it removed).

## Alternatives Considered

- **Reap only ast-grep.** Rejected by M: a stray from a passing check is the same leak.
- **Keep the last N logs.** Rejected: the temp directory is shared by every checkout, and a count
  would delete another run's fresh log; an age cannot.

## Component / Boundary Impact

`internal/subproc`, `internal/check`, one call in `internal/read/astgrep.go`, and the reports in
`cmd/mrw`. The engine packages `apply`, `plan`, `lines`, `iter`, `seen`, `state` and `rooted` are
unchanged.

## Wiring & Contract Changes

Exit 3 replaces exit 2 for a check interrupted before it started. `pruned_logs` joins the check's JSON
result. Contract §162, §163.

## Inter-task Contracts

None across tasks.

## Implementation

See `tasks/`.

## Consequences

- A project whose check deliberately leaves a process running (a server started for later) finds it
  killed when mrw returns; a check that means to start one detaches it (`setsid`).
- The temp directory holds at most a week of mrw's check logs.

## Out of Scope

- A grandchild that called `setsid` (permanent: fact: it is in a process group of its own, which mrw did not start)
- A group id reused in the microseconds after its last member exits (permanent: fact: a kill of an empty group finds none; the window is the kernel's)
- Windows job objects for the grandchild (deferred: docs/adr/BACKLOG.md, with ADR-072's)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A check that relies on a background process surviving it | Low | Medium | `setsid` detaches one on purpose; the record says so |
| Pruning another user's logs in a shared /tmp | Low | Low | only files this user can remove are removed; errors are passed over |

## Rollback

Revert the three tasks.

## Follow-ups

- [ ] Release with ADR-076 to ADR-079 as v1.27.0.
