# Task ADR-092-T1: `internal/check` runs an ordered list of steps

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `check.Step`, `check.StepResult`, `check.StepsResult`, `check.RunSteps`, `Config.Steps`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `steps run in the order given`, `the first step that does not pass stops the rest`, `a step that could not start stops the rest`, `a cancelled sequence runs nothing`, `an interrupt between steps is caught and reported`, `a malformed step is refused when asked`, `pruned logs are counted`, `the check's own verdict is unchanged`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

One runner grades a list of shell steps in order, stops at the first that does not pass, and names
the rest `not_run`, using the same process handling as the check.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Config.Steps` validated in `Load`; `Run`'s body after choosing the command line becomes an unexported runner; `Step`, `StepResult` (with `Ran`), `StepsResult` (`Steps`, `Pruned`), `RunSteps` |
| `internal/check/steps092_test.go` | add | four platform-neutral tests |
| `internal/check/steps092_unix_test.go` | add | the interrupt test, which signals the test process |

## Ordered Steps

1. [S1] Write the five failing tests in the Tests table; they fail to compile before `RunSteps` exists. [proof: mutation]
2. [S2] Extract the runner from `Run` — timeout, `subproc.Command`, file output, verdict, tail, log
   removal — taking a command line and an already-interruptible context; `Run` wraps it in
   `Interruptible` as today and still prunes after the verdict. [proof: mutation]
3. [S3] `Config.Steps` (`json:"steps"`), read as it stands; `Config.StepCommands()` refuses an empty
   name, a name with whitespace, or an empty command, naming the step — only when a step is asked for
   (amended by T5: `Load` refused it for every write). [proof: mutation]
4. [S4] `RunSteps(ctx, root, cfg, []Step) StepsResult`: ONE `Interruptible` for the whole sequence;
   each step through the runner; a step's `status` from its result (`interrupted` keeps `Ran`); after
   the first non-pass every later step is `not_run`; a context already done before a step starts marks
   it `interrupted` with `Ran` false; an unexported `afterStep(ctx, i)` hook, a no-op in production and
   called after every step, is the seam a test uses to land a signal BETWEEN steps; prune once at the
   end into `Pruned`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ -count=1 -timeout 300s -run 'TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass|TestAStepThatCouldNotStartStopsTheSequence|TestACancelledSequenceReportsEveryStepNotRun|TestAnInterruptDuringASequenceStopsItAndNamesTheRest|TestAStepWithNoCommandIsRefusedWhenAsked' -v 2>&1 | tee /tmp/adr092-T1.out \
  && missing=$(for t in TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass TestAStepThatCouldNotStartStopsTheSequence TestACancelledSequenceReportsEveryStepNotRun TestAnInterruptDuringASequenceStopsItAndNamesTheRest TestAStepWithNoCommandIsRefusedWhenAsked; do grep -qE "^--- PASS: $t \(" /tmp/adr092-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/check/ -count=1 -timeout 300s \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

The fence runs on Unix, where the interrupt test exists; on Windows the Go suite in CI runs the four
platform-neutral tests.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass` | `internal/check/steps092_test.go` | three steps each append to a file; the second exits 1; the file holds steps 1 and 2 only; statuses are pass, fail, not_run with the real exit code; an old log planted in a temp dir is counted in `Pruned` | — | S1, S2, S4 |
| `TestAStepThatCouldNotStartStopsTheSequence` | `internal/check/steps092_test.go` | a step whose shell cannot start is `could_not_start`, and the next is `not_run` | — | S4 |
| `TestACancelledSequenceReportsEveryStepNotRun` | `internal/check/steps092_test.go` | a context cancelled before the call: the first step is `interrupted` with `Ran` false, the rest `not_run`, no marker written | — | S4 |
| `TestAnInterruptDuringASequenceStopsItAndNamesTheRest` | `internal/check/steps092_unix_test.go` | two subtests. DURING: step 1 passes, step 2 sleeps, the test sends SIGINT to its own process once step 2's start marker exists; step 2 is `interrupted` with `Ran` true, step 3 `not_run`, its marker absent. BETWEEN: `afterStep` after step 1 sends SIGINT and waits (bounded) for the sequence context to end; step 1 `pass`, step 2 `interrupted` with `Ran` false, step 3 `not_run`, no later marker. In both the test process survives — a handler scoped to one child leaves the BETWEEN signal unhandled, and the default action kills the test | — | S4 |
| `TestAStepWithNoCommandIsRefusedWhenAsked` | `internal/check/steps092_test.go` | asked for, an empty command, an empty name and a name with a space are each refused, naming the step; a sound step resolves | — | S3 |

The second command of the fence runs every existing `internal/check` test, which is what proves S2
changed no check verdict (`TestRunReportsTheRealExitCode` and its siblings).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `RunSteps` and its tests |
| 2 — something selects it | nothing yet — T2's `write` and `check` call it |
| 3 — the caller can discover it | n/a: no declared interface until T2 |
| 4 — it is used | telemetry is refused (ADR-009); the evidence for building it is the zeus finding `1aca63aa` |

## Mutation Log
(empty until execute)
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · the sequence goes on after a failed step · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:the first step that does not pass stops the rest
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · steps run in reverse order · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:steps run in the order given
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · a step that never started is graded fail · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:a step that could not start stops the rest
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · an interrupted step is graded fail · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:a cancelled sequence runs nothing
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · no handler over the sequence · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:an interrupt between steps is caught and reported
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · a step with no command loads · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:a malformed step is refused at load
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · the sequence prunes nothing · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:pruned logs are counted
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · the check stops pruning after the refactor · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:the check's own verdict is unchanged
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/check/check.go` · check.go is not gofmt-clean · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:the tree is gofmt-clean
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · covers:no other engine package changes
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · the sequence goes on after a failed step · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:the first step that does not pass stops the rest
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · steps run in reverse order · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:steps run in the order given
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · a step that never started is graded fail · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:a step that could not start stops the rest
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · an interrupted step is graded fail · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:a cancelled sequence runs nothing
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · no handler over the sequence · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:an interrupt between steps is caught and reported
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · an asked-for malformed step resolves · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:a malformed step is refused when asked
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · the sequence prunes nothing · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:pruned logs are counted
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · the check stops pruning after the refactor · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:the check's own verdict is unchanged
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · check.go is not gofmt-clean · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:the tree is gofmt-clean
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · covers:no other engine package changes

## Invariants

- `Run`'s result for every existing check case is unchanged.
- A step is started only through `subproc.Command`, as the check is.

## Risks

- Moving `Run`'s body drops a guard (the never-started `Ran`, the interrupted-before-start case). The
  second fence command runs every existing check test against it.

## Out of Scope

- The flags and receipt (T2's job)

## Stop Condition

The fence exits 0. Stop and ask if extracting the runner needs any existing `internal/check` test
changed.

## Verification Log
(empty until execute)
- 2026-09-28 · 8ecb059* · exit 1 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:484 · test-lock-sha256:9442c497265e200517d1f750df5b7eaf9111f2a29f58ba11a1e27012ade59d69 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3Rlc3QuZ28JVGVzdEFDYW5jZWxsZWRTZXF1ZW5jZVJlcG9ydHNFdmVyeVN0ZXBOb3RSdW4JMjkxYzJmNGU4OTkzOGU0YzE0MDM0ZGQxZWY3ZjFiNDcxNzNmMDg1YjA3NzM3NTJlYTU0ZjI4MWY3ZjFjNzgzYgpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3Rlc3QuZ28JVGVzdEFTdGVwVGhhdENvdWxkTm90U3RhcnRTdG9wc1RoZVNlcXVlbmNlCTRhNDNlZDcwNjdhYjU1YTdhMmUwMDY0N2UzN2ZhYWIzYWQ1Mzk1YTg1YTI2MTczZWYxNWZkZWZiNWI1YWVjNjgKYm9keQlpbnRlcm5hbC9jaGVjay9zdGVwczA5Ml90ZXN0LmdvCVRlc3RMb2FkUmVmdXNlc0FTdGVwV2l0aE5vQ29tbWFuZAk2ZjJmMTNjODFlNmM5MGUxNjQwNDJjZjdmZWM2NDU3YWE1OGJiM2QzOWQ1ZDBlNDA3NDY1MDhkNzBiMjU3OTM1CmJvZHkJaW50ZXJuYWwvY2hlY2svc3RlcHMwOTJfdGVzdC5nbwlUZXN0U3RlcHNSdW5Jbk9yZGVyQW5kU3RvcEF0VGhlRmlyc3RUaGF0RG9lc05vdFBhc3MJNmY4Y2VmMzJiYjU0NWY4NDc1ZGE3YTk4ZTZkZGJjNWZkNmE5Mzc2NmNmNDYyY2RjYmE2N2E5MThlYjBhNjY3OQpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3VuaXhfdGVzdC5nbwlUZXN0QW5JbnRlcnJ1cHREdXJpbmdBU2VxdWVuY2VTdG9wc0l0QW5kTmFtZXNUaGVSZXN0CTc2NmI1ZWRmNTI1OWU5MTcwN2E5MWNhZmEyZWZjMWNkNzQ0ZDYyNmEzMjc1NzI3YjE2YmQ0ZGY1Mzg3Yjk5NGEKYm9keQlpbnRlcm5hbC9jaGVjay9zdGVwczA5Ml91bml4X3Rlc3QuZ28JYmV0d2VlbgkzYWU1ZDIwZGUxMWM1MTNjNzRhNTRlZjIxMGU4OWMzZjU2ZTM1Mjc3MDFmZGIwY2RkNWNjN2MwYmQzM2Q2ZDBjCmJvZHkJaW50ZXJuYWwvY2hlY2svc3RlcHMwOTJfdW5peF90ZXN0LmdvCWR1cmluZwkwZDllNWE1NTY0ZmU0MmEzMTE2NjdhZWY1NTg4NzI3YmUzNDk1ZWU4NTFkYmNkMmM0M2E4OTVlOTAxNTljNDlh
  ```
  --- last 10 line(s) of stdout (of 27 after folding 27 raw)
  === RUN   TestAnInterruptDuringASequenceStopsItAndNamesTheRest/during
      steps092_unix_test.go:49: statuses [], want [pass interrupted not_run]
  === RUN   TestAnInterruptDuringASequenceStopsItAndNamesTheRest/between
      steps092_unix_test.go:80: statuses [], want [pass interrupted not_run]
  --- FAIL: TestAnInterruptDuringASequenceStopsItAndNamesTheRest (0.00s)
      --- FAIL: TestAnInterruptDuringASequenceStopsItAndNamesTheRest/during (0.00s)
      --- FAIL: TestAnInterruptDuringASequenceStopsItAndNamesTheRest/between (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.216s
  FAIL
  ```
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:5149
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:5002
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:5297
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:5372
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:5399
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:4933
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:4887
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:4921
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:4898
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:790d8715f576a90c8e982e8c6407dcbfe5e67a9c1778606ccfb6e24b99adfe5e · ms:4898
- 2026-09-28 · d8576c3* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:0 · test-lock-sha256:d95463c2033e581bb16dfcd878cd668c05a0da641cc215a22978db56206190eb · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3Rlc3QuZ28JVGVzdEFDYW5jZWxsZWRTZXF1ZW5jZVJlcG9ydHNFdmVyeVN0ZXBOb3RSdW4JMjkxYzJmNGU4OTkzOGU0YzE0MDM0ZGQxZWY3ZjFiNDcxNzNmMDg1YjA3NzM3NTJlYTU0ZjI4MWY3ZjFjNzgzYgpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3Rlc3QuZ28JVGVzdEFTdGVwVGhhdENvdWxkTm90U3RhcnRTdG9wc1RoZVNlcXVlbmNlCTRhNDNlZDcwNjdhYjU1YTdhMmUwMDY0N2UzN2ZhYWIzYWQ1Mzk1YTg1YTI2MTczZWYxNWZkZWZiNWI1YWVjNjgKYm9keQlpbnRlcm5hbC9jaGVjay9zdGVwczA5Ml90ZXN0LmdvCVRlc3RBU3RlcFdpdGhOb0NvbW1hbmRJc1JlZnVzZWRXaGVuQXNrZWQJMjQ4ZmU5YzVlZmEwMDU2ZWQ2ZTQ5OGJkN2Y1ZTNjNzg0MThmY2RkZGVlNWE0NDQyN2RjZjU2MTExOTg1YmRmZgpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3Rlc3QuZ28JVGVzdFN0ZXBzUnVuSW5PcmRlckFuZFN0b3BBdFRoZUZpcnN0VGhhdERvZXNOb3RQYXNzCTZmOGNlZjMyYmI1NDVmODQ3NWRhN2E5OGU2ZGRiYzVmZDZhOTM3NjZjZjQ2MmNkY2JhNjdhOTE4ZWIwYTY2NzkKYm9keQlpbnRlcm5hbC9jaGVjay9zdGVwczA5Ml91bml4X3Rlc3QuZ28JVGVzdEFuSW50ZXJydXB0RHVyaW5nQVNlcXVlbmNlU3RvcHNJdEFuZE5hbWVzVGhlUmVzdAk3NjZiNWVkZjUyNTllOTE3MDdhOTFjYWZhMmVmYzFjZDc0NGQ2MjZhMzI3NTcyN2IxNmJkNGRmNTM4N2I5OTRhCmJvZHkJaW50ZXJuYWwvY2hlY2svc3RlcHMwOTJfdW5peF90ZXN0LmdvCWJldHdlZW4JM2FlNWQyMGRlMTFjNTEzYzc0YTU0ZWYyMTBlODljM2Y1NmUzNTI3NzAxZmRiMGNkZDVjYzdjMGJkMzNkNmQwYwpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkyX3VuaXhfdGVzdC5nbwlkdXJpbmcJMGQ5ZTVhNTU2NGZlNDJhMzExNjY3YWVmNTU4ODcyN2JlMzQ5NWVlODUxZGJjZDJjNDNhODk1ZTkwMTU5YzQ5YQ · test-lock-kind:replace
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:5120
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4881
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4891
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4882
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4887
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4856
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4885
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4914
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4883
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:03b11445a882ce5d65a76dddb5b665100d42e145a1560288c9ce52eb37b42a22 · ms:4875
