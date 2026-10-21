# Task ADR-163-T6: the side arms G and C, read against P

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:**
- configurations G and C;
- `bench run --qh-binary FILE --qh-sums FILE`, and C's `turnEnd` written into its `.quality-harness.json`;
- `run.json`'s `side_configs` and `side` (tag, asset sha, setting), and the same in the side receipts;
- the core export projection (`plugin` and `scaffold` from A, P and W only);
- `bench side --control P --arm G|C`.

**Consumes:** `--profile stage6`, the shims (T1); P's profile and plugin, W's configuration shape, the declared check file, `--max-turns` (T2)
**Data dependency:** hermetic tests with a stub plugin, a stub binary and fixture receipts. The arms themselves need, at the stage's one frozen plugin tag:
- C: the `turnEnd` setting (quality-harness 3.8.51 or later);
- G: a documented setting that selects the Go hook runtime, with `qh-darwin-arm64` in `SHA256SUMS`.
An arm not available at that tag is not run in this stage, and nothing here blocks A, P, W or T4.
**Proof map:** v1
**Rests-on:** `G differs from P only in what runs the hooks`, `C differs from P only in turnEnd`, `ADR-112's rows are byte-identical with or without side arms`, `the side reader uses P as control and the registered bar`

## Goal

Under `stage6`, two side arms run beside P at the stage's one plugin tag, paired by task. They are read against P by their own reader, and never reach ADR-112's rows or stop rule (ADR-163 Decision 9).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["G"]` and `Configs["C"]` built from P's. G pre-stages the verified binary and sets the tag's runtime setting, with the plugin's fetching off. Every plugin arm gets a fresh plugin data directory per attempt. C's `.quality-harness.json` gains `turnEnd`. `side_configs` and `side` go into `run.json` and the side receipts. |
| `internal/bench/materialize.go` | edit | C's check file is the declared one plus `turnEnd`; the other arms' files are unchanged |
| `internal/bench/profile.go` | edit | G and C only under `stage6`, only beside P, and only when the registration names their setting or asset |
| `internal/bench/canary.go` | edit | P's hooks ran under Node and G's under the staged binary; C's file carries `turnEnd` and no other arm's does |
| `internal/bench/export.go` | edit | `plugin` and `scaffold.configs` built from the core arms only; side arms never exported to the scorer |
| `internal/bench/analyze.go` | edit | `judgeWith`'s control becomes a parameter, A by default, so `bench side` reuses its intervals |
| `internal/bench/side.go` | add | `bench side`: P as control, the registered thresholds, the 60-pair floor, exclusions, descriptive-only below the floor |
| `internal/bench/verify.go` | edit | `bench verify` checks each side arm's attempts beside P's |
| `internal/bench/sidearms_test.go` | add | the tests below |
| `cmd/bench/main.go` | edit | `G` and `C` in `--configs`, `--qh-binary`, `--qh-sums`, the `side` subcommand, the usage text |
| `docs/bench/README.md` | edit | the side arms, what each differs in, and how they are read |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] G and C: G refuses a binary whose sha256 is not its `SHA256SUMS` line, and pre-stages it with fetching off. Every plugin arm gets an equivalent fresh plugin data directory. C's file carries `turnEnd` and nothing else extra. Both are refused without P in the block or without their registration. [proof: mutation]
3. [S3] The core export projection: `--pair A,P` and `--pair A,W` exports are byte-identical whether side arms ran or not, and the cross-block equality check compares the core projection. [proof: mutation]
4. [S4] `bench side`: P as control and the arm as the other side, paired by task, using `bench analyze`'s 99% bootstrap for cost and Tango's interval for solved. It applies the registered thresholds, and reads no verdict below 60 paired tasks per model. [proof: mutation]
5. [S5] The canary's G and C checks, `bench verify` for side arms, the flags and the subcommand in `cmd/bench/main.go`, the usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode|TestConfigurationCDiffersFromPOnlyInTurnEnd|TestCoreExportsAreIdenticalWithOrWithoutSideArms|TestTheSideReaderUsesPAsControlAndTheRegisteredBar)$' 2>&1) \
  && for t in TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode TestConfigurationCDiffersFromPOnlyInTurnEnd TestCoreExportsAreIdenticalWithOrWithoutSideArms TestTheSideReaderUsesPAsControlAndTheRegisteredBar; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--qh-binary' \
  && { go run ./cmd/bench side 2>&1 || true; } | grep -q -- '--control'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2); the rows below replace it, with their Steps, in the commit that adds the tests | none | — |

Planned tests, which this table names in the commit that adds them:

- **TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode** (internal/bench/sidearms_test.go, steps S2, S5):
  - G's arguments, hook set and plugin tag are P's, and only the runtime setting and the staged binary differ;
  - P's stub hooks record Node and G's record the binary;
  - every plugin arm has its own fresh data directory;
  - fetching is off;
  - a binary not matching `SHA256SUMS`, G without P, or G without its registration is refused.
- **TestConfigurationCDiffersFromPOnlyInTurnEnd** (internal/bench/sidearms_test.go, steps S2, S5):
  - C's tree differs from P's only in the `turnEnd` key of `.quality-harness.json`;
  - A, P and W files are unchanged;
  - C without P, or without its registration, is refused.
- **TestCoreExportsAreIdenticalWithOrWithoutSideArms** (internal/bench/sidearms_test.go, step S3):
  - the `--pair A,P` and `--pair A,W` export bytes of one fixture block are equal with side arms absent and present;
  - two blocks, one with side arms, pass the cross-block equality check;
  - `--pair A,G` and `--pair A,C` are refused.
- **TestTheSideReaderUsesPAsControlAndTheRegisteredBar** (internal/bench/sidearms_test.go, step S4):
  - fixture receipts give the registered verdict at each threshold boundary;
  - pairing drops an unpaired task and names it;
  - an exclusion is counted;
  - 59 paired tasks give descriptive only, and 60 give a verdict.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--configs …,G,C`, `--qh-binary` and `bench side` in `cmd/bench/main.go` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's agent pilot and T5's blocks, for each arm available at the frozen tag |

## Mutation Log
(empty until execute)

## Invariants

- ADR-112's rows, `configs` and `configs_for` are byte-identical with or without G and C.
- One plugin tag for every arm. A side arm never joins after the pilot.

## Risks

- The frozen tag may offer no Go-runtime setting, or a different `turnEnd` spelling. The arm then waits or follows the release notes, and the registration says which.
- At 60 pairs the solved interval is only just inside the 10-point margin. A real difference of a few points will read as no verdict, and that is reported.

## Stop Condition

The fence exits 0. An arm not available at the frozen tag is skipped and named in the registration; T4 proceeds without it.

## Out of Scope

- Reading G or C against A (permanent: boundary: each differs from P in one thing, so P is its control)
- G before quality-harness wires a Go hook launcher (external: quality-harness: ADR-116 defers the launcher until ADR-110's measurements)

## Verification Log
(empty until execute)
