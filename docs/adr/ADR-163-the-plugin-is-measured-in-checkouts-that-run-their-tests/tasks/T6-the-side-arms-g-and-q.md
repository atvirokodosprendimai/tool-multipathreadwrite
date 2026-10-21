# Task ADR-163-T6: the side arms G and Q, read against P

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configurations G and Q; `bench run --qh-binary FILE --qh-sums FILE` and `--stop-switch NAME=VALUE`; `run.json` and P-family receipts' `plugin.qh_sha256` and `plugin.stop_switch`; `bench export-paired --pair P,G` and `--pair P,Q`
**Consumes:** `--profile stage6`, the shims (T1); P's profile and plugin, W's configuration shape, `--max-turns` (T2)
**Data dependency:** hermetic tests with a stub plugin and a stub binary. G needs a quality-harness release that ships `qh-darwin-arm64` with `SHA256SUMS` (announced as 3.8.52); Q needs a plugin tag whose Stop hook has a documented switch for continuing a turn when no check can start (quality-harness ADR-120). Neither blocks A, P or W.
**Proof map:** v1
**Rests-on:** `G differs from P only in what runs the hooks`, `Q differs from P only in the Stop switch`, `the staged binary is the release's, by its published sha`, `a side pair never enters ADR-112's stop rule`

## Goal

Under `stage6`, two side arms run beside P, paired by task, and are read against P, never against A and never by ADR-112's scorer (ADR-163 Decision 9):
- **G:** P with the plugin's hooks run by its Go binary;
- **Q:** P with the Stop continuation switched off when no check can start.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["G"]` and `Configs["Q"]` built from P's; G stages the binary into the attempt's plugin data directory after checking it against `SHA256SUMS`; Q sets the switch; `plugin.qh_sha256` and `plugin.stop_switch` in `run.json` and the receipts |
| `internal/bench/profile.go` | edit | G and Q allowed only under `stage6`, and only beside P |
| `internal/bench/canary.go` | edit | G's hooks ran from the staged binary and its sha matches; Q's switch is set in Q and in no other configuration |
| `internal/bench/export.go` | edit | `--pair P,G` and `--pair P,Q`, paired by task like every pair |
| `internal/bench/sidearms_test.go` | add | the tests below |
| `cmd/bench/main.go` | edit | `G` and `Q` in `--configs`; `--qh-binary`, `--qh-sums`, `--stop-switch`; the usage text |
| `docs/bench/README.md` | edit | the side arms, what each differs in, and how they are read |

## Ordered Steps

1. [S1] Write the three failing tests and see them red. [proof: acceptance]
2. [S2] G: refuse a binary whose sha256 is not the `SHA256SUMS` line for `qh-darwin-arm64`; stage it where the plugin reads it (`CLAUDE_PLUGIN_DATA` for the attempt); record the sha. Q: set the switch named by `--stop-switch` for Q alone. Both refuse to run without P in the same block. [proof: mutation]
3. [S3] `export-paired --pair P,G` and `--pair P,Q`; the side pairs are absent from what `--pair A,P` and `--pair A,W` export, so ADR-112's scorer never reads them. [proof: mutation]
4. [S4] The canary's G and Q checks; the flags and `--configs` in `cmd/bench/main.go`; the usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationGIsPWithTheGoHooksBinary|TestConfigurationQIsPWithTheStopContinuationOff|TestTheSideArmsAreReadAgainstPAndNeverEnterTheStopRule)$' 2>&1) \
  && for t in TestConfigurationGIsPWithTheGoHooksBinary TestConfigurationQIsPWithTheStopContinuationOff TestTheSideArmsAreReadAgainstPAndNeverEnterTheStopRule; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--qh-binary' \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--stop-switch'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2); the rows below replace it, with their Steps, in the commit that adds the tests | none | — |

Planned tests, which this table names in the commit that adds them:

- TestConfigurationGIsPWithTheGoHooksBinary in internal/bench/sidearms_test.go: G's arguments, tools and plugin are P's; the binary is staged for G alone and its sha is in `run.json` and G's receipts; a binary not matching `SHA256SUMS`, or G without P, or G without `stage6`, is refused (steps S2)
- TestConfigurationQIsPWithTheStopContinuationOff in internal/bench/sidearms_test.go: Q's arguments, tools and plugin are P's; the switch is set for Q alone and recorded; Q without `--stop-switch`, without P, or without `stage6` is refused (steps S2)
- TestTheSideArmsAreReadAgainstPAndNeverEnterTheStopRule in internal/bench/sidearms_test.go: `--pair P,G` and `--pair P,Q` export paired rows; `--pair A,P` and `--pair A,W` exports carry no G or Q row; `--pair A,G` and `--pair A,Q` are refused (steps S3)

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--configs …,G,Q`, `--qh-binary`, `--stop-switch` in `cmd/bench/main.go` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's agent pilot and T5's blocks, from the first block where each arm's plugin release exists |

## Mutation Log
(empty until execute)

## Invariants

- A, P and W are byte-identical in arguments, tools and receipts with or without G and Q in the block.
- ADR-112's scorer never reads a G or Q row.

## Risks

- The plugin may expose no switch for Q, or ship no darwin binary for G; that arm then waits, and the record says so at the block it would have joined.

## Stop Condition

The fence exits 0; or quality-harness's release offers neither the binary nor the switch, which goes back to Zy and quality-harness.

## Out of Scope

- Reading G or Q against A (permanent: boundary: each differs from P in one thing, so P is its control)

## Verification Log
(empty until execute)
