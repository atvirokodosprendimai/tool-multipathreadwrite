# Task ADR-181-T2: each printed read line serves the log in its own shell

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestEveryPrintedReadLineServesTheLogInItsShell`
**Consumes:** `runReadLines` (T1)
**Data dependency:** a Windows runner with Windows PowerShell 5.1, cmd and Git for Windows' sh (the CI `windows` job); pwsh 7 when present
**Proof map:** v1

## Goal

Decision 3: on Windows, each line T1 prints is run in the shell its label names, against a real `mrw run` log, and serves the log's last lines byte for byte.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/runread181_windows_test.go` | new | build-tagged windows; drives the built binary (`mrwExe`, `cmd/mrw/shell_windows_test.go:64`) |
| `.github/workflows/ci.yml` | edit | the shard whose package list holds `cmd/mrw` also runs `go test -count=1 -v -run '^TestEveryPrintedReadLineServesTheLogInItsShell$' ./cmd/mrw/`, so its `--- PASS` line and the shell versions it logs are in the job log (the shards run without `-v`, `:223`) |

## Ordered Steps

1. [S1] Write `TestEveryPrintedReadLineServesTheLogInItsShell`: in a directory whose path holds a space and an apostrophe, `mrw run` a command printing 45 lines, the last `ä € ü`; take the receipt's three lines; run the PowerShell line under `powershell.exe -NoProfile -Command` and under `pwsh.exe` when on PATH, the cmd line under `cmd.exe /d /s /c` with the whole printed command in an outer quote pair as a raw command line (`cmd/mrw/cmdquote_windows_test.go:38`), the Git Bash line under `check.Shell()`'s sh; each exits 0 and serves lines 6-45 with `ä € ü` intact; `t.Log` names each shell's version. A path holding `!` run under `cmd /v:on` shows why that line is left out. It never skips a missing PowerShell 5.1, cmd or sh. See it RED on the commit before T1 (only the PowerShell line exists). [proof: human: the Windows CI job is where this runs; the run's log is the evidence]
2. [S2] Mutant: the Git Bash line uses the PowerShell form — the sh case must fail on the CI runner. [proof: human: observed in the Windows CI job's log]

## Acceptance

```bash
# human-observed: the Windows shard holding cmd/mrw runs TestEveryPrintedReadLineServesTheLogInItsShell with -v;
# sign-off names the run id, the shard, the head sha, the PowerShell, pwsh and Git versions it logged, and `--- PASS`.
grep -q 'func TestEveryPrintedReadLineServesTheLogInItsShell' cmd/mrw/runread181_windows_test.go
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryPrintedReadLineServesTheLogInItsShell` | `cmd/mrw/runread181_windows_test.go` | each labelled line, run in its shell, serves the log's last lines byte for byte | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | the CI `windows` job runs `cmd/mrw` |
| 3 — the caller can discover it | — (a test) |
| 4 — it is used | every PR's Windows job |

## Mutation Log

## Invariants

- The test drives the built binary through each real shell; none is simulated.
- A missing required shell fails the test, never skips it.

## Risks

- The CI image's shells change version: the test logs each version it ran.

## Stop Condition

The Windows job passes with the test, and the sign-off line names the run.

## Out of Scope

- Interactive terminals (mintty, Windows Terminal) (permanent: boundary: the test runs each shell non-interactively; a paste into an interactive prompt is the same parser, and the v1.77.1 manual smoke covers the prompt)

## Verification Log
