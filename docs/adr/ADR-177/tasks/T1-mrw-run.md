# Task ADR-177-T1: `mrw run 'CMD'` keeps the output in a log and prints one receipt

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw run`; its receipt, exit codes and `--json` type `run`; `<state home>/mrw-runs/`; the widened `guide.ThenShCaveat()`; contract §302
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the run tests run`, `the receipt test passed`, `the short-log test passed`, `the exit test passed`, `the could-not-start test passed`, `the one-argument test passed`, `the depth test passed`, `the interrupt test passed`, `the help test passed`, `the ceiling test passed`, `the prune test passed`, `the receipts test passed`, `the whole check and cmd packages pass`, `the receipts file holds run`, `the contract run`, `section 302 exists`, `the receipt row passes`, `the exit row passes`, `the read-line row passes`, `the two-argument row passes`, `the short-log row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1-9: `mrw run 'CMD'` runs CMD through an exported runner entry with the 24-hour ceiling and the interrupt handler, keeps every log in `<state home>/mrw-runs/` (renamed from `.running.log` when the command exits), prunes completed logs by age and count, prints the receipt with a working read line for the platform, exits with the command's code (128+N, 124, 130 named), carries the widened trust caveat in its help, and is refused at depth 8.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/run177_test.go` | new | the red tests driving the built binary |
| `internal/check/runlog177_test.go` | new | the red tests of the entry and of pruning |
| `internal/check/check.go` | edit | an exported entry over `run` (`:346`) taking log directory and name, timeout, keep-the-log, wrapping `subproc.Interruptible` (as `Run` at `:319`) and returning the wait status; the passing-log deletion (`:486`) skipped when keep-the-log is set; `pruneLogs` (`:258`) takes the pattern and a newest-N bound; the check's calls (`:338`, `:1087`) keep today's values |
| `cmd/mrw/main.go` | edit | the `run` subcommand in the command list, its one-argument rule, the depth refusal, the rename, the receipt per platform, the exit mapping, `--json` |
| `internal/guide/guide.go` | edit | `ThenShCaveat()` (`:141`) names `mrw run` beside `--then-sh` |
| `AGENTS.md` | edit | a backticked `mrw run` in §5, which `TestEverySubcommandReachesTheAgentFacingGuide` (`cmd/mrw/agentsdoc_test.go:29`) needs the moment the subcommand exists |
| `cmd/mrw/receipts111_test.go` | edit | the `run` receipt type held to `docs/receipts.txt` both ways |
| `docs/receipts.txt` | edit | the `run` legend line and its eleven rows |
| `scripts/contract.sh` | edit | §302 |

## Ordered Steps

1. [S1] Write the tests in the table below and see them RED: the receipt (a 500-line command: four receipt lines on stdout and none of the 500; the log holds all 500; the printed `read:` line, run verbatim with `XDG_STATE_HOME` under a directory whose name holds a space and a `'`, serves the last 40 lines); a short and an empty passing command keep their logs; exits 3, 7, `kill -TERM $$` → 143 (unix); could-not-start (no `sh` resolvable: exit 2, a `mrw:` line, no `exit` line; `--json` `ran` false, `exit_code` -1); two arguments (exit 2 naming `mrw run 'CMD'`; `--json` gives `{"error": …}`); depth 8 (exit 2, nothing run, no log made); an INT to mrw during `sleep 30` (unix: exit 130, `interrupted`, the group gone); `mrw run --help` holds `guide.ThenShCaveat()`; the entry with a 1-second timeout over `sleep 5` reports the timeout and `mrw run` asks for the 24-hour ceiling; pruning (70 completed logs, 3 older than 7 days, and a fresh `.running.log`: 3 + 3 removed, the newest 64 and the running one kept, a non-matching file untouched).
2. [S2] The entry and the subcommand. Mutant: the output goes to the receipt instead of the log. [proof: mutation]
3. [S3] Keeping every log. Mutant: the keep-the-log option ignored, so the check's deletion at `:486` removes a short passing log. [proof: mutation]
4. [S4] The exit code. Mutant: a signalled command reported as exit 0. [proof: mutation]
5. [S5] Retention. Mutant: `.running.log` files matched by the prune. [proof: mutation]
6. [S6] The depth guard. Mutant: the `DepthRefusal` call removed. [proof: mutation]
7. [S7] `--json` and `docs/receipts.txt`. Mutant: the `last` key renamed. [proof: mutation]
8. [S8] The read line. Mutant: the path printed unquoted. [proof: mutation]
9. [S9] Contract §302: the receipt and exit through the built binary, the `read:` line run as printed, a failing command's exit passed through, a short passing run's log kept, two arguments refused. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/check/ -count=1 -v -run '^(TestRunKeepsTheOutputAndPrintsOneReceipt|TestRunKeepsTheLogOfAShortOrEmptySuccess|TestRunExitsWithTheCommandsOwnCode|TestRunThatCannotStartSaysSoAndExitsTwo|TestRunTakesExactlyOneCommandString|TestRunIsRefusedAtTheStepDepthLimit|TestRunStoppedByAnInterruptExits130|TestRunHelpNamesTheShellTrust|TestTheRunEntryTakesItsOwnTimeout|TestRunLogsArePrunedByAgeAndCount|TestNoShippedReceiptFieldDisappears)$' 2>&1) \
  && grep -qE '^--- PASS: TestRunKeepsTheOutputAndPrintsOneReceipt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunKeepsTheLogOfAShortOrEmptySuccess \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunExitsWithTheCommandsOwnCode \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunThatCannotStartSaysSoAndExitsTwo \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunTakesExactlyOneCommandString \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunIsRefusedAtTheStepDepthLimit \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunStoppedByAnInterruptExits130 \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunHelpNamesTheShellTrust \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheRunEntryTakesItsOwnTimeout \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunLogsArePrunedByAgeAndCount \(' <<<"$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' <<<"$out" \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 >/dev/null \
  && grep -qx 'run output_file' docs/receipts.txt \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 302\.' scripts/contract.sh \
  && grep -qF 'PASS  mrw run prints a receipt, not the output, and keeps the output in its log' <<<"$c" \
  && grep -qF 'PASS  mrw run exits with the command'"'"'s own code' <<<"$c" \
  && grep -qF 'PASS  the read line mrw run prints reaches the log' <<<"$c" \
  && grep -qF 'PASS  mrw run with two arguments is refused' <<<"$c" \
  && grep -qF 'PASS  a short passing mrw run keeps its log' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRunKeepsTheOutputAndPrintsOneReceipt` | `cmd/mrw/run177_test.go` | receipt only on stdout; the log holds the output; the `read:` line works verbatim with a space and a quote in the path | none | S1, S2, S8 |
| `TestRunKeepsTheLogOfAShortOrEmptySuccess` | `cmd/mrw/run177_test.go` | a one-line and an empty passing run keep their logs | none | S1, S3 |
| `TestRunExitsWithTheCommandsOwnCode` | `cmd/mrw/run177_test.go` | 3, 7 and 128+signal pass through | none | S1, S4 |
| `TestRunThatCannotStartSaysSoAndExitsTwo` | `cmd/mrw/run177_test.go` | exit 2, `mrw:` line, no receipt; JSON `ran` false, `exit_code` -1 | none | S1, S7 |
| `TestRunTakesExactlyOneCommandString` | `cmd/mrw/run177_test.go` | two arguments refused naming the form; `{"error": …}` under `--json` | none | S1 |
| `TestRunIsRefusedAtTheStepDepthLimit` | `cmd/mrw/run177_test.go` | depth 8 refused, nothing run, no log | none | S1, S6 |
| `TestRunStoppedByAnInterruptExits130` | `cmd/mrw/run177_test.go` | an INT to mrw stops the group; exit 130; `interrupted` (unix) | none | S1, S4 |
| `TestRunHelpNamesTheShellTrust` | `cmd/mrw/run177_test.go` | `mrw run --help` holds `guide.ThenShCaveat()`, which names `mrw run` | none | S1 |
| `TestTheRunEntryTakesItsOwnTimeout` | `internal/check/runlog177_test.go` | the entry's timeout, not the check's; `mrw run` asks for the 24-hour ceiling | none | S1, S2 |
| `TestRunLogsArePrunedByAgeAndCount` | `internal/check/runlog177_test.go` | 7 days, then newest 64; a running log and other names untouched | none | S1, S5 |
| `TestNoShippedReceiptFieldDisappears` | `cmd/mrw/receipts111_test.go` | the `run` type's keys and `docs/receipts.txt` agree | none | S7 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the `run` subcommand in `cmd/mrw/main.go`'s command list; §302 drives the built binary |
| 3 — the caller can discover it | `mrw run --help` (fenced by `TestRunHelpNamesTheShellTrust`); AGENTS.md (held by `TestEverySubcommandReachesTheAgentFacingGuide`); the recipe in T2 |
| 4 — it is used | nothing measures this yet; the pre-registered re-measure (BACKLOG) is the first look |

## Mutation Log
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `internal/check/check.go` · the command output goes to mrw stdout instead of the log (S2) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the receipt test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `internal/check/check.go` · the keep-the-log option is ignored, so a short passing log is deleted (S3) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the short-log test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `cmd/mrw/run177.go` · a signalled command is reported as exit 0 (S4) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the exit test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `internal/check/runlog177.go` · running logs are matched by the completed-log prune (S5) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the prune test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the DepthRefusal call is removed (S6) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the depth test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the last key is renamed (S7) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the receipts test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the read line prints the path unquoted (S8) · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · covers:the receipt test passed

## Invariants

- The check, its temp-directory logs, its deletion of passing logs and its receipt are unchanged.
- Nothing is written in the checkout or inside `state.Base()` (unless the state home is itself in the checkout, Decision 3).
- No line of the command's output reaches stdout.

## Risks

- The could-not-start case needs an unresolvable shell, made by a PATH holding no `sh`; on Windows Git's fallback may still find one, so that case and the signal and interrupt cases run on unix, and the Windows read-line form is asserted by its text there.

## Stop Condition

The fence exits 0. Stop and ask if the entry needs a change to the check's own behaviour.

## Out of Scope

- The instructions beyond AGENTS.md's mention (T2's job).

## Verification Log
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:126663
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:131036
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:133476
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:130709
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:129413
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:129700
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:130579
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:129551
- 2026-10-27 · e9186d50* · exit 1 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:630 · test-lock-sha256:e39478eb743a348ba1b3fec70a4699a0e1fe70384d25cc9e4c77830aaebc5d78 · test-lock-file:docs/adr/locks/e39478eb743a348ba1b3fec70a4699a0e1fe70384d25cc9e4c77830aaebc5d78.lock
  ```
  ```
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:634dd474b5c72b9103ae155c87b7342ca128fe15ef2d69e8d80d7119dedde630 · ms:166357
