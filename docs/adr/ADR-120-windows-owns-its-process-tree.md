# ADR-120: Windows owns its process tree

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — the gap list of 2026-10-01, item 6, kept over ADR-080's boundary ("Keep ADR-120"); "start suspended, resume through Toolhelp32, kernel32 only, no taskkill fallback"; and after a Windows peer's measurement, "Drop BREAKAWAY_OK". The record's text was drafted after those answers and not shown to Zy before execution: these are the answers, not a review of these words.
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-072, ADR-074, ADR-080, ADR-082, ADR-095
**Invalidates:** ADR-080's "on Windows a grandchild can outlive the child" boundary and its Out of Scope job-object deferral
**Governs:** `internal/subproc/subproc.go`, `internal/subproc/job.go`, `internal/subproc/subproc_windows.go`, `internal/subproc/subproc_other.go`, `internal/subproc/subproc_unix.go`, `internal/read/astgrep.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/subproc/job120_test.go::TestTheJobSequenceAssignsBeforeResuming`
**Served-path change:** On Windows a check, a step or ast-grep runs in a job object that is closed when it exits, is cancelled or times out, so every process it started stops with it and releases what it held, the check's log included. A Windows process cannot leave that job, so there is no equivalent of `setsid` there.

## Context

`internal/subproc` stops a child's whole process group on unix (ADR-072, ADR-080, ADR-095). On Windows it had no group to stop: `subproc_other.go`'s `group` and `reap` are empty, so a cancel killed only the `sh.exe` mrw started. A Windows peer measured v1.42.0 on a real desktop (BACKLOG "From the Windows peers", 2026-10-02): a timed-out check left the inner `sh.exe` and both `sleep.exe` grandchildren running; a timed-out `--then-sh` step left both sleeps; a killed mrw left all four descendants. In every case the survivors held the check's log open until killed. Their parent ids were dead MSYS fork stubs, so a walk of parent ids from mrw's child would not have found them.

A job object is what Windows offers instead of a group: every process created by a member joins it, and `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` kills every member when the last handle closes, including when mrw itself dies. The same peer then measured that Git for Windows' sh passes `CREATE_BREAKAWAY_FROM_JOB` whenever the job allows it: with `JOB_OBJECT_LIMIT_BREAKAWAY_OK` set, both grandchildren left the job and survived its close. Zy dropped that flag.

**Audit of the class** — *every child mrw starts that can have descendants*: `mrw read --grep 'subproc\.(Command|Run|Output)\(' --exclude '*_test.go' internal/ cmd/` — 2 sites: the check and its steps (`internal/check/check.go`, `subproc.Command` then `subproc.Run`) and ast-grep (`internal/read/astgrep.go`, `subproc.Command` then `subproc.Output`, which calls `Run`). Both start through `Command` and wait through `Run`, which is where the job is created; nothing else starts a child.

## Existing Primitives Audit

- **`subproc.Command` / `subproc.Run`** — the one start and the one wait; the job belongs between them, so no caller changes.
- **`syscall.NewLazyDLL("kernel32.dll")`** — already used by `internal/state/lock_windows.go` and `internal/apply/attrs_windows.go`; `go.mod` keeps one requirement, so no `golang.org/x/sys`.
- **The fake filesystem behind `rooted.throughLinks`** (ADR-071) — the precedent for driving Windows-only logic on every platform: here a fake job API drives the sequence.

## Decision

1. **Every child starts suspended in its own job.** On Windows `Command` adds `CREATE_SUSPENDED`; `Run` starts the child, creates a job whose only limit is `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, assigns the child to it, and only then resumes the child's main thread, found through a Toolhelp32 snapshot. A grandchild the child starts at once is therefore created inside the job.
2. **Nothing escapes the job.** `JOB_OBJECT_LIMIT_BREAKAWAY_OK` is not set, because Git for Windows' sh asks to break away whenever it may. A process that must outlive mrw is started outside it.
3. **The job ends with the call.** A cancel or a timeout terminates the job (or kills the still-unassigned child); after the child exits, `Run` terminates and closes the job, so a background grandchild of a check that passed stops too (ADR-080's reap, on Windows). If creating, assigning or resuming fails, the child is killed and `Run` returns the error: the check reports it could not run rather than running uncontained.
4. **The sequence is portable; the calls are not.** The order — start, create, assign, resume, wait, terminate, close — lives in `job.go` behind a small interface, driven in tests by a fake on every platform; `subproc_windows.go` implements it with kernel32. Unix keeps its process group; other platforms keep the wait bound only.

## Alternatives Considered

- **`taskkill /T`** — rejected by Zy's answer: it walks parent ids, which the measurement showed are dead stubs under MSYS.
- **`BREAKAWAY_OK` as the `setsid` equivalent** — rejected after measurement: under Git for Windows' sh the job would contain nothing.
- **`PROC_THREAD_ATTRIBUTE_JOB_LIST` at creation** — would avoid the suspended start, but `syscall.SysProcAttr` exposes no attribute list for it and `go.mod` takes no new requirement.

## Component / Boundary Impact

`internal/subproc`, which is not an engine package, and six lines of `internal/read/astgrep.go`, which is: the ast-grep caller must refuse `subproc.ErrNotContained` rather than read the killed child's empty output as no matches (the review of #325). `internal/check` changes only in a test file (an un-skip). `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `subproc.Run` on Windows | creates, fills and closes a job around the child | T1 | `internal/check`, `internal/read` |
| `subproc.Command` on Windows | `CREATE_SUSPENDED`, a cancel that terminates the job | T1 | same |
| `AGENTS.md`, `README.md` | the Windows clause of the steps paragraph | T2 | callers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `subproc.runInJob`, `subproc.jobAPI` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** on Windows nothing a check, step or ast-grep starts outlives the call, and the log is released.
- **Negative:** a Windows process cannot opt out of the job; a check that must leave a daemon running has to start it some other way.
- **Neutral:** unix and other platforms are unchanged.

## Out of Scope

- A contract row (permanent: boundary: `scripts/contract.sh` drives a POSIX shell on Linux only; the real-process tests run on the windows CI shards)
- A breakaway opt-in for projects that need a surviving child (permanent: boundary: Zy chose no escape; arm on a request)
- The rules hook on NTFS (deferred: docs/adr/BACKLOG.md "From ADR-080")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a suspended child never resumed hangs the check | Low | High | `Run` is the only wait and resumes or kills; the fake drives the failure paths |
| the struct layout differs from Win32 | Low | High | `TestTheJobLimitStructHasTheWin32Size` pins 144 bytes on 64-bit; on windows/386 Go aligns `int64` to 4 bytes and the struct would be 108 against Win32's 112, so a 386 build would refuse every check — only windows/amd64 is released |
| a Windows-only path regresses unseen locally | Medium | Medium | the windows CI shards run the real grandchild tests on every push |

## Rollback

Revert the tasks: Windows goes back to killing only the direct child. No receipt key, exit code or stored state changes.

## Follow-ups

- None — the record carries no open follow-up.
