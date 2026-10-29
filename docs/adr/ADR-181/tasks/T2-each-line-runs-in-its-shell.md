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

Acceptance is human-observed: the Windows shard holding `cmd/mrw` runs `TestEveryPrintedReadLineServesTheLogInItsShell` with `-v`; the sign-off names the run id, the shard, the head sha, the PowerShell, pwsh, cmd and Git versions it logged, and `--- PASS`, and the CI run in which the S2 mutant failed the Git Bash case. No local fence can observe it: the test runs only on Windows. (Amended 2026-10-29: the placeholder fence, a `grep` for the function name, passed on any platform whatever the shells did, so it was evidence of nothing; adr-lint asked a tool-run mutant of it, which only deleting the test could kill.)

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
- 2026-10-29 · human-observed · mutant killed · test exit 1 · `cmd/mrw/run177.go` · line 245 · from `return append(out, "read (Git Bash): "+runReadLine("unix", exe, path, lines))` · to `return append(out, "read (Git Bash): "+line)` · test `go test -count=1 -v -run '^TestEveryPrintedReadLineServesTheLogInItsShell$' ./cmd/mrw/ (Windows CI shard 3, run 40824370162, head 822b2c25)` · S2: the Git Bash line in PowerShell form; sh: unexpected EOF while looking for matching backtick, the Git Bash case failed exit 2

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
- 2026-10-29 · human-observed · Claude (session mac-tool-multipathreadwrite-main-dev) read the job log of CI run 40825388009, windows-shard (3), head 551d00e28e56ebaca15b8f449d091505e759200b: '--- PASS: TestEveryPrintedReadLineServesTheLogInItsShell (2.77s)' under -v, logging powershell.exe 5.1.26100.33648, pwsh.exe 7.6.7, cmd.exe 10.0.26100.33650, Git sh GNU bash 5.3.15(2). S1 red before T1's lines existed: not run on CI as a separate commit (only ADR-177's single line existed, so receiptField finds no labelled line). S2 mutant (Git Bash line in PowerShell form, 822b2c25) killed in CI run 40824370162, shard 3: 'Git Bash: the printed line did not serve lines 6-45 byte for byte (err exit status 2)', sh 'unexpected EOF while looking for matching'. Also confirmed on Windows 11 10.0.26200 by the repository's Windows session at f58e7423 (PS 5.1.26100.9711, pwsh 7.6.9, cmd 10.0.26200.9711, Git bash 5.2.37). · acceptance-sha256:58726be3a4f369b2f09c696550d2391e8b5d26f811b1313037ba3d5a7e9349d3
- 2026-10-29 · 551d00e2* · exit 0 · `# human-observed: the Windows shard holding cmd/mrw runs TestEveryPrintedReadLineServesTheLogInItsShell with -v; …` · acceptance-sha256:58726be3a4f369b2f09c696550d2391e8b5d26f811b1313037ba3d5a7e9349d3 · ms:43
- 2026-10-29 · human-observed · Claude (session mac-tool-multipathreadwrite-main-dev) read the job log of CI run 40825388009, windows-shard (3), head 551d00e28e56ebaca15b8f449d091505e759200b: '--- PASS: TestEveryPrintedReadLineServesTheLogInItsShell (2.77s)' under -v, logging powershell.exe 5.1.26100.33648, pwsh.exe 7.6.7, cmd.exe 10.0.26100.33650, Git sh GNU bash 5.3.15(2). The S2 mutant (Git Bash line in PowerShell form, 822b2c25) failed the Git Bash case in CI run 40824370162, shard 3 (sh: unexpected EOF while looking for matching backtick). The repository's Windows session confirmed the three lines at f58e7423 on Windows 11 10.0.26200 (PS 5.1.26100.9711, pwsh 7.6.9, cmd 10.0.26200.9711, Git bash 5.2.37). · acceptance-sha256:cab0c0cfaef08aab369c59d5095ebc14f2d31d1c69c69946f4cd03c4bcdc8f7e
