# ADR-181: A run receipt on Windows names a log-read line for each shell

**Status:** Accepted
**Accepted:** 2026-10-29 by Zy — *"accepted ADR-180 and ADR-181, implement them"*: the three labelled lines and no cmd line for a path holding `!`, as Decision 1 states.
**Date:** 2026-10-28
**Amended:** 2026-10-28, before acceptance, from a Codex xhigh challenge (3 P2, 1 P3), each finding checked against the source: the Windows shards run `go test` without `-v`, so T2 adds a verbose step for its one test and names the shard (T2, `.github/workflows/ci.yml`); the guide's recipe 12, the generated `.claude/skills/mrw/SKILL.md` and `docs/agents.md` promise a four-line receipt and are in scope (T1, Governs, Invalidates); cmd expands `!NAME!` under delayed expansion even inside double quotes, so the cmd line is left out for a path holding `!` (Decision 1); the empty-log test rejects every read-line label, not just `read:` (T1).
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-082, ADR-142, ADR-177, README.md, AGENTS.md
**Governs:** `cmd/mrw/run177.go`, `cmd/mrw/run177_test.go`, `cmd/mrw/runread181_windows_test.go`, `.github/workflows/ci.yml`, `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md`, `docs/agents.md`, `README.md`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/run177_test.go::TestTheWindowsRunReceiptLabelsAReadLineForEachShell`
**Invalidates:** ADR-177 — Decision 5's single Windows `read:` line (the form typed at a PowerShell prompt), which becomes three labelled lines, and ADR-177 T2's "four-line receipt" wording in the guide's recipe 12, the generated skill and `docs/agents.md`, which on Windows is six lines; ADR-177's verification evidence stands as recorded. The unix `read:` line, the receipt's other lines, `--json` (which carries no read line, `docs/receipts.txt` `run` keys) and every exit code are unchanged. Checked: contract §313 drives the unix line only (Linux); ADR-142 is about piped plans, not this.
**Served-path change:** on Windows `mrw run` prints `read (PowerShell): …`, `read (cmd): …` and `read (Git Bash): …` in place of the one `read:` line, each a command that serves the log's last 40 lines when pasted into that shell; on unix the receipt is unchanged.

## Context

ADR-177 Decision 5 prints one Windows read line, `cmd /c "`"EXE`" read -- -:N-`$ < `"LOG`""`, a PowerShell double-quoted string, chosen because cmd's redirection keeps the log's bytes where a PowerShell pipe does not (measured 2026-10-27, issue #538). Pasted into Git Bash or cmd it fails: the backticks are PowerShell's escape and mean nothing — or a command substitution — elsewhere. Reported by the Windows validation of v1.77.1 (memoryd tool-multipathreadwrite/73843a76, 2026-10-28, maintenance item a). Claude Code on Windows runs its commands in Git Bash, so the shell most likely to paste the line is the one it fails in. mrw cannot know the caller's interactive shell: the receipt is printed by a process whose parent may be `sh` started by an agent from PowerShell, and the robustness plan (memoryd 644d167a, item 1(a)) forbids guessing it from GOOS or incidental environment variables.

## Existing Primitives Audit

- `runReadLine(goos, exe, path, lines)` (`cmd/mrw/run177.go:232`): the one producer; it gains the per-shell forms. Reuse.
- `posixQuote` (the unix line's quoting) serves the Git Bash line: Git for Windows' bash opens a single-quoted `C:\…` path. Reuse.
- `psEscaper` (`cmd/mrw/run177.go:250`): the PowerShell line, unchanged. Reuse.
- The `"`/`%` refusal (`:238`): a Windows path cannot hold `"`, and cmd expands `%VAR%` inside double quotes; with either, no Windows line is printed, as now. cmd also expands `!NAME!` inside double quotes when delayed expansion is on and NAME is a defined variable — an undefined one is left as written, measured 2026-10-29 in CI run 40822546802 on cmd 10.0.26100.33650 — (`cmd /v:on`, or the registry's `DelayedExpansion`; https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/cmd), which mrw cannot see, so a path holding `!` loses only the cmd line.

## Decision

**1. Three labelled lines on Windows.** In place of `read: …`, the Windows receipt prints, in this order:

```
read (PowerShell): cmd /c "`"EXE`" read -- -:N-`$ < `"LOG`""
read (cmd): "EXE" read -- -:N-$ < "LOG"
read (Git Bash): 'EXE' read -- '-:N-$' < 'LOG'
```

The PowerShell line is ADR-177's, byte for byte. The cmd line is the same command typed at a cmd prompt; the Git Bash line is the unix form with the Windows paths single-quoted (`'` written `'\''`). Each serves the last 40 lines (`-:1-$` when there are 40 or fewer), through standard input, licensing and recording nothing, as ADR-177 Decision 5 says. When a path holds `"` or `%`, none of the three is printed, as today; when a path holds `!`, the cmd line is left out and the other two are printed.

**2. Unix is unchanged**: one `read:` line, the POSIX form.

**3. Each line is executed in its shell before it ships.** A Windows test runs each printed line in pwsh 7 (when the runner has it), Windows PowerShell 5.1, cmd and Git for Windows' sh against a log holding non-ASCII text and a path with a space and an apostrophe, and compares the served lines byte for byte. A Mac or Linux test of the strings is not that evidence.

**What would make this decision wrong:** a line that does not serve the log in its shell, which the Windows test exists to catch; or a reader who cannot tell which line is theirs, which the labels address. The three lines cost two more receipt lines on Windows only.

## Alternatives Considered

- **A flag naming the shell (`mrw run --shell=cmd`).** Explicit, one line. Rejected: the caller who pastes the wrong line is the one who did not know to pass a flag, and an agent would have to pass it on every run.
- **Detect the shell.** Rejected: forbidden by the plan, and unreliable — the receipt's parent process is often a shell other than the one the person types in.
- **One line that works in every shell.** None found: PowerShell reserves `<`, cmd has no single-quote quoting, and bash reads backticks as command substitution.
- **A mrw-native tail (`mrw run --tail LOG`) with no redirection.** Removes the shell's redirection from the recipe, but still needs the binary's path quoted per shell, and adds a command that reads a file outside the root. Rejected for this record.

## Component / Boundary Impact

- `cmd/mrw` (`run177.go`): the receipt's read lines. No engine package; ownership unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw run` receipt on Windows | `read:` becomes three labelled lines | `cmd/mrw` | people, agents pasting a line |
| `mrw run --json` | none — it carries no read line | — | — |
| README "mrw run", AGENTS.md `mrw run` | name the three lines | docs | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the three labelled lines | T1 | T2 | No |

## Implementation

`tasks/README.md`: T1 (the lines, the unit test, the docs), T2 (the Windows test that runs each line in its shell). One pull request.

## Consequences

- **Positive:** the line a Git Bash or cmd user pastes works.
- **Negative:** two more lines in a Windows receipt.
- **Neutral:** a consumer that grepped `^read: ` on Windows finds `^read \(PowerShell\): ` instead.

## Out of Scope

- A read line in `--json` (permanent: boundary: the JSON receipt carries `output_file`, from which a program builds its own read; ADR-177 left the line out of it)
- WSL's bash reading a Windows log path (permanent: boundary: WSL is a Linux process whose mrw is a Linux binary with its own state directory; it prints the unix line)
- Paths holding `"` or `%` (permanent: boundary: ADR-177 Decision 5 prints no Windows line for them, unchanged here)
- The cmd line for a path holding `!` (permanent: boundary: whether delayed expansion is on is the caller's cmd setting, which mrw cannot read)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Git Bash converts `-:N-$` or the quoted `C:\` paths | Low | Med | T2 runs the line in Git for Windows' sh on the CI runner; if it is converted the line is changed before it ships |
| A runner lacks pwsh 7 | Med | Low | T2 runs Windows PowerShell 5.1, which every Windows has, and pwsh when present, naming which ran |

## Rollback

Before a release: revert to the one line. After a release: a later release restores it; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
