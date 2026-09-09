# Task ADR-037-T2: `mrw instructions` prints the CLI pamphlet

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw instructions` subcommand (T2)
**Consumes:** `guide.Shared()` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the subcommand being in rootCommand`, `stdout being CLI()`, `exit 0`, `AGENTS.md naming mrw instructions`, `contract §75 driving the built binary`

## Goal

Register `mrw instructions` so a caller with only the binary can print `guide.CLI()`, and prove
the built binary does it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `CLI()` = `Shared()` plus the pipe trap, exit 3, MSYS, and glob+address. |
| `internal/guide/guide_test.go` | edit | `TestCLIContainsSharedAndTheOperatorTraps`. |
| `cmd/mrw/main.go` | edit | `instructionsCmd` in `rootCommand().Commands` — the line that selects it. |
| `cmd/mrw/instructions_test.go` | add | `TestInstructionsCommandPrintsCLI` — exec or invoke the command, assert stdout and exit 0. |
| `AGENTS.md` | edit | `` `mrw instructions` `` so `TestEverySubcommandReachesTheAgentFacingGuide` stays green. |
| `scripts/contract.sh` | edit | §75: `$MRW instructions` exits 0, prints the trigger and "through a pipe", and `instructions nope` is a usage error. Reserve 75 with `grep -oE '^# [0-9]+\. ' scripts/contract.sh \| sort -k2,2n \| tail -1`. |

## Ordered Steps

1. [S1] Write `TestCLIContainsSharedAndTheOperatorTraps` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestInstructionsCommandPrintsCLI` and confirm it is RED: no such command, or stdout is not `guide.CLI()`. [proof: mutation]
3. [S3] Implement `guide.CLI()` and `instructionsCmd`; add it to `rootCommand().Commands`. [proof: mutation]
4. [S4] Add `` `mrw instructions` `` to AGENTS.md and confirm `TestEverySubcommandReachesTheAgentFacingGuide` is green. Deleting the `Commands` entry must fail this test — that is rung 2. [proof: acceptance]
5. [S5] Write §75 against a binary that does **not** yet have the command (or with the entry temporarily removed) and confirm it is RED, then rebuild and confirm it is GREEN. Pair the good case (`instructions` exits 0 and names the trigger) with a failing case (unknown extra argument is exit 2). [proof: mutation]
6. [S6] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 75\. ' scripts/contract.sh \
  && go test ./internal/guide/ ./cmd/mrw/ -count=1 -v \
    -run 'TestCLIContainsSharedAndTheOperatorTraps|TestInstructionsCommandPrintsCLI|TestEverySubcommandReachesTheAgentFacingGuide' 2>&1 | tee /tmp/adr037-t2.out \
  && grep -q '^--- PASS: TestCLIContainsSharedAndTheOperatorTraps' /tmp/adr037-t2.out \
  && grep -q '^--- PASS: TestInstructionsCommandPrintsCLI' /tmp/adr037-t2.out \
  && grep -q '^--- PASS: TestEverySubcommandReachesTheAgentFacingGuide' /tmp/adr037-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr037-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/guide/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCLIContainsSharedAndTheOperatorTraps` | `internal/guide/guide_test.go` | `CLI()` contains `Shared()` and the four CLI-only traps | — | S1, S3 |
| `TestInstructionsCommandPrintsCLI` | `cmd/mrw/instructions_test.go` | The command exits 0 and stdout equals `guide.CLI()` | — | S2, S3 |
| `TestEverySubcommandReachesTheAgentFacingGuide` | `cmd/mrw/agentsdoc_test.go` | AGENTS.md names `` `mrw instructions` `` — the existing #73 gate | — | S4 |
| `§75` | `scripts/contract.sh` | The built binary prints the trigger at exit 0, and an extra argument is exit 2 | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestInstructionsCommandPrintsCLI` |
| 2 — something selects it | the `Commands` entry; deleting it fails S4 and §75 |
| 3 — the caller can discover it | `mrw --help` lists it; AGENTS.md names `` `mrw instructions` `` |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log

- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `internal/guide/guide.go` · S1: CLI drops the pipe trap, so a pamphlet that only reprints Shared still fails TestCLIContainsSharedAndTheOperatorTraps · acceptance-sha256:19f347999444ee7b2257f0e96102b917b2770073db8ee9ccffa4d9e0a5cf8044 · covers:stdout being CLI()
- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: unregister instructions so the Commands table no longer selects it — TestInstructionsCommandPrintsCLI and the AGENTS.md gate both go red · acceptance-sha256:19f347999444ee7b2257f0e96102b917b2770073db8ee9ccffa4d9e0a5cf8044 · covers:the subcommand being in rootCommand

## Invariants

- Exit 0 on success; stdout is exactly `guide.CLI()`.
- No flags. Extra arguments are usage (exit 2), not a file to append to.
- `Shared()` is unchanged by this task except that `CLI()` calls it.

## Risks

- Wiring the command without AGENTS.md. Mitigation: S4 is the existing gate, inside the fence.

## Stop Condition

Stop if the command needs to read a file from this checkout to produce its output — the point is
that the binary is enough.

## Out of Scope

- README and the central skill (that's T3)
- Changing help text of other subcommands

## Verification Log
- 2026-09-09 · 85b7455* · exit 1 · `set -o pipefail …` · acceptance-sha256:19f347999444ee7b2257f0e96102b917b2770073db8ee9ccffa4d9e0a5cf8044 · ms:24
  ```
  ```
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:19f347999444ee7b2257f0e96102b917b2770073db8ee9ccffa4d9e0a5cf8044 · ms:611
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:19f347999444ee7b2257f0e96102b917b2770073db8ee9ccffa4d9e0a5cf8044 · ms:578
