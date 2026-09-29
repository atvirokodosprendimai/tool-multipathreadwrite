# Task ADR-095-T2: A group is stopped with TERM first

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `stopGroup`, contract §182, contract §183
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a cancel sends TERM before KILL`, `a cancel waits for a slow member`, `a cancel kills what ignores TERM`, `a straggler after a clean exit hears TERM first`, `a reap after a cancel sends nothing`, `an EPERM answer is not an empty group`, `every group signal goes through the seam`, `a timeout reaches the check of a nested mrw`, `an exit 0 after a cancel is never a pass`, `the group tests of ADR-072, ADR-074 and ADR-080 still pass`, `the contract rows exist`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

On unix, wherever `internal/subproc` stops a child's process group — the cancel and the reap — it sends
SIGTERM, waits until the group is empty or `waitDelay` has passed, and only then sends SIGKILL, never
after the group was seen empty (ADR-095 Decisions 4–5).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/subproc/subproc_unix.go` | edit | `stopGroup(pid int) error`: SIGTERM to `-pid`, then `kill(-pid, 0)` every few milliseconds until ESRCH or `waitDelay` since the TERM — any other answer, EPERM included, is a member left — then SIGKILL to what is left, nothing after an ESRCH; it returns the TERM's error. Every signal goes through a package variable `signalGroup` (`syscall.Kill` in production), the seam the two seam tests replace. SELECTS: `group` sets `c.Cancel` to a `sync.Once`-guarded call of it (`:14-16`), and `reap` (`:23-26`) calls `c.Cancel()`, so a reap after a cancel sends nothing |
| `internal/subproc/subproc.go` | edit | the package, `Run` and `Output` comments say what a cancel and a reap send |
| `internal/subproc/stop095_unix_test.go` | add | the six mechanism tests (`//go:build unix`, as `subproc_unix_test.go` has) |
| `cmd/mrw/nested095_unix_test.go` | add | the end-to-end test through a built binary, and the ast-grep that answers on TERM (`//go:build unix`) |
| `internal/check/term095_unix_test.go` | add | a check and a step that exit 0 from a TERM trap are not a pass (`//go:build unix`); a test file only — `internal/check`'s code is T1's |
| `scripts/contract.sh` | edit | §182, §183 |

`internal/subproc/subproc_other.go` is unchanged: there are no groups on Windows (ADR-095 Decision 6).

## Ordered Steps

1. [S1] Write the nine tests in the Tests table. The four that need TERM to arrive first are red
   today, because the cancel and the reap send SIGKILL and no TERM trap ever runs; the two seam tests
   do not build until S2 adds `signalGroup`, and today's reap signals after the cancel has;
   `TestACancelKillsWhatIgnoresTerm` and the two exit-0 tests pass today and guard what T2 must not
   break. [proof: mutation]
2. [S2] `stopGroup` and `signalGroup`; `group`'s `Cancel` is a once-only call of `stopGroup`. The
   cancel returns only when `stopGroup` does, which `os/exec` waits for before `Wait` returns
   (go1.27.1 `src/os/exec/exec.go:944`, `:952`). [proof: mutation]
3. [S3] `reap` calls `c.Cancel()`: after a cancel the once has already run and nothing is sent; after
   a clean exit the TERM finds an empty group and nothing else is sent. [proof: mutation]
4. [S4] Contract §182: (a) a fixture whose check (`timeout_seconds` 3) is `"$MRW" -C inner check
   --full; true` — `sh` stays the leader — and whose `inner` check is `echo $$ > pid182; exec sleep
   300`: the outer run exits 3 and the inner pid is gone within 3 s, killed in the failing branch
   (the run-wide `pgrep -g $$` sweep cannot see it: it is in a group of its own); (b) a check `trap ''
   TERM; echo $$ > pid182b; sleep 300` with `timeout_seconds` 1 returns exit 3 within 10 s with that
   pid gone. Contract §183: a check that passes leaving `(trap 'echo term > marker183; exit 0' TERM;
   touch ready183; sleep 300 & wait) &` — the leader waiting for `ready183` before it exits 0 — exits
   0 with `marker183` written and the straggler gone. Each check script is written with `$MRW`'s absolute path in it: `$MRW` is not exported (`scripts/contract.sh:115`). [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §182's and §183's rows printed]

## Acceptance

```bash
set -o pipefail
go test ./internal/subproc/ ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestACancelSendsTermBeforeKill|TestACancelWaitsForASlowMemberToLeave|TestACancelKillsWhatIgnoresTerm|TestAStragglerAfterACleanExitGetsTermFirst|TestAReapAfterACancelSendsNothing|TestAnEPERMAnswerIsNotAnEmptyGroup|TestATimedOutCheckStopsTheCheckOfTheMrwItRan|TestACheckOrStepThatExitsZeroOnTermIsNotAPass|TestAnAstGrepThatAnswersOnTermStillTimesOut' -v 2>&1 | tee /tmp/adr095-T2.out \
  && missing=$(for t in TestACancelSendsTermBeforeKill TestACancelWaitsForASlowMemberToLeave TestACancelKillsWhatIgnoresTerm TestAStragglerAfterACleanExitGetsTermFirst TestAReapAfterACancelSendsNothing TestAnEPERMAnswerIsNotAnEmptyGroup TestATimedOutCheckStopsTheCheckOfTheMrwItRan TestACheckOrStepThatExitsZeroOnTermIsNotAPass TestAnAstGrepThatAnswersOnTermStillTimesOut; do grep -qE "^--- PASS: $t \(" /tmp/adr095-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/subproc/ ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestATimedOutCommandTakesItsGrandchildWithIt|TestAHeldPipeIsNotWaitedOnForever|TestAGrandchildOfACleanExitIsReaped|TestATimedOutCheckLeavesNoGrandchild|TestAnInterruptedCheckSaysSo|TestAHangingAstGrepTimesOut' -v 2>&1 | tee /tmp/adr095-T2-reg.out \
  && regmissing=$(for t in TestATimedOutCommandTakesItsGrandchildWithIt TestAHeldPipeIsNotWaitedOnForever TestAGrandchildOfACleanExitIsReaped TestATimedOutCheckLeavesNoGrandchild TestAnInterruptedCheckSaysSo TestAHangingAstGrepTimesOut; do grep -qE "^--- PASS: $t \(" /tmp/adr095-T2-reg.out || echo "$t"; done) \
  && [ -z "$regmissing" ] \
  && grep -q '^# 182\. ' scripts/contract.sh \
  && grep -q '^# 183\. ' scripts/contract.sh \
  && ! grep -qF 'syscall.Kill(' internal/subproc/subproc_unix.go \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && [ -z "$(git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACancelSendsTermBeforeKill` | `internal/subproc/stop095_unix_test.go` | a `Command` whose shell traps TERM to write a marker, cancelled once its pid file exists, leaves the marker when `Run` returns | — | S1, S2 |
| `TestACancelWaitsForASlowMemberToLeave` | `internal/subproc/stop095_unix_test.go` | the leader `sh` has no trap and dies of TERM at once; a member traps TERM, sleeps 0.3 s and writes a marker; with `Wait` running in a goroutine, the marker is there the moment `c.Cancel()` returns — no timing bound | — | S1, S2 |
| `TestACancelKillsWhatIgnoresTerm` | `internal/subproc/stop095_unix_test.go` | a group whose members ignore TERM is gone after `c.Cancel()` returns (polled with the file's `waitGone`, 3 s) | — | S1, S2 |
| `TestAStragglerAfterACleanExitGetsTermFirst` | `internal/subproc/stop095_unix_test.go` | a leader that exits 0 after its background member has installed a TERM trap: `Run` returns with the member's marker written and the member gone | — | S1, S3 |
| `TestAReapAfterACancelSendsNothing` | `internal/subproc/stop095_unix_test.go` | with `signalGroup` recording every call: a `Command` whose group obeys TERM, cancelled, then `Run` returning — no call follows the first ESRCH answer, and the reap adds none | — | S1, S2, S3 |
| `TestAnEPERMAnswerIsNotAnEmptyGroup` | `internal/subproc/stop095_unix_test.go` | with `signalGroup` answering EPERM to every poll, `stopGroup` returns no sooner than `waitDelay` after its TERM and has sent SIGKILL; answering ESRCH to the first poll, it sends nothing after it | — | S1, S2 |
| `TestATimedOutCheckStopsTheCheckOfTheMrwItRan` | `cmd/mrw/nested095_unix_test.go` | an in-process `mrw check --full` whose check (`timeout_seconds` 3) runs a built mrw on an inner tree, `sh` the leader, exits 3 "timed out", and the inner check's `sleep` pid is gone within 3 s; a `t.Cleanup` kills that pid if it survives, so a red run leaves nothing behind | — | S1, S2 |
| `TestACheckOrStepThatExitsZeroOnTermIsNotAPass` | `internal/check/term095_unix_test.go` | a check (`timeout_seconds` 1) and a step whose shell traps TERM with `exit 0` each come back `timed out` — `OK()` false, step status `timed_out` — never PASS (ADR-003 on a path T2 makes reachable) | — | S1, S2 |
| `TestAnAstGrepThatAnswersOnTermStillTimesOut` | `cmd/mrw/nested095_unix_test.go` | a fake `ast-grep` on PATH that prints a valid hit and exits 0 from a TERM trap is reported `timed out`, exit 2, and its hit is not served | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `stopGroup`; the three cancel tests |
| 2 — something selects it | `group`'s `Cancel` and `reap` in `internal/subproc/subproc_unix.go`, the only two group signals (ADR-095 audit); delete either call and its test goes red; §182 and §183 drive the built binary |
| 3 — the caller can discover it | n/a: no declared interface — T3 documents the behaviour |
| 4 — it is used | every check, step and `--ast-grep` run on unix goes through it; telemetry is refused (ADR-009) |

## Mutation Log
(empty until execute)
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the stop sends SIGKILL first, as before · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a cancel sends TERM before KILL
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the stop returns right after its TERM without waiting for the group to empty · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a cancel waits for a slow member
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · nothing is killed after the grace · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a cancel kills what ignores TERM
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the reap after a clean exit sends SIGKILL at once · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a straggler after a clean exit hears TERM first
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the stop runs on every call, so the reap after a cancel signals again · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a reap after a cancel sends nothing
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · any error answer, EPERM included, is read as an empty group · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:an EPERM answer is not an empty group
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the TERM bypasses the seam · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:every group signal goes through the seam
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · the TERM reaches only the leader, so a nested mrw never hears it before the KILL · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:a timeout reaches the check of a nested mrw
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · a process that exits 0 from its TERM trap is judged by its exit status · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:an exit 0 after a cancel is never a pass
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc.go` · Wait waits without bound for a pipe a grandchild holds · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:the group tests of ADR-072, ADR-074 and ADR-080 still pass
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `scripts/contract.sh` · the section header 182 is gone (the line becomes a no-op command) · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:the contract rows exist
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · subproc_unix.go is not gofmt-clean · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:the tree is gofmt-clean
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · covers:no other engine package changes

## Invariants

- A group that obeys TERM ends within a poll interval of how fast it did; one that ignores TERM ends up to `waitDelay` later (ADR-095 Consequences); a group already empty gets one TERM that finds none and nothing else.
- No signal is sent after the group was seen empty, and at most one stop runs per command.
- `Run` and `Output` never return while a member of the group they started is still running, except one that left it (`setsid`).
- Windows is unchanged.

## Risks

- Timing tests flake on a loaded machine. Mitigation: the mechanism tests assert state when
  `c.Cancel()` returns, not elapsed time; the end-to-end test bounds only the survivor check, at 3 s.
- A member that is a zombie with a live parent keeps the poll busy for the whole grace. Mitigation:
  bounded at `waitDelay`, then KILL, today's end state.

## Stop Condition

Stop and bring it back if ESRCH from `kill(-pgid, 0)` cannot be trusted to mean the group is empty on
a supported unix, or if an engine package has to change.

## Out of Scope

- The depth guard (that's T1's job)
- Teaching the behaviour and the record amendments (that's T3's job)

## Verification Log
(empty until execute)
- 2026-09-29 · 5794966* · exit 1 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8894 · test-lock-sha256:1096551c6f1ca21b2c112c07ca6744ca7d1262cc64f294fbe33967ea624fed0a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbmVzdGVkMDk1X3VuaXhfdGVzdC5nbwlUZXN0QVRpbWVkT3V0Q2hlY2tTdG9wc1RoZUNoZWNrT2ZUaGVNcndJdFJhbgljZGU0MGJhYjQyOGFhZWU5ZDFhY2QwNTdiM2U2YTI4ZDhmM2MyYWVlNDYxMDkwMDczYjVkZjMzZWE0MjJhNTRjCmJvZHkJY21kL21ydy9uZXN0ZWQwOTVfdW5peF90ZXN0LmdvCVRlc3RBbkFzdEdyZXBUaGF0QW5zd2Vyc09uVGVybVN0aWxsVGltZXNPdXQJZThlOWIyNjFlYzlhMTRjMzE1NjdmZDUxYmE2NDNkNDhjZTZhNWRjNTkwYjVlZDBmYjE4NDcyNzg2YTU3ZWNiZApib2R5CWludGVybmFsL2NoZWNrL3Rlcm0wOTVfdW5peF90ZXN0LmdvCVRlc3RBQ2hlY2tPclN0ZXBUaGF0RXhpdHNaZXJvT25UZXJtSXNOb3RBUGFzcwk1MWY5NDUxMDY1OWM0YjEyMGE1MjFiYmRmZjJmMzhhNzkyNTNmOGFmOTdhODgxMDdhODA4ZDhkZmVmNzM2OGExCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9zdG9wMDk1X3VuaXhfdGVzdC5nbwlUZXN0QUNhbmNlbEtpbGxzV2hhdElnbm9yZXNUZXJtCTM4ZDdiZTJiYThmOTE3MmZmY2E4YzM2YjIyNDhkMWJiZDk4ZTM5OTFhMTc0OWNmZDcyNTdkNjBhNTQ4NDljNDEKYm9keQlpbnRlcm5hbC9zdWJwcm9jL3N0b3AwOTVfdW5peF90ZXN0LmdvCVRlc3RBQ2FuY2VsU2VuZHNUZXJtQmVmb3JlS2lsbAlkZGQ4NjVlNjk0NTM2MzQ5Yzk1OWVjNTk3ZWVjOTZjMWE5M2IxYjYzNTJjZWZlNjBlYWQ5N2UxMjE2ZmY2YzYwCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9zdG9wMDk1X3VuaXhfdGVzdC5nbwlUZXN0QUNhbmNlbFdhaXRzRm9yQVNsb3dNZW1iZXJUb0xlYXZlCTViNDdkYTA5MTJjMDk3NDI1MDgzN2M1ODljNzkxNDY1YjU4MzVkMmNmNDI3MjU3NWYxYTkwZGQwZWRmYWY2NDAKYm9keQlpbnRlcm5hbC9zdWJwcm9jL3N0b3AwOTVfdW5peF90ZXN0LmdvCVRlc3RBUmVhcEFmdGVyQUNhbmNlbFNlbmRzTm90aGluZwkzNmJjMWE0YjhhZjdmYjA1YzQ2YWQ5OGRhMjE1N2QzMzIxZTM4NjFmMzdjZjhiYjExMzk3ZDcyZmE2MTgyZTM5CmJvZHkJaW50ZXJuYWwvc3VicHJvYy9zdG9wMDk1X3VuaXhfdGVzdC5nbwlUZXN0QVN0cmFnZ2xlckFmdGVyQUNsZWFuRXhpdEdldHNUZXJtRmlyc3QJMGYyYjI5Y2IzNDI0NTQyYTdkNjM2NWVkNzIxYzc0MDIwNmRjMDRkNDdiNDBhZjJjMjBmNzU2MmM0YThkMzM3ZApib2R5CWludGVybmFsL3N1YnByb2Mvc3RvcDA5NV91bml4X3Rlc3QuZ28JVGVzdEFuRVBFUk1BbnN3ZXJJc05vdEFuRW1wdHlHcm91cAkwMTdiNGJmZTRiYTg4ZmEzY2MzM2FjODNiY2M2YmI0NWM1MDE2NWY0MjczMjhiODQxOTc4OTM3NGJhNTgxNjgyCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9zdG9wMDk1X3VuaXhfdGVzdC5nbwlhbiBFU1JDSCBhbnN3ZXIgZW5kcyBpdAk5YzkwOGNiMmFhMTY3YzZiODg5ZDdhN2Q5ZTdmMTgzNjRhMDljYTc4ZWEzMjFiNDk3MzUyZWZiYzE4MWJkOTAy
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	2.118s
  === RUN   TestATimedOutCheckStopsTheCheckOfTheMrwItRan
      nested095_unix_test.go:55: the inner check's sleep 53539 outlived the outer timeout
  --- FAIL: TestATimedOutCheckStopsTheCheckOfTheMrwItRan (6.26s)
  === RUN   TestAnAstGrepThatAnswersOnTermStillTimesOut
  --- PASS: TestAnAstGrepThatAnswersOnTermStillTimesOut (2.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	8.377s
  FAIL
  ```
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8157
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8020
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8014
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8051
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8128
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8028
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8337
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8294
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8102
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8129
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8045
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8116
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b679d11d8d6d94bf610dcf5d366974c97950a0f064335784baaaa36aa665a08 · ms:8039
- 2026-09-29 · human-observed · S4 observed: PR #279 records 'contract.sh holds'; contract.sh exit 0 with §182 and §183 on main db39d42 on 2026-09-29 (this session)
