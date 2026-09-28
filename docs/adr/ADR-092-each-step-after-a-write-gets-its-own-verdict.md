# ADR-092: Each step after a write gets its own verdict

**Status:** Accepted
**Accepted:** 2026-09-28 by Zy — "accept", on the record as drafted after two Codex reviews; step authorship "Both, ad-hoc marked" chosen the same day
**Date:** 2026-09-28
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-009, ADR-016, ADR-044, ADR-054, ADR-056, ADR-059, ADR-062, ADR-069, ADR-072, ADR-080, ADR-082
**Governs:** `internal/check/check.go`, `cmd/mrw/main.go`, `internal/guide/guide.go`
**Enforced-by:** `internal/check/steps092_test.go::TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass`
**Invalidates:** none — checked. ADR-056's pricing keeps reading the declared check alone (Decision 7), so its pre-registered bar keeps its meaning; ADR-003's exit table is extended, not changed.
**Served-path change:** `mrw write` and `mrw check` take `--then NAME` (a step the project declares in `.quality-harness.json` `steps`) and `--then-sh 'CMD'` (an ad-hoc shell step), repeatable and run in command-line order after the write lands and its declared check passes; the receipt gains a `then` list with one verdict per step, the first step that does not pass stops the rest, which are named `not_run`, and a failed step exits 3.

## Context

A zeus finding filed into this wing's inbox on 2026-09-24 (`1aca63aa`, from zeus eval runs
`20260924T13*Z-full_mode-lmstudio-qwen-moe`, 3 runs at each of zeus commits e0e862e4 and 2589296a,
one local model, 12 small Python tasks) measured what a model does after an edit: of the 21 shell
calls zeus still refused at 2589296a, 16 were `&&` chains and 4 used `$(…)`, almost all
edit-then-verify — `python3 -c '…assert…' && python3 tests/test_m.py`,
`rm -f app.log && python3 demo.py && grep -q INFO app.log && echo ok`. The model wants ordered,
dependent steps where each success gates the next, with one answer at the end. zeus asked whether
mrw should offer that; M, 2026-09-28: *"write as proposed ADR, seems a needed feature"*.

mrw already chains ONE step to a write: the project's check (ADR-003, run by default since ADR-054).
Everything after it is the caller's own `&&` chain, where the verdicts ADR-003 exists for are lost:
a chain's exit status is the last command's (or the first failure's, unnamed), output is interleaved,
and a step that never ran looks like one that passed. The same finding's point 2 bounds the design:
zeus refused shell syntax, the model re-sent the chain as `bash -c '…'`, and "a refusal-based policy
the caller can route around is cost without safety" — so this record does not filter command words.

Asked who authors a step (2026-09-28), M chose **both**: steps the project declares, which the caller
names and orders, and ad-hoc shell under a separately marked flag.

## Existing Primitives Audit

- `check.Run` (`internal/check/check.go:194`) already does everything one step needs: output to a
  file, never a pipe (ADR-003 rule 1); `Ran` set only once a process existed (rule 2); a process group
  killed on timeout or signal (`subproc.Command`, ADR-072, ADR-080); Git's `sh` on Windows (ADR-082);
  the harness timeout, `fenceTimeout` alias included (ADR-059); a bounded tail; old logs pruned
  (ADR-080). **Reshape**: its body after the command line is chosen becomes one unexported runner,
  and `Run` calls it; the steps call the same runner. No second way to start a process.
- `check.Load` reads `.quality-harness.json` before anything is written (ADR-072). **Reuse**: it gains
  a `steps` object and validates it there, so an unknown or empty step refuses before a byte lands.
- The write command's receipt (`cmd/mrw/main.go:1372`) carries `check`; hooks read it. **Keep** it
  unchanged and add `then` beside it.
- The tally (`authoring.Reclassify`, ADR-009) already files a landed write as `failed_check` or
  `check_not_run`. **Reuse**, per Decision 7.
- Class enumerated 2026-09-28 — places mrw starts a child process and reports its verdict:
  `rg -n 'subproc\.Command|exec\.Command' --glob '!*_test.go' internal cmd` → 2 production sites:
  `internal/check/check.go:261` (the check; this record's runner) and `internal/read/astgrep.go:89`
  (ast-grep on the read path, ADR-058). ast-grep is deliberately left out: it is a finder whose output
  is served, not a verdict about the tree.

## Decision

1. **Two flags, one ordered list.** `--then NAME` runs the step the project declares as `NAME`;
   `--then-sh 'CMD'` runs `CMD` as given. Both are repeatable and share one list in the order they
   appear on the command line, so `--then vet --then-sh 'make e2e' --then contract` runs three steps
   in that order. Each value is one step: neither flag splits on commas. (urfave/cli v3.11.0 calls a
   flag value's `Set` in argument order; two value instances, one per flag, append to one collector.)
   A value holding a padded path-like argument follows ADR-069's guard as every other flag value does.
2. **Declared steps live in `.quality-harness.json`** as `"steps": {"vet": "go vet ./...",
   "contract": "./scripts/contract.sh"}`. A step name is non-empty and holds no whitespace; its command
   is non-empty. `check.Load` refuses otherwise. A `--then` naming a step the file does not declare,
   and a `--then-sh` whose command is empty or only whitespace, are refused, exit 2, naming the
   declared steps — both before anything is written (ADR-072's order).
3. **Where steps run.** On `write`, only after the plan LANDED: a failed hunk, `--dry-run` or a
   refusal runs no step. After the declared check when one is due (ADR-054, including `--check` on a
   prose plan) and only if it passed; with `--no-check`, the steps run without it. On `check`, after
   the check passes. Each step runs as the check does — `sh -c` in the root, Git's `sh` on Windows (a
   step is POSIX shell on every platform), output to a file, the harness's timeout per step, and on
   Unix its own process group killed on timeout or signal and reaped after it exits (ADR-072, ADR-080;
   on Windows, as for the check, a grandchild can outlive its step — a job object stays deferred).
4. **The first step that does not pass stops the sequence.** A step passes only if it ran and exited 0.
   A step that failed, timed out, was interrupted or could not start stops it, and every later step is
   `not_run`. One interrupt handler covers the whole sequence, so an interrupt while any step runs, or
   between two steps, marks that step `interrupted` (with `ran` saying whether its process had started)
   and the rest `not_run`; it never kills mrw with steps unreported. Old step logs are pruned once,
   after the last step, and the count is reported (below).
5. **The receipt.** `then` is an object: `{"steps": [...], "pruned_logs": N}`, each step
   `{name, command, adhoc, status, ran, exit_code, duration_ms, output_file, tail, truncated_lines,
   skipped}`, `status` one of `pass`, `fail`, `timed_out`, `interrupted`, `could_not_start`, `not_run`.
   `write --json` carries it beside the unchanged `check`; `check --json` keeps its flat `check.Result`
   fields and gains a top-level `then` beside them. `then` is present whenever a step was asked for and
   the command got as far as a receipt:

   | Outcome | Steps | Exit |
   |---|---|---|
   | unknown `--then` name, empty `--then-sh`, malformed `steps` | none; the ADR-072 single refusal document, no `then` | 2 |
   | a hunk failed | every step `not_run` | 1 |
   | `--dry-run` | every step `not_run` | 0 |
   | the check failed / timed out / was interrupted | every step `not_run` | 3 |
   | the check could not start | every step `not_run` | 2 |
   | a step failed, timed out or was interrupted | that step's status; later steps `not_run` | 3 |
   | a step could not start | `could_not_start`; later steps `not_run` | 2 |
   | every step passed | every step `pass` | 0 |

   The human receipt prints one line per step after the check's report — `then 2/3 contract:
   ./scripts/contract.sh — FAIL exit 1`, the tail and a `then last:` line for a failure,
   `then 3/3 e2e — NOT RUN` for the rest — and a `removed N check log(s)` line when steps pruned any.
6. **Exit codes extend ADR-003's table, unchanged in meaning** — as the outcome table says: a step that
   ran and did not pass, or was interrupted, is 3 (on `write` the tree is changed and unverified); one
   that could not start is 2, as a check that could not start is.
7. **The tally.** A landed write whose step ran and did not pass is `failed_check`; one whose step could
   not start, or was interrupted before its process started, is `check_not_run` — ADR-009's buckets,
   mapped as the check's are (ADR-080). ADR-056's strict-balance pricing keeps reading the declared
   check alone: a write whose check passed and whose step failed is priced `held`, because the bar was
   pre-registered against the check's verdict.
8. **What this grants.** `--then NAME` runs only what the project declared, as the check already
   does. `--then-sh` runs any shell the caller writes: **a harness rule that allows `mrw` without
   reading its arguments allows arbitrary shell through `--then-sh`**. The write help, `mrw
   instructions`, AGENTS.md and the README say so beside the flag. Nothing filters the command's words
   (the zeus lesson above).

**What would falsify it:** a sequence whose second step exits 1 while mrw reports exit 0, runs the
third step, or reports the third as passed. Contract §172 builds exactly that — the third step
touches a marker file, and the row asserts the marker is absent — plus a declared step, an unknown
name and a refused write that must run nothing. Valid for the CLI on Linux (contract.sh) and on every
CI platform through the Go tests.

**Amendment, 2026-09-28 (T5), after a stress round** — six local peer sessions drove the release
candidate d8576c3 (argv, JSON, signals, config, real use, concurrency). M chose, of the two open
questions, "Validate only when asked" and "Depth guard":

- Decision 2 is narrowed: `check.Load` holds `"steps"` as written, and `Config.StepCommands()`
  decodes and validates it only when a step is asked for by name. A typo in a block a write never
  uses no longer refuses that write, which is what the Neutral consequence below always promised.
- A step runs with `MRW_STEP_DEPTH` one higher than its caller's, and `--then`/`--then-sh` are
  refused (exit 2, nothing written) at depth 8, so a step that re-runs mrw with steps stops instead of
  recursing with each level resetting the step timeout.
- A step name or command holding control bytes is printed quoted; "could not start" is said once.
- Clarified, not changed: `could_not_start` means the shell could not start (a command the shell
  cannot find is `fail`, exit 127); a process that leaves the step's group (`setsid`) is not reaped;
  a signal mrw inherited as ignored stays ignored (ADR-072); an ad-hoc step has no `name`, and a
  step not run carries no `duration_ms`, `output_file` or `skipped`.
- The round's lower findings are in BACKLOG "From ADR-092".

## Alternatives Considered

- **Caller-named shell strings only (`--then 'CMD'`).** Closest to the zeus finding. Rejected as the
  only form: any harness allow rule for `mrw` would grant arbitrary shell with no flag saying so. M
  kept it, marked, as `--then-sh`.
- **Project-declared steps only.** Rejected as the only form: the finding's chains were ad-hoc checks
  the model chose (`python3 -c 'assert …'`), which no project would pre-declare.
- **A plan-level `@@ - then` hunk.** Rejected: a plan is edits, applied whole or not at all (ADR-001);
  a command inside it would make a plan something to run, and every plan reader would need to learn it.
- **Tell callers to put `a && b` in `check`.** It is what exists; rejected because the verdict is the
  chain's, with no step named and no step marked as not run.
- **Parse the chain and run its parts.** Rejected: zeus's own splitter needed three fixes Codex found
  (`#`, quoted `NAME=`, backslash-newline); a shell parses shell.

## Component / Boundary Impact

`internal/check` owns running a step and grading it; `cmd/mrw` owns the flags, the order, the receipt
and the exit code; `internal/guide` teaches it. `internal/check` is an engine package; this record owns
that change, and every other engine package stays byte-identical against 8ecb059.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `.quality-harness.json` | new optional `steps` object: name → shell command | project | `check.Load` |
| `mrw write`, `mrw check` flags | `--then NAME`, `--then-sh CMD`, repeatable, ordered together | `cmd/mrw` | callers |
| `write --json` / `check --json` receipt | new `then` object `{steps, pruned_logs}`; `check` on `write` and the flat fields on `check` unchanged | `cmd/mrw` | hooks, agents |
| human receipt | `then i/n …` lines after the check report | `cmd/mrw` | callers |
| exit codes | 3 for a step that ran and did not pass; 2 for one that could not start | `cmd/mrw` | callers |
| `mrw instructions`, AGENTS.md "Using mrw", README, `write --help` | teach both flags and the allow-rule caveat | T3 | callers; the centralised `mrw` skill mirrors AGENTS.md at the next release |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `check.Step`, `check.StepResult`, `check.RunSteps`, `Config.Steps` | T1 | T2 | No — additive |
| `--then`, `--then-sh`, the `then` receipt | T2 | T3 | No — additive |

## Implementation

See `tasks/README.md`.

## Consequences

- **Positive:** edit → check → step → step is one call with a verdict per step; a step that did not
  run is named, never passed; a project can name its gates once and a caller composes them.
- **Negative:** `--then-sh` makes an argument-blind `mrw` allow rule an arbitrary-shell allow rule;
  documented, not prevented.
- **Neutral:** without either flag nothing changes — receipt, exit code and tally are as today.

## Out of Scope

- Steps on the MCP surface (`mrw_write` runs no check today; running caller-named shell from a host that may have withheld shell is a trust decision of its own) (deferred: docs/adr/BACKLOG.md — "From ADR-092")
- Filtering or parsing a step's command words (permanent: boundary: a refusal the caller routes around is cost without safety — the zeus finding, point 2; confinement, not word filters, would be the boundary)
- Running steps in parallel, or continuing after a failure (permanent: boundary: the finding's chains are dependent — each success gates the next)
- A per-step timeout or tail setting (deferred: docs/adr/BACKLOG.md — "From ADR-092")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A harness allows `mrw` blind to its arguments and so grants shell through `--then-sh` | Med | High | Decision 8: named beside the flag in help and README; `--then NAME` is the declared-only path |
| Refactoring `check.Run` changes a check's verdict | Low | High | T1 keeps every existing `internal/check` test green in its fence; contract §§ for the check still run |
| A ^C between steps leaves a child running or a step unreported | Low | Med | one `Interruptible` over the sequence; T1 signals its own process mid-sequence and asserts the rest `not_run` |
| A step's grandchild outlives it on Windows | Med | Low | as for the check since ADR-072/ADR-080 — `subproc` has no process groups there; a job object stays deferred, and Decision 3 says so |
| An urfave/cli slice flag splits a step at a comma | Med | Med | both flags share an order-keeping value that never splits; T2 tests a comma |

## Rollback

Revert the tasks. `steps` in `.quality-harness.json` is ignored by a binary without this record, so a
project that declared steps keeps working, minus the flags.

## Follow-ups

- [ ] Update the centralised `mrw` skill from AGENTS.md at the release that ships this record.
