# Task ADR-177-T1: `mrw run 'CMD'` keeps the output in a log and prints one receipt

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw run`; its receipt, exit codes and `--json` type `run`; `<state home>/mrw-runs/`; the widened `guide.ThenShCaveat()`; contract §302
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the run tests run`, `the receipt test passed`, `the short-log test passed`, `the exit test passed`, `the could-not-start test passed`, `the one-argument test passed`, `the depth test passed`, `the interrupt test passed`, `the help test passed`, `the ceiling test passed`, `the prune test passed`, `the receipts test passed`, `the whole check and cmd packages pass`, `the receipts file holds run`, `the contract run`, `the own-log test passed`, `the completion-name test passed`, `the ceiling default test passed`, `the stopped-before-start test passed`, `the exit-code table passed`, `the usage-json test passed`, `the last-line test passed`, `the json-flag test passed`, `the boundary test passed`, `the json row passes`, `section 302 exists`, `the receipt row passes`, `the exit row passes`, `the read-line row passes`, `the two-argument row passes`, `the short-log row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1-9: `mrw run 'CMD'` runs CMD through an exported runner entry with the 24-hour ceiling and the interrupt handler, keeps every log in `<state home>/mrw-runs/` (renamed from `.running.log` when the command exits), prunes completed logs by age and count, prints the receipt with a working read line for the platform, exits with the command's code (128+N, 124, 130 named), carries the widened trust caveat in its help, and is refused at depth 8.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/run177_test.go` | new | the red tests driving the built binary |
| `internal/check/runlog177_test.go` | new | the red tests of the entry and of pruning |
| `internal/check/check.go` | edit | an exported entry over `run` (`:346`) taking log directory and name, timeout, keep-the-log, wrapping `subproc.Interruptible` (as `Run` at `:319`) and returning the wait status; the passing-log deletion (`:486`) skipped when keep-the-log is set; `pruneMatching` takes the match and a newest-N bound beside `pruneLogs`, which the check's calls and its test keep |
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
10. [S10] A run's own log is never pruned: 64 runs that finished before it do not remove it, and the exemption is the caller's named log. Mutant: `PruneRunLogs` ignores `own`. [proof: mutation]
11. [S11] The completed name is taken when the command ends, so name order is completion order. Mutant: the done name is taken before `Exec`. [proof: mutation]
12. [S12] The entry's zero timeout is `RunCeiling`. Mutant: `spec.Timeout = RunCeiling` dropped. [proof: mutation]
13. [S13] The exit mapping is a table, and an interrupt or deadline before the command starts is 130 or 124. Mutants: the 124 case of `runExitCode` deleted; the before-start switch returns 2. [proof: mutation]
14. [S14] A parser usage error under `--json` is one `{"error": …}` document. Mutant: the `usageError` hook removed. [proof: mutation]
15. [S15] `last` is judged over the whole line. Mutant: the whole-line blank test replaced by the kept prefix's. [proof: mutation]
16. [S16] Contract §302 parses stdout as exactly one document and drives a `--json` refusal through the binary. [proof: acceptance]
17. [S17] The protected log counts toward the 64: it and the newest 63 others stay, 64 in all. Mutant: `keep--` dropped, so 65 stay. [proof: mutation]
18. [S18] The `--json` fallback for a usage error reads the arguments as the parser reads a boolean flag. Mutant: the last-occurrence rule replaced by first-wins. [proof: mutation]
19. [S19] A multi-byte space cut by a read boundary does not make a blank line the last line. Mutant: the carried partial character dropped. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/check/ -count=1 -v -run '^(TestRunKeepsTheOutputAndPrintsOneReceipt|TestRunKeepsTheLogOfAShortOrEmptySuccess|TestRunExitsWithTheCommandsOwnCode|TestRunThatCannotStartSaysSoAndExitsTwo|TestRunTakesExactlyOneCommandString|TestRunIsRefusedAtTheStepDepthLimit|TestRunStoppedByAnInterruptExits130|TestRunHelpNamesTheShellTrust|TestTheRunEntryTakesItsOwnTimeout|TestRunLogsArePrunedByAgeAndCount|TestARunKeepsItsOwnLogWhenSixtyFourOthersFinishedBeforeIt|TestTheRunEntryWithNoTimeoutAsksForTheCeiling|TestRunStoppedBeforeItStartsExits130Or124|TestRunExitCodeNamesWhatStoppedTheCommand|TestRunUsageErrorUnderJSONIsOneRefusalDocument|TestRunLastLineIsJudgedWholeNotByItsKeptPrefix|TestRunLogIsNamedWhenTheCommandEnds|TestRunJSONFlagIsReadAsTheParserReadsIt|TestRunLastLineBlankIsJudgedAcrossFragmentBoundaries|TestNoShippedReceiptFieldDisappears)$' 2>&1) \
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
  && grep -qE '^--- PASS: TestARunKeepsItsOwnLogWhenSixtyFourOthersFinishedBeforeIt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheRunEntryWithNoTimeoutAsksForTheCeiling \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunStoppedBeforeItStartsExits130Or124 \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunExitCodeNamesWhatStoppedTheCommand \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunUsageErrorUnderJSONIsOneRefusalDocument \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunLastLineIsJudgedWholeNotByItsKeptPrefix \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunLogIsNamedWhenTheCommandEnds \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunJSONFlagIsReadAsTheParserReadsIt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestRunLastLineBlankIsJudgedAcrossFragmentBoundaries \(' <<<"$out" \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 >/dev/null \
  && grep -qx 'run output_file' docs/receipts.txt \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 302\.' scripts/contract.sh \
  && grep -qF 'PASS  mrw run prints a receipt, not the output, and keeps the output in its log' <<<"$c" \
  && grep -qF 'PASS  mrw run exits with the command'"'"'s own code' <<<"$c" \
  && grep -qF 'PASS  the read line mrw run prints reaches the log' <<<"$c" \
  && grep -qF 'PASS  mrw run with two arguments is refused' <<<"$c" \
  && grep -qF 'PASS  a short passing mrw run keeps its log' <<<"$c" \
  && grep -qF 'PASS  mrw run --json refuses a flag the parser rejects with one {error} document' <<<"$c" \
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
| `TestARunKeepsItsOwnLogWhenSixtyFourOthersFinishedBeforeIt` | `internal/check/runlog177_test.go` | 64 later-named logs leave the caller's own log; without the exemption the oldest name goes | none | S10 |
| `TestTheRunEntryWithNoTimeoutAsksForTheCeiling` | `internal/check/runlog177_test.go` | a zero timeout names `24h0m0s` under an outer 1 s deadline | none | S12 |
| `TestRunStoppedBeforeItStartsExits130Or124` | `cmd/mrw/run177_test.go` | a cancelled or expired context before the start is 130 or 124, with `--json` `ran` false, no command, no log | none | S13 |
| `TestRunExitCodeNamesWhatStoppedTheCommand` | `cmd/mrw/run177_test.go` | the exit mapping: timeout 124, interrupt 130, signal 128+N, the command's own code | none | S13 |
| `TestRunUsageErrorUnderJSONIsOneRefusalDocument` | `cmd/mrw/run177_test.go` | `--json --bogus` is one `{"error": …}` document, exit 2; without `--json` none | none | S14 |
| `TestRunLastLineIsJudgedWholeNotByItsKeptPrefix` | `cmd/mrw/run177_test.go` | 4,096 spaces then text is the last line (`…`); a blank long line is not | none | S15 |
| `TestRunLogIsNamedWhenTheCommandEnds` | `cmd/mrw/run177_test.go` | the log's name carries the time the command ended | none | S11 |
| `TestRunJSONFlagIsReadAsTheParserReadsIt` | `cmd/mrw/run177_test.go` | the raw `--json` forms: last wins, `=true`/`=1`/`=false`/`=0`, stop at `--`; through the binary's usage error | none | S18 |
| `TestRunLastLineBlankIsJudgedAcrossFragmentBoundaries` | `cmd/mrw/run177_test.go` | 4,097 spaces then 21,000 U+2003 after a meaningful line leave that line the last; the pair ending in FAIL is `…` | none | S19 |

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
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `internal/check/runlog177.go` · the caller's own log is no longer exempt from the prune (S10) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the own-log test passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the done name is taken from the start time, not the end (S11) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the completion-name test passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `internal/check/runlog177.go` · a zero timeout no longer becomes the 24-hour ceiling (S12) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the ceiling default test passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the 124 case of runExitCode is deleted (S13) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the exit-code table passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `cmd/mrw/run177.go` · an interrupt before the command starts exits 2 again (S13) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the stopped-before-start test passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `cmd/mrw/main.go` · the usage-error hook for mrw run --json is removed (S14) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the usage-json test passed
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `cmd/mrw/run177.go` · blankness judged from the kept prefix only (S15) · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · covers:the last-line test passed
- 2026-10-27 · e3f5ea87* · mutant killed · exit 1 · `internal/check/runlog177.go` · the protected log no longer counts toward the 64, so 65 stay (S17) · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · covers:the own-log test passed
- 2026-10-27 · e3f5ea87* · mutant killed · exit 1 · `cmd/mrw/run177.go` · a later --json=false no longer overrides an earlier true (S18) · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · covers:the json-flag test passed
- 2026-10-27 · e3f5ea87* · mutant killed · exit 1 · `cmd/mrw/run177.go` · the partial character is not carried across fragments (S19) · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · covers:the boundary test passed
- 2026-10-27 · 80bf726a* · mutant killed · exit 1 · `cmd/mrw/run177.go` · S18: an empty --json= value read as false · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · covers:the json-flag test passed

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
- 2026-10-27 · 5e342885* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:0 · test-lock-sha256:c61a6bda98b390b167a0dd95a2d4dadec6aa7665db2c325efadbdaa3b9d50c60 · test-lock-file:docs/adr/locks/c61a6bda98b390b167a0dd95a2d4dadec6aa7665db2c325efadbdaa3b9d50c60.lock · test-lock-kind:replace
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:134195
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:141817
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:141920
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:136820
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:156799
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:133794
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:136157
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:01e464a5eaa936fb18d2fc6f77022fe180a4346098cb561136712de13c10b46f · ms:144493
- 2026-10-27 · e3f5ea87* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:0 · test-lock-sha256:8c95d241bc7e9ca65bd47ba3798cc8a76b911e5c8969ab6964a69ba991b7f1ce · test-lock-file:docs/adr/locks/8c95d241bc7e9ca65bd47ba3798cc8a76b911e5c8969ab6964a69ba991b7f1ce.lock · test-lock-kind:replace
- 2026-10-27 · e3f5ea87* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:153899
- 2026-10-27 · e3f5ea87* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:189514
- 2026-10-27 · e3f5ea87* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:158221
- 2026-10-27 · e3f5ea87* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:142509
- 2026-10-27 · 80bf726a* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:0 · test-lock-sha256:a1c49e2414df1a53ff6e6376c3fbe6632de04c0acf1ddb2a3920141d2b6e4201 · test-lock-file:docs/adr/locks/a1c49e2414df1a53ff6e6376c3fbe6632de04c0acf1ddb2a3920141d2b6e4201.lock · test-lock-kind:replace
- 2026-10-27 · 80bf726a* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:139066
- 2026-10-27 · 80bf726a* · exit 0 · `set -o pipefail …` · acceptance-sha256:90c1da2c18aaca85f28f20991c3ef9c2dc29d0e2c255915b82925e1e6b89763c · ms:134823
