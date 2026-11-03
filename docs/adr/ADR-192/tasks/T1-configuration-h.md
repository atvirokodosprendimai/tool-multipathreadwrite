# Task ADR-192-T1: configuration H and the stage7 profile

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration H, profile `stage7`, `run.json` `h_sentence_sha256`, receipt `host_reads`, `hook_runs` and `hook_stderr_bytes`, `<attempt>.hook.stderr` and `<attempt>.hook.fired`, the stage7 canary
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `H is B plus Read, the read hook and the sentence`, `A and B carry neither the hook nor the sentence`, `the canary needs a PostToolUse:Read event from H`, `the sentence cannot change between blocks`, `the bench suite still passes`, `the usage text names H`

## Goal

`bench run --profile stage7 --configs A,B,H` and `bench canary` under stage7 run a third configuration that is B
with the host Read, ADR-191's PostToolUse read hook and one fixed sentence after `mrw instructions`, and the canary
proves the hook fired (ADR-192 Decision 1).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["H"]` (`Bash, Glob, Read`, `Mrw: true`, `ReadHook: true`); `invocation`: the PostToolUse Read hook writing `hook.stderr` and `hook.fired`, the sentence after `mrw instructions`; copy both files beside the receipt before the scratch is removed (`:785`); receipt `host_reads`, `hook_runs`, `hook_stderr_bytes`; `run.json` `h_sentence_sha256`, and the block-2+ check (`:263-272`) refusing a changed sha; `mrw_version` recorded for any Mrw configuration, not only when B runs (`:255`); `checkCanary` (`:632-640`) admits H |
| `internal/bench/transcript.go` | edit | count `Read` tool calls, not only Bash inputs (`:92-105`) |
| `internal/bench/profile.go` | edit | `ProfileStage7`, stage5's profile admitting H; the sentence as a constant |
| `internal/bench/canary.go` | edit | under stage7 H's canary instruction asks for a Read of the task's file first; H's init tools equal `Bash, Glob, Read, Skill` and its hook events include `PostToolUse:Read`; A and B show neither (`:246-251`); one canary per model |
| `internal/bench/hook.go` | edit | under stage7 H's refusal says "read them with mrw read or the Read tool" (`:52`) |
| `internal/bench/look.go` | edit | block 1's sentence sha is the stage's (`:93`) |
| `cmd/bench/main.go` | edit | `stage7` in `--profile`, `H` in `--configs`, H in the canary's configurations under stage7 (`:259-272`), the usage text |
| `internal/bench/runner_test.go` | edit | the two runner tests |
| `internal/bench/canary_test.go` | edit | the canary test |
| `docs/bench/README.md` | edit | configuration H, profile stage7 |

## Ordered Steps

1. [S1] Write the three failing tests and see them red. [proof: acceptance]
2. [S2] `Configs["H"]` and its invocation: tools, the PostToolUse entry beside B's PreToolUse Bash hook, the prompt file `mrw instructions` then a newline then the sentence, byte for byte as ADR-192 Decision 1 gives it; H's refusal text. [proof: mutation]
3. [S3] The receipt's `host_reads`, `hook_runs`, `hook_stderr_bytes` and the kept files; `h_sentence_sha256` in `run.json`, refused when block 2 or later names another; `mrw_version` for any Mrw configuration. H outside stage7 is refused. [proof: mutation]
4. [S4] The canary: H's init tools, a `PostToolUse:Read` hook event in H's transcript, none in A's or B's; `bench canary --profile stage7` runs A, B and H. [proof: mutation]
5. [S5] Wire `stage7` and `H` in `cmd/bench/main.go` and its usage text; document them in `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationHIsBPlusReadAndTheReadHook|TestAAndBCarryNoReadHookInStage7|TestTheStage7CanaryRefusesAnHWithoutItsHook)$' 2>&1) \
  && for t in TestConfigurationHIsBPlusReadAndTheReadHook TestAAndBCarryNoReadHookInStage7 TestTheStage7CanaryRefusesAnHWithoutItsHook; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/bench/ ./cmd/bench/ -count=1 \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'stage7'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConfigurationHIsBPlusReadAndTheReadHook` | `internal/bench/runner_test.go` | H's `--tools`; its PostToolUse Read hook writing `hook.stderr` and `hook.fired`; the prompt ending in the sentence; H's refusal text; `h_sentence_sha256` in `run.json` and a block 2 naming another sha refused; `host_reads` 2 and `hook_runs` 2 from a transcript and fired file with two Reads; both files kept beside the receipt; `mrw_version` recorded for H; H outside stage7 refused | none | S2, S3 |
| `TestAAndBCarryNoReadHookInStage7` | `internal/bench/runner_test.go` | under stage7, A's and B's arguments, tools, prompt, settings and refusal text equal stage5's | none | S2 |
| `TestTheStage7CanaryRefusesAnHWithoutItsHook` | `internal/bench/canary_test.go` | an H transcript without `Read` in its init tools, or with no `PostToolUse:Read` event, fails the canary; a B with that event fails it; the stage7 canary's configurations are A, B and H | none | S4 |
| `TestStage7RunsOnlyAThroughHAndHNeedsBInBlockOne` | `internal/bench/runner_test.go` | under stage7 every list is a subset of A, B and H, and a model that runs H runs B in block 1 or a stage without blocks (A,H and A,B,H,P refused); a later block may drop B | none | S3 |
| `TestRunCanaryFailsAnHWhoseReadHookSpoke` | `internal/bench/runner_test.go` | `RunCanary` records, for H alone, a problem when its hook wrote to standard error | none | S4 |
| `TestTheReadHookMustHaveRunQuietlyForEveryRead` | `internal/bench/canary_test.go` | an H receipt with hook standard error bytes, or fewer hook runs than host Reads, is a canary problem; a quiet hook that ran for every Read, and B, are not | none | S4 |
| `TestTheCanaryRunsHOnlyUnderStage7` | `cmd/bench/main_test.go` | the canary command's configurations are A and B, with H under stage7, then P and W | none | S4, S5 |
| `TestBenchHookReadToolFlagNamesTheReadTool` | `cmd/bench/main_test.go` | `bench hook` refuses `rm x` naming the Read tool with `--read-tool` and not without | none | S2, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | `Configs["H"]` read by `--configs` and by the canary's list; deleting either fails a named test |
| 3 — the caller can discover it | the usage text names `stage7`, the fence's last clause |
| 4 — it is used | T3's canary and pilot, T4's blocks |

## Mutation Log
- 2026-11-03 · 3d8c806f* · mutant killed · exit 1 · `internal/bench/runner.go` · S2: H's PostToolUse hook is attached to Glob, not Read; the settings test must see no hook on Read · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca
- 2026-11-03 · 3d8c806f* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: mrw_version recorded only when B runs; H alone in a stage with no blocks must still name its mrw · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca
- 2026-11-03 · 3d8c806f* · mutant killed · exit 1 · `internal/bench/canary.go` · S4: the canary no longer demands a PostToolUse:Read event of H · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca
- 2026-11-03 · 551c2544* · mutant killed · exit 1 · `internal/bench/look.go` · S3: the look no longer refuses a block that ran another H sentence than block 1 · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca
- 2026-11-03 · 551c2544* · mutant killed · exit 1 · `cmd/bench/main.go` · S5: the canary command takes A and B whatever the profile, never H · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca
- 2026-11-03 · 551c2544* · mutant killed · exit 1 · `cmd/bench/main.go` · S5: bench hook --read-tool is dead, H refusals never name the Read tool · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca

## Invariants

- Without `--profile stage7`, every configuration's invocation and refusal text are byte-identical to today's.
- B's prompt under stage7 is `mrw instructions` alone, as in Stage 5.

## Risks

- The tests stand in for Claude Code's init line, hook events and settings; T3's canary and preflight meet the real CLI.

## Stop Condition

The pinned Claude Code shows no `PostToolUse:Read` hook event for a Read under `--settings`: stop to Zy.

## Out of Scope

- The comparison against B in `bench look` (T2's job).

## Verification Log
- 2026-11-03 · 3d8c806f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:220023
- 2026-11-03 · 3d8c806f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:173942
- 2026-11-03 · 3d8c806f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:212936
- 2026-11-03 · 3d8c806f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:157516
- 2026-11-03 · no-git · exit 1 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:194 · test-lock-sha256:8dc043bee046c6849b570feabd83a53240edc37ccd0364900ff8ee8bcb8330de · test-lock-file:docs/adr/locks/8dc043bee046c6849b570feabd83a53240edc37ccd0364900ff8ee8bcb8330de.lock
  ```
  ```
- 2026-11-03 · 3d8c806f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:166665
- 2026-11-03 · 551c2544* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:154937
- 2026-11-03 · 551c2544* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:153381
- 2026-11-03 · 551c2544* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:156605
- 2026-11-03 · 551c2544* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:0 · test-lock-sha256:33c0214723d3b269beaf0a30e85f95e6af56ee4f5da5ea6348ddf759d1054d35 · test-lock-file:docs/adr/locks/33c0214723d3b269beaf0a30e85f95e6af56ee4f5da5ea6348ddf759d1054d35.lock · test-lock-kind:replace
- 2026-11-03 · 551c2544* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1c8cd1246b0dea18c82dd11cba066a3f599fb72f4944395e32617e385e12ca · ms:154470
