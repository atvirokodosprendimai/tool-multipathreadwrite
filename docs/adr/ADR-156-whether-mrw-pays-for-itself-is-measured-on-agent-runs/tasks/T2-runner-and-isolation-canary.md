# Task ADR-156-T2: the runner, its receipts, configuration B's hook and the isolation canary

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** Zy
**Produces:** `bench run`, `bench verify`, `bench canary`, `bench hook`, the receipt format
**Consumes:** `internal/bench.Oracle` (T1), `internal/bench.LoadTask` (T1)
**Data dependency:** hermetic with a stub `claude`; the one real canary run needs Zy's word and his subscription login in the dedicated config directory
**Proof map:** v1
**Rests-on:** `a receipt per attempt`, `no overwrite of an earlier run`, `resume repeats no started attempt`, `configuration B's hook refuses other writes and counts reads`, `a slipped write is found from the ledger`, `the canary fails on a leak`, `the isolation holds on a real run`

## Goal

Decisions 1, 2, 4 and 6: the runner that starts the CLI, grades with the oracle and records a receipt, and the canary that proves isolation before any measured run.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/bench/main.go` | add | `run`, `canary`, `verify`, `hook`; `analyze` (T6) and `swebench-select` (T5) beside them |
| `internal/bench/runner.go` | add | the gates, a tree per attempt, seeded sequential order, the CLI's arguments, environment and `PATH`, the receipt, `run.json` and its refusal of another plan, resume, `Verify` |
| `internal/bench/transcript.go` | add | what bench reads from a stream-json run |
| `internal/bench/hook.go` | add | configuration B's PreToolUse hook (`bench hook`), its log, and the slipped-write check |
| `internal/bench/canary.go` | add | the leak checks over a stream-json transcript, and `RunCanary` |
| `internal/bench/runner_test.go`, `canary_test.go`, `hook_test.go` | add | the tests, with a stub `claude` |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestTheRunnerRecordsAReceiptPerAttemptFromAStubClaude`, `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts`, `TestTheRunnerResumesWithoutRepeatingAStartedAttempt`, `TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads` (a list of commands it must refuse, a list it must pass, `mrw …` among them, and reads it must count without refusing), `TestASlippedWriteIsFoundFromTheLedger`, `TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions` (a stub transcript that names `mrw` in configuration A, one that quotes a CLAUDE.md sentence, one with an MCP server in `init`: each fails; a clean one passes).
2. [S2] Runner and receipts. [proof: mutation]
3. [S3] Configuration B's hook and the slipped-write check. [proof: mutation]
4. [S4] The canary's checks. [proof: mutation]
5. [S5] The real canary, once, on Zy's word: Zy logs in once in the dedicated `CLAUDE_CONFIG_DIR`; `bench canary` runs one attempt per configuration on Sonnet; both pass, and Zy's own session still authenticates. If no configuration both isolates and authenticates with the subscription, stop and report (Decision 2). [proof: human: Zy observes the canary's verdict and his own session afterwards]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ ./cmd/bench/ -count=1 -timeout 600s -run 'TestTheRunnerRecordsAReceiptPerAttemptFromAStubClaude|TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts|TestTheRunnerResumesWithoutRepeatingAStartedAttempt|TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads|TestASlippedWriteIsFoundFromTheLedger|TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheRunnerRecordsAReceiptPerAttemptFromAStubClaude \(' "$out" \
  && grep -qE '^--- PASS: TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts \(' "$out" \
  && grep -qE '^--- PASS: TestTheRunnerResumesWithoutRepeatingAStartedAttempt \(' "$out" \
  && grep -qE '^--- PASS: TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads \(' "$out" \
  && grep -qE '^--- PASS: TestASlippedWriteIsFoundFromTheLedger \(' "$out" \
  && grep -qE '^--- PASS: TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions \(' "$out" \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRecordsAReceiptPerAttemptFromAStubClaude` | `internal/bench/runner_test.go` | one receipt per attempt with usage, oracle exit, diff stat, versions | none | S1, S2 |
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | an earlier run's receipts are never replaced | none | S1, S2 |
| `TestTheRunnerResumesWithoutRepeatingAStartedAttempt` | `internal/bench/runner_test.go` | resume skips finished and started attempts | none | S1, S2 |
| `TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads` | `internal/bench/hook_test.go` | the hook's refuse, pass and count lists | none | S1, S3 |
| `TestASlippedWriteIsFoundFromTheLedger` | `internal/bench/hook_test.go` | a changed file with no `w` record is reported | none | S1, S3 |
| `TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions` | `internal/bench/canary_test.go` | each leak fails the canary; a clean transcript passes | none | S1, S4 |
| `TestTheCanaryRunsBothConfigurationsAndRecordsItsVerdict` | `internal/bench/runner_test.go` | `RunCanary` with a stub: a clean stub passes, one naming mrw in A fails, and a stage refuses to start on it | none | — |
| `TestVerifyNamesAnAttemptWithNoReceipt` | `internal/bench/runner_test.go` | a stage verifies only with every planned receipt, and against its own counts | none | — |
| `TestTheRunnerRefusesWithoutTheGatesZyHolds` | `internal/bench/runner_test.go` | no containment, a dedicated user (not built until Zy chooses it), no canary, a failing canary, no config directory, or the user's own `~/.claude`: nothing starts | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `bench run`, `bench canary`, the hook |
| 2 — something selects it | T3, T4 and T5 run through `bench run`; `bench run` refuses to start a stage without a passing canary receipt for the same CLI version |
| 3 — the caller can discover it | `bench --help`; `docs/bench/README.md` |
| 4 — it is used | the canary run in S5 |

## Mutation Log
(empty until execute)
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/runner.go` · the receipt drops the cost the CLI reported · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:a receipt per attempt
- 2026-10-14 · 05ec3f84* · mutant inconclusive · exit 1 · `internal/bench/runner.go` · a run with another plan reuses an earlier run's directory · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:no overwrite of an earlier run
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/runner.go` · a started attempt with no receipt is run again · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:resume repeats no started attempt
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/hook.go` · the hook never refuses · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:configuration B's hook refuses other writes and counts reads
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/hook.go` · every changed file counts as written by mrw · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:a slipped write is found from the ledger
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/canary.go` · configuration A may name mrw · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:the canary fails on a leak
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/runner.go` · a run with another plan reuses an earlier run's directory · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · covers:no overwrite of an earlier run

## Verification Log
(empty until execute)
- 2026-10-14 · 05ec3f84* · exit 1 · `set -o pipefail …` · acceptance-sha256:3afbd305e98536e4f55ad761e9601c7000569e3e41dcb4fb70b85122a005f621 · ms:250 · test-lock-sha256:f31ecfa151dee5c5d064c0d0b4cf165a149f06e2187205bac735f4057bb0a853 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNhbmFyeUZhaWxzV2hlblRoZVRyYW5zY3JpcHROYW1lc01yd09yVXNlckluc3RydWN0aW9ucwkzMzNjOTU0NjExMjkwZjFhNjZlNWE5ZjMyMGI3ZmYzMGZiZGZjZDY3MTFiNGZjZTIwYzQ2YjA0ZDczYzFjNjFlCmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RBU2xpcHBlZFdyaXRlSXNGb3VuZEZyb21UaGVMZWRnZXIJNWE3MzVmMWNjMDlhNDE0YmFhNzUxMDA0MzA4NTBjN2QxMjdjY2MzNWViMTY1N2Y0ZmYxMmZmZGJlOTBjYTllNApib2R5CWludGVybmFsL2JlbmNoL2hvb2tfdGVzdC5nbwlUZXN0VGhlQXJtQkhvb2tSZWZ1c2VzQVJlYWRPcldyaXRlQnlBbnlNZWFuc0J1dE1ydwk5MjVlYjBlYjY2ZjlhODljYTM1YTFkNDBjMWU5N2IxMjAzZjVhNzY2NGEyNGQ4MjVjODdhNjgxNzJmZWI1ZmU5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlY29yZHNBUmVjZWlwdFBlckF0dGVtcHRGcm9tQVN0dWJDbGF1ZGUJMDUwOGIzMzc2Yzc2YjgwZGNkMjc0YjViNTM5YjdlYWM0YzQ0YzQ2NDI2YjYwZGM5MWZkNDllYmVjNTNlZjkwOQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5uZXJSZWZ1c2VzVG9PdmVyd3JpdGVBbkVhcmxpZXJSdW5zUmVjZWlwdHMJOGRjNDlhNzZlYzBkNDUyNjM0OWIwMTQ4ZDFlNjM1NDRhNjM3NTU1MjBlYjc1ODNlOWM3MTFmMjM5YzM3NDQ4Nwpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5uZXJSZWZ1c2VzV2l0aG91dFRoZUdhdGVzWnlIb2xkcwk1YWExNGQyNmVlOWEzYWJjMzM4NjljNDRjZjkwZGIwNzhhNmU2NTliNzAzYTY3YzQxMjA4NDA5MjFkNTY5M2Y4CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlc3VtZXNXaXRob3V0UmVwZWF0aW5nQVN0YXJ0ZWRBdHRlbXB0CWIyMDBkYzRlOTg5Mzk4YWU1ZjdmZWRjMGJmYmE2NDE2YWM0Yzg0OWI5YTNkNjgyZDg0OWZkYjA0MGEzMjdhMjQ
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  internal/bench/analyze_test.go:16:106: undefined: Verdict
  internal/bench/analyze_test.go:17:4: undefined: Receipt
  internal/bench/analyze_test.go:17:106: undefined: Verdict
  internal/bench/analyze_test.go:39:18: undefined: Receipt
  internal/bench/oracle_test.go:99:35: undefined: Task
  internal/bench/runner_test.go:47:45: undefined: Task
  internal/bench/runner_test.go:70:32: undefined: RunOptions
  internal/bench/analyze_test.go:39:18: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench [build failed]
  FAIL
  ```
- 2026-10-14 · 05ec3f84* · exit 1 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:332
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  internal/bench/analyze_test.go:16:106: undefined: Verdict
  internal/bench/helpers_test.go:12:9: undefined: Task
  internal/bench/helpers_test.go:23:9: undefined: Task
  internal/bench/helpers_test.go:42:9: undefined: Task
  internal/bench/oracle_test.go:100:35: undefined: Task
  internal/bench/runner_test.go:52:45: undefined: Task
  internal/bench/runner_test.go:75:32: undefined: RunOptions
  internal/bench/analyze_test.go:16:106: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench [build failed]
  FAIL
  ```
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:2995
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:2706
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:2572
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:2780
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:3778
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:4087
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:c16246fb5a76d8875fcd829a52be70b4d04cd2c35fc5cc65771b1b34428687a2 · ms:5764

## Invariants

- The CLI is started by absolute path with `PATH` set by `bench`; configuration A's `PATH` holds no mrw.
- Nothing is sent anywhere by `bench` itself (ADR-009).

## Risks

- The dedicated login disturbs Zy's own session: S5 checks it, and stops on any doubt.

## Out of Scope

- An API-key fallback (deferred: docs/adr/BACKLOG.md, "Pre-registered for ADR-156", only on Zy's word if the canary fails)

## Stop Condition

The fence exits 0 and S5's canary passed; or S5 shows no isolating configuration authenticates with the subscription, which stops the record.
