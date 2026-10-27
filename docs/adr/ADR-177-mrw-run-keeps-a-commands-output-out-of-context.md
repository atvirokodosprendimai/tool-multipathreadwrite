# ADR-177: `mrw run` keeps a command's output out of the context and prints one receipt

**Status:** Accepted
**Accepted:** 2026-10-27 by Zy — *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction"*, and, after the plan was presented, *"go on, implement all of it"*.
**Date:** 2026-10-27
**Amended:** 2026-10-27, before acceptance, from four Codex challenges and a Claude adr-challenger pass, each finding checked against the code. The printed read form puts flags before `--` and quotes the path (it exited 2 as first drafted); the runner keeps a 24-hour ceiling, not "no timeout" (`internal/check/check.go:82-86`); every run keeps its log, including the short passing ones the check deletes (`:486`); a running log is never pruned; signal, timeout and interrupt exits are named; Windows gets its own read form; the baseline is re-measured over a fixed window, which shrinks the bucket this record addresses to non-read Bash output (Context). ADR-177 ships on its own branch from `main` (tasks/README.md).
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-004, ADR-034, ADR-077, ADR-080, ADR-082, ADR-092, ADR-095, ADR-100, ADR-101, ADR-104, ADR-111, ADR-120, ADR-149, ADR-161, ADR-178, ADR-179, docs/bench/output-study/README.md, docs/receipts.txt, docs/agents.md, README.md, AGENTS.md
**Governs:** `cmd/mrw/main.go`, `internal/check/check.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/run177_test.go::TestRunKeepsTheOutputAndPrintsOneReceipt`
**Invalidates:** none — checked: a new subcommand; `--then-sh`, the check, its temp-directory logs, their deletion and pruning (ADR-080) are unchanged. `guide.ThenShCaveat()`'s sentence is widened to name `mrw run` beside `--then-sh`; every test holds it by `Contains`, not by its words.
**Served-path change:** `mrw run 'CMD'` runs CMD under `sh -c`, writes its combined output to a log beside mrw's state directory, prints a receipt of four lines (exit, counts, the last non-empty line cut to 200 characters, the log, and a read command that reaches it) instead of the output, and exits with the command's own code.

## Context

Zy, 2026-10-27, on the output study: *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction."* The goal: cut the tool output agents pull into their context when they use mrw.

**The registered baseline** is `docs/bench/output-study/baseline-2026-10-27.json`: `measure_output.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, every tool result whose transcript line is stamped in that window, 129 main-session transcripts, 100,931,328 characters of tool output.

- Bash outputs over 4 KB, not mrw: 7.81M characters in 952 calls — **of which 6.27M are `cat`/`grep`/`rg`/`sed`**, reads that `mrw read` addresses, not this record. **What `mrw run` can address is the rest: 1.54M characters in 142 calls (1.5% of all output)** — builds, tests, gates, git.
- **11,478 of 23,850 non-mrw Bash calls (48.1%) pipe into `head` or `tail`** — the shape AGENTS.md warns loses the exit code ("Never read an exit code through a pipe").

The first study run (2026-10-27, files chosen by mtime, the figures this record was first written with) reported 10.2M characters in 1,251 calls over 4 KB; that count included the reads. The challenge found the confound; the fixed-window re-run separates it.

Confounds, written before any code: subagent transcripts (subdirectories) are not scanned; one machine, one owner; the saving assumes the agent needed none of the output it did not see, which is false for a failing test it must then read — that is why the log is kept and the receipt names how to reach it. The head/tail share is the larger lever: a command already trimmed with `| tail` costs little output and loses its exit code, which `mrw run` keeps.

## Existing Primitives Audit

- **The step runner** `run(ctx, root, cfg, cmdline, env)` (`internal/check/check.go:346`): `sh -c` from `Shell()` (`internal/check/shell.go:26`, Git's `sh.exe` on Windows, ADR-082), output to a file and never a pipe (`:386-399`), the process group or job object through `subproc.Command`/`subproc.Run` (`internal/subproc/subproc.go:47`, `:94`; ADR-080, ADR-095, ADR-120), the verdict from the process (ADR-003), `ExitCode -1` when nothing exited with a status (ADR-101). Two of its behaviours are the check's, not a run's: a deadline it always installs (`defaultTimeout`, 5 minutes, `:82`, `:357-384`; ceiling `maxTimeoutSeconds`, 24 hours, `:86`), and deletion of a passing log that withheld nothing (`:486-490`). Its caller installs the interrupt handler (`subproc.Interruptible`, `:319`; `run`'s comment at `:342-345`). **Reshape:** T1 adds an exported entry in `internal/check` — an engine package this record owns (lifecycle §6) — that takes a log directory, a log name, a timeout and a keep-the-log policy, wraps `subproc.Interruptible` as `Run` does, and calls the same body; the check's calls keep today's arguments and behaviour. Duplicating the runner in `cmd/mrw` was rejected (Alternatives).
- **Exit grading** (`internal/check/check.go:418-469`): a signalled process reports `ExitCode() == -1` through `ProcessState`, and a timeout or an interrupt sets `Skipped`. The entry also returns the wait status, so `mrw run` can report 128+N.
- **`--then-sh 'CMD'`** (`cmd/mrw/main.go:2266-2292`): the same trust surface and the same `MRW_STEP_DEPTH` guard, `check.DepthRefusal` and `deeper()` (`internal/check/check.go:1130-1156`, ADR-095); its help carries `guide.ThenShCaveat()` (`cmd/mrw/main.go:2271`, held by `cmd/mrw/teach092_test.go:25`). Reuse all three.
- **Log pruning** `pruneLogs(dir, cutoff)` (`internal/check/check.go:258`, called at `:338` and `:1087`), `LogRetention` 7 days (`:253`): hard-wired to the `mrw-check-*.log` name. Reshape: T1 adds `pruneMatching(dir, match, cutoff, keepNewest)` beside it and leaves `pruneLogs(dir, cutoff)` as the check's wrapper, because `internal/check/logs080_test.go` calls it with that signature and the check's tests stay unchanged.
- **The tail** `lastLines` (`internal/check/check.go:940`): a bounded read of a log's last lines (ADR-104). Reuse for `last:` and the line count.
- **The sibling directory beside the state home**: ADR-176 (branch `adr-176`, unmerged) keeps refused plans in `<state home>/mrw-refused/`, beside `state.Base()` (`internal/state/state.go:48`), so ADR-077's "nothing inside the state directory is served or edited" stays whole and `mrw seen --prune` is untouched. This record takes the same shape and does not depend on ADR-176's code.
- **Reading a file outside the root** is refused (`mrw read /etc/hosts` under `--root /tmp`, probed 2026-10-27). **Reading standard input with an address** works and records nothing: `printf 'x\ny\nzzz\n' | mrw read -- '-:2-3'` serves lines 2-3 (ADR-161, probed 2026-10-27). Flags go before `--`: `mrw read -- '-:/P/' -C 3` exits 2 (`cmd/mrw/stdin.go:35`, probed by the challenge), `mrw read -C 3 -- '-:/P/'` exits 0.
- **Every subcommand is held by tests over the whole command list**: `TestEverySubcommandReachesTheAgentFacingGuide` (`cmd/mrw/agentsdoc_test.go:29`) needs a backticked `mrw run` in AGENTS.md; `cmd/mrw/rootflag165_test.go:33` and `cmd/mrw/paddedstress_test.go:82` ask every subcommand about the root flag and padded arguments. T1 runs the whole `cmd/mrw` package.
- **Receipts only add keys** (ADR-111, `docs/receipts.txt`, `cmd/mrw/receipts111_test.go`); a refusal under `--json` is a document `{"error": …}` (ADR-100's shape for `mrw check`).

## Decision

**1. The command.** `mrw run [--json] 'CMD'` takes exactly one argument, the command string, and runs it as `sh -c CMD` in the root through the exported runner entry: Git's `sh.exe` on Windows when `sh` is not on PATH (ADR-082), a process group on unix and a job object on Windows (ADR-120), SIGTERM then kill a second later when stopped (ADR-095). More than one argument is a usage error, exit 2, naming the one-string form (`mrw run 'go test ./...'`): joining arguments would drop the caller's quoting. Its standard input is the null device, so a command that would wait on input ends instead of hanging — the `rg` with no path that AGENTS.md records hanging an agent shell.

**2. Time.** The run's deadline is the runner's ceiling, 24 hours (`maxTimeoutSeconds`), not the check's 5 minutes: a build or a suite is the main use, and the host bounds its own call. While it runs, an interrupt, terminate or hangup sent to mrw stops the command's group, as for the check (`subproc.Interruptible`).

**3. The log.** Standard output and standard error go, interleaved, to one file in `<state home>/mrw-runs/`, the sibling of `state.Base()`, mode 0600, directory 0700. While the command runs it is `run-<UTC time to the nanosecond>-<pid>-<8 hex of crypto/rand>.running.log (the salt keeps two runs of one process that end in the same instant from sharing a name, as ADR-176 names its kept plans)`; after it exits, mrw renames it to `run-<UTC time the command ended>-<pid>-<salt>.log`, the name the receipt prints, never replacing a log that is already there (a hard link then a remove, not a rename; on a clash the log stays a `.running.log` as for a failed rename), so names sort in the order runs finished (named at the start, a long run's log would sort oldest and be the first the count bound removes). The log is created with O_EXCL. If the rename fails the log stays a `.running.log`, mrw says so on stderr, and it is removed after 25 hours. Every run keeps its log — passing, failing, short or empty: the point is to read it afterwards. The directory is beside mrw's state directory, never in it, so ADR-077 is unchanged and `mrw seen --prune` does not look at it. **When the state home is inside a checkout** (`XDG_STATE_HOME` in the tree, or `--root "$HOME"`; ADR-077), `mrw-runs/` is inside it too and is an ordinary directory there: a walk serves it and git sees it. The README says so; a caller who puts the state home in a tree ignores `mrw-runs/` there.

**4. Retention.** After its command exits and its log is renamed, a run removes, best effort, completed logs (`run-*.log`, never `*.running.log`) in `mrw-runs/` older than `check.LogRetention` (7 days), then all but the newest 64 completed logs by name (this run's own just-renamed log is never a candidate and counts toward the 64: it and the newest 63 others stay), and a `*.running.log` older than 25 hours (beyond the ceiling, so its run is gone). Nothing else removes them: a log is **eligible** for removal when it is older than 7 days **or** beyond the newest 64 completed ones, and is removed by the next run that completes. The directory serves every checkout of this user, as ADR-176's does, so the count bound is needed beside the age bound.

**5. The receipt.** On standard output, and nothing else from the command:

```
exit 1 · 2,314 lines · 187,902 bytes · 41.2 s
last: FAIL	github.com/x/y/internal/read	3.112s
log: /Users/zy/.local/state/mrw-runs/run-20261027T101500.123Z-4242.log
read: mrw read -- '-:2275-$' < '/Users/zy/.local/state/mrw-runs/run-20261027T101500.123Z-4242.log'
```

`last:` is the last non-empty line, its first 200 characters followed by `…` when it is longer; absent when the output is empty. A line that is blank in the first 4,096 bytes mrw keeps of it but not after them is the last line, shown as `…`; blankness is judged by character over the whole line, a multi-byte space cut by a read boundary included. The `read:` line is a working command that serves the last 40 lines (`-:1-$` when there are 40 or fewer): the log is outside every checkout (Decision 3's exception aside), so a named-path read is refused, and standard input reaches it, licenses nothing (a log is not a tree file) and records nothing in the ledger, though like any mrw call it makes the root's state directory (checked 2026-10-27). The path is single-quoted for a POSIX shell (`'` written `'\''`). On Windows the line is the PowerShell form, `Get-Content -Raw -LiteralPath 'PATH' | mrw read -- '-:N-$'` (`'` doubled). The recipe (T2) teaches `mrw read --grep P -C 3 - < 'LOG'` for a pattern. A pruning count is printed only when non-zero. A run that timed out or was interrupted says so in the first line (`timed out after 24h0m0s`, `interrupted`).

**6. The exit code is the command's** — 0 to 255 as `sh` reports it; 128+N for a command killed by signal N on unix, taken from the wait status (`syscall.WaitStatus.Signal()`), since `ProcessState.ExitCode()` is -1 for it; 124 for a run that reached the ceiling (GNU `timeout`'s code); 130 for a run mrw stopped on an interrupt, terminate or hangup — and both when the interrupt or the deadline lands before the command started: no command ran, so there is no receipt line, a `mrw:` line names it, and `--json` is the `ran: false` document. This gives the subcommand's codes a meaning of their own, as `mrw version --check-latest` does (ADR-149); README "Exit status" and AGENTS.md "Exit codes are the contract" say so (T2). mrw's own failures — a usage error, the log directory that cannot be made, a command that could not start, the depth guard — exit 2 with a `mrw:` line on stderr and no `exit` receipt line, so a 2 from the command (receipt present, `exit 2`) and a 2 from mrw (no receipt) are told apart.

**7. `--json`.** One document on stdout, a new receipt type `run`: `command`, `ran`, `exit_code`, `signal`, `skipped`, `duration_ms`, `lines`, `bytes`, `last`, `output_file`, `pruned_logs`. A command that could not start is that document with `"ran": false`, `"exit_code": -1` and `skipped` naming why (ADR-101), exit 2. A refusal before anything runs — usage (a flag the parser rejects included), depth — is `{"error": …}` with no `exit_code` (ADR-100's shape), exit 2. Each key is appended to `docs/receipts.txt` under the type `run`, with its legend line, in T1.

**8. Depth.** `mrw run` is refused at `MRW_STEP_DEPTH` 8 with `check.DepthRefusal("mrw run is")`, before anything runs or any log is made, and its command runs one level deeper, as a step does (ADR-095): `mrw run 'mrw run …'` stops.

**9. Trust.** `mrw run` runs any shell command it is given, as `--then-sh` already does: a harness rule that allows mrw without reading its arguments allows arbitrary shell. `guide.ThenShCaveat()` is widened to name both, so `mrw run --help`, the `--then-sh` help, the recipes, README "What allowing mrw allows" and `docs/agents.md` §7 say it in one sentence.

**10. CLI only.** The MCP surface is for hosts with no shell (ADR-170); such a host has no command to run.

**What would make this decision wrong:** agents running `mrw run` and then reading the whole log back, which pays the output plus the receipt. The pre-registered re-measure (`docs/adr/BACKLOG.md`, "Pre-registered for ADR-177, ADR-178 and ADR-179") counts reads of `mrw-runs/` logs beside the bucket.

## Alternatives Considered

- **Copy the runner into `cmd/mrw`.** Rejected: a second copy of the shell lookup, the group handling and the exit grading is the drift ADR-082, ADR-095 and ADR-120 each fixed once.
- **No deadline at all.** The first draft. Rejected: the runner has none to offer below its ceiling without a new code path, and a run killed with its host leaves its group running; the 24-hour ceiling is the existing bound.
- **The check's 5-minute deadline.** Rejected: a suite or a build is the main use.
- **Logs in the temp directory, like the check's.** Rejected: on Linux `/tmp` is shared and its lifetime is the OS's; ADR-176 made the same choice for the same reason.
- **Logs inside the state directory, with an ADR-077 exception.** Rejected: ADR-176's challenge showed the exception can be reached by a hard link, and a log there could not be read with mrw at all.
- **The receipt names `mrw --root <state home>/mrw-runs read NAME.log:/P/`.** Works, and licenses the log's lines; rejected as the default form because it makes a state directory keyed on the log directory, which `mrw seen --prune` then treats as a checkout (ADR-034), and the licence is of no use: nobody edits a log.
- **A hard-coded `/FAIL|panic/` in the read line.** The first draft. Rejected: which pattern matters is the caller's; the last 40 lines are the one answer every command has.
- **Summarise the output (first and last N lines).** Rejected: the receipt gives the counts, the last line and the address, and `mrw read --grep` answers the rest.
- **Arguments after `--` joined into one string.** Rejected: `mrw run -- grep 'a b' f` would run `grep a b f`.
- **On MCP too.** Rejected: Decision 10.

## Component / Boundary Impact

- `internal/check` (engine package, owned here): the exported entry over `run` (log directory and name, timeout, keep-the-log, the interrupt handler, the wait status), and `pruneLogs` taking the pattern and the bound; the check's behaviour, its logs and its receipt are unchanged.
- `cmd/mrw`: the `run` subcommand, its receipt, the read line per platform, `--json`.
- `internal/guide`: `ThenShCaveat()` widened (T1, since `mrw run --help` carries it), the recipe (T2).
No package changes ownership. ADR-178 and ADR-179 change `internal/read`, which this record's fences hold byte-identical, and hold `internal/check` byte-identical themselves; so this record ships on its own branch from `main`, and those two share another.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw run 'CMD'` | new subcommand; exits with the command's code, 124 at the ceiling, 130 when stopped | `cmd/mrw` | agents, people |
| `<state home>/mrw-runs/run-*.log` | new files; `.running.log` while running; completed ones eligible after 7 days, newest 64 | `cmd/mrw`, `internal/check` | `mrw read -- '-:…' < LOG`, people |
| `mrw run --json` | new receipt type `run` and its eleven keys; refusals `{"error": …}` | `cmd/mrw` | adapters |
| `docs/receipts.txt` | legend line and rows for `run` | T1 | `TestNoShippedReceiptFieldDisappears` |
| `guide.ThenShCaveat()` | names `mrw run` beside `--then-sh` | T1 | `--then-sh` and `mrw run` help, the recipes |
| `mrw instructions`, `mrw skill`, `.claude/skills/mrw/SKILL.md`, README, AGENTS.md, `docs/agents.md` | a recipe, the exit codes, the trust note | T2 (AGENTS.md's mention in T1) | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `mrw run`, its receipt lines and exit codes | T1 | T2 | No |

## Implementation

`tasks/README.md`: T1 (the command, the log, retention, the receipt, `--json`, the widened caveat, AGENTS.md's mention, contract §302), T2 (the instructions). One pull request, on a branch of its own from `main`.

## Consequences

- **Positive:** a long build or test output costs the context four lines; the full output stays one address away, with its exit code intact.
- **Negative:** logs of command output persist in the user's state home, up to 64 completed files, eligible for removal after 7 days but removed only by a later run; nothing bounds one log's size (Risks).
- **Neutral:** a second way to run shell through mrw, with the same trust as `--then-sh`.

## Out of Scope

- `mrw run` on the MCP surface (permanent: boundary: MCP is for hosts with no shell, ADR-170)
- A `run` tool in the opencode plugin (deferred: docs/adr/BACKLOG.md "From ADR-177")
- A size cap on one run's log (deferred: docs/adr/BACKLOG.md "From ADR-177")
- Folding or projecting the log as it is written (permanent: boundary: the log is the command's output byte for byte; ADR-179's `--fold` and ADR-178's projection act when it is read)
- Replacing `--then-sh` or the check's temp-directory logs (permanent: boundary: both are published behaviour this record does not need to change)
- Getting agents off `cat`/`grep` in Bash (permanent: boundary: adoption guidance, `docs/agents.md` §6, not a feature of this record)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A command writes gigabytes and fills the disk | Low | Med | the receipt names the size; the 64-file bound limits the count, not the bytes (deferred cap) |
| The log holds a secret the command printed | Med | Med | 0600 in the user's own state home; eligible for removal after 7 days, removed by the next completed run; the README says it stays until then |
| A state home inside a checkout puts logs in the tree | Low | Med | Decision 3 names it; README says to ignore `mrw-runs/` there |
| A caller reads the exit code of a `mrw run` through a pipe | Med | Med | the recipe never pipes it; the receipt is short enough not to need `head` |
| A harness allow-rule for mrw now allows arbitrary shell | High | Med | it already did through `--then-sh`; one caveat names both |

## Rollback

Before a release: remove the subcommand and the `run` rows; nothing else changes. After a release: `mrw run` is deprecated — it keeps working and says so for at least one minor release — before removal; its receipt keys stay listed (ADR-111). Completed logs become eligible for removal 7 days on.

## Open Questions

- **Retention numbers.** 7 days reuses `LogRetention`; 64 files is a guess (ADR-176 chose 16 for plans, which are smaller and rarer). Zy may set either.
- **A per-log size cap.** None is decided; a cap would cut what the caller came to read. Deferred with a trigger.

## Follow-ups

- [ ] none at authoring
