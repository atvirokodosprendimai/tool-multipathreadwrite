# Task ADR-141-T1: `--core` prints the eight rules, held to the binary and the full text

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `guide.Core`, `mrw instructions --core`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `instructions --core prints eight short rules that name only real flags and that the full contract also carries`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `Core()`; the one sentence at the start of the full contract |
| `cmd/mrw/main.go` | edit | the `--core` flag on `instructions` |
| `internal/guide/core141_test.go`, `cmd/mrw/core141_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §242 |
| `AGENTS.md` | edit | the `mrw instructions` bullet |

## Ordered Steps

1. [S1] Write `TestTheCoreIsEightShortRulesTheFullContractStillCarries` (eight numbered rules, under 300 words, last line names the full form, and each rule's marker phrase is also in `CLI()`), and in `cmd/mrw` `TestTheCoreNamesOnlyFlagsTheBinaryHas` (every `--flag` in the core is a flag of `read`, `write` or the root command) and `TestInstructionsCoreFlagPrintsTheCore` (`instructions --core` prints `Core()`; plain `instructions` prints `CLI()`; `--core` with an argument is exit 2). Confirm RED.
2. [S2] `Core()`, the flag, the start sentence. Mutants: the core names a flag the binary lacks; the flag prints the full text. [proof: mutation]
3. [S3] Contract §242 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/guide/ -count=1 -timeout 300s -run 'TestTheCoreIsEightShortRulesTheFullContractStillCarries' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCoreIsEightShortRulesTheFullContractStillCarries \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheCoreNamesOnlyFlagsTheBinaryHas|TestInstructionsCoreFlagPrintsTheCore|TestInstructionsCommandPrintsCLI' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesOnlyFlagsTheBinaryHas \(' "$out" \
  && grep -qE '^--- PASS: TestInstructionsCoreFlagPrintsTheCore \(' "$out" \
  && go test ./internal/guide/ -count=1 -timeout 900s \
  && grep -q '^# 242\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCoreIsEightShortRulesTheFullContractStillCarries` | `internal/guide/core141_test.go` | eight numbered rules under 300 words, the full form named, each rule's marker also in `CLI()` | none | S1, S2 |
| `TestTheCoreNamesOnlyFlagsTheBinaryHas` | `cmd/mrw/core141_test.go` | every flag the core names exists on `read`, `write` or the root | none | S1, S2 |
| `TestInstructionsCoreFlagPrintsTheCore` | `cmd/mrw/core141_test.go` | `--core` prints the core, plain prints the full contract, an argument is exit 2 | none | S1, S2 |
| `TestInstructionsCommandPrintsCLI` | `cmd/mrw/instructions_test.go` | the plain command is still exactly `guide.CLI()` | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `guide.Core()` |
| 2 — something selects it | the `--core` flag of `instructions` |
| 3 — the caller can discover it | the first line of `mrw instructions`; `mrw instructions --help`; AGENTS.md |
| 4 — it is used | the 2026-10-09 survey; no telemetry (ADR-009) |

## Invariants

- `mrw instructions` without the flag is the full contract plus one sentence.
- The MCP handshake document is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the eight rules cannot stay under 300 words without dropping one the survey shows costing turns: the record would then name nine.

## Out of Scope

- Shortening the full text or the prose copies (permanent: boundary: the record's Out of Scope)

## Mutation Log

## Verification Log
