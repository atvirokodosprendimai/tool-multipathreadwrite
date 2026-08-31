# Task ADR-003-T1: The check runs scoped to what changed, and its verdict is the process's real exit code

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `check.Config` (T1), `check.Run` (T1), `check.Result` (T1), `check.Result.OK` (T1)
**Consumes:** none
**Data dependency:** hermetic

## Goal

`internal/check` reads the project's declared command, scopes it to the changed
paths, runs it with output to a file, and grades it from `ProcessState` — never
from what was printed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | add | Config loading, scope derivation, the run, the tail, the grade |
| `internal/check/check_test.go` | add | Real exit code, no-check-is-not-a-pass, tail announcement, timeout, scope derivation |
| `.quality-harness.json` | add | Declares this project's own check, so it is not inferred |
| `cmd/mrw/main.go` | edit | `checkCmd` and `write --check` are what SELECT this package |

## Ordered Steps

1. Confirm the failing tests exist and can go red. **Retrofit note:** the
   original red run is historical (`f0e12a9`); the substitute proof is
   `adr-verify --mutant` recording `killed` against the `ProcessState` read.
2. Load `.quality-harness.json`; fall back to a Go-shaped default only when a
   `go.mod` is present, and mark that config as NOT declared.
3. Derive scope: every changed path must be a `.go` file for the scoped form;
   otherwise return the full command. Dedupe and sort the package list.
4. Run via `exec.CommandContext` with an in-process timeout — not `timeout(1)`,
   which is absent on macOS here.
5. Send stdout and stderr to a temp FILE. Read the tail back from the file
   afterwards. Nothing may sit between the process and its exit code.
6. Grade: `OK()` is true only when the run happened AND the code is 0. A timeout
   is a failure, not a skip.
7. Report the tail with a count of what was withheld — a silent tail reads as
   the whole output.

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ -v 2>&1 | tee /tmp/adr003t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr003t1.out \
  && go test ./internal/check/
```

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestRunReportsTheRealExitCode` | `internal/check/check_test.go` | A command printing output and exiting 7 reports 7, and `OK()` is false | — |
| `TestNoCheckIsNotAPass` | `internal/check/check_test.go` | A config with no command reports `Ran: false`, `OK(): false`, and explains why | — |
| `TestTailAnnouncesWhatItLeftOut` | `internal/check/check_test.go` | 100 lines with `tail_lines: 5` reports 5 plus 95 withheld, and the FILE keeps all 100 | — |
| `TestTimeoutIsReportedAsAFailureNotAPass` | `internal/check/check_test.go` | A 5s command with a 1s timeout is not `OK()`, and says it timed out | — |
| `TestScopeDerivation` | `internal/check/check_test.go` | One package, two packages, dedup, root, and the non-Go fallback to the full command | — |
| `TestLoadInfersGoCheckButSaysSo` | `internal/check/check_test.go` | An inferred command reports `Declared() == false` | — |
| `TestLoadPrefersTheDeclaredCheck` | `internal/check/check_test.go` | `.quality-harness.json` wins and is marked declared | — |
| `TestLoadOnABareDirectoryHasNoCheck` | `internal/check/check_test.go` | No config and no go.mod invents nothing | — |
| `TestRunPasses` | `internal/check/check_test.go` | A passing command is `OK()` | — |
| `TestFilesPlaceholder` | `internal/check/check_test.go` | `{files}` expands to the changed paths | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestRunReportsTheRealExitCode` |
| 2 — something selects it | `cmd/mrw/main.go` `checkCmd` and the `--check` branch of `writeCmd` call `check.Run`; `scripts/contract.sh` rows 2, 3 and 5 fail if either call is removed |
| 3 — the caller can discover it | `.quality-harness.json`'s keys are the declared interface, documented in README; `mrw check --help` names `--full` |
| 4 — it is used | This repository's own `.quality-harness.json` is read on every `--check` run; nothing counts them |

## Mutation Log

## Invariants

- No pipe, ever, between the checked process and its recorded exit code.
- `Ran == false` implies `OK() == false`.
- The full output survives in the file even when the tail is trimmed.
- A scoped command is only used when EVERY changed path is a Go file.

## Risks

- Temp output files are never deleted. Small, named in the report, and deferred
  in the parent ADR rather than left unsaid.
- `sh -c` runs the declared command, so a project's config is arbitrary code
  execution by design — the same trust level as a Makefile in the repo.

## Stop Condition

Stop if a caller wants the verdict derived from output (a pattern, a JSON test
report). That is the exact thing rule 1 of the parent ADR forbids, and changing
it needs a superseding record.

## Out of Scope

- Mapping the result to a process exit status — that is T2.
- Cleaning up temp output files (deferred: docs/adr/BACKLOG.md)
- Scope derivation for languages other than Go (deferred: docs/adr/BACKLOG.md)

## Verification Log
