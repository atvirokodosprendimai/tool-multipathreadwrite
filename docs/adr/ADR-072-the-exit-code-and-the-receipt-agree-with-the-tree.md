# ADR-072: The exit code and the receipt agree with the tree

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"plan to address these, properly, no looping on small details"*; the plan that groups the v1.25.1 adversarial round into five records was approved the same day
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-003, ADR-009, ADR-054, ADR-055, ADR-056, ADR-058, ADR-059, ADR-066, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/harness_before_apply_test.go`, `cmd/mrw/receipt_before_check_test.go`, `cmd/mrw/json_refusal_test.go`, `internal/check/check.go`, `internal/check/group_unix_test.go`, `internal/subproc/**`, `internal/authoring/authoring.go`, `internal/authoring/reclassify_test.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/harness_before_apply_test.go::TestAMalformedHarnessRefusesTheWriteBeforeAnythingIsWritten`
**Invalidates:** ADR-059 Out of Scope, "Loading the harness before apply" (recorded as permanent; its citation had drifted from `cmd/mrw/main.go:1056` to `:1120`)
**Served-path change:** `mrw write` refuses before writing anything when `.quality-harness.json` cannot be read, exit 2. The human receipt is printed before the check starts, so a write killed during its check still says what landed, and `mrw stats` counts it. Under `--json` every refusal after the plan is named is a JSON document with an `error` field, a plan that does not parse included. A check that times out or is interrupted takes its process group with it on unix, and an interrupted check is reported as `interrupted`, exit 3.

## Context

**What was observed.** The v1.25.1 adversarial round (2026-09-25; `docs/adr/BACKLOG.md`, "From the
v1.25.1 adversarial round") found three ways the tree and the report disagree after a write.

1. **A malformed `.quality-harness.json` applied the write, then exited 2 with only the JSON
   error.** `check.Load` ran after `apply.Apply` committed (`cmd/mrw/main.go:1120`), so the tree
   changed and no receipt said so. ADR-059 had recorded loading earlier as permanently out of scope.
2. **A write killed during its check printed nothing.** The receipt was rendered only after
   `check.Run` returned (`:1134` then `:1153`). mrw bounds a check at five minutes by default
   (`internal/check/check.go:68`), but an outer kill before that printed zero bytes, and `mrw stats`
   never counted the landing because the tally was written after the check too.
3. **`--json` printed text on a plan that did not parse** (`:1013`), and on every other refusal
   between opening the plan and applying it.

And one the round's read side found in the same code shape: the check's timeout killed only `sh`
(`check.go:209`), so a check run as `sh -c 'make test'` left its children running.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `check.Load` | `internal/check/check.go:79-108` | Names the file in its error; pure read. Safe to call before apply. |
| `report`, `reportCheck` | `cmd/mrw/main.go:1443-1494` | Each flushes its own buffer on return. |
| `receipt` | `cmd/mrw/main.go:1263-1270` | `apply.Result` plus the check and the pattern; `files` and `hunks` carry no `omitempty`. |
| `authoring.Record` | `internal/authoring/authoring.go:143-166` | Load, increment, rewrite; errors swallowed by design. |
| `exec.Cmd.Cancel`, `WaitDelay` | Go `os/exec` | Cancel replaces the default kill; WaitDelay bounds the wait for pipes a grandchild holds. |

## Decision

1. **The harness is read before anything is written.** Unless `--no-check` or `--dry-run` is given,
   `check.Load` runs before `apply.Apply`; its error refuses the write, exit 2, and says nothing was
   written. A markdown-only plan is refused too: it was already broken by a bad harness (the load
   ran before the prose gate), and deciding "prose only" before apply would be a second reading of
   what the plan touches beside the receipt's.
2. **The human receipt is printed before the check runs, and the landing is counted before it.**
   The tally records `applied` before the check and moves that one count to `failed_check` or
   `check_not_run` after it (`authoring.Reclassify`), so a kill between the two leaves the write
   counted as landed. `--json` stays one document, rendered after the check.
3. **Under `--json` a refusal after the plan is named is a receipt with an `error` field.**
   `applied` false, `files` and `hunks` empty arrays, exit 2. A refusal after the write landed (the
   ledger, a check that could not start) carries the real result beside the error. Argument errors
   before any plan exists stay text.
4. **A check's child is stopped with its descendants.** `internal/subproc` starts a child in its own
   process group on unix and kills the group on cancel, and bounds the wait for held pipes at one
   second everywhere. While the check runs, an interrupt, terminate or hangup sent to mrw (unless the process started with it ignored, as nohup and a shell's background jobs do) cancels it: the
   group is killed and the check reports `interrupted`, exit 3, the tree changed and unverified. On
   Windows only the wait bound applies.

**What would make this decision fail:** a check that depends on sharing the terminal's process
group; its stdin is already `/dev/null` and its output a file, so none is known. And a signal
mrw cannot catch: a SIGKILL sent to mrw's process group, or ^Z, no longer reaches the check,
which runs on (or keeps running) with nothing left to enforce its timeout.

## Alternatives Considered

- **Keep the load after apply and print a receipt before the error.** Rejected: the write would
  still land in a tree whose verification cannot even be configured.
- **A process-wide signal handler in `main`.** Rejected: a first ^C during a long `read`, or under
  `mrw mcp`, would then do nothing visible. The handler lives only as long as a check runs.
- **Stream `--json` as several documents.** Rejected: a consumer parses one document; the ledger and
  the tally carry the landing when mrw is killed.

## Component / Boundary Impact

`internal/check` is an engine package and this record owns its run changes. `internal/subproc` is
new. `cmd/mrw` and `internal/authoring` are not engine code. Byte-identical: `internal/read`,
`internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`,
`internal/iter`, `internal/rooted`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` | harness read before apply; exit 2, nothing written | T1 | CLI callers, hooks |
| `mrw write` output | receipt before the check; `stats` counts a killed landing | T2 | CLI callers |
| `mrw write --json` | an `error` field on every post-plan refusal | T3 | hooks, quality gates |
| `check.Run` | group kill; `interrupted` | T4 | `mrw write`, `mrw check` |
| contract §142–§145 | T1–T4 | T1–T4 | CI, `adr-verify` |

## Inter-task Contracts

T3's refusal helper renders T1's refusal under `--json`; T2 reorders the code T1 and T3 touch, so
they land in the order T4, T1, T3, T2.

## Implementation

See `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree/tasks/README.md`.

## Consequences

- **Positive:** exit 2 after `mrw write` again means nothing was written, except where the receipt
  says what landed.
- **Negative:** a malformed harness blocks every write that would read it, prose included.
- **Neutral:** no exit code changes meaning.

## Out of Scope

- A partial `--json` document when mrw is killed (permanent: boundary: one document per call; the ledger and the tally carry the landing)
- Stopping a grandchild on Windows, which needs a job object (deferred: docs/adr/BACKLOG.md "From the v1.25.1 adversarial round", the killed-check entry)
- JSON for argument errors before a plan exists (permanent: fact: they precede any plan; citation: file `cmd/mrw/main.go:944`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A check reads the terminal | Low | Med | its stdin is `/dev/null`; a TTY read from a background group stops it, and the timeout still ends it |
| A second ^C during the one-second wait is swallowed | Low | Low | the wait is bounded; mrw exits right after |
| SIGKILL to mrw's process group, or ^Z (SIGTSTP), during a check: the check, in its own group, is neither killed nor stopped | Low | Med | no handler can catch SIGKILL or pass ^Z on; the check's own command finishes as it would have |
| ^\\ (SIGQUIT) during a check ends mrw with a goroutine dump and leaves the check running | Low | Low | deliberately not caught, so ^\\ keeps its dump |

## Rollback

Revert the four tasks. Nothing persistent moves.

## Follow-ups

- [x] ADR-074 T1 moves `--ast-grep` onto `internal/subproc`.
