# Task ADR-192-T1: configuration H and the stage7 profile

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration H, profile `stage7`, `run.json` `h_sentence_sha256`, receipt `host_reads` and `hook_stderr_bytes`, the stage7 canary
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `H is B plus Read, the read hook and the sentence`, `A and B carry neither the hook nor the sentence`, `the canary refuses an H without its hook`, `the bench suite still passes`, `the usage text names H`

## Goal

`bench run --profile stage7 --configs A,B,H` runs a third configuration that is B with the host Read, ADR-191's
PostToolUse read hook and one fixed sentence after `mrw instructions`, and the canary proves it (ADR-192 Decision 1).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["H"]` (`Bash, Glob, Read`, `Mrw: true`, `ReadHook: true`); `invocation`: the PostToolUse Read hook with stderr to the scratch's `hook.stderr`, the sentence appended after `mrw instructions`; receipt `host_reads` (from the transcript's `tool_use` names) and `hook_stderr_bytes`; `run.json` `h_sentence_sha256` |
| `internal/bench/profile.go` | edit | `ProfileStage7`: stage5's profile, the one that admits H; the sentence as a constant |
| `internal/bench/canary.go` | edit | H's init tools equal `Bash, Glob, Read, Skill`; its settings carry the Read hook; A and B carry neither |
| `internal/bench/runner_test.go` | edit | H's invocation; A's and B's unchanged under stage7 |
| `internal/bench/canary_test.go` | edit | the canary refuses an H without its hook |
| `cmd/bench/main.go` | edit | `stage7` in `--profile`, `H` in `--configs`, the usage text (what selects them) |
| `docs/bench/README.md` | edit | configuration H, profile stage7 |

## Ordered Steps

1. [S1] Write the three failing tests and see them red. [proof: acceptance]
2. [S2] `Configs["H"]` and its invocation: tools `Bash, Glob, Read` plus the profile's `Skill`; the PostToolUse entry, matcher `Read`, command `<mrw> hook read 2>> <scratch>/hook.stderr`, beside B's PreToolUse Bash hook; the prompt file is `mrw instructions` then a newline then the sentence, byte for byte as ADR-192 Decision 1 gives it. [proof: mutation]
3. [S3] H's receipt carries `host_reads` (the count of `Read` tool calls) and `hook_stderr_bytes`; `run.json` carries `h_sentence_sha256` under stage7. H outside stage7 is refused, exit 2. [proof: mutation]
4. [S4] The canary: H's init tools equal `Bash, Glob, Read, Skill` and its settings carry the hook; an A or B with either is refused. [proof: mutation]
5. [S5] Wire `stage7` and `H` in `cmd/bench/main.go` and its usage text; document them in `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationHIsBPlusReadAndTheReadHook|TestAAndBCarryNoReadHookInStage7|TestTheStage7CanaryRefusesAnHWithoutItsHook)$' 2>&1) \
  && for t in TestConfigurationHIsBPlusReadAndTheReadHook TestAAndBCarryNoReadHookInStage7 TestTheStage7CanaryRefusesAnHWithoutItsHook; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/bench/ -count=1 \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'stage7'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConfigurationHIsBPlusReadAndTheReadHook` | `internal/bench/runner_test.go` | H's `--tools`, its PostToolUse Read hook writing to `hook.stderr`, the prompt file ending in the sentence, `h_sentence_sha256` in `run.json`, `host_reads` counted from a transcript with two Reads; H outside stage7 refused | none | S2, S3 |
| `TestAAndBCarryNoReadHookInStage7` | `internal/bench/runner_test.go` | under stage7, A's and B's arguments, tools, prompt and settings equal stage5's | none | S2 |
| `TestTheStage7CanaryRefusesAnHWithoutItsHook` | `internal/bench/canary_test.go` | an H init line without `Read`, or settings without the hook, fails the canary; a B with the hook fails it | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | `Configs["H"]` read by `--configs`; deleting the entry fails `TestConfigurationHIsBPlusReadAndTheReadHook` |
| 3 — the caller can discover it | the usage text names `stage7`, checked by the fence's last clause |
| 4 — it is used | T3's pilot and T4's blocks |

## Mutation Log

## Invariants

- Without `--profile stage7`, every configuration's invocation is byte-identical to today's.
- B's prompt under stage7 is `mrw instructions` alone, as in Stage 5.

## Risks

- The test doubles stand in for Claude Code's init line and settings; T3's pilot meets the real CLI.

## Stop Condition

The pinned Claude Code does not run a PostToolUse hook for `Read` under `--settings`: stop to Zy.

## Out of Scope

- The comparison against B in `bench look` (T2's job).

## Verification Log
