# ADR-095: A nested mrw stays bounded — the depth guard reaches the check, and a timeout reaches a nested check

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — "ADR-095 nested bounded", selected under "Which records do you accept as written, so I can execute them?", on the record as drafted after a Codex review, including the depth-8 refusal of a write whose check is due
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-009, ADR-054, ADR-058, ADR-072, ADR-074, ADR-080, ADR-082, ADR-083, ADR-092, docs/adr/BACKLOG.md
**Governs:** `internal/check/check.go`, `internal/subproc/**`, `cmd/mrw/main.go`, `internal/guide/guide.go`
**Enforced-by:** `internal/check/depth095_test.go::TestTheCheckAndEveryStepRunOneLevelDeeper`
**Invalidates:** ADR-092 — its T5 amendment (lines 129-131: a step runs one level deeper, `--then`/`--then-sh` are refused at depth 8) now counts the check as a level too and also refuses, at depth 8, a write whose check is due and `mrw check`; its outcome table (lines 88-97) gains that refusal as a row. Narrowed with it: ADR-080 Decision 1 and ADR-072 Decision 4 ("kill" the group, on a cancel and after every exit) become TERM, then KILL at most a second later; ADR-058 Decision 5 and ADR-074 Decision 2 (a hanging `ast-grep` is "killed at 2 s") become TERM at 2 s and KILL by 3 s for one that ignores TERM.
**Served-path change:** A declared check now runs with `MRW_STEP_DEPTH` one higher than mrw's own, and at depth 8 or more `mrw write` whose check is due and `mrw check` are refused, exit 2, before anything is written or run — so a step at depth 7 that runs a plain `mrw write` of a code file, which ran on v1.31.0, is now refused; and on unix a cancelled or reaped check, step or `--ast-grep` process group gets SIGTERM, and SIGKILL only if something is still in it a second later (it got SIGKILL at once), so a nested mrw stops its own check before it goes.

## Context

The 2026-09-29 gap survey of v1.31.0 (workflow `wf_d4444a2d-6cc`; agentsmemory drawer `349dbd2b`, wing
`wing_tool-multipathreadwrite`, room `findings`; items loops-L1 = C10, loops-L2, loops-L5) found two
ways an mrw that runs mrw escapes its bounds. Both were re-probed for this record on 2026-09-29 on
macOS (Darwin 27.0.0, APFS) with the PATH binary `mrw version v1.31.0 (2ea8bd5)`; every line cited
below is `main` at 8cbb89e.

**1. A check that runs mrw is outside the depth guard.** ADR-092 T5 gives each step `MRW_STEP_DEPTH`
one higher than its caller's (`internal/check/check.go:792`) and refuses `--then`/`--then-sh` at
depth 8 (`cmd/mrw/main.go:1603-1604`). The declared check gets nothing: `Run` calls
`run(ctx, root, cfg, cmdline, nil)` (`internal/check/check.go:242`), and `run` adds variables only when some were passed
(`internal/check/check.go:316-323`), so the check inherits its caller's value unchanged — unset at the top, unset at
every level below. Probe: a check `echo "${MRW_STEP_DEPTH:-unset}" >> depth; … exec mrw check --full`,
stopped by its own counter after four levels, recorded `unset unset unset unset`; the survey drove the
same shape to 12. Each level's `timeout_seconds` starts fresh, so no timeout bounds the chain either.

**2. A timeout does not reach the check of a nested mrw.** On a deadline or an interrupt, and after
every exit (ADR-080), `internal/subproc` sends SIGKILL to the child's process group
(`internal/subproc/subproc_unix.go:14-16` for the cancel, `:23-26` for the reap). A nested mrw in that
group cannot catch SIGKILL, and its own check is a group of its own (ADR-072), so the check runs on
with nothing left to enforce its timeout. Probe: outer check `mrw --root inner check --full` with
`timeout_seconds: 1`, inner check `echo $$ > pid; exec sleep 20`. The outer exited 3, "timed out
after 1s"; the inner `sleep` was still running, parent 1, in a group of its own. A second probe sent
SIGTERM to a nested mrw directly: it exited 3, "interrupted", and its check's `sleep` was gone. A
nested mrw already stops its own check when TERMed, because `subproc.Interruptible` listens for TERM
while a check runs (`internal/subproc/subproc.go:58-69`). ADR-072 named this failure for a sender
outside mrw — "a SIGKILL sent to mrw's process group … no longer reaches the check"
(`docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree.md:65-68`) — and rated it Low
(`:124`). Here mrw is the sender.

**Why swapping the signal is not enough** (reasoned from the code, not probed; T2's end-to-end test
keeps `sh` the leader to prove it). `subproc.Run` reaps with SIGKILL as soon as the leader exits
(`internal/subproc/subproc.go:77-80`). When the leader is `sh` — as it is for a check of more than one
command, which the shell cannot `exec` in place — it dies of TERM at once, the reap follows, and the
nested mrw is killed before it has stopped its own group: the
orphan survives exactly as today. In Go's `os/exec` (go1.27.1, `src/os/exec/exec.go`) `Cancel` runs
in the watcher goroutine (`:807-831`), `Wait` reaps the leader first (`:944`) and then waits for that
goroutine (`:952`), and the `WaitDelay` timer starts only once `Cancel` has returned (`:837`). So a
`Cancel` that waits for the GROUP to empty holds `Run` for exactly as long as the group needs.

**3. What else the guard cannot see** (loops-L5). A command that clears the environment — `env -i`,
`sudo` with its default `env_reset` — hands the next mrw no depth, as `setsid` hands a grandchild no
group. `StepDepth` reads an unset or unreadable value as zero (`internal/check/check.go:845-851`); nothing pins that
today (`TestAStepRunsOneLevelDeeper` sets a readable value).

## Existing Primitives Audit

- `check.StepDepth`, `check.MaxStepDepth` (`internal/check/check.go:838-851`). **Reuse**: one counter, one limit. The
  variable keeps its name; it is taught in three places and read by callers.
- `run(…, env)` (`internal/check/check.go:260`, `:316-323`). **Reuse**: `Run` passes the variable `RunSteps` passes,
  built by one helper both call.
- `askedStepsError` (`cmd/mrw/main.go:1595-1612`). **Reshape**: its depth clause uses the shared
  refusal the write and `mrw check` use.
- `checkDue` (`cmd/mrw/main.go:1301-1311`) and `writeCheckPaths` (`:1482-1511`). **Reshape**: the
  rule that decides whether a write's check is due is factored so the pre-apply guard and the
  post-apply decision are one predicate.
- `refuseWith` (`cmd/mrw/main.go:1072-1110`). **Reuse**: a refusal after the parse and before the
  apply already counts `refused_apply` (`:1083-1085`, ADR-083) and renders ADR-072's single JSON
  document.
- `subproc.group` and `subproc.reap` (`internal/subproc/subproc_unix.go:12-27`). **Reshape**: both
  go through one stop per command, which runs at most once: `Run` reaps after every exit
  (`internal/subproc/subproc.go:77-80`), a cancelled one included, so a stateless routine both
  called would signal a group its cancel had already seen empty.
- `subproc.Signals`, `subproc.Interruptible` (`internal/subproc/subproc.go:41-69`). **Reuse
  unchanged**: the nested mrw's side already works (probe 2).
- `waitDelay` (`internal/subproc/subproc.go:29`, one second). **Reuse** as the grace; no new knob.
- **The classes, enumerated 2026-09-29 at 8cbb89e:**
  - Project commands mrw starts — `mrw read --grep 'run\(ctx, root, cfg' --exclude '*_test.go' internal`
    → 2: `internal/check/check.go:242` (the check), `:792` (each step). Their CLI callers —
    `mrw read --grep 'check\.Run\(|check\.RunSteps\(' --exclude '*_test.go' cmd internal` → 3:
    `cmd/mrw/main.go:1384` (write's check), `:1666` (steps), `:1935` (`mrw check`). The MCP surface
    runs none (ADR-054, ADR-092). All covered.
  - Places mrw signals a child's group — `mrw read --grep 'syscall\.Kill\(' --exclude '*_test.go' cmd internal`
    → 2: `internal/subproc/subproc_unix.go:15` (cancel), `:25` (reap). What starts those children —
    `mrw read --grep 'subproc\.Command\(' --exclude '*_test.go' cmd internal` → 2:
    `internal/check/check.go:309` (check and steps), `internal/read/astgrep.go:89` (`--ast-grep`). All
    covered; ast-grep is in on purpose (Decision 5).

## Decision

1. **The check counts a level.** `check.Run` gives the check's shell `MRW_STEP_DEPTH` =
   `StepDepth()+1`, as `RunSteps` gives each step; one unexported helper builds that variable for both.
   The variable now counts every hop through a project command, check or step, on every platform.
2. **At depth 8 or more, mrw starts no project command**, and says so before anything is written or
   run, exit 2:
   - `--then`/`--then-sh`, as ADR-092 T5 already refuses;
   - a write whose check would be due, by ADR-054's rule — never under `--no-check` or `--dry-run`, and
     otherwise `--check`, or a non-prose path in a tree with a declared or inferred command — judged before the apply over the plan's own paths: every
     hunk path after pointer resolution (so an unlink target and a rename source count by their
     extension, as `writeCheckPaths` counts removed paths) and every rename destination. The same
     predicate decides `checkDue` after the apply, so the two cannot disagree for a plan that lands whole;
   - `mrw check`, in every form (named paths, the working set, `--full`), whether or not a check is
     declared or inferred: that command asks for nothing but a check.

   The refusal is one function in `internal/check`, and names the depth and the variable — shape:
   `a check is refused 8 deep (MRW_STEP_DEPTH=8): a check or step that runs mrw again would recurse
   without end`. What a caller at depth 8 sees:

   | Call at `MRW_STEP_DEPTH=8` | v1.31.0 | After |
   |---|---|---|
   | `mrw write P`, P touches a code file, a check is declared or inferred | writes, runs the check | exit 2; the refusal plus `--no-check writes without it: nothing was written`; tree unchanged; tallied `refused_apply`; `--json`: ADR-072's single refusal document, `applied` false, no `then` |
   | `mrw write --check P`, P prose only | writes, runs the check | exit 2, as above |
   | `mrw write --no-check P`, `--dry-run`, a prose-only plan, a tree with no check | lands, exit 0 | unchanged |
   | `mrw check`, any form | runs | exit 2, the refusal; `--json`: `{"error": …}` |
   | `--then` / `--then-sh` | exit 2 | exit 2, the shared wording (still names `MRW_STEP_DEPTH`) |
   | `mrw read`, `mrw_read`, `mrw_write` | unchanged | unchanged — they start no project command |

   At depth 7 each of these runs, and the check or step it starts sees 8.
3. **An unset or unreadable value counts as zero, and is not refused** — `""`, `x`, `-3`, ` 8`, a
   value `strconv.Atoi` overflows. The variable is mrw's own count; a value mrw did not write means
   the chain was already broken outside it (an export, a wrapper), exactly as `env -i` breaks it, and
   failing closed would refuse every check in that shell over a number the guard cannot trust either way.
4. **A group is stopped with TERM first, and once.** On unix, wherever `internal/subproc` stops a
   child's group — the cancel on a deadline or an interrupt, and ADR-080's reap after an exit — it
   sends SIGTERM to the group, polls it (`kill(-pgid, 0)`, every few milliseconds) until no member is
   left or `waitDelay` has passed since the TERM, and only then sends SIGKILL to what is left. The poll
   reads ESRCH as empty and any other answer, EPERM included, as a member left. No signal is sent once
   the group has been seen empty: a signal after that can land on a reused id (the review of #241,
   `internal/subproc/subproc.go:83-88`). So the stop runs at most once per command: `Run` reaps after
   every exit, a cancelled one included (`internal/subproc/subproc.go:77-80`), and a reap after a
   cancel sends nothing. The cancel returns only when the stop does, so `Run` and `Output` return
   with the group empty or killed. A nested mrw hears the TERM while its check or steps run, stops its
   own group the same way, and exits 3 "interrupted" — so the outermost timeout now reaches every
   level, instead of each level's timeout bounding only its own. A process that exits 0 from a TERM
   trap is still reported timed out or interrupted: after a cancel the verdict comes from the context,
   never from the exit status (`internal/check/check.go:361-371`, `internal/read/astgrep.go:95-101`),
   and the cancel returns the TERM's error, which `os/exec` turns into the context's error or a
   wrapped one, never nil (go1.27.1 `src/os/exec/exec.go:815-831`). That path is newly reachable —
   nothing exited 0 from a SIGKILL — so T2 pins it for the check, a step and `--ast-grep`.
5. **Every child `subproc` starts gets it**: the check, each step, and `--ast-grep`. A hanging
   `ast-grep` is TERMed at 2 s on unix; one that ignores TERM is killed by 3 s (on Windows it is killed at 2 s, Decision 6). The real binary obeys TERM, and
   so does every fixture here: `mrw read --grep 'trap ' cmd internal` finds no TERM trap in a Go test
   (2026-09-29), and the only `trap` lines in `scripts/contract.sh` are its runner's own prologue
   (`:82`), not a check or a fake. So they still return at 2 s. One routine, no second constructor for
   one caller.
6. **What still escapes, said beside the `setsid` limit** (T3 teaches the first three; the Windows
   limit is taught already, ADR-080):
   - A command that clears or rewrites the environment — `env -i`, `sudo` with `env_reset`, a wrapper
     that drops variables, a check that sets `MRW_STEP_DEPTH` itself — restarts the count below it.
   - A process that calls `setsid` leaves the group and gets neither signal (ADR-080).
   - A leaf two levels down that ignores TERM keeps its mrw waiting that mrw's full grace; the outer
     grace started a moment earlier, ends first and KILLs that mrw, so the leaf can outlive both.
     Rated Low, as ADR-072:124 rates the outside sender; not closed (Alternatives).
   - A nested mrw listens for TERM only while its check or steps run; a TERM during its apply ends it
     by the default action, as the SIGKILL did.
   - **Windows** has no process groups (ADR-080). There a cancel still kills only the direct child
     (os/exec's default), nothing is TERMed first, and a nested mrw's check outlives it as any
     grandchild does. Decisions 1–3 hold on Windows unchanged; Decisions 4–5 change nothing there.

   Not on this list: a check that ignores TERM (`trap '' TERM`) does not silence an mrw it runs. The
   Go runtime installs its own SIGTERM handler whatever disposition it inherits — only SIGHUP and
   SIGINT stay ignored (go1.27.1 `src/runtime/signal_unix.go:155-164`) — so `subproc.Signals` keeps
   TERM, and that mrw stops its own check; the shell that ignored TERM is KILLed at the grace.

**What would falsify it.** (a) A declared check that runs `mrw check --full` again returning anything
but exit 3 with its deepest level recorded as 8, or not returning in bounded time — contract §181
builds exactly that, from a check script that stops itself past 12 levels, so a missing guard fails
the row instead of running away. (b) At depth 8, a `.go` write landing or its check marker appearing — §181's
pair, beside the `--no-check` write that must land. (c) After an outer check times out, the inner
check of the mrw it ran still alive — §182, with `sh` kept as the leader so the TERM kills it at once
and only the grace keeps the nested mrw alive long enough. (d) A check that ignores TERM not killed by
the grace — §182's pair. (e) A straggler left by a passing check dying without having heard TERM —
§183. Valid for the CLI on Linux (contract.sh) and on macOS (the probes above); Decisions 1–3 also on
Windows through the Go tests.

## Alternatives Considered

- **Land the write and do not start its check** (exit 2, tallied `check_not_run`). Rejected: at the
  limit a tree would change unverified, and `--then` at the limit already refuses before writing
  (ADR-092 T5, ADR-072's order). One shape for every depth refusal.
- **Refuse every write at depth 8 unless `--no-check`.** Simpler, rejected: it refuses prose-only
  writes and trees with no check, and its message would name a check that would not have run.
- **Guard inside `check.Run`.** Rejected: on `write` the plan has landed by then. The two CLI call
  sites are the whole class (Existing Primitives Audit), and both guard before anything runs.
- **A second counter (`MRW_CHECK_DEPTH`), or renaming the variable.** Rejected: two counters each
  bounded at 8 let a chain that alternates check and step hops reach 64; a rename breaks the callers
  it is taught to.
- **A chain-wide deadline passed down in the environment.** Rejected: Decision 4 carries the outermost
  timeout down through the signal every mrw already listens for, and a variable is one more thing
  `env -i` drops.
- **SIGINT instead of SIGTERM.** Rejected: a non-interactive shell starts a `&` job with SIGINT
  ignored, and mrw keeps an ignored signal ignored, so `mrw check &` inside a check would never hear it.
- **Swap SIGKILL for SIGTERM in the cancel alone.** Rejected — the reap kills the nested mrw as soon as
  an `sh` leader dies (Context, "Why swapping the signal is not enough"). T2's end-to-end test keeps
  `sh` the leader for this reason.
- **Sleep the grace, then KILL.** Rejected: a KILL after the group emptied can land on a reused id
  (#241); poll, and stop at the first empty answer.
- **Scale the grace with depth, so each level outwaits the one below.** Rejected: eight levels of one
  second at the top, paid by every cancelled nested chain, for the rare TERM-ignoring leaf Decision 6
  names.
- **Keep `--ast-grep` on an immediate KILL.** Rejected: a second constructor in `subproc` for one
  caller, and an `ast-grep` that obeys TERM keeps its 2 s.
- **Leave ADR-080's reap on SIGKILL.** Rejected: `mrw check & exit 0` inside a check puts a nested mrw
  in the reap path, the same orphan; one routine serves both.

## Component / Boundary Impact

No architecture document exists in this repository; the impact is stated here. `internal/check` (an
engine package; this record owns its change: the environment helper, `Run`'s call, the refusal
function) · `internal/subproc` (the stop routine; not an engine package) · `cmd/mrw` (the write's
pre-apply guard, `mrw check`'s guard, the factored predicate) · `internal/guide` (the teaching). Each
keeps one reason to change. `internal/read` changes behaviour through `subproc` for `--ast-grep`
without a line changing there. Byte-identical against the merge-base in every task: `internal/read`,
`internal/apply`, `internal/plan`, `internal/seen`, `internal/state`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a check's environment | `MRW_STEP_DEPTH` = caller's + 1 (a step's already was) | `internal/check` (T1) | project checks; a nested mrw |
| `mrw write`, `mrw check` at `MRW_STEP_DEPTH` ≥ 8 | refused, exit 2, before anything is written or run | `cmd/mrw` (T1) | callers, hooks |
| `write --json` at the limit | ADR-072's single refusal document: `applied` false, `files` and `hunks` empty, `error`, no `then` | `cmd/mrw` (T1) | hooks |
| `check --json` at the limit | one document holding only `error`, as `check`'s other refusals (`cmd/mrw/main.go:1915-1924`) | `cmd/mrw` (T1) | hooks |
| `mrw stats` | the write refusal is `refused_apply` (after the parse, ADR-083) | `cmd/mrw` (T1) | callers |
| process-group signals on unix | SIGTERM, then SIGKILL at most `waitDelay` later; nothing after the group is empty; one stop per command, so the reap after a cancel sends nothing | `internal/subproc` (T2) | the check, steps, `--ast-grep` |
| `mrw instructions`, AGENTS.md, README.md | the check counts a level; what is refused at 8; what clears the count; TERM first; the ast-grep bound | T3 | callers; the centralised `mrw` skill mirrors AGENTS.md at the next release |
| ADR-058, ADR-072, ADR-074, ADR-080, ADR-092 | one "Amended by ADR-095" line each on the Decision this narrows; ADR-092's outcome table gains the depth row | T3 | readers, `adr-context` |
| contract | §181 (T1), §182 and §183 (T2) | T1, T2 | CI, `adr-verify` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `check.DepthRefusal`, the check's `MRW_STEP_DEPTH` | T1 | T3 | Yes for a caller already 8 deep (Served-path change); no for anyone shallower |
| `stopGroup` (TERM, poll, KILL) | T2 | T3 | No — a group that obeys TERM ends as it did |

## Implementation

See `docs/adr/ADR-095-a-nested-mrw-stays-bounded/tasks/README.md`. T1 and T2 are independent; T3
teaches both.

## Consequences

- **Positive:** a check that runs mrw is bounded like a step; the outermost timeout stops the whole
  chain; a check that traps TERM gets to clean up before it is killed.
- **Negative:** a caller already 8 deep that ran a plain code write is refused; a cancelled check,
  step or `--ast-grep` whose process ignores TERM returns up to a second later; a straggler left by a
  passing check that ignores TERM costs up to a second.
- **Neutral:** this repository's own check (`go test ./...`) now runs with `MRW_STEP_DEPTH=1` under
  `mrw write`, so a test that counts levels sets the variable itself (T1's tests do). At the limit, a
  plan that would have failed validation is refused for its depth first.

## Out of Scope

- Breadth under the cap — a check that runs mrw k times at each level can start k^8 processes in turn (permanent: boundary: the guard bounds depth; the outermost timeout, which Decision 4 carries down the chain, bounds time)
- Carrying the depth anywhere but the environment (permanent: boundary: the environment is the one channel that crosses `sh -c`, Git's `sh` on Windows and a wrapper without mrw's help; what clears it is named in Decision 6)
- Failing closed on an unreadable `MRW_STEP_DEPTH` (permanent: boundary: Decision 3 — a value mrw did not write is not evidence of depth)
- A TERM-ignoring leaf two levels down outliving both mrws (permanent: boundary: closing it means each level outwaits the one below, which compounds — Alternatives; rated Low as ADR-072 rates the outside sender)
- A depth guard or a check on the MCP surface (permanent: boundary: `mrw_write` runs no check (ADR-054) and no step (ADR-092), so it starts nothing a depth could bound)
- Stopping a grandchild on Windows, which needs a job object, delivered by ADR-120 on 2026-10-02: a nested mrw's check is in the same job object and stops with it (permanent: fact: delivered by ADR-120; citation: file `docs/adr/BACKLOG.md:2359`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller legitimately 8 deep is refused | Low | Med | the refusal names the variable and `--no-check`; depth 7 still runs (T1's pair); no known use nests mrw eight times |
| The pre-apply predicate drifts from `checkDue` | Med | Med | one predicate for both (Decision 2); T1 tests at the limit a `.go` replace, a `.go` unlink, a `.go` → `.md` rename, a `.md` → `.go` rename, `--check` on prose, a code write under an inferred check, and a prose-only plan and a code write in a tree with no check that both land |
| A ^C or a timeout returns up to a second later when the check ignores TERM | Med | Low | bounded by `waitDelay`; the KILL still comes |
| `--ast-grep`'s 2 s becomes 3 s for one that ignores TERM | Low | Low | the real binary and every fake obey TERM; the docs say so (T3) |
| A zombie whose live parent never waits keeps the poll busy for the whole grace | Low | Low | bounded at one second, then KILL, which is today's end state |
| ADR-094 (drafted in parallel) edits the step environment at `internal/check/check.go:792` and adds a refusal to `askedStepsError` (`cmd/mrw/main.go:1599-1612`), which T1 reshapes; ADR-096 owns `internal/read/astgrep.go`, whose lines Decision 4 cites | Med | Low | the variable is built in one helper and the depth clause is one call, so either merge is mechanical; a moved `astgrep.go` line is a citation to refresh, and each task's `internal/read` go/no-go is taken against its own branch's merge-base |

## Rollback

Revert the three tasks. Nothing persistent moves: an older binary ignores the check's
`MRW_STEP_DEPTH`, and the signal a group gets is not stored anywhere.

## Follow-ups

- [ ] Update the centralised `mrw` skill from AGENTS.md at the release that ships this record.
