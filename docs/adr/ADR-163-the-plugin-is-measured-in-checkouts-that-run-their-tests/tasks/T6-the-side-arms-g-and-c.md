# Task ADR-163-T6: the side arms G and C, read against P

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:**
- configurations G and C;
- `bench run --qh-binary FILE --qh-sums FILE`, and C's `turnEnd` written into its `.quality-harness.json`;
- `run.json`'s `side_admitted`, `side_configs_for` and `side` (tag, asset sha, setting), and the same in the side receipts;
- the core export projection (`plugin` and `scaffold` from A, P and W only);
- `bench side --control P --arm G|C`.

**Consumes:** `--profile stage6`, the shims (T1); P's profile and plugin, W's configuration shape, the declared check file, `--max-turns` (T2)
**Data dependency:** hermetic tests with a stub plugin, a stub binary and fixture receipts. The arms themselves need, at the stage's one frozen plugin tag:
- C: the `turnEnd: notify` setting, or the frozen tag's spelling of it;
- G: a documented setting that selects the Go hook runtime, with `qh-darwin-arm64` in `SHA256SUMS`.
An arm not available at that tag is not run in this stage, and nothing here blocks A, P, W or T4.
**Proof map:** v1
**Rests-on:** `G differs from P only in what runs the hooks`, `C differs from P only in turnEnd`, `ADR-112's rows are byte-identical with or without side arms`, `the side reader uses P as control and the registered bar`

## Goal

Under `stage6`, two side arms run beside P at the stage's one plugin tag, paired by task. They are read against P by their own reader, and never reach ADR-112's rows or stop rule (ADR-163 Decision 9).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["G"]` and `Configs["C"]` built from P's. G pre-stages the verified binary and sets the tag's runtime setting, with the plugin's fetching off. Every plugin arm gets a fresh plugin data directory per attempt. C's `.quality-harness.json` gains `turnEnd`. `side_admitted`, `side_configs_for` and `side` go into `run.json` and the side receipts. |
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
| `TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode` | `internal/bench/sidearms_test.go` | G's arguments, settings and environment are P's plus the registered runtime setting and the fetch switch, its data directory holds the verified binary alone; P's stub hooks record Node and G's the staged binary; every plugin arm's data directory is fresh; a binary that is not its SHA256SUMS line, missing from it, malformed sums, a wrong registered sha, G without P, without its registration or outside stage6 are refused | none | S2, S5 |
| `TestConfigurationCDiffersFromPOnlyInTurnEnd` | `internal/bench/sidearms_test.go` | C's tree differs from P's in the turnEnd key of `.quality-harness.json` alone, in the working tree and the base commit; A, P, W and G trees are unchanged; arguments, settings and environment are P's; the canary's tree check; C without P, without its registration or outside stage6 is refused | none | S2, S5 |
| `TestCoreExportsAreIdenticalWithOrWithoutSideArms` | `internal/bench/sidearms_test.go` | a stage run with G and C exports `--pair A,P` and `--pair A,W` byte for byte as one run without them, with a canary that names them or not; run.json keeps configs and configs_for and has the side fields; side receipts are in side/; mixed blocks export as one; `--pair` with a side arm is refused; a side arm cannot join or leave after block 1 | none | S3 |
| `TestTheSideReaderUsesPAsControlAndTheRegisteredBar` | `internal/bench/sidearms_test.go` | the arm is read against P, not A, paired by task; the thresholds hold at each boundary on both sides; 59 paired tasks are descriptive only and 60 give a verdict; an unpaired task is named, an exclusion counted, a pilot never counted; each model has its own count | none | S4 |

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
- 2026-10-23 · e1ef1b00* · mutant survived · exit 0 · `internal/bench/sidearms.go` · a binary whose sha256 is not its SHA256SUMS line is staged · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a binary whose sha256 is not the registered one is staged · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · an asset listed twice in SHA256SUMS is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a registration for another plugin tag is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · G without a runtime setting is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · G's plugin may fetch · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant survived · exit 0 · `internal/bench/sidearms.go` · a side arm runs outside stage6 · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a side arm runs without P · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a side arm runs without its registration · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · plugin arms share a data directory · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · C's tree lacks turnEnd · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · every arm's tree gets turnEnd · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · side arms reach run.json's configs · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/export.go` · side arms reach the exported scaffold · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/export.go` · --pair A,G is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant survived · exit 0 · `internal/bench/sidearms.go` · a side arm joins or leaves after block 1 · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · 60 pairs give no verdict · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · 59 pairs give a verdict · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · a cost bound exactly at the bar passes · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · a solved bound exactly at -0.10 passes · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · the side reader compares against A, not P · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · pilot attempts count toward a side verdict · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a registration with an unknown key is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/export.go` · the exported canary carries side keys · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a binary whose sha256 is not its SHA256SUMS line is staged · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a side arm runs outside stage6 · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a side arm joins or leaves after block 1 · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · the plugin data directory is not the one the CLI derives · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a plugin attempt starts in the data directory an earlier attempt left · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · an earlier attempt's plugin data is deleted, not kept · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · P's plugin data directory is not fresh · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · a canary for another plugin data directory is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/canary.go` · a canary whose plugin wrote nothing in its data directory passes · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/canary.go` · the canary does not record the plugin data directory · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/runner.go` · a canary for another side registration is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a canary for another side asset is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a canary for another side registration body is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/export.go` · the exported core canary carries side and plugin-data keys · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/canary.go` · a failed hook's marker counts as G's binary running · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/canary.go` · a marker in a hook name, id or stderr counts as the binary running · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant survived · exit 0 · `internal/bench/side.go` · a block that did not run the arm is read for it · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · blocks with another side registration are pooled · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · blocks with another plugin are pooled · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · a pilot block at another tag stops the side reader · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/sidearms.go` · a registration with trailing content is accepted · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8
- 2026-10-23 · e1ef1b00* · mutant killed · exit 1 · `internal/bench/side.go` · a block that did not run the arm is read for it · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8

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
- 2026-10-23 · e1ef1b00* · exit 1 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:286 · test-lock-sha256:108d201fcbaef6b0115c340481f1660dd4fb2b5b4cb14ffbdeec737e7e00d21e · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlHIGlzIFAgcGx1cyB0aGUgcnVudGltZSBzZXR0aW5nLCB0aGUgZmV0Y2ggc3dpdGNoIGFuZCB0aGUgYmluYXJ5CTYxYThlZTc5YmJiNWU2YjFhNDc4NmIyYWI5YmZiMGIyODNiZGE4ZDAzN2QwYjdlYTMwZjQ5ZTNlNzBmMzc2NzYKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCVAgaXMgdGhlIGNvbnRyb2wsIG5ldmVyIEEJMGU1OTQzMTYzODg4ZDc2NTgxZTY3NmIyMjI2MWUwYzU4N2Y2OGIwMTI0ODA5MjI0ZGU3MjM0M2E0YjM3NWE0ZQpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25DRGlmZmVyc0Zyb21QT25seUluVHVybkVuZAkwZGUyOGIzNjNmODcxZDlhNjlmOGNmZDlmN2I5ODAxMjdmNTc3Mzk5ZmRkOTE0YmNiNTczZDk0MTkyNjA0OGY4CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlUZXN0Q29uZmlndXJhdGlvbkdSdW5zVGhlSG9va3NUaHJvdWdoVGhlQmluYXJ5QW5kUFRocm91Z2hOb2RlCTMwZTNkNDZmNzkxYjc0ZmVhNjJiMWM4NjBkODVmYjE1NmNlMzk5NjJhYTc4MGFiMzMzZDQ1ODg5Nzc2ZWU5OTIKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCVRlc3RDb3JlRXhwb3J0c0FyZUlkZW50aWNhbFdpdGhPcldpdGhvdXRTaWRlQXJtcwkyNGZhZmIyMDI1OTVhNjMwMzJmZmE4MTM1MTU3N2VhMmI1OWZiNDQ4ZWZmMTRiYWFlYzRmMzQ1ZjQ2YmRkZGM5CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlUZXN0VGhlU2lkZVJlYWRlclVzZXNQQXNDb250cm9sQW5kVGhlUmVnaXN0ZXJlZEJhcgkxYjNlMTQxN2ZhZDQ5NTg3ODZmM2QyYTViZmIwNjVjNDk1M2RkZmVlOTM4YmEyZmQ5MzI5YjU0ZDRkYjc0N2UzCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlhIHBpbG90IGlzIG5ldmVyIGNvdW50ZWQJZmVmYzg4YmM1YmMwNmFlYjlmM2YyODY0N2NlMzE1MmM4NzEwYmUzYTQ0YzAwZGFmZmYyYjc5YWI2ZTkyMjA3MQpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JYSBzaWRlIGFybSBjYW5ub3Qgam9pbiBvciBsZWF2ZSBhZnRlciBibG9jayAxCTNiMTYxYThhMTk5OWQ2NGY1ODAzZDM2ODMyZjg0NjY1MTUxMDZmZTVhYzI4ODY1OGE4NTM0MDg1YzBmNWIwYTEKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCWEgc2lkZSBhcm0gaXMgbm90IGEgcGFpcgk3ODJmMzBjYzgxZjlhNDliNzVmYWE1ZGQxNzdlM2IyNTNhYjZmMGNiZWMyNDk4Mjg1NWY3YWQ4OTEzNzUyYmM2CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlibG9ja3Mgd2l0aCBhbmQgd2l0aG91dCBzaWRlIGFybXMgZXhwb3J0IGFzIG9uZQk1OTE0YjgxYzVhOTAwZjkwOWYwN2RlNzA4YTI4OGZlYmM1ODlhN2EwYWE1YjZhZGY1NjUzNDgwNjNhNjc0OGY1CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwljb3N0IGF0IGVhY2ggYm91bmRhcnksIGVuZCB0byBlbmQJNDA2OWVlMDc3MTE3YjQ1NGM4MmIwMGJhNDMzMDE2MzhmYjNiODUxNTNjNzRiYTFjYTcwNzg3OGQ3ZmI3NDU0Mwpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JZWFjaCBtb2RlbCBpcyBpdHMgb3duIGNvdW50LCBhbmQgYmxvY2tzIGFyZSBjdW11bGF0ZWQJYWZjZmNmYjJmZDRhMWM1MDUyMzM1ZGYxZWZmZGNlYTYwNmMyZTg3ZWFlZDg1NzIzMzg4MDRjYjkxOWRiNzM4ZQpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JZXZlcnkgcGx1Z2luIGFybSBnZXRzIGEgZnJlc2ggZGF0YSBkaXJlY3RvcnkJNTA5ZDM5N2I3NWZiYTAyNTgzYjNjNWUzMTg3YzRjODBjZTMxZjYwZTY0MDgxYWI4ZWIyOGI4MTQwMDkwMmFmNApib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28Jbm8gcmVjZWlwdHMgb2YgdGhlIGFybSBpcyBhbiBlcnJvciwgbm90IGEgdmVyZGljdAkzMGM0MWZmNTAxMGQ5MzEwN2NhZTkyYTEzNDE5MmNiOGZkNGIxY2QwZDEwNmExNDhmYWE4YWYwZWYxMjA1OGFkCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlwYWlyaW5nLCBleGNsdXNpb25zIGFuZCB0aGUgNjAtcGFpciBmbG9vcgkwOWIzMWZmMzk3OTg5NmNlZWQ1NTAxNzZhNTQ3ZGQ5OTdjZGQ3ZDA5ZDAzMGI5YTc3MGRkZDA2MWNmZDM2MGNhCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlyZWZ1c2Fscwk2OGE2MTA1OTVhNTAxMDEzMjU4NTEwY2IwMDhiMmViM2Q2MTFhMjc3MWU2Yjk2MGU0MGNiZDhjODQ0NWViYzI4CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwl0aGUgYmFyIGF0IGVhY2ggYm91bmRhcnksIGJvdGggc2lkZXMJZTVkOGUzOWNiOWUwN2MyMGFiMWRmNzZkYWEyNGViZDMxYmI2MmZjYjE0YTRlN2Y3YWNiNzljYTI4ZTdhZDk3MApib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JdGhlIGNhbmFyeSByZWFkcyB0aGUgcnVudGltZSBmcm9tIHRoZSBob29rIG91dHB1dAkzNDAxNGEyNzE5MTQ5MmFjNzIwOTEzOWMwMDExZTZhYzE5MWRmNDFlNDViYzFjYjg0YjY2YjhiZjM5ZWI0ZDEw
  ```
  ```
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15061
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:18615
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15837
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:24012
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:22148
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16982
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:18031
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16176
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16904
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15384
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15854
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16002
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:37795
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:28260
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16470
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15821
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16510
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17372
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16353
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:18008
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15975
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17565
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15765
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17217
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16244
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:16042
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15378
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:15567
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17651
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17752
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17889
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17384
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17379
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17415
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17453
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17749
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17801
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:18417
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17490
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17432
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17666
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:18515
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17782
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17773
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17609
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17590
- 2026-10-23 · e1ef1b00* · exit 0 · `set -o pipefail …` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:17734
- 2026-10-23 · e1ef1b00* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:da9b71897e14322039d6a1892da124016aaaa2203e5138c50ba80fdc912400c8 · ms:0 · test-lock-sha256:91a64a119138dbfb5caf9f187c45118a469db48584a44ffe21f90afc49bcfce2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlHIGlzIFAgcGx1cyB0aGUgcnVudGltZSBzZXR0aW5nLCB0aGUgZmV0Y2ggc3dpdGNoIGFuZCB0aGUgYmluYXJ5CWMyZmZkODUzMmE0OTkwYTRkOGFkOWM1YjkzMGZkZTFiMzNmOGExZDdlZThjOWNlNzVkYTdiNzQ3NGRhMjE1MTcKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCVAgaXMgdGhlIGNvbnRyb2wsIG5ldmVyIEEJMGU1OTQzMTYzODg4ZDc2NTgxZTY3NmIyMjI2MWUwYzU4N2Y2OGIwMTI0ODA5MjI0ZGU3MjM0M2E0YjM3NWE0ZQpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25DRGlmZmVyc0Zyb21QT25seUluVHVybkVuZAlhOTQ3YzBmYzVjMmY0MGE5OTE1NGUzZDgyZjE0MDg5ZTA4ODAxNjZkYTIzODNmZWFhNzkyNTc3NzJiYjVlMjY3CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlUZXN0Q29uZmlndXJhdGlvbkdSdW5zVGhlSG9va3NUaHJvdWdoVGhlQmluYXJ5QW5kUFRocm91Z2hOb2RlCTBhMWRhOWFmN2UxZDY2Yjk1ZDNmZjNiZDEyZTRlODY1ODhkNDdhY2I1NzI2NDYyOWQ0MGU0NWM5ZTRmNzVmNTEKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCVRlc3RDb3JlRXhwb3J0c0FyZUlkZW50aWNhbFdpdGhPcldpdGhvdXRTaWRlQXJtcwkxMTA0ZWU3ODdmZWU4ZmU1YTI4YWIwZGIyMWI3OGZhNmE3M2ExYzllNzdiOGQ2NTg3NjhjMDFjMGI4NDlmYTM2CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlUZXN0VGhlU2lkZVJlYWRlclVzZXNQQXNDb250cm9sQW5kVGhlUmVnaXN0ZXJlZEJhcgllZjZjNmExODY3MDQ1Yjk3ZTEwZDcwMGQyOTJlMDZiZTY3M2JhY2E4OGJhOGI3NWExZWViNGQ5ZmMwODQzZWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlhIGNhbmFyeSBhdXRob3JpemVzIG9ubHkgdGhlIGRhdGEgZGlyZWN0b3J5IGFuZCBzaWRlIHJlZ2lzdHJhdGlvbiBpdCByYW4JMjYyN2NjZWZkMTQzYTI5Y2EwNzYyMzUxNjIwODQzZjhmODdlOWY0ZWZkZWQ0ZTIxNjY5ZjUzMDQwMTMyZjE0Ywpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JYSBwaWxvdCBpcyBuZXZlciBjb3VudGVkCTY5YzZiMWMxN2MzZmZhN2NjYzg0ZGQ1ZmRmZDdkOGRhOGUyYTNiMTc3YzM5YTA1OWM1M2E1OTkzYjYzZjhkYjkKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCWEgc2lkZSBhcm0gY2Fubm90IGpvaW4gb3IgbGVhdmUgYWZ0ZXIgYmxvY2sgMQk5YTU5Nzk0ZTFiZTZiN2Q0OTYxY2QxNjY0NGJjZjFlZmYxN2Q5NWZjNjlhNzkyNDAwM2Y2MTc0OGMwYmI3OTc3CmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlhIHNpZGUgYXJtIGlzIG5vdCBhIHBhaXIJNzgyZjMwY2M4MWY5YTQ5Yjc1ZmFhNWRkMTc3ZTNiMjUzYWI2ZjBjYmVjMjQ5ODI4NTVmN2FkODkxMzc1MmJjNgpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JYmxvY2tzIHRoYXQgcmFuIGFub3RoZXIgcGx1Z2luLCBhc3NldCBvciBzZXR0aW5nIGFyZSBub3QgcG9vbGVkCWI2ZjBiNjhhNzZmMDFkMzI5NzViZTZkOTBmYmE2MmY2YjA5Yzc0OWM2ODllMjdiYTU1NDVhZGY4NWE5MmNlYTMKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCWJsb2NrcyB3aXRoIGFuZCB3aXRob3V0IHNpZGUgYXJtcyBleHBvcnQgYXMgb25lCTU5MTRiODFjNWE5MDBmOTA5ZjA3ZGU3MDhhMjg4ZmViYzU4OWE3YTBhYTViNmFkZjU2NTM0ODA2M2E2NzQ4ZjUKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCWNvc3QgYXQgZWFjaCBib3VuZGFyeSwgZW5kIHRvIGVuZAk0MDY5ZWUwNzcxMTdiNDU0YzgyYjAwYmE0MzMwMTYzOGZiM2I4NTE1M2M3NGJhMWNhNzA3ODc4ZDdmYjc0NTQzCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwllYWNoIG1vZGVsIGlzIGl0cyBvd24gY291bnQsIGFuZCBibG9ja3MgYXJlIGN1bXVsYXRlZAlhZmNmY2ZiMmZkNGExYzUwNTIzMzVkZjFlZmZkY2VhNjA2YzJlODdlYWVkODU3MjMzODgwNGNiOTE5ZGI3MzhlCmJvZHkJaW50ZXJuYWwvYmVuY2gvc2lkZWFybXNfdGVzdC5nbwlldmVyeSBwbHVnaW4gYXJtIGdldHMgYSBmcmVzaCBkYXRhIGRpcmVjdG9yeSB3aGVyZSBDbGF1ZGUgQ29kZSBwdXRzIGl0CWEwMzRjMWE2M2I2MTVhMDI0MmI5OTc2MTc5Y2I4YzJhZTcwYmM3NDNkNjQ0MjJlMTllZjlhMjhmYTUyNTk1NzQKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCW5vIHJlY2VpcHRzIG9mIHRoZSBhcm0gaXMgYW4gZXJyb3IsIG5vdCBhIHZlcmRpY3QJMzBjNDFmZjUwMTBkOTMxMDdjYWU5MmExMzQxOTJjYjhmZDRiMWNkMGQxMDZhMTQ4ZmFhOGFmMGVmMTIwNThhZApib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JcGFpcmluZywgZXhjbHVzaW9ucyBhbmQgdGhlIDYwLXBhaXIgZmxvb3IJMDliMzFmZjM5Nzk4OTZjZWVkNTUwMTc2YTU0N2RkOTk3Y2RkN2QwOWQwMzBiOWE3NzBkZGQwNjFjZmQzNjBjYQpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JcmVmdXNhbHMJNmUwMjlmM2U0YTQ3ODNhMjkxMWQ2NmU1MTI5ODgxZWM2MjA1MDVmNDhmOWIwOWZmOWIwODQ2MTIwODlkZmU2Zgpib2R5CWludGVybmFsL2JlbmNoL3NpZGVhcm1zX3Rlc3QuZ28JdGhlIGJhciBhdCBlYWNoIGJvdW5kYXJ5LCBib3RoIHNpZGVzCWU1ZDhlMzljYjllMDdjMjBhYjFkZjc2ZGFhMjRlYmQzMWJiNjJmY2IxNGE0ZTdmN2FjYjc5Y2EyOGU3YWQ5NzAKYm9keQlpbnRlcm5hbC9iZW5jaC9zaWRlYXJtc190ZXN0LmdvCXRoZSBjYW5hcnkgYXR0ZXN0cyB0aGUgcnVudGltZSBvbmx5IGZyb20gdGhlIG91dHB1dCBvZiBhIGhvb2sgdGhhdCBzdWNjZWVkZWQJY2UxOTI2YjM3MmQ4ZjJmOTk1OGJlMTFmM2RmYTQxNzM0NmJlYTEwNmU2MTRkYzZlMjNhYmNhZjRiYzM0YzY1MA · test-lock-kind:replace
