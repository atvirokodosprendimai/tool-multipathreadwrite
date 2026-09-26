# ADR-082: A check finds Git's shell on Windows

**Status:** Accepted
**Accepted:** 2026-09-26 by M — "PowerShell is one of the main targets of usage", then chose "Find Git's sh" over skipping the tests or running checks in PowerShell
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, ADR-071, ADR-080, docs/adr/BACKLOG.md
**Governs:** `internal/check/shell.go`, `internal/check/check.go`, `internal/check/shell082_test.go`, `internal/check/shell082_windows_test.go`, `internal/check/logs080_test.go`, `cmd/mrw/writecheck_test.go`, `cmd/mrw/pathop_check_test.go`, `cmd/mrw/planpath_test.go`, `cmd/mrw/receipt_before_check_test.go`, `cmd/mrw/checklog080_test.go`, `cmd/mrw/shell082_test.go`, `internal/adversarial/adr054_test.go`, `internal/adversarial/shell082_test.go`, `AGENTS.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/check/shell082_windows_test.go::TestACheckRunsUnderGitsShellWhenShIsNotOnPath`
**Invalidates:** none
**Served-path change:** on Windows, when no `sh` is on PATH, a check runs under the `sh.exe` Git for Windows installs beside `git.exe` (`usr\bin\sh.exe` or `bin\sh.exe`, found by walking up from `git.exe`); with neither, "could not start" says to install Git for Windows or put an `sh` on PATH. Elsewhere nothing changes.

## Context

**What was observed** (a Windows peer, 2026-09-26, PowerShell 7.6.6, v1.27.0): three `go test ./...`
runs each failed the same 13 tests with `could not start: exec: "sh": executable file not found in
%PATH%`. A plain PowerShell PATH holds Git's `cmd` directory and not its `usr\bin`, so mrw's check,
which runs a declared command with `sh -c`, never starts there: the suite is red, and a PowerShell
user gets no default check at all. With Git's `usr\bin` on PATH everything passes. BACKLOG's
"Fixed by ADR-071 T4 for the check tests" did not hold on that PATH. M: PowerShell is a main target.

## Existing Primitives Audit

- `check.Run` starts the check with `subproc.Command(ctx, "sh", "-c", cmdline)` — the only place mrw
  starts a shell.
- Git for Windows ships `sh.exe` in `<Git>\usr\bin` (and `<Git>\bin`), and puts `<Git>\cmd\git.exe`
  on PATH.

## Decision

1. `check.Shell()` returns `sh` from PATH; on Windows, failing that, the `sh.exe` found by walking up
   at most three directories from the `git.exe` on PATH (`usr\bin\sh.exe`, then `bin\sh.exe`).
   `check.Run` runs the check with it.
2. When no shell is found, the could-not-start report says so: on Windows, "install Git for Windows,
   or put an sh on PATH".
3. A test that needs a check to run skips, naming the reason, only when `check.Shell()` finds none.

## Alternatives Considered

- **Only skip the tests.** Rejected by M: the suite goes green while a PowerShell user still gets no
  check.
- **Run the check through PowerShell.** Rejected by M: a declared check is a POSIX shell command
  (`&&`, `2>&1 |`, `$VAR`), and it would mean something else, or fail, depending on the host shell.

## Component / Boundary Impact

`internal/check` only; the tests that need a shell in `cmd/mrw`, `internal/adversarial` and
`internal/check` skip through it. The engine packages `apply`, `plan`, `read`, `rooted`, `lines`,
`iter`, `seen`, `state` and `subproc` are unchanged.

## Wiring & Contract Changes

A check that could not start on Windows now starts where Git for Windows is installed. No contract
row: `scripts/contract.sh` runs on Linux only (AGENTS.md); the Windows CI shard runs the test named in
Enforced-by, which takes `sh` off PATH.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- A PowerShell session with Git for Windows installed runs the project's default check.
- `mrw check` and `mrw write --check` behave the same in PowerShell, cmd.exe and Git Bash.

## Out of Scope

- A Windows machine with no Git and no sh (permanent: boundary: a declared check is a POSIX shell command, so without a POSIX shell mrw reports that it could not start and names what to install)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A `git.exe` on PATH that is not Git for Windows (a shim, WSL's) | Low | Low | only a regular `sh.exe` in Git's layout is taken; otherwise the report names what to install |

## Rollback

Revert the task.

## Follow-ups

- [ ] Release with the next tag, and ask a Windows peer to run `go test ./...` under plain PowerShell.
