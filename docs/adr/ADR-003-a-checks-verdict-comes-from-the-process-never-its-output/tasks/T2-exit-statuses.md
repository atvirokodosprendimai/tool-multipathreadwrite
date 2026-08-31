# Task ADR-003-T2: Four exit statuses, each meaning a different next move

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `mrw` exit statuses 0/1/2/3 (T2), `mrw check` subcommand (T2), `mrw write --check` (T2)
**Consumes:** `check.Result` (T1), `check.Result.OK` (T1)
**Data dependency:** hermetic

## Goal

`cmd/mrw` maps outcomes to four distinct statuses — 0 succeeded, 1 nothing
written, 2 fix the call, 3 a check ran and failed — and the check runs only
after a real, successful write.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | The status constants, the `--check` gating, and the three-way outcome switch |
| `scripts/contract.sh` | edit | Rows 2, 3 and 5 assert the statuses against the built binary |
| `README.md` | edit | The exit-status table — the declared interface |
| `.claude/skills/mrw/SKILL.md` | edit | Repeats the 1-vs-3 distinction where an agent reads it |

## Ordered Steps

1. Confirm the failing checks exist and can go red. **Retrofit note:** the
   original red run is historical; the substitute is `adr-verify --mutant`
   against the outcome switch. `scripts/contract.sh` rows 2/3/5 are the fence
   that would catch a collapse of 1 and 3.
2. Name the statuses as constants with a comment saying WHY they differ — the
   caller's next move, not a severity ordering.
3. Gate `--check` on `res.Applied && res.Failed == 0`: verifying a tree the plan
   did not touch would attribute someone else's red suite to this edit.
4. Map: any failed hunk → 1 (and say nothing was written); a check that could
   not RUN → 2, naming the config remedy; a check that ran and failed → 3, and
   say the tree is changed and unverified.
5. Make `mrw check` with no arguments scope to the working set, and `--full`
   ignore scope.
6. State in README and SKILL that 1 and 3 are opposites, because collapsing them
   is the mistake this contract exists to prevent.

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw && ./scripts/contract.sh 2>&1 | tee /tmp/adr003t2.out \
  && ! grep -qE "^  FAIL|assertion\(s\) FAILED" /tmp/adr003t2.out \
  && grep -q "output says PASS, process exits 1 -> reported FAIL" /tmp/adr003t2.out \
  && go test ./cmd/mrw/
```

The fence asserts the adversarial row is PRESENT as well as passing — a
contract script that silently stopped running that case would otherwise pass.

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestVersionFlagPrintsTheStampedVersion` | `cmd/mrw/version_test.go` | The CLI wiring is exercised at all — the only Go test in this package | — |
| `TestVersionIsAVariableTheLinkerCanStamp` | `cmd/mrw/version_test.go` | `version` stays a stampable string var | — |

The statuses themselves are asserted by `scripts/contract.sh` rather than by Go
tests: an exit status is a property of the PROCESS, and a test calling
`rootCommand().Run` in-process observes a returned error, not the status the
shell sees. That gap is real and named here rather than hidden — the contract
script is the only place the four statuses are checked as statuses.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `scripts/contract.sh` rows 2, 3, 5 |
| 2 — something selects it | `main()` calls `os.Exit(exitCode(err))`; deleting it makes every failure exit 0 and turns contract rows 1, 3, 4, 5, 6 red |
| 3 — the caller can discover it | README's exit-status table and SKILL.md's copy; `mrw --help` per subcommand |
| 4 — it is used | CI runs `scripts/contract.sh`, which branches on all four; nothing else measures caller behaviour |

## Mutation Log

- 2026-08-31 · f679354* · mutant survived · exit 0 · `cmd/mrw/main.go` · Changes the message the nothing-written path emits. contract.sh row 1 asserts exit 1 with the offender named, so this proves the fence binds to the status path rather than merely to the exit code. · acceptance-sha256:bc92ab8726f1e3d0075c104d37cb175ea715f9d7d4f25811782f71781ffb7a51
  ```
  the fence passed with the mechanism broken
  ```
- 2026-08-31 · f679354* · mutant killed · exit 1 · `cmd/mrw/main.go` · Collapses "nothing written" into success — the mechanism this task IS. The previous mutant on this task mutated only a message string and survived, which was a badly chosen mutant rather than a decoration test; this one targets the status mapping itself. · acceptance-sha256:bc92ab8726f1e3d0075c104d37cb175ea715f9d7d4f25811782f71781ffb7a51

## Invariants

- 1 promises an untouched tree. If anything was written, the status is not 1.
- A check is never run after a failed or dry run.
- A check that could not run is 2, never 3 — nothing ran, so nothing failed.

## Risks

- **A survived mutant is recorded above, deliberately left in.** Changing the
  wording of the "nothing was written" message does NOT turn the fence red:
  `scripts/contract.sh` row 1 asserts the exit status and greps for the named
  offender, not for the summary line. So the message text is genuinely
  unasserted — a real, small hole, recorded rather than tidied away. The mutant
  was badly chosen (a message string is the trivial escape hatch the template
  warns about); the killed entry beneath it targets the status mapping, which is
  the mechanism this task actually is. Asserting exact message wording was
  considered and rejected: it would make every copy edit a test failure.
- The statuses are asserted only by a shell script, so a Go-level refactor that
  changes the mapping is caught by CI but not by `go test`. Accepted: the
  property genuinely lives at the process boundary.

## Stop Condition

Stop if a fifth status is proposed. Four exist because four distinct next moves
exist; a fifth needs a fifth answer to "what should the caller do differently",
and a vocabulary that grows without that is where vocabularies rot.

## Out of Scope

- Running or grading the check — that is T1.
- A machine-readable status beyond the exit code (the `--json` receipt already
  carries `check.exit_code`) (permanent: two sources for one fact is how they
  disagree)

## Verification Log
- 2026-08-31 · f679354* · exit 0 · `set -o pipefail …` · acceptance-sha256:bc92ab8726f1e3d0075c104d37cb175ea715f9d7d4f25811782f71781ffb7a51
