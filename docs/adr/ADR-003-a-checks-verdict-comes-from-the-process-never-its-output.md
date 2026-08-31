# ADR-003: A check's verdict comes from the process, never its output

**Status:** Accepted
**Date:** 2026-08-31
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (owns the write this check runs after, and the exit statuses 0/1 that report the write itself), ADR-002 (owns the precondition that the write was allowed to happen at all)
**Governs:** `internal/check/**`, `cmd/mrw/main.go`, `.quality-harness.json`
**Enforced-by:** `internal/check/check_test.go::TestRunReportsTheRealExitCode`
**Invalidates:** none — checked (grepped ADR-001 and ADR-002 for `exit`, `check`, `revert`: ADR-001 defines exit 1 for a failed hunk and ADR-002 adds no status of its own; this record adds 2 and 3 and does not change 0 or 1)
**Served-path change:** `mrw write --check` applies the plan and then runs the project's own tests scoped to the files it touched, reporting a distinct exit status for "written but unverified" — collapsing edit, test, read-the-output from three tool calls into one.

## Context

**This is a retrofit**, shipped in `f0e12a9` with the exit-status corrections in
`6655113`, released as v0.0.1 and v0.0.2. The TDD-red step is historical; the
substitute proof is a mutation.

Editing, running the tests, and reading the output is three round trips and
three result blocks. It is the same "chain the check with the action" habit that
makes `gofmt -l x && go vet ./... && git commit` one call: the verification and
the thing it gates belong in one output.

The hazard is not the chaining, it is the reporting. Two failures observed
before this record, both from checks that lied:

- Piping a test run through `tail` makes the pipeline's exit status the tail's,
  so a failing suite exits 0. This repository's own verification rule exists
  because of it.
- A CI job reporting success in seconds was a guard step skipping, not a
  rollout. Reading the conclusion instead of the steps is the same error one
  level up.

## Existing Primitives Audit

- **`go test ./...` and the repo's declared check:** the thing being run.
  **Reused unchanged** — mrw does not reimplement a test runner, it runs the
  project's own command and reports its status.
- **`.quality-harness.json`:** an existing convention in this workspace for
  declaring a project's check. **Reused** rather than inventing a new config
  file.
- **`timeout(1)`:** would bound the run. **Rejected as the mechanism** — it is
  absent on macOS here, and `exec.CommandContext` does the same job in-process
  and portably.
- **`set -o pipefail` / `${PIPESTATUS[0]}`:** the shell-level answer to the same
  problem. **Reused as the principle**, not the implementation: mrw reads
  `ProcessState` directly, which has no pipeline to get wrong.

## Decision

`mrw check` and `mrw write --check` run the project's declared command and
report its verdict, under three rules:

1. **The exit status is never inferred from output.** Output goes to a file; a
   bounded tail is shown; the process's real `ProcessState` is reported. There
   is no pipe between the process and its verdict.
2. **A check that did not run is not a pass.** `OK()` is false when `Ran` is
   false. "No evidence" and "evidence of success" are different answers, and a
   missing check is exit 2 — a configuration problem — not exit 3.
3. **A red check never reverts.** The caller is told, with a distinct status.
   Undoing an edit for someone can destroy work they wanted to inspect.

The scope is derived from the written paths: `{packages}` expands to the Go
packages containing them, `{files}` to the paths. **If any changed path is not a
Go file the scoped form is abandoned for the full one** — a scoped run that
quietly omits a changed file is worse than a slow complete one.

The four exit statuses are a contract, because the caller's next move differs:

| code | meaning | next move |
|---|---|---|
| 0 | everything asked for succeeded | — |
| 1 | a hunk failed; **nothing written** | fix the plan |
| 2 | usage, parse, missing check, bad pointer | fix the call |
| 3 | a check ran and did not pass | read the test output |

**1 and 3 must never collapse.** 1 promises an untouched tree; 3 means the tree
changed and is unverified.

An inferred command (no `.quality-harness.json`, but a `go.mod`) is labelled
`inferred`, because an inferred check can be red on a tree the caller never
touched — and that finding is about the machine, not about the change.

**What would falsify rule 1:** a check whose output says `PASS` while the
process exits non-zero. `scripts/contract.sh` constructs exactly that case and
asserts mrw reports FAIL; if it ever reported PASS, the rule is broken. That
case is synthetic on purpose — the real-world version is a runner that dies
after printing its summary, which is rarer and harder to stage.

## Alternatives Considered

- **Parse the runner's output for pass/fail:** rejected — it is the exact
  failure being designed against, it is runner-specific, and a build failure
  prints `FAIL <pkg> [build failed]` with no `--- FAIL` line, so a parser
  counting those reads a package that does not compile as a pass.
- **Revert the write when the check fails:** rejected. It destroys work the
  caller may want to inspect, and it makes the tool's blast radius larger than
  the plan it was given. Reporting is enough; exit 3 says exactly what happened.
- **Always run the full project check:** rejected as the default because it
  throws away the round-trip saving on a large suite — but kept as the automatic
  fallback whenever a non-Go path is touched, and as `--full`.
- **Treat a missing check as a pass (exit 0):** rejected — it is the "green you
  did not compute" failure written into the tool.
- **Treat a missing check as exit 3:** rejected after being shipped that way.
  Nothing ran, so nothing failed; telling the caller to go read output that does
  not exist is the same imprecision one layer down. Corrected in `6655113`.
- **No `--check` at all; let the caller run tests:** rejected because it is the
  three-round-trip status quo, and because the caller who most needs the check
  is the one who will skip it.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/check` | Choosing the command, running it, grading it. Knows nothing about hunks or plans. | Yes — changes when check policy changes |
| `cmd/mrw` | Deciding WHEN to check (after a successful apply only) and mapping verdicts to exit statuses | Yes |

`internal/check` does not import `internal/apply`: the caller passes the written
paths as strings. A check is a thing you run over paths, not a phase of a write.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `.quality-harness.json` keys `check`, `scoped_check`, `timeout_seconds`, `tail_lines` | new public contract | the project | `internal/check` |
| `{packages}` / `{files}` placeholders in `scoped_check` | new public contract | `internal/check` | the project's config |
| `mrw write --check`, `mrw check`, `mrw check --full` | new public contract | `cmd/mrw` | callers, CI |
| Exit statuses 2 and 3 | new public contract | `cmd/mrw` | scripts, hooks, CI |
| `check.Result` in the `--json` receipt (`ran`, `exit_code`, `tail`, `output_file`) | new public contract | `internal/check` | hooks, quality gates |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `check.Config`, `check.Run`, `check.Result` (T1) | T1 | T2 | No — T2 maps the result to a status, it does not define it |

## Implementation

See `ADR-003-a-checks-verdict-comes-from-the-process-never-its-output/tasks/README.md`.
Two tasks: running and grading the check, and the exit-status contract.

## Consequences

- **Positive:** edit → test → read collapses to one call, and the verdict cannot
  be faked by output.
- **Positive:** "no evidence" is reportable, which is what makes an honest green
  distinguishable from an unexamined one.
- **Negative:** `mrw write --check` can leave the tree changed and red. That is
  the honest outcome and exit 3 names it, but it means the tool has a state a
  caller must act on rather than ignore.
- **Negative:** the scoped/full fallback is coarse — one Markdown file in the
  plan promotes the whole run to the full suite.
- **Neutral:** the output file is left behind in the system temp directory and
  is not cleaned up; it is named in the report so the caller can read it.

## Out of Scope

- Reverting or stashing a write when the check fails (permanent: reporting is
  the decision; a tool that undoes your work is a worse tool)
- Parsing runner output to attribute failures to specific tests (permanent:
  runner-specific, and rule 1 exists to stop mrw reading output for verdicts)
- Cleaning up the temp output files (deferred: docs/adr/BACKLOG.md)
- Per-language scope derivation beyond Go (deferred: docs/adr/BACKLOG.md)
- Running the check on a dry run (permanent: nothing changed, so there is
  nothing new to verify)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller reads exit 3 as "nothing happened" and re-runs the plan | Med | High — the tree is already changed, and ADR-002's ledger will now refuse the re-run, which is the safe direction | The message says "the tree is changed and unverified"; README and skill both state that 1 and 3 are opposites |
| An inferred check is red on an untouched tree and the caller blames their change | Med | Med | The report labels the command `inferred`; the README says the finding is about the machine |
| The scoped run omits a changed non-Go file | Low | High | It cannot: any non-Go path forces the full command. Asserted by `TestScopeDerivation`'s "non-Go falls back" case |
| Temp output files accumulate | High | Low | Small; deferred to BACKLOG |

## Rollback

No persistent state and no migration. Reverting means dropping the `--check`
flag and the `check` subcommand: `internal/check` is only reachable from
`cmd/mrw`, and `.quality-harness.json` is read-only input that other tools in
this workspace already use for their own purposes, so leaving the file in place
breaks nothing. Exit statuses 2 and 3 would disappear from the contract, which
is a breaking change for any script branching on them — as of 2026-08-31 the
only such consumer is `scripts/contract.sh` in this repository.

## Follow-ups

- [ ] Decide whether the temp output files should be cleaned up or moved under the repo (see BACKLOG.md)
