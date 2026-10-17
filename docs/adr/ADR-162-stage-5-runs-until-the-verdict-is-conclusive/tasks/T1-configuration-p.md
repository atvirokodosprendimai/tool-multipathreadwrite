# Task ADR-162-T1: configuration P and the Stage 5 profile

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration P, `bench run --profile stage5 --plugin-dir --node --codex-stub`, the canary per configuration, the receipt's and `run.json`'s `plugin: {tag, commit}`
**Consumes:** none
**Data dependency:** hermetic, except S1's one uncounted canary probe on Sonnet (a real P transcript the canary rule is written against)
**Proof map:** v1
**Rests-on:** `P is A plus the plugin`, `the profile is the stage's, not the arms'`, `the canary proves engagement and nothing synced`, `no configuration reaches an external model`, `the usage text names --plugin-dir`

## Goal

`bench run --profile stage5 --configs A,B,P` runs a third configuration that is A with the quality-harness plugin loaded, inside a profile that every configuration and every block of the stage shares, and the canary proves P engaged and nothing else loaded (ADR-162 Decisions 1 and 2).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["P"]`; the Stage 5 profile in `RunOptions` and `run.json`; `invocation`: `--plugin-dir` for P, and under the profile for every configuration no `--disable-slash-commands`, `Skill` offered, `--node` linked on PATH, `CODEX_CLI_PATH` at the stub, `--include-hook-events`; `plugin: {tag, commit}` |
| `internal/bench/codexstub.go` | add | the stub: exits 2 saying external models are not part of this stage |
| `internal/bench/canary.go` | edit | `CheckCanary` per configuration: the init line's skills against the built-in set (plus `quality-harness:` in P), its plugins, the plugin's hook events, the markers allowed only inside them in P, the mrw check over agent text |
| `internal/bench/transcript.go` | edit | read the init line's skills and plugins, and hook events with their source |
| `internal/bench/runner_test.go` | edit | P's invocation; the profile with P absent from a block |
| `internal/bench/canary_test.go` | edit | the canary per configuration |
| `cmd/bench/main.go` | edit | `--profile`, `--plugin-dir`, `--node`, `--codex-stub`, `P`; the usage text (what selects them) |
| `docs/bench/README.md` | edit | configuration P and the profile |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. Then one uncounted canary probe, stage `probe`, Sonnet, P with the plugin at its latest tag: record its init line's skills and plugins, its hook events, and whether any account-synced skill or plugin from the config directory loaded; find the setting or layout that keeps those out of every configuration, or stop back to Zy. [proof: human: the probe's init line, hook events, and the synced entries' fate]
2. [S2] `Configs["P"]` is A's tools. The profile, set by `--profile stage5` and written in `run.json`, applies to every configuration a block runs: skills on, `Skill` offered, `--node` on PATH, `CODEX_CLI_PATH` at the stub and no `codex` on PATH, `--include-hook-events`. P adds `--plugin-dir DIR`; the plugin's tag and commit (`git -C DIR describe --tags --exact-match`, `rev-parse HEAD`; a dirty or untagged worktree refused) go into `run.json` and P's receipts. [proof: mutation]
3. [S3] `CheckCanary` takes the configuration and the built-in skill set S1 recorded.
   - A's and B's init skills equal that set, and their init line lists no plugin.
   - P's init skills are that set plus `quality-harness:` skills and nothing else, and its init line lists only that plugin.
   - P shows a hook event whose source is the plugin.
   - In P, `quality-harness:` is allowed in the validated init skills, in `Skill` calls to them, and in the plugin's hook events; "SessionStart hook" and "UserPromptSubmit hook" only in the plugin's hook events. Any other plugin or hook source is refused in every configuration.
   - The mrw check reads the agent's text and tool results.

   [proof: mutation]
4. [S4] Wire the flags and `P` in `cmd/bench/main.go` and its usage text; `docs/bench/README.md` names P, the profile and its limitation (no `Agent`, no external model). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationPIsAPlusThePluginAndSkills|TestTheProfileOutlivesP|TestTheCanaryRefusesAPluginThatIsNotEngaged|TestAAndBCarryNoPluginInAStageWithP)$' 2>&1) \
  && for t in TestConfigurationPIsAPlusThePluginAndSkills TestTheProfileOutlivesP TestTheCanaryRefusesAPluginThatIsNotEngaged TestAAndBCarryNoPluginInAStageWithP; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--plugin-dir'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2) | none | — |

Added by this task, and rowed here when they exist (`TestEveryTestNamedByARecordExists` checks every row):
- `TestConfigurationPIsAPlusThePluginAndSkills` (`internal/bench/runner_test.go`; S2): P's arguments are A's plus `--plugin-dir`; the receipt carries `{tag, commit}`; an untagged worktree is refused.
- `TestTheProfileOutlivesP` (`internal/bench/runner_test.go`; S2): a profiled block that runs only A and B keeps skills, `Skill`, `node` and the stub.
- `TestTheCanaryRefusesAPluginThatIsNotEngaged` (`internal/bench/canary_test.go`; S3): a P transcript with no plugin skill or no plugin hook event fails. So does a transcript of any configuration listing a skill outside the built-in set or another plugin. P with both passes.
- `TestAAndBCarryNoPluginInAStageWithP` (`internal/bench/runner_test.go`; S2, S3): A's and B's arguments hold no `--plugin-dir`, and a plugin marker in their transcript fails the canary.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--profile` and `--configs …,P` in `cmd/bench/main.go`; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's pilot and T5's stage |

## Mutation Log
(empty until execute)

## Invariants

- Without `--profile stage5`, A and B get exactly today's arguments, so Stages 1 to 4b stay reproducible.
- The plugin is read from a worktree and never written.

## Risks

- The headless init line may not attribute skills to a plugin; S1's probe settles the shape before S3's rule is written.
- Synced account entries may have no off switch; then S1 stops and Zy decides.

## Stop Condition

The fence exits 0 and S1's probe is recorded; or no layout keeps the synced entries out, or no headless transcript shows a plugin engaged, either of which goes back to Zy.

## Out of Scope

- Offering `Agent`, or allowing an external model, in any configuration (deferred: docs/adr/BACKLOG.md — "From ADR-162")

## Verification Log
(empty until execute)
