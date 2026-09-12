# Task ADR-040-T3: `mrw version` prints `versionString()`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw version` subcommand (T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the subcommand being in rootCommand`, `stdout being versionString()`, `AGENTS.md naming mrw version`

## Goal

If Accept quotes the version fork: `mrw version` exits 0 and prints the same string `-v` already
prints, so a skill that taught the subcommand name is not lying on a *new* binary. Old PATH
binaries stay without it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `versionCmd` in `rootCommand().Commands` — the line that selects it. Reuse `versionString()`. |
| `cmd/mrw/version_test.go` | add | `TestVersionCommandPrintsVersionString` — exec or invoke; stdout equals `versionString()`; exit 0. |
| `AGENTS.md` | edit | `` `mrw version` `` so `TestEverySubcommandReachesTheAgentFacingGuide` stays green. Keep `-v`. |
| `scripts/contract.sh` | edit | a row (next free after whatever T1 reserved) driving `$MRW version` paired with `version nope` as exit 2. |

## Ordered Steps

1. [S1] Write `TestVersionCommandPrintsVersionString` and confirm it is RED: no such command, or stdout is not `versionString()`. [proof: mutation]
2. [S2] Implement `versionCmd`; add it to `rootCommand().Commands`. Extra arguments are usage (exit 2). [proof: mutation]
3. [S3] Add `` `mrw version` `` to AGENTS.md and confirm `TestEverySubcommandReachesTheAgentFacingGuide` is green. Deleting the `Commands` entry must fail this test — that is rung 2. Keep teaching `-v`. [proof: acceptance]
4. [S4] Drive the built binary: `version` exits 0 and equals `-v`; an extra argument is exit 2. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v \
  -run 'TestVersionCommandPrintsVersionString|TestEverySubcommandReachesTheAgentFacingGuide' 2>&1 | tee /tmp/adr040-t3.out \
  && grep -q '^--- PASS: TestVersionCommandPrintsVersionString' /tmp/adr040-t3.out \
  && grep -q '^--- PASS: TestEverySubcommandReachesTheAgentFacingGuide' /tmp/adr040-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr040-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestVersionCommandPrintsVersionString` | `cmd/mrw/version_test.go` | The command exits 0 and stdout equals `versionString()` | — | S1, S2 |
| `TestEverySubcommandReachesTheAgentFacingGuide` | `cmd/mrw/agentsdoc_test.go` | AGENTS.md names `` `mrw version` `` — the existing #73 gate | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestVersionCommandPrintsVersionString` |
| 2 — something selects it | the `Commands` entry; deleting it fails S3 |
| 3 — the caller can discover it | `mrw --help` lists it; AGENTS.md names `` `mrw version` `` and still names `-v` |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `cmd/mrw/main.go` · dropping versionCmd from Commands must fail T3 · acceptance-sha256:64334cc515bb92d6995aebaf2d70f89abb0d75eda2fc677d9528fe5ab827ebf8

## Invariants

- `-v` / `--version` keep working and print the same string.
- Extra arguments are usage (exit 2), not a file.
- Accept quoted the version fork (*"good, accepted all"*).
- Do not claim a pre-existing PATH binary grew the command.

## Risks

- Wiring the command without AGENTS.md. Mitigation: S3 is the existing gate, inside the fence.
- Teaching only `mrw version` so the next old-binary session repeats the field report. Mitigation: keep `-v` in AGENTS.md.

## Stop Condition

Stop if the command needs to read a file from this checkout to produce its output.

## Out of Scope

- Teach (that's T1)
- Parse (that's T2)
- A `mrw version --json` or build-time schema

## Verification Log
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:64334cc515bb92d6995aebaf2d70f89abb0d75eda2fc677d9528fe5ab827ebf8 · ms:460
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:64334cc515bb92d6995aebaf2d70f89abb0d75eda2fc677d9528fe5ab827ebf8 · ms:454
