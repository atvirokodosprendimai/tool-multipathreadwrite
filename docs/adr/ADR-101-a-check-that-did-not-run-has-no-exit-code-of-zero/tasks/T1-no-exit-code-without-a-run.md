# Task ADR-101-T1: A check that did not run reports `exit_code` -1

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `ExitCode` -1 on every `check.Result` with no exit status; contract §195; the ADR-100 BACKLOG entry closed
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `no status is -1`, `a real status is unchanged`, `a contract row drives the binary`, `only internal/check among the engine packages changes`

## Goal

`check.Run` returns `ExitCode` -1 when no command is found, when the scope is refused, and when the check's
log cannot be created; `mrw check --json` and `mrw write --check --json` show it; a check that ran still
reports its own status.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Run`'s two early returns (`:232`, `:236`) carry `ExitCode: -1`; `run` starts `res` at -1 (`:274`); the `ExitCode` field's comment says so |
| `internal/check/noexit101_test.go` | add | the unit test below |
| `cmd/mrw/noexit101_test.go` | add | the receipt test below |
| `scripts/contract.sh` | edit | §195 |
| `README.md`, `AGENTS.md` | edit | ADR-100's "read `ran` before `exit_code`" becomes "-1 when no check process exited" |
| `docs/adr/BACKLOG.md` | edit | the ADR-100 `exit_code` entry closed |

**What selects it:** `check.Run` is the one producer of `Result`; `checkCmd` and `writeCmd` serialize it.

## Ordered Steps

1. [S1] Write the failing tests `TestARunWithNoProcessHasExitCodeMinusOne` and
   `TestACheckThatDidNotRunReportsNoExitCodeOfZero`; confirm both RED on `e9620ea`. [proof: mutation]
2. [S2] Set -1 on the three paths. [proof: mutation] Mutants: the no-command return without -1; `run`'s
   `res` without -1; the refused-scope return without -1.
3. [S3] Contract §195, driving `$MRW`: in a tree with no check, `check --json --full` exits 2 with
   `ran` false and `exit_code` -1, and `write --check --json` carries the same `check` block; paired, a
   declared passing check reports `ran` true and `exit_code` 0.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §195's rows printed — the fence greps the section, since the whole contract takes minutes]
4. [S4] README, AGENTS and the BACKLOG entry updated. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestARunWithNoProcessHasExitCodeMinusOne|TestACheckThatDidNotRunReportsNoExitCodeOfZero' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestARunWithNoProcessHasExitCodeMinusOne \(' "$out" \
  && grep -qE '^--- PASS: TestACheckThatDidNotRunReportsNoExitCodeOfZero \(' "$out" \
  && grep -q '^# 195\. ' scripts/contract.sh \
  && grep -q 'Closed\*\* by ADR-101' docs/adr/BACKLOG.md \
  && [ -z "$(gofmt -l cmd/mrw internal/check)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARunWithNoProcessHasExitCodeMinusOne` | `internal/check/noexit101_test.go` | `Run` with no command (no harness, no `go.mod`), with a scope outside the root, and with a temp directory that does not exist each returns `Ran` false and `ExitCode` -1; paired, `Run` of `true` returns `Ran` true and `ExitCode` 0 | — | S1, S2 |
| `TestACheckThatDidNotRunReportsNoExitCodeOfZero` | `cmd/mrw/noexit101_test.go` | in a tree with no check, `check --json --full` exits 2 with one document, `ran` false and `exit_code` -1, and `write --check --json` exits 2 with its `check` block the same | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the -1 in `Run` and `run` |
| 2 — something selects it | `checkCmd` and `writeCmd`; the tests run `check.Run` and `rootCommand`, §195 the built binary |
| 3 — the caller can discover it | the receipts' `exit_code` |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 probe |

## Mutation Log
- 2026-09-30 · e9620ea* · mutant killed · exit 1 · `internal/check/check.go` · the no-command return reports exit_code 0 again · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · covers:no status is -1
- 2026-09-30 · e9620ea* · mutant killed · exit 1 · `internal/check/check.go` · run starts at 0 again, so the log that cannot be created reads exit 0 · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · covers:no status is -1
- 2026-09-30 · e9620ea* · mutant killed · exit 1 · `internal/check/check.go` · the refused-scope return reports exit_code 0 again · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · covers:no status is -1
- 2026-09-30 · e9620ea* · mutant killed · exit 1 · `internal/check/check.go` · a check that ran and exited 0 reports -1 · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · covers:a real status is unchanged
- 2026-09-30 · e9620ea* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · covers:only internal/check among the engine packages changes

## Invariants

- A check that ran reports its own exit status; `OK()` is unchanged.
- Exit codes and the human report are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The human report (permanent: boundary: ADR-101 Out of Scope)

## Verification Log
- 2026-09-30 · e9620ea* · exit 1 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:436 · test-lock-sha256:370da73e7e3259d63a20366b266d45b26045df121b3cef2b3195662aca7cd91e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbm9leGl0MTAxX3Rlc3QuZ28JVGVzdEFDaGVja1RoYXREaWROb3RSdW5SZXBvcnRzTm9FeGl0Q29kZU9mWmVybwk0NWRkZDAzYWNhMmEwYzAzMzYzMjFiOTEyOWFiYTg1NzM1MzJlZWVlNWQ0ZTJiNzU0MjQwMTBjNWZjNjNjZWI0CmJvZHkJaW50ZXJuYWwvY2hlY2svbm9leGl0MTAxX3Rlc3QuZ28JVGVzdEFSdW5XaXRoTm9Qcm9jZXNzSGFzRXhpdENvZGVNaW51c09uZQlmYWM5ZDQ5YmNiZGYwMjgwZDlmNmUyMTg4YzQxYWJiMDA4ZDkzY2JlM2U1ODA5YzU1Y2Y1Yjg0ODdjYjY1MWI3
  ```
  --- last 10 line(s) of stdout (of 59 after folding 59 raw)
            "pattern": {
              "advisory_writes": 0,
              "window": 1,
              "fires": false
            }
          }
  --- FAIL: TestACheckThatDidNotRunReportsNoExitCodeOfZero (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.084s
  FAIL
  ```
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:355
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:336
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:325
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:315
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:354
- 2026-09-30 · e9620ea* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ced7dfd5a14a6ac2781a96d6080dc7ee8e39879c8d1b836b8969d5089fe16c2 · ms:329
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped on this branch (base e9620ea), exit 0 'contract holds', with §195 printed: in a tree with no harness and no go.mod, check --json exits 2 with ran false and exit_code -1, write --check --json exits 2 with its check block ran false and exit_code -1; the pair, a declared passing check, ran true and exit_code 0
