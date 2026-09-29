# Task ADR-092-T5: What the stress round of 2026-09-28 found

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `Config.StepCommands`, `check.StepDepth`, `check.MaxStepDepth`, `MRW_STEP_DEPTH`
**Consumes:** `--then`, `--then-sh`, the `then` receipt (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a plain write ignores a malformed steps block`, `a malformed steps block is refused when asked`, `a step runs one level deeper`, `the depth guard refuses past the limit`, `step names and commands are quoted`, `could not start is said once`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

Close what six peer sessions found stress-testing the release candidate (d8576c3) on 2026-09-28 and
M chose to fix before release: "Validate only when asked" and "Depth guard".

1. A malformed `steps` block refused EVERY write and check in the tree, `--then` or not
   (`internal/check/check.go:105`), against the record's Neutral line — one typo in a key a project
   may never use turned every write into exit 2 on upgrade (klientams-front-v2, infrastructure-7b).
2. A step that re-runs mrw with steps recursed without bound; each level reset the per-step timeout
   (playtrix-48: ~107 live processes before it killed them).
3. Control bytes in a step name reached the terminal raw on the `then` line and in the declared list,
   so a name could clear the screen or forge the list (infrastructure-7b).
4. "could not start: could not start: …" — the exit message and the `then` line both prefixed a
   `skipped` that already carried it (klientams-front-v2).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Config.Steps` held raw and read only by `StepCommands()`; `Load` no longer validates it; `run` takes extra environment; `RunSteps` passes `MRW_STEP_DEPTH` one deeper; `StepDepth`, `MaxStepDepth` |
| `internal/check/steps092t5_test.go` | add | the load and depth tests |
| `internal/check/steps092_test.go` | edit | T1's loader test renamed to what it now proves |
| `cmd/mrw/main.go` | edit | `resolveSteps` asks `StepCommands` only when a step is asked for; the depth guard refuses before anything is written; names and commands printed quoted when they hold control bytes; one "could not start" |
| `cmd/mrw/then092t5_test.go` | add | the four CLI tests |
| `internal/guide/guide.go` | edit | teaches the depth guard and what could_not_start means |
| `AGENTS.md`, `README.md` | edit | the same, plus the setsid and inherited-signal limits the round confirmed |
| `scripts/contract.sh` | edit | §174 drives the built binary |
| `docs/adr/BACKLOG.md` | edit | the round's lower findings, receipted |

## Ordered Steps

1. [S1] Write the six failing tests. [proof: mutation]
2. [S2] `Config.Steps json.RawMessage`; `StepCommands()` decodes and validates; `Load` stops calling it. [proof: mutation]
3. [S3] `run(…, env)`; `RunSteps` gives each step `MRW_STEP_DEPTH=StepDepth()+1`; the CLI refuses `--then`/`--then-sh` at `StepDepth() >= MaxStepDepth` (8), exit 2, before anything is written. [proof: mutation]
4. [S4] Names and commands with control bytes print quoted (`strconv.Quote`); the doubled prefix goes. [proof: mutation]
5. [S5] Teaching, BACKLOG, and contract §174: a plain write beside a malformed block exits 0 and a real recursion ends at the guard with nothing left running. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §174's rows printed]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAMalformedStepsBlockStillLoads|TestAStepRunsOneLevelDeeper|TestAPlainWriteIgnoresAMalformedStepsBlock|TestAStepNameWithControlBytesIsQuoted|TestACouldNotStartStepSaysSoOnce|TestTheStepDepthGuardRefusesADeepSequence' -v 2>&1 | tee /tmp/adr092-T5.out \
  && missing=$(for t in TestAMalformedStepsBlockStillLoads TestAStepRunsOneLevelDeeper TestAPlainWriteIgnoresAMalformedStepsBlock TestAStepNameWithControlBytesIsQuoted TestACouldNotStartStepSaysSoOnce TestTheStepDepthGuardRefusesADeepSequence; do grep -qE "^--- PASS: $t \(" /tmp/adr092-T5.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 174\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMalformedStepsBlockStillLoads` | `internal/check/steps092t5_test.go` | a name with a space, a number, an array and an empty command each still load; `StepCommands` refuses each | — | S1, S2 |
| `TestAStepRunsOneLevelDeeper` | `internal/check/steps092t5_test.go` | under `MRW_STEP_DEPTH=3` a step sees 4 | — | S3 |
| `TestAPlainWriteIgnoresAMalformedStepsBlock` | `cmd/mrw/then092t5_test.go` | a write and a check asking for no step exit 0 beside a malformed block; `--then` then refuses it, nothing written, naming the step | — | S2 |
| `TestAStepNameWithControlBytesIsQuoted` | `cmd/mrw/then092t5_test.go` | no raw ESC on the then lines or in the declared list; the name printed quoted | — | S4 |
| `TestACouldNotStartStepSaysSoOnce` | `cmd/mrw/then092t5_test.go` | no line says "could not start" twice | — | S4 |
| `TestTheStepDepthGuardRefusesADeepSequence` | `cmd/mrw/then092t5_test.go` | depth 8 refuses on write and check with nothing written; depth 7 runs; the instructions teach it | — | S3, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `StepCommands`, `StepDepth`, the quoting |
| 2 — something selects it | `resolveSteps`, `RunSteps`, `reportSteps`, `stepsExit`; §174 drives the binary into a real recursion |
| 3 — the caller can discover it | `MRW_STEP_DEPTH` in the instructions, AGENTS.md and README |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the stress round's six reports |

## Mutation Log
(empty until execute)
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · a write reads the steps block whether asked or not · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:a plain write ignores a malformed steps block
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · an asked-for malformed block is not refused as such · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:a malformed steps block is refused when asked
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/check/check.go` · a step runs at its caller's depth · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:a step runs one level deeper
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · the guard lets depth 8 run · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:the depth guard refuses past the limit
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · nothing is ever quoted · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:step names and commands are quoted
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · the exit message says could not start twice · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:could not start is said once
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:the tree is gofmt-clean
- 2026-09-28 · d8576c3* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · covers:no other engine package changes

## Invariants

- Without `--then`/`--then-sh`, a write and a check read no step, whatever the `steps` block holds.

## Risks

- A project relying on a malformed `steps` being caught at every write now learns only when it asks for a step. That is the Neutral line's promise; recorded in the amendment.

## Out of Scope

- The round's lower findings (signal name, duplicate keys, log size, dangling-symlink wording, a plan editing the harness, stats on a missing root) (deferred: docs/adr/BACKLOG.md — "From ADR-092")

## Stop Condition

The fence exits 0 and contract §174 passes.

## Verification Log
(empty until execute)
- 2026-09-28 · d8576c3* · exit 1 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:788 · test-lock-sha256:c3115dd3ab26edc22b465cd00f1aac7dd72d40314bc3bb70d550dc0eef306f00 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGhlbjA5MnQ1X3Rlc3QuZ28JVGVzdEFDb3VsZE5vdFN0YXJ0U3RlcFNheXNTb09uY2UJNjliNDMxYTgzMWY2OWJjODc3MjQ3NWMwOTQ0MWE2OWM2ODNmNjY2MDJiZGY3YjE4ZTc2MDJlODRhZjNjMWI2NQpib2R5CWNtZC9tcncvdGhlbjA5MnQ1X3Rlc3QuZ28JVGVzdEFQbGFpbldyaXRlSWdub3Jlc0FNYWxmb3JtZWRTdGVwc0Jsb2NrCTg3NWViNGFkOTY3ZTBmMDI5YzJlNGQ4MDhjYjdiNTY4MTc0MWIzNTNhNDcxZDdkYWU5ZmRlZWU3ZTEzYzdiY2UKYm9keQljbWQvbXJ3L3RoZW4wOTJ0NV90ZXN0LmdvCVRlc3RBU3RlcE5hbWVXaXRoQ29udHJvbEJ5dGVzSXNRdW90ZWQJOTViMjhhNmI4MGY4NzVkZmQ3Zjk4NDIyZjBmMmY1ZTFiNmNmNTJiNWQ5ZWM5NmNlNjE3OTM3YzA1NTNjNjg0Mwpib2R5CWNtZC9tcncvdGhlbjA5MnQ1X3Rlc3QuZ28JVGVzdFRoZVN0ZXBEZXB0aEd1YXJkUmVmdXNlc0FEZWVwU2VxdWVuY2UJM2IwMTY5ZDA1MmRlZjViODdkMWQ1MzdmODk1MGI3ZmFiZWQ2ZjEyNGQ2OGE5YWJlMmM0M2ZiYjNkYzRlMmM5Ngpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDkydDVfdGVzdC5nbwlUZXN0QU1hbGZvcm1lZFN0ZXBzQmxvY2tTdGlsbExvYWRzCTZhMWYyYzlkMWRjMWQzYmJjMTg0YTI4MmEzMGYwNmRlZjE1MWY4NDk1MWQxNmM3NjNhNGU1OGU4MjExN2VlMTgKYm9keQlpbnRlcm5hbC9jaGVjay9zdGVwczA5MnQ1X3Rlc3QuZ28JVGVzdEFTdGVwUnVuc09uZUxldmVsRGVlcGVyCWNmNGQ5NjNkOThlNTA5NTllNTQxZmZhYjQ0MzExYjBhMjhjODUyYmExMTU1YTNkNWUzMWU3ZGYzYzA0N2FkZWU
  ```
  --- last 10 line(s) of stdout (of 45 after folding 45 raw)
      then092t5_test.go:98: at depth 7: exit 0, log "x\na\nx\n":
          ok   a.go 2 replace  -1 +1
          wrote a.go  2L -> 2L  sha 86ab99a1
          1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
          then 1/1 --then-sh: echo x >> log — PASS
      then092t5_test.go:101: mrw instructions does not teach the depth guard
  --- FAIL: TestTheStepDepthGuardRefusesADeepSequence (0.03s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.292s
  FAIL
  ```
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:460
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:380
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:395
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:398
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:410
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:390
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:408
- 2026-09-28 · d8576c3* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b5d98d680757148479f0afe836569df0994ec7f5ac75623a708dfb23a3f6f39 · ms:400
- 2026-09-29 · human-observed · S5 observed: PR #269 records './scripts/contract.sh exits 0, including §174's rows'; contract.sh exit 0 on main db39d42 on 2026-09-29 (this session)
