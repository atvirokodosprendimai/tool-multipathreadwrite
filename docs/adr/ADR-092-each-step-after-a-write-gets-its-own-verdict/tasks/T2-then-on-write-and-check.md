# Task ADR-092-T2: `write` and `check` take `--then` and `--then-sh`

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `--then`, `--then-sh`, the `then` receipt; contract §172
**Consumes:** `check.RunSteps`, `check.Step`, `check.StepResult`, `check.StepsResult`, `Config.Steps` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `declared steps run in the order given`, `the two flags keep their command-line order`, `an unknown step is refused before anything is written`, `a failed step exits 3 and names the steps not run`, `a refused write runs no step`, `a failed check runs no step`, `check runs steps after a passing check`, `pricing reads the check alone`, `a comma does not split a step`, `a padded value follows the argument guard`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

A caller writes and verifies in one call — `mrw write plan --then vet --then-sh 'CMD'` — and reads
one verdict per step.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | both flags on `write` and `check`: two value instances appending to one collector, never splitting on commas; harness loaded when either is given; unknown name or empty `--then-sh` refused before `writer.Apply`; `RunSteps` after the check; `then` in the receipt (beside `check` on `write`, beside the flat fields on `check`); human lines; exit code and tally per the outcome table |
| `cmd/mrw/then092_test.go` | add | the ten tests below |
| `cmd/mrw/paddedstress_test.go` | edit | the random differential's value recorder reads the two `GenericFlag`s as it reads `StringFlag` |
| `scripts/contract.sh` | edit | §172 drives the built binary |

## Ordered Steps

1. [S1] Write the ten failing tests; they fail before the flags exist. [proof: mutation]
2. [S2] The collector and the two `GenericFlag`s on `write` and `check`; the padded-argument recorder
   extended to them. [proof: mutation]
3. [S3] On `write`: load the harness when either flag is given (ADR-072 order); refuse an unknown name
   or empty `--then-sh`, exit 2, before `writer.Apply`; after the check (or with `--no-check`),
   `RunSteps` only on a landed plan whose check passed; on `check`, after a passing check. [proof: mutation]
4. [S4] The `then` object, human `then i/n` lines and `removed N check log(s)` after `reportCheck`,
   exit and tally per the parent's outcome table; strict-balance pricing untouched. [proof: mutation]
5. [S5] Contract §172: all-pass exits 0; step 2 fails → exit 3 and step 3's marker absent; a declared
   step runs; an unknown name exits 2 with the file unchanged; a failed hunk runs no step. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §172's rows printed — the whole contract is minutes long and a fence runs at least three times per task, so the fence greps the section and lifecycle §6 runs it]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestThenRunsDeclaredStepsInTheOrderGiven|TestThenShAndThenKeepTheirCommandLineOrder|TestAnUnknownStepIsRefusedBeforeAnythingIsWritten|TestAFailedStepExitsThreeAndNamesTheStepsNotRun|TestARefusedWriteRunsNoStep|TestAFailedCheckRunsNoStep|TestCheckThenRunsStepsAfterAPassingCheck|TestAStepFailureIsPricedByTheCheckAlone|TestAValueWithACommaIsOneStep|TestAPaddedThenValueFollowsTheArgumentGuard' -v 2>&1 | tee /tmp/adr092-T2.out \
  && missing=$(for t in TestThenRunsDeclaredStepsInTheOrderGiven TestThenShAndThenKeepTheirCommandLineOrder TestAnUnknownStepIsRefusedBeforeAnythingIsWritten TestAFailedStepExitsThreeAndNamesTheStepsNotRun TestARefusedWriteRunsNoStep TestAFailedCheckRunsNoStep TestCheckThenRunsStepsAfterAPassingCheck TestAStepFailureIsPricedByTheCheckAlone TestAValueWithACommaIsOneStep TestAPaddedThenValueFollowsTheArgumentGuard; do grep -qE "^--- PASS: $t \(" /tmp/adr092-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestTheGuardsAgreeWithTheParserOnRandomArgv' \
  && grep -q '^# 172\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestThenRunsDeclaredStepsInTheOrderGiven` | `cmd/mrw/then092_test.go` | `--then b --then a` runs the declared b then a, both pass, exit 0; JSON `then.steps` lists both and `then.pruned_logs` is present with `--no-check` | — | S1, S3, S4 |
| `TestThenShAndThenKeepTheirCommandLineOrder` | `cmd/mrw/then092_test.go` | `--then a --then-sh X --then b` runs a, X, b in that order; `adhoc` true only for X | — | S2 |
| `TestAnUnknownStepIsRefusedBeforeAnythingIsWritten` | `cmd/mrw/then092_test.go` | an undeclared `--then` and a whitespace-only `--then-sh` each exit 2 naming the declared steps; the file is byte-identical; `--json` gives one refusal document | — | S3 |
| `TestAFailedStepExitsThreeAndNamesTheStepsNotRun` | `cmd/mrw/then092_test.go` | step 2 exits 1: exit 3, step 3's marker absent, statuses pass/fail/not_run in JSON and a `NOT RUN` line in the human receipt; the tally counts failed_check | — | S4 |
| `TestARefusedWriteRunsNoStep` | `cmd/mrw/then092_test.go` | a failed hunk (exit 1) and a `--dry-run` (exit 0) run no step, marker absent, every step `not_run` | — | S3 |
| `TestAFailedCheckRunsNoStep` | `cmd/mrw/then092_test.go` | a declared check that exits 1 on a `--check` prose write: exit 3, every step `not_run`, marker absent | — | S3 |
| `TestCheckThenRunsStepsAfterAPassingCheck` | `cmd/mrw/then092_test.go` | `mrw check --then a --json` runs a after a passing check; the flat check fields are unchanged and `then` sits beside them | — | S3, S4 |
| `TestAStepFailureIsPricedByTheCheckAlone` | `cmd/mrw/then092_test.go` | a write whose check passes and whose step fails: `mrw stats --json` pricing records `held`, not `broke` | — | S4 |
| `TestAValueWithACommaIsOneStep` | `cmd/mrw/then092_test.go` | `--then-sh 'printf a,b > m'` is one step and `m` holds `a,b` | — | S2 |
| `TestAPaddedThenValueFollowsTheArgumentGuard` | `cmd/mrw/then092_test.go` | an attached `--then-sh=' x '`-style value ending in whitespace is refused as ADR-069 refuses others; the separate form is kept | — | S2 |
| `TestTheGuardsAgreeWithTheParserOnRandomArgv` | `cmd/mrw/paddedstress_test.go` | the random differential still agrees with the parser once the new flags are in the recorder | — | S2 |

Two early Mutation Log rows are not kills. `a declared step runs a stand-in` was INCONCLUSIVE: it
left `c` unused and did not compile; `a declared step runs as an echo of itself` replaced it. `a dry
run runs the steps` SURVIVED because it was equivalent: `apply` returns `Applied` false for a dry run
(`internal/apply/apply.go:593`) and for a failed hunk, so `!res.DryRun` and `res.Failed == 0` beside
`res.Applied` could never change the gate. They were removed, and `a plan that did not land runs its
steps` kills the gate that remains.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the flags and the receipt |
| 2 — something selects it | `write` and `check` actions; contract §172 drives the built binary |
| 3 — the caller can discover it | `--help` lists both flags (T3 teaches them) |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the zeus finding `1aca63aa` |

## Mutation Log
(empty until execute)
- 2026-09-28 · 8ecb059* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · a declared step runs a stand-in · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:declared steps run in the order given
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · an ad-hoc step jumps the queue · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:the two flags keep their command-line order
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · an undeclared name runs as an empty step · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:an unknown step is refused before anything is written
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · a failed step exits 2 · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a failed step exits 3 and names the steps not run
- 2026-09-28 · 8ecb059* · mutant survived · exit 0 · `cmd/mrw/main.go` · a dry run runs the steps · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a refused write runs no step
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · steps run after a failed check · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a failed check runs no step
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · check never runs its steps · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:check runs steps after a passing check
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · a declared step runs as an echo of itself · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:declared steps run in the order given
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · a plan that did not land runs its steps · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a refused write runs no step
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · steps run after a failed check · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a failed check runs no step
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · an ad-hoc value splits at a comma · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a comma does not split a step
- 2026-09-28 · 8ecb059* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · no attached value is judged · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a padded value follows the argument guard
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:the tree is gofmt-clean
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:no other engine package changes
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · pricing reads the steps · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:pricing reads the check alone
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · no attached value is judged · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · covers:a padded value follows the argument guard

## Invariants

- Without `--then`/`--then-sh`, the receipt, exit code and tally are unchanged.
- `check` in the write receipt and the flat fields of `check --json` are unchanged; strict-balance
  pricing reads the check alone.

## Risks

- A new flag breaks the padded-argument differential (ADR-069); the fence runs it.

## Out of Scope

- Teaching the flags (T3's job)

## Stop Condition

The fence exits 0 and contract §172 passes. Stop and ask if urfave/cli cannot keep two flags in one
order without re-parsing `os.Args`.

## Verification Log
(empty until execute)
- 2026-09-28 · 8ecb059* · exit 1 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:836 · test-lock-sha256:7cbc20201734c9a416ef9f98ba66a529c6cb76fa8eba68aa3ed243633853bbe2 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkc3RyZXNzX3Rlc3QuZ28JVGVzdFRoZUd1YXJkc0FncmVlV2l0aFRoZVBhcnNlck9uUmFuZG9tQXJndglkNzlhNjFiYTE5MTdmYzllYjUxZjFiZTNlMTgwODRiNDRkNWY1N2Y2NWE1ZGM5MGU0NmU0NWY1MjExYmZhMzgxCmJvZHkJY21kL21ydy90aGVuMDkyX3Rlc3QuZ28JVGVzdEFGYWlsZWRDaGVja1J1bnNOb1N0ZXAJMDYwZDE2MmU3YzhiZWJmZTgyYmYxYjNkNDJiOTcyNWYzMGI2ZjdjYTkyY2MxZTE4NDVkMjU5OWUyODk0NTE1OApib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RBRmFpbGVkU3RlcEV4aXRzVGhyZWVBbmROYW1lc1RoZVN0ZXBzTm90UnVuCTIyYjE2MmJmYTFhNjJhMGMzMTVjYThlOGIyYzBmYThkZTUyZDU3MGJhMGJlYTI3Nzg5MjNhM2NhNzc2OTQ4ZjcKYm9keQljbWQvbXJ3L3RoZW4wOTJfdGVzdC5nbwlUZXN0QVBhZGRlZFRoZW5WYWx1ZUZvbGxvd3NUaGVBcmd1bWVudEd1YXJkCTNiOTdlZDAzYjdkZGI3MTI4NGE1ZTJmNDNmZGRjMGE2ZDNkZDVjNzU1MjljMjdjMzM1ZTg5YWFkODBhNmI3OGEKYm9keQljbWQvbXJ3L3RoZW4wOTJfdGVzdC5nbwlUZXN0QVJlZnVzZWRXcml0ZVJ1bnNOb1N0ZXAJNTY0MDMxNzRhMWM1YThjMDBjZmFkNmM4ZjJhNDg1NTY5YmQ0ODVlNjU3NjQwYmNmNzIwMGJhMzIzMzRiMzZhYgpib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RBU3RlcEZhaWx1cmVJc1ByaWNlZEJ5VGhlQ2hlY2tBbG9uZQk2Y2YyMzk5MWQ2NzkwNjRjMmYyODY1YmQxZjJiMTQ2NTIzYTMyNmZkZjM1ZGQzYzRlODEwMDQ2ZjA5OWNhYTk0CmJvZHkJY21kL21ydy90aGVuMDkyX3Rlc3QuZ28JVGVzdEFWYWx1ZVdpdGhBQ29tbWFJc09uZVN0ZXAJYjA0YjIyZDI0ZWFmM2ZlYjRmMmQ0ZjFjNjExM2JhYzFkZTMzNjNiYzU1OTE5MzUzOTY1Y2Y5YzY5MDk0MDEwYQpib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RBblVua25vd25TdGVwSXNSZWZ1c2VkQmVmb3JlQW55dGhpbmdJc1dyaXR0ZW4JODFiZDUzYWUyYzc1MWMwMzA3NTM1YWUzMWU5NDlmMzk5Mjk5NGFhMDVlOTg4ZjFkNzQ3NjY4NDVhY2MwZTM2OApib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RDaGVja1RoZW5SdW5zU3RlcHNBZnRlckFQYXNzaW5nQ2hlY2sJODdhNTJjOGE2ODE5Y2EwZDBiMWU4NGM4MDQwZDg4MDM5NThkYmU0OTFjYWNkMTdkMzc3ZjQzZDVjMjg3NjA1Ywpib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RUaGVuUnVuc0RlY2xhcmVkU3RlcHNJblRoZU9yZGVyR2l2ZW4JOGEyNTVjYzY2ZjMxYTU1YWZkZDIzYTcxNjg1N2VkYTZhNDhhZjBiYzg3M2JlNTRiNGZlZDdhOTYwMmNlZTY4Mgpib2R5CWNtZC9tcncvdGhlbjA5Ml90ZXN0LmdvCVRlc3RUaGVuU2hBbmRUaGVuS2VlcFRoZWlyQ29tbWFuZExpbmVPcmRlcgk3OGU5NDVkNDYzZWY3ZWY0YTU3YWY5NDgyYTY5MDA2NDZhYmY5OWRiMDNmMTk3Y2U5NGQ3MGZiNWI0MjY2OGEz
  ```
  --- last 10 line(s) of stdout (of 36 after folding 36 raw)
  --- FAIL: TestAStepFailureIsPricedByTheCheckAlone (0.00s)
  === RUN   TestAValueWithACommaIsOneStep
      then092_test.go:302: exit 2:
  --- FAIL: TestAValueWithACommaIsOneStep (0.00s)
  === RUN   TestAPaddedThenValueFollowsTheArgumentGuard
      then092_test.go:326: the separate form did not run as typed: exit 2, log "":
  --- FAIL: TestAPaddedThenValueFollowsTheArgumentGuard (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.285s
  FAIL
  ```
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:840
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:823
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:803
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:795
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:819
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:799
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:799
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:1167
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:825
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:786
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:796
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:791
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:770
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:779
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:774
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c19d2826067c9365fc247381c222a02b29927248ac8c9062b1c6f4cfbebaca2 · ms:793
- 2026-09-29 · human-observed · S5 observed: contract.sh unpiped exit 0 with §172 on main db39d42 on 2026-09-29 (this session, several runs); pre-merge, PR #266's CI test job (which runs contract.sh) passed
